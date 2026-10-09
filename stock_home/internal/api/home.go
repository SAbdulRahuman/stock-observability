package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/seenimo/stock_home/internal/source"
)

// marketOpen keeps only samples taken while the exchange session was open.
const marketOpen = ` and on() (max(stock_market_open) == 1)`

const (
	sparkLookback = 5 * 24 * time.Hour // Covers a weekend plus a holiday.
	sparkStep     = 5 * time.Minute
)

// sessionRange runs expr over [now-lookback, now], optionally only while the market was open,
// and with lastDay keeps each series' points from its latest exchange day only.
func sessionRange(q *querier, expr string, now time.Time, lookback, step time.Duration, open, lastDay bool, loc *time.Location) []source.Series {
	if open {
		expr = "(" + expr + ")" + marketOpen
	}
	start := now.Add(-lookback).Truncate(step)
	series := q.rangeQuery(expr, start, now, step)
	if !lastDay {
		return series
	}
	for i, s := range series {
		series[i].Points = lastDayPoints(s.Points, loc)
	}
	return series
}

func lastDayPoints(pts []source.Point, loc *time.Location) []source.Point {
	if len(pts) == 0 {
		return pts
	}
	day := func(p source.Point) string { return time.UnixMilli(p.T).In(loc).Format(time.DateOnly) }
	last := day(pts[len(pts)-1])
	i := len(pts)
	for i > 0 && day(pts[i-1]) == last {
		i--
	}
	return pts[i:]
}

// sessionSparks returns the latest session's last-price points per symbol; matchers narrows the selector.
func sessionSparks(q *querier, matchers string, symbols []string, now time.Time, loc *time.Location) map[string][]source.Point {
	expr := fmt.Sprintf(`max by (symbol) (stock_last_price{symbol=~%q%s})`, symbolAlternation(symbols), matchers)
	out := map[string][]source.Point{}
	for _, s := range sessionRange(q, expr, now, sparkLookback, sparkStep, true, true, loc) {
		if len(s.Points) > 1 {
			out[s.Labels["symbol"]] = s.Points
		}
	}
	return out
}

func symbolAlternation(symbols []string) string {
	quoted := make([]string, len(symbols))
	for i, s := range symbols {
		quoted[i] = regexp.QuoteMeta(s)
	}
	return strings.Join(quoted, "|")
}

type chartRange struct {
	lookback, step, ttl time.Duration
	open, lastDay       bool
}

var chartRanges = map[string]chartRange{
	"1D": {5 * 24 * time.Hour, 2 * time.Minute, 30 * time.Second, true, true},
	"1W": {7 * 24 * time.Hour, 10 * time.Minute, 2 * time.Minute, true, false},
	"1M": {30 * 24 * time.Hour, time.Hour, 10 * time.Minute, true, false},
	"3M": {90 * 24 * time.Hour, 3 * time.Hour, 10 * time.Minute, true, false},
	"1Y": {365 * 24 * time.Hour, 24 * time.Hour, 30 * time.Minute, false, false},
	"5Y": {5 * 365 * 24 * time.Hour, 7 * 24 * time.Hour, time.Hour, false, false},
}

// Chart is the price history of an index (with core-equity traded volume) or of one equity.
type Chart struct {
	Symbol      string         `json:"symbol"`
	Range       string         `json:"range"`
	Points      []source.Point `json:"points"`
	Volume      []source.Point `json:"volume"`
	VolumeLabel string         `json:"volumeLabel"`
	Last        *float64       `json:"last"`
}

const (
	chartVolumeLabel  = "Core equities traded volume"
	equityVolumeLabel = "Traded volume"
)

func (s *Server) chartHandler(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	rng := strings.ToUpper(r.URL.Query().Get("range"))
	if rng == "" {
		rng = "1D"
	}
	cr, ok := chartRanges[rng]
	if !ok {
		writeJSON(w, http.StatusBadRequest, envelope{Error: "range must be one of 1D, 1W, 1M, 3M, 1Y, 5Y", UpdatedAt: s.now()})
		return
	}
	if s.isIndex(symbol) {
		s.cached(w, "chart", "chart:"+symbol+":"+rng, cr.ttl, func(ctx context.Context) (any, error) {
			return s.chart(ctx, symbol, rng, cr, true)
		})
		return
	}
	if !symbolPattern.MatchString(symbol) {
		writeJSON(w, http.StatusBadRequest, envelope{Error: "symbol must be a configured index or an NSE trading symbol", UpdatedAt: s.now()})
		return
	}
	s.direct(w, r, "chart", func(ctx context.Context) (any, error) { return s.chart(ctx, symbol, rng, cr, false) })
}

func (s *Server) isIndex(symbol string) bool {
	for _, sym := range append(append([]string{}, s.cfg.Indices...), s.cfg.SectorIndices...) {
		if strings.EqualFold(sym, symbol) {
			return true
		}
	}
	return false
}

