package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/seenimo/stock_home/internal/source"
)

// Top Movers page: one cached snapshot per universe; sector and price filters and rankings are applied per request.

type moverRow struct {
	exchange, symbol, sector                    string
	price, change, pct, volume, relVol, hi, low *float64
}

var universes = map[string]string{
	"core": `instrument_type="EQ", tier="core"`,
	"all":  `instrument_type="EQ"`,
}

var priceBands = map[string][2]float64{
	"all":      {0, math.Inf(1)},
	"lt100":    {0, 100},
	"100-500":  {100, 500},
	"500-2000": {500, 2000},
	"gt2000":   {2000, math.Inf(1)},
}

var sectorPattern = regexp.MustCompile(`^[A-Za-z0-9 &.,()/'-]{1,40}$`)

const (
	scanTop      = 10
	scanRangeTop = 5
)

// RangeItem is an equity near its 52-week high or low.
type RangeItem struct {
	Quote
	High52          float64 `json:"high52"`
	Low52           float64 `json:"low52"`
	DistancePercent float64 `json:"distancePercent"`
	Position        float64 `json:"position"` // 0 at the 52-week low, 1 at the high.
}

// VolumeItem is an equity trading above its usual volume pace.
type VolumeItem struct {
	Quote
	RelativeVolume float64 `json:"relativeVolume"`
}

// Bin is one bucket of the day-change distribution; From/To are nil for open ends.
type Bin struct {
	Label string   `json:"label"`
	From  *float64 `json:"from"`
	To    *float64 `json:"to"`
	Count int      `json:"count"`
}

// SectorMovers ranks the best and worst equity within one sector.
type SectorMovers struct {
	Name          string  `json:"name"`
	ChangePercent float64 `json:"changePercent"`
	Count         int     `json:"count"`
	Advancers     int     `json:"advancers"`
	Decliners     int     `json:"decliners"`
	Gainer        *Quote  `json:"gainer"`
	Loser         *Quote  `json:"loser"`
}

// MoversScan is the Top Movers page payload.
type MoversScan struct {
	Universe     string         `json:"universe"`
	Sector       string         `json:"sector"`
	Price        string         `json:"price"`
	Total        int            `json:"total"`
	Advancers    int            `json:"advancers"`
	Decliners    int            `json:"decliners"`
	Unchanged    int            `json:"unchanged"`
	Gainers      []Quote        `json:"gainers"`
	Losers       []Quote        `json:"losers"`
	MostActive   []Quote        `json:"mostActive"`
	VolumeChange []VolumeItem   `json:"volumeChange"`
	High52       []RangeItem    `json:"high52"`
	Low52        []RangeItem    `json:"low52"`
	Distribution []Bin          `json:"distribution"`
	Sectors      []SectorMovers `json:"sectors"`
	SectorNames  []string       `json:"sectorNames"`
}

func (s *Server) moversScanHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	universe, price, sector := or(q.Get("universe"), "core"), or(q.Get("price"), "all"), strings.TrimSpace(q.Get("sector"))
	bad := func(msg string) { writeJSON(w, http.StatusBadRequest, envelope{Error: msg, UpdatedAt: s.now()}) }
	if _, ok := universes[universe]; !ok {
		bad("universe must be core or all")
		return
	}
	band, ok := priceBands[price]
	if !ok {
		bad("price must be one of all, lt100, 100-500, 500-2000, gt2000")
		return
	}
	if sector != "" && !sectorPattern.MatchString(sector) {
		bad("invalid sector")
		return
	}
	res, err := s.cache.Get("movers-scan:"+universe, 10*time.Second, 2*time.Minute, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.QueryTimeout)
		defer cancel()
		return s.moverRows(ctx, universes[universe])
	})
	if err != nil {
		s.errors.WithLabelValues("movers-scan").Inc()
		s.logger.Warn("Movers scan failed", "universe", universe, "err", err)
		writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
		return
	}
	scan := buildScan(res.Value.([]moverRow), universe, sector, price, band)
	// Sparklines only for the ranked lists; the symbol sets follow the filters, not free user input.
	syms := append(symbolsOf(scan.Gainers), symbolsOf(scan.Losers)...)
	if len(syms) > 0 {
		key := "movers-sparks:" + strings.Join(syms, ",")
		if sp, err := s.cache.Get(key, 30*time.Second, 5*time.Minute, func() (any, error) {
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.QueryTimeout)
			defer cancel()
			q := s.querier(ctx)
			out := sessionSparks(q, ", "+universes[universe], syms, s.now(), s.cfg.Location)
			return out, q.err
		}); err == nil {
			sparks := sp.Value.(map[string][]source.Point)
			for _, list := range [][]Quote{scan.Gainers, scan.Losers} {
				for i := range list {
					list[i].Spark = sparks[list[i].Symbol]
				}
			}
		} else {
			s.logger.Warn("Movers sparklines unavailable", "err", err)
		}
	}
	writeJSON(w, http.StatusOK, envelope{Data: scan, UpdatedAt: res.UpdatedAt, Stale: res.Stale})
}

