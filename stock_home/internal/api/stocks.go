package api

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Stock Explorer: search, filter, sort and page the all-equities snapshot.

var (
	queryPattern  = regexp.MustCompile(`^[A-Z0-9&.\- ]{0,32}$`)
	changeFilters = map[string]func(p float64) bool{
		"all":       func(float64) bool { return true },
		"up":        func(p float64) bool { return p > 0 },
		"down":      func(p float64) bool { return p < 0 },
		"unchanged": func(p float64) bool { return p == 0 },
		"gt2":       func(p float64) bool { return p >= 2 },
		"lt-2":      func(p float64) bool { return p <= -2 },
	}
	// Numeric sort keys; nil means the instrument has no value and sorts last.
	stockSorts = map[string]func(moverRow) *float64{
		"symbol": nil,
		"price":  func(r moverRow) *float64 { return r.price },
		"change": func(r moverRow) *float64 { return r.pct },
		"volume": func(r moverRow) *float64 { return r.volume },
	}
	perPage = map[int]bool{10: true, 25: true, 50: true}
)

// Stocks is one page of Stock Explorer results.
type Stocks struct {
	Items       []Quote  `json:"items"`
	Total       int      `json:"total"`
	Page        int      `json:"page"`
	Per         int      `json:"per"`
	SectorNames []string `json:"sectorNames"`
}

func (s *Server) stocksHandler(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	bad := func(msg string) { writeJSON(w, http.StatusBadRequest, envelope{Error: msg, UpdatedAt: s.now()}) }
	q := strings.ToUpper(strings.TrimSpace(v.Get("q")))
	sector, price, change := strings.TrimSpace(v.Get("sector")), or(v.Get("price"), "all"), or(v.Get("change"), "all")
	sortKey, dir := or(v.Get("sort"), "symbol"), or(v.Get("dir"), "asc")
	page, err1 := strconv.Atoi(or(v.Get("page"), "1"))
	per, err2 := strconv.Atoi(or(v.Get("per"), "25"))
	band, okPrice := priceBands[price]
	keep, okChange := changeFilters[change]
	key, okSort := stockSorts[sortKey]
	switch {
	case !queryPattern.MatchString(q):
		bad("q must be part of an NSE trading symbol")
	case sector != "" && !sectorPattern.MatchString(sector):
		bad("invalid sector")
	case !okPrice:
		bad("price must be one of all, lt100, 100-500, 500-2000, gt2000")
	case !okChange:
		bad("change must be one of all, up, down, unchanged, gt2, lt-2")
	case !okSort || (dir != "asc" && dir != "desc"):
		bad("sort must be symbol, price, change or volume; dir asc or desc")
	case err1 != nil || page < 1 || err2 != nil || !perPage[per]:
		bad("page must be >= 1 and per one of 10, 25, 50")
	default:
		res, err := s.moverSnapshot("all")
		if err != nil {
			s.errors.WithLabelValues("stocks").Inc()
			s.logger.Warn("Stock explorer query failed", "err", err)
			writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
			return
		}
		out := filterStocks(res.Value.([]moverRow), q, sector, band, price != "all", keep, key, dir == "desc", page, per)
		writeJSON(w, http.StatusOK, envelope{Data: out, UpdatedAt: res.UpdatedAt, Stale: res.Stale})
	}
}

func filterStocks(all []moverRow, q, sector string, band [2]float64, byPrice bool, keep func(float64) bool, key func(moverRow) *float64, desc bool, page, per int) Stocks {
	out := Stocks{Items: []Quote{}, Page: page, Per: per, SectorNames: []string{}}
	names := map[string]bool{}
	var rows []moverRow
	for _, r := range all {
		if r.sector != "" && !names[r.sector] {
			names[r.sector] = true
			out.SectorNames = append(out.SectorNames, r.sector)
		}
		switch {
		case q != "" && !strings.Contains(r.symbol, q),
			sector != "" && r.sector != sector,
			byPrice && (r.price == nil || *r.price < band[0] || *r.price >= band[1]),
			!keep(*r.pct):
			continue
		}
		rows = append(rows, r)
	}
	sort.Strings(out.SectorNames)
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if key == nil {
			return (a.symbol < b.symbol) != desc
		}
		va, vb := key(a), key(b)
		switch {
		case va == nil || vb == nil:
			return va != nil && vb == nil
		case *va == *vb:
			return a.symbol < b.symbol
		}
		return (*va < *vb) != desc
	})
	// An exact symbol match leads the results of a search.
	if q != "" {
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].symbol == q && rows[j].symbol != q })
	}
	out.Total = len(rows)
	for _, r := range rows[min((page-1)*per, len(rows)):min(page*per, len(rows))] {
		out.Items = append(out.Items, r.quote())
	}
	return out
}
