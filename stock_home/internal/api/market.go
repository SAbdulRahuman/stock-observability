package api

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/seenimo/stock_home/internal/source"
)

// Quote is the price and day change of one instrument.
type Quote struct {
	Exchange      string   `json:"exchange"`
	Symbol        string   `json:"symbol"`
	Sector        string   `json:"sector,omitempty"`
	Price         *float64 `json:"price"`
	ChangePercent *float64 `json:"changePercent"`
	ExploreURL    string   `json:"exploreUrl"`
	// Change is last price minus previous close.
	Change *float64       `json:"change,omitempty"`
	Volume *float64       `json:"volume,omitempty"`
	Spark  []source.Point `json:"spark,omitempty"`
}

// Sector is the average day change of one sector.
type Sector struct {
	Name          string  `json:"name"`
	ChangePercent float64 `json:"changePercent"`
	Constituents  int     `json:"constituents,omitempty"`
}

// Market is section 3: indices, breadth and sectors.
type Market struct {
	Indices        []Quote  `json:"indices"`
	Advancers      *float64 `json:"advancers"`
	Decliners      *float64 `json:"decliners"`
	Unchanged      *float64 `json:"unchanged"`
	AdvanceDecline *float64 `json:"advanceDecline"`
	Sectors        []Sector `json:"sectors"`
	Spark          *Spark   `json:"spark"`
}

// Spark is a small price chart for one instrument.
type Spark struct {
	Symbol string         `json:"symbol"`
	Points []source.Point `json:"points"`
}

func (s *Server) market(ctx context.Context) (any, error) {
	q := s.querier(ctx)
	prices := bySymbol(q.vector(`max by (exchange, symbol) (stock_last_price{instrument_type="INDEX"})`))
	changes := bySymbol(q.vector(`max by (exchange, symbol) (stock_change_percent{instrument_type="INDEX"})`))
	abs := bySymbol(q.vector(absChangeQuery(`instrument_type="INDEX"`)))
	m := Market{
		Indices:        []Quote{},
		Advancers:      q.scalar(`market:advancers:count`),
		Decliners:      q.scalar(`market:decliners:count`),
		Unchanged:      q.scalar(`count(stock_change_percent{instrument_type="EQ"} == 0) or (count(stock_change_percent{instrument_type="EQ"}) * 0)`),
		AdvanceDecline: q.scalar(`market:advance_decline:ratio`),
		Sectors:        []Sector{},
	}
	members := byLabel(q.vector(`count by (sector) (stock_change_percent{instrument_type="EQ", tier="core"})`), "sector")
	for _, smp := range q.vector(`sector:change_percent:avg`) {
		if name := smp.Labels["sector"]; name != "" {
			m.Sectors = append(m.Sectors, Sector{Name: name, ChangePercent: smp.Value, Constituents: int(members[name])})
		}
	}
	sort.Slice(m.Sectors, func(i, j int) bool { return m.Sectors[i].ChangePercent > m.Sectors[j].ChangePercent })

	for _, sym := range s.cfg.Indices {
		p, ok := prices[sym]
		if !ok {
			continue
		}
		qt := Quote{Exchange: p.Labels["exchange"], Symbol: sym, Price: &p.Value, ExploreURL: symbolExploreURL(p.Labels["exchange"], sym)}
		if c, ok := changes[sym]; ok {
			qt.ChangePercent = &c.Value
		}
		if c, ok := abs[sym]; ok {
			qt.Change = &c.Value
		}
		m.Indices = append(m.Indices, qt)
	}
	if len(m.Indices) > 0 {
		syms := make([]string, len(m.Indices))
		for i, qt := range m.Indices {
			syms[i] = qt.Symbol
		}
		sparks := sessionSparks(q, `, instrument_type="INDEX"`, syms, s.now(), s.cfg.Location)
		for i := range m.Indices {
			m.Indices[i].Spark = sparks[m.Indices[i].Symbol]
		}
	}
	if len(m.Indices) > 0 {
		sym := m.Indices[0].Symbol
		end := s.now()
		if series := q.rangeQuery(fmt.Sprintf(`max(stock_last_price{symbol=%q})`, sym), end.Add(-7*time.Hour), end, 2*time.Minute); len(series) > 0 {
			m.Spark = &Spark{Symbol: sym, Points: series[0].Points}
		}
	}
	if q.err != nil {
		return nil, q.err
	}
	return m, nil
}

// Movers is section 7: top gainers and losers among core equities.
type Movers struct {
	Gainers    []Quote `json:"gainers"`
	Losers     []Quote `json:"losers"`
	HighVolume []Quote `json:"highVolume"`
}

const moversN = 5