func or(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func symbolsOf(qs []Quote) []string {
	out := make([]string, len(qs))
	for i, q := range qs {
		out[i] = q.Symbol
	}
	return out
}

func (s *Server) moverRows(ctx context.Context, sel string) (any, error) {
	q := s.querier(ctx)
	pct := q.vector(fmt.Sprintf(`max by (exchange, symbol, sector) (stock_change_percent{%s})`, sel))
	get := func(expr string) map[string]float64 {
		out := map[string]float64{}
		for _, smp := range q.vector(expr) {
			out[smp.Labels["exchange"]+":"+smp.Labels["symbol"]] = smp.Value
		}
		return out
	}
	prices := get(fmt.Sprintf(`max by (exchange, symbol) (stock_last_price{%s})`, sel))
	abs := get(absChangeQuery(sel))
	volume := get(fmt.Sprintf(`max by (exchange, symbol) (stock_day_volume_shares{%s})`, sel))
	relVol := get(fmt.Sprintf(`max by (exchange, symbol) (stock:relative_volume:ratio{%s})`, sel))
	hi := get(fmt.Sprintf(`max by (exchange, symbol) (stock_52w_high_price{%s} > 0)`, sel))
	low := get(fmt.Sprintf(`max by (exchange, symbol) (stock_52w_low_price{%s} > 0)`, sel))
	if q.err != nil {
		return nil, q.err
	}
	ptr := func(m map[string]float64, k string) *float64 {
		if v, ok := m[k]; ok {
			return &v
		}
		return nil
	}
	rows := make([]moverRow, 0, len(pct))
	for _, smp := range pct {
		k := smp.Labels["exchange"] + ":" + smp.Labels["symbol"]
		v := smp.Value
		rows = append(rows, moverRow{
			exchange: smp.Labels["exchange"], symbol: smp.Labels["symbol"], sector: smp.Labels["sector"], pct: &v,
			price: ptr(prices, k), change: ptr(abs, k), volume: ptr(volume, k), relVol: ptr(relVol, k), hi: ptr(hi, k), low: ptr(low, k),
		})
	}
	return rows, nil
}

func (m moverRow) quote() Quote {
	return Quote{Exchange: m.exchange, Symbol: m.symbol, Sector: m.sector, Price: m.price, Change: m.change, ChangePercent: m.pct,
		Volume: m.volume, ExploreURL: symbolExploreURL(m.exchange, m.symbol)}
}

var distributionEdges = []float64{-5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5}

func buildScan(all []moverRow, universe, sector, price string, band [2]float64) MoversScan {
	sc := MoversScan{Universe: universe, Sector: sector, Price: price, Gainers: []Quote{}, Losers: []Quote{}, MostActive: []Quote{},
		VolumeChange: []VolumeItem{}, High52: []RangeItem{}, Low52: []RangeItem{}, Sectors: []SectorMovers{}, SectorNames: []string{}}
	names := map[string]bool{}
	var rows []moverRow
	for _, r := range all {
		if r.sector != "" {
			names[r.sector] = true
		}
		if price != "all" && (r.price == nil || *r.price < band[0] || *r.price >= band[1]) {
			continue
		}
		if sector != "" && r.sector != sector {
			continue
		}
		rows = append(rows, r)
	}
	for n := range names {
		sc.SectorNames = append(sc.SectorNames, n)
	}
	sort.Strings(sc.SectorNames)
	sc.Total = len(rows)

	bins := make([]Bin, len(distributionEdges)+2)
	for i := range bins {
		bins[i] = binFor(i)
	}
	bySector := map[string]*SectorMovers{}
	sums := map[string]float64{}
	for _, r := range rows {
		p := *r.pct
		switch {
		case p > 0:
			sc.Advancers++
		case p < 0:
			sc.Decliners++
		default:
			sc.Unchanged++
		}
		bins[binIndex(p)].Count++
		name := or(r.sector, "unknown")
		sm := bySector[name]
		if sm == nil {
			sm = &SectorMovers{Name: name}
			bySector[name] = sm
		}
		sm.Count++
		switch {
		case p > 0:
			sm.Advancers++
		case p < 0:
			sm.Decliners++
		}
		sums[name] += p
		if q := r.quote(); sm.Gainer == nil || p > *sm.Gainer.ChangePercent {
			sm.Gainer = &q
		}
		if q := r.quote(); sm.Loser == nil || p < *sm.Loser.ChangePercent {
			sm.Loser = &q
		}
	}
	sc.Distribution = bins
	for name, sm := range bySector {
		sm.ChangePercent = sums[name] / float64(sm.Count)
		if *sm.Gainer.ChangePercent <= 0 {
			sm.Gainer = nil
		}
		if *sm.Loser.ChangePercent >= 0 {
			sm.Loser = nil
		}
		sc.Sectors = append(sc.Sectors, *sm)
	}
	sort.Slice(sc.Sectors, func(i, j int) bool { return sc.Sectors[i].ChangePercent > sc.Sectors[j].ChangePercent })

	ranked := func(keep func(moverRow) bool, less func(a, b moverRow) bool, n int) []moverRow {
		var out []moverRow
		for _, r := range rows {
			if keep(r) {
				out = append(out, r)
			}
		}
		sort.SliceStable(out, func(i, j int) bool { return less(out[i], out[j]) })
		if len(out) > n {
			out = out[:n]
		}
		return out
	}
	for _, r := range ranked(func(r moverRow) bool { return *r.pct > 0 }, func(a, b moverRow) bool { return *a.pct > *b.pct }, scanTop) {
		sc.Gainers = append(sc.Gainers, r.quote())
	}
	for _, r := range ranked(func(r moverRow) bool { return *r.pct < 0 }, func(a, b moverRow) bool { return *a.pct < *b.pct }, scanTop) {
		sc.Losers = append(sc.Losers, r.quote())
	}
	for _, r := range ranked(func(r moverRow) bool { return r.volume != nil && *r.volume > 0 }, func(a, b moverRow) bool { return *a.volume > *b.volume }, scanTop) {
		sc.MostActive = append(sc.MostActive, r.quote())
	}
	for _, r := range ranked(func(r moverRow) bool { return r.relVol != nil }, func(a, b moverRow) bool { return *a.relVol > *b.relVol }, scanRangeTop) {
		sc.VolumeChange = append(sc.VolumeChange, VolumeItem{Quote: r.quote(), RelativeVolume: *r.relVol})
	}
	hasRange := func(r moverRow) bool { return r.price != nil && r.hi != nil && r.low != nil && *r.hi > *r.low }
	rangeItem := func(r moverRow, ref float64) RangeItem {
		return RangeItem{Quote: r.quote(), High52: *r.hi, Low52: *r.low, DistancePercent: (*r.price/ref - 1) * 100,
			Position: math.Max(0, math.Min(1, (*r.price-*r.low)/(*r.hi-*r.low)))}
	}
	for _, r := range ranked(hasRange, func(a, b moverRow) bool { return *a.price / *a.hi > *b.price / *b.hi }, scanRangeTop) {
		sc.High52 = append(sc.High52, rangeItem(r, *r.hi))
	}
	for _, r := range ranked(hasRange, func(a, b moverRow) bool { return *a.price / *a.low < *b.price / *b.low }, scanRangeTop) {
		sc.Low52 = append(sc.Low52, rangeItem(r, *r.low))
	}
	return sc
}

// binIndex: bin 0 is below -5%, the middle bin holds exactly 0%, the last is 5% and above.
func binIndex(p float64) int {
	switch {
	case p == 0:
		return 6
	case p < 0:
		for i, e := range distributionEdges[:6] {
			if p < e {
				return i
			}
		}
		return 5
	default:
		for i, e := range distributionEdges[6:] {
			if p < e {
				return 7 + i
			}
		}
		return 12
	}
}

func binFor(i int) Bin {
	f := func(v float64) *float64 { return &v }
	switch {
	case i == 0:
		return Bin{Label: "< -5%", To: f(-5)}
	case i < 6:
		return Bin{Label: fmt.Sprintf("%g%% to %g%%", distributionEdges[i-1], distributionEdges[i]), From: f(distributionEdges[i-1]), To: f(distributionEdges[i])}
	case i == 6:
		return Bin{Label: "0%", From: f(0), To: f(0)}
	case i < 12:
		return Bin{Label: fmt.Sprintf("%g%% to %g%%", distributionEdges[i-2], distributionEdges[i-1]), From: f(distributionEdges[i-2]), To: f(distributionEdges[i-1])}
	default:
		return Bin{Label: "≥ 5%", From: f(5)}
	}
}