func (s *Server) chart(ctx context.Context, symbol, rng string, cr chartRange, index bool) (any, error) {
	q := s.querier(ctx)
	now := s.now()
	window := promDuration(cr.step)
	priceExpr := fmt.Sprintf(`max(stock_last_price{symbol=%q, instrument_type="INDEX"})`, symbol)
	volumeExpr := fmt.Sprintf(`sum(increase(stock_day_volume_shares{instrument_type="EQ", tier="core"}[%s]))`, window)
	label := chartVolumeLabel
	if !index {
		priceExpr = fmt.Sprintf(`max(stock_last_price{symbol=%q, instrument_type="EQ"})`, symbol)
		volumeExpr = fmt.Sprintf(`sum(increase(stock_day_volume_shares{symbol=%q, instrument_type="EQ"}[%s]))`, symbol, window)
		label = equityVolumeLabel
	}
	price := sessionRange(q, priceExpr, now, cr.lookback, cr.step, cr.open, cr.lastDay, s.cfg.Location)
	volume := sessionRange(q, volumeExpr, now, cr.lookback, cr.step, cr.open, cr.lastDay, s.cfg.Location)
	if q.err != nil {
		return nil, q.err
	}
	c := Chart{Symbol: symbol, Range: rng, Points: []source.Point{}, Volume: []source.Point{}, VolumeLabel: label}
	if len(price) > 0 {
		c.Points = price[0].Points
	}
	if len(volume) > 0 && len(c.Points) > 0 {
		from := c.Points[0].T
		for _, p := range volume[0].Points {
			if p.T >= from {
				c.Volume = append(c.Volume, p)
			}
		}
	}
	if n := len(c.Points); n > 0 {
		v := c.Points[n-1].V
		c.Last = &v
	}
	return c, nil
}

// promDuration formats a duration as a PromQL range such as "2m" or "1d".
func promDuration(d time.Duration) string {
	switch {
	case d%(24*time.Hour) == 0:
		return fmt.Sprintf("%dd", d/(24*time.Hour))
	case d%time.Hour == 0:
		return fmt.Sprintf("%dh", d/time.Hour)
	default:
		return fmt.Sprintf("%dm", d/time.Minute)
	}
}

// Quotes is the watchlist snapshot for a list of symbols, in request order.
type Quotes struct {
	Items []Quote `json:"items"`
}

const maxQuoteSymbols = 20

func (s *Server) quotesHandler(w http.ResponseWriter, r *http.Request) {
	var symbols []string
	seen := map[string]bool{}
	for _, p := range strings.Split(r.URL.Query().Get("symbols"), ",") {
		sym := strings.ToUpper(strings.TrimSpace(p))
		if sym == "" || seen[sym] {
			continue
		}
		if !symbolPattern.MatchString(sym) {
			writeJSON(w, http.StatusBadRequest, envelope{Error: "symbols must be NSE trading symbols", UpdatedAt: s.now()})
			return
		}
		seen[sym] = true
		symbols = append(symbols, sym)
	}
	if len(symbols) == 0 || len(symbols) > maxQuoteSymbols {
		writeJSON(w, http.StatusBadRequest, envelope{Error: fmt.Sprintf("symbols: 1 to %d comma-separated symbols required", maxQuoteSymbols), UpdatedAt: s.now()})
		return
	}
	// Not cached: arbitrary symbol lists would grow the cache without bound.
	s.direct(w, r, "quotes", func(ctx context.Context) (any, error) { return s.quotes(ctx, symbols) })
}

func (s *Server) quotes(ctx context.Context, symbols []string) (Quotes, error) {
	q := s.querier(ctx)
	sel := fmt.Sprintf(`symbol=~%q`, symbolAlternation(symbols))
	prices := bySymbol(q.vector(fmt.Sprintf(`max by (exchange, symbol, sector) (stock_last_price{%s})`, sel)))
	pct := bySymbol(q.vector(fmt.Sprintf(`max by (exchange, symbol) (stock_change_percent{%s})`, sel)))
	abs := bySymbol(q.vector(absChangeQuery(sel)))
	volume := bySymbol(q.vector(fmt.Sprintf(`max by (symbol) (stock_day_volume_shares{%s})`, sel)))
	sparks := sessionSparks(q, "", symbols, s.now(), s.cfg.Location)
	if q.err != nil {
		return Quotes{}, q.err
	}
	out := Quotes{Items: make([]Quote, 0, len(symbols))}
	for _, sym := range symbols {
		qt := Quote{Symbol: sym, Spark: sparks[sym]}
		if p, ok := prices[sym]; ok {
			v := p.Value
			qt.Exchange, qt.Sector, qt.Price = p.Labels["exchange"], p.Labels["sector"], &v
			qt.ExploreURL = symbolExploreURL(qt.Exchange, sym)
		}
		if c, ok := pct[sym]; ok {
			v := c.Value
			qt.ChangePercent = &v
		}
		if c, ok := abs[sym]; ok {
			v := c.Value
			qt.Change = &v
		}
		if c, ok := volume[sym]; ok {
			v := c.Value
			qt.Volume = &v
		}
		out.Items = append(out.Items, qt)
	}
	return out, nil
}

// direct serves an uncached query bounded by the request context and the query timeout.
func (s *Server) direct(w http.ResponseWriter, r *http.Request, section string, fn func(context.Context) (any, error)) {
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.QueryTimeout)
	defer cancel()
	res, err := fn(ctx)
	if err != nil {
		s.errors.WithLabelValues(section).Inc()
		s.logger.Warn("Query failed", "section", section, "err", err)
		writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
		return
	}
	if nf, ok := res.(notFound); ok {
		writeJSON(w, http.StatusNotFound, envelope{Error: string(nf), UpdatedAt: s.now()})
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: res, UpdatedAt: s.now()})
}

// notFound is returned by direct query functions when the subject has no data.
type notFound string

// cached serves a parameterised query through the cache with the generic upstream error.
func (s *Server) cached(w http.ResponseWriter, section, key string, ttl time.Duration, fn func(context.Context) (any, error)) {
	res, err := s.cache.Get(key, ttl, 10*ttl+time.Minute, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.QueryTimeout)
		defer cancel()
		return fn(ctx)
	})
	if err != nil {
		s.errors.WithLabelValues(section).Inc()
		s.logger.Warn("Query failed", "section", section, "key", key, "err", err)
		writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: res.Value, UpdatedAt: res.UpdatedAt, Stale: res.Stale})
}