func (s *Server) movers(ctx context.Context) (any, error) {
	q := s.querier(ctx)
	const sel = `stock_change_percent{instrument_type="EQ", tier="core"}`
	top := q.vector(fmt.Sprintf(`topk(%d, max by (exchange, symbol, sector) (%s))`, moversN, sel))
	bottom := q.vector(fmt.Sprintf(`bottomk(%d, max by (exchange, symbol, sector) (%s))`, moversN, sel))
	volume := q.vector(fmt.Sprintf(`topk(%d, max by (exchange, symbol, sector) (stock_day_volume_shares{instrument_type="EQ", tier="core"} > 0))`, moversN))
	prices := bySymbol(q.vector(`max by (exchange, symbol) (stock_last_price{instrument_type="EQ", tier="core"})`))
	abs := bySymbol(q.vector(absChangeQuery(`instrument_type="EQ", tier="core"`)))
	pct := bySymbol(q.vector(`max by (exchange, symbol) (` + sel + `)`))
	if q.err != nil {
		return nil, q.err
	}
	m := Movers{Gainers: quotes(top, prices), Losers: quotes(bottom, prices), HighVolume: []Quote{}}
	sort.Slice(m.Gainers, func(i, j int) bool { return *m.Gainers[i].ChangePercent > *m.Gainers[j].ChangePercent })
	sort.Slice(m.Losers, func(i, j int) bool { return *m.Losers[i].ChangePercent < *m.Losers[j].ChangePercent })
	m.Gainers = filter(m.Gainers, func(q Quote) bool { return *q.ChangePercent > 0 })
	m.Losers = filter(m.Losers, func(q Quote) bool { return *q.ChangePercent < 0 })
	for _, smp := range volume {
		qt := quote(smp, prices)
		v := smp.Value
		qt.Volume, qt.ChangePercent = &v, nil
		if c, ok := pct[qt.Symbol]; ok {
			qt.ChangePercent = &c.Value
		}
		m.HighVolume = append(m.HighVolume, qt)
	}
	sort.Slice(m.HighVolume, func(i, j int) bool { return *m.HighVolume[i].Volume > *m.HighVolume[j].Volume })
	var syms []string
	for _, list := range [][]Quote{m.Gainers, m.Losers} {
		for _, qt := range list {
			syms = append(syms, qt.Symbol)
		}
	}
	var sparks map[string][]source.Point
	if len(syms) > 0 {
		if sparks = sessionSparks(q, `, instrument_type="EQ", tier="core"`, syms, s.now(), s.cfg.Location); q.err != nil {
			return nil, q.err
		}
	}
	for _, list := range [][]Quote{m.Gainers, m.Losers, m.HighVolume} {
		for i := range list {
			if c, ok := abs[list[i].Symbol]; ok {
				list[i].Change = &c.Value
			}
			list[i].Spark = sparks[list[i].Symbol]
		}
	}
	return m, nil
}

// absChangeQuery returns last price minus a known previous close, by exchange and symbol.
func absChangeQuery(sel string) string {
	return fmt.Sprintf(`max by (exchange, symbol) (stock_last_price{%[1]s} - (stock_previous_close_price{%[1]s} > 0))`, sel)
}

// CircuitItem is an instrument at or near a circuit limit.
type CircuitItem struct {
	Quote
	Side            string  `json:"side"` // upper or lower.
	DistancePercent float64 `json:"distancePercent"`
	Hit             bool    `json:"hit"`
}

// Circuits is section 8: circuit watch.
type Circuits struct {
	Hits      []CircuitItem `json:"hits"`
	Near      []CircuitItem `json:"near"`
	Threshold float64       `json:"thresholdPercent"`
}

const (
	circuitNearPercent = 2.0
	maxCircuitItems    = 10
)

func (s *Server) circuits(ctx context.Context) (any, error) {
	q := s.querier(ctx)
	query := func(side string) string {
		return fmt.Sprintf(`max by (exchange, symbol, sector) (
  (stock:%[1]s_circuit_distance:percent < %[2]g)
  and on (exchange, symbol) (stock_circuit_%[1]s_limit_price > 0)
)`, side, circuitNearPercent)
	}
	upper := q.vector(query("upper"))
	lower := q.vector(query("lower"))
	prices := bySymbol(q.vector(`max by (exchange, symbol) (stock_last_price)`))
	if q.err != nil {
		return nil, q.err
	}
	c := Circuits{Hits: []CircuitItem{}, Near: []CircuitItem{}, Threshold: circuitNearPercent}
	add := func(side string, v []source.Sample) {
		for _, smp := range v {
			it := CircuitItem{Quote: quote(smp, prices), Side: side, DistancePercent: math.Max(smp.Value, 0), Hit: smp.Value <= 0}
			it.ChangePercent = nil
			if it.Hit {
				c.Hits = append(c.Hits, it)
			} else {
				c.Near = append(c.Near, it)
			}
		}
	}
	add("upper", upper)
	add("lower", lower)
	sort.Slice(c.Near, func(i, j int) bool { return c.Near[i].DistancePercent < c.Near[j].DistancePercent })
	sort.Slice(c.Hits, func(i, j int) bool { return c.Hits[i].Symbol < c.Hits[j].Symbol })
	if len(c.Near) > maxCircuitItems {
		c.Near = c.Near[:maxCircuitItems]
	}
	return c, nil
}

// Portfolio is section 4: holdings, positions and margin.
type Portfolio struct {
	Enabled          bool          `json:"enabled"`
	HoldingValue     *float64      `json:"holdingValue"`
	DayChange        *float64      `json:"dayChange"`
	DayChangePercent *float64      `json:"dayChangePercent"`
	HoldingPnL       *float64      `json:"holdingPnl"`
	PositionPnL      *float64      `json:"positionPnl"`
	MarginUsed       *float64      `json:"marginUsed"`
	MarginTotal      *float64      `json:"marginTotal"`
	UpdatedAt        *time.Time    `json:"updatedAt"`
	TopHoldings      []HoldingItem `json:"topHoldings"`
}

// HoldingItem is one holding.
type HoldingItem struct {
	Exchange  string  `json:"exchange"`
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	Value     float64 `json:"value"`
	PnL       float64 `json:"pnl"`
	DayChange float64 `json:"dayChange"`
}

const maxHoldings = 8

func (s *Server) portfolio(ctx context.Context) (any, error) {
	q := s.querier(ctx)
	updated := q.scalar(`max(stock_portfolio_last_update_timestamp_seconds)`)
	if q.err != nil {
		return nil, q.err
	}
	if updated == nil {
		return Portfolio{TopHoldings: []HoldingItem{}}, nil
	}
	p := Portfolio{
		Enabled:      true,
		UpdatedAt:    unixTime(updated),
		HoldingValue: q.scalar(`portfolio:holding_value:sum`),
		DayChange:    q.scalar(`portfolio:day_change:sum`),
		HoldingPnL:   q.scalar(`portfolio:holding_pnl:sum`),
		PositionPnL:  q.scalar(`portfolio:position_pnl:sum`),
		MarginUsed:   q.scalar(`sum(max by (segment) (stock_portfolio_margin_utilised))`),
		MarginTotal:  q.scalar(`sum(max by (segment) (stock_portfolio_margin_total))`),
		TopHoldings:  []HoldingItem{},
	}
	qty := bySymbol(q.vector(`max by (exchange, symbol) (stock_portfolio_holding_quantity)`))
	last := bySymbol(q.vector(`max by (exchange, symbol) (stock_portfolio_holding_last_price)`))
	pnl := bySymbol(q.vector(`max by (exchange, symbol) (stock_portfolio_holding_pnl)`))
	day := bySymbol(q.vector(`max by (exchange, symbol) (stock_portfolio_holding_day_change)`))
	if q.err != nil {
		return nil, q.err
	}
	if p.HoldingValue != nil && p.DayChange != nil {
		if base := *p.HoldingValue - *p.DayChange; base > 0 {
			v := *p.DayChange / base * 100
			p.DayChangePercent = &v
		}
	}
	for sym, qs := range qty {
		p.TopHoldings = append(p.TopHoldings, HoldingItem{
			Exchange: qs.Labels["exchange"], Symbol: sym, Quantity: qs.Value,
			Value: qs.Value * last[sym].Value, PnL: pnl[sym].Value, DayChange: day[sym].Value,
		})
	}
	sort.Slice(p.TopHoldings, func(i, j int) bool {
		return math.Abs(p.TopHoldings[i].DayChange) > math.Abs(p.TopHoldings[j].DayChange)
	})
	if len(p.TopHoldings) > maxHoldings {
		p.TopHoldings = p.TopHoldings[:maxHoldings]
	}
	return p, nil
}

// bySymbol indexes samples by their symbol label.
func bySymbol(v []source.Sample) map[string]source.Sample {
	out := make(map[string]source.Sample, len(v))
	for _, s := range v {
		out[s.Labels["symbol"]] = s
	}
	return out
}

func quote(smp source.Sample, prices map[string]source.Sample) Quote {
	sym, ex := smp.Labels["symbol"], smp.Labels["exchange"]
	v := smp.Value
	qt := Quote{Exchange: ex, Symbol: sym, Sector: smp.Labels["sector"], ChangePercent: &v, ExploreURL: symbolExploreURL(ex, sym)}
	if p, ok := prices[sym]; ok {
		pv := p.Value
		qt.Price = &pv
	}
	return qt
}

func quotes(v []source.Sample, prices map[string]source.Sample) []Quote {
	out := make([]Quote, 0, len(v))
	for _, smp := range v {
		out = append(out, quote(smp, prices))
	}
	return out
}

func filter[T any](in []T, keep func(T) bool) []T {
	out := in[:0]
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
