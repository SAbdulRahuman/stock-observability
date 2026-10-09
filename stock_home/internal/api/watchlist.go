package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Instrument is the day summary of one equity for the Watchlist detail panel.
type Instrument struct {
	Quote
	Open         *float64 `json:"open"`
	High         *float64 `json:"high"`
	Low          *float64 `json:"low"`
	PrevClose    *float64 `json:"prevClose"`
	AvgVolume20d *float64 `json:"avgVolume20d"`
	ChartURL     string   `json:"chartUrl"`
}

func (s *Server) instrumentHandler(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	if !symbolPattern.MatchString(symbol) {
		writeJSON(w, http.StatusBadRequest, envelope{Error: "symbol must be an NSE trading symbol", UpdatedAt: s.now()})
		return
	}
	s.direct(w, r, "instrument", func(ctx context.Context) (any, error) { return s.instrument(ctx, symbol) })
}

func (s *Server) instrument(ctx context.Context, symbol string) (any, error) {
	q := s.querier(ctx)
	sel := fmt.Sprintf(`symbol=%q, instrument_type="EQ"`, symbol)
	values := q.vector(fmt.Sprintf(`max by (__name__, exchange, sector) ({__name__=~"stock_(last|open|high|low|previous_close)_price|stock_change_percent|stock_day_volume_shares", %s})`, sel))
	// stock:session_volume:shares is only recorded after the close; 28 calendar days span about 20 sessions.
	avg := q.scalar(fmt.Sprintf(`max(avg_over_time(stock:session_volume:shares{%s}[28d]))`, sel))
	if q.err != nil {
		return nil, q.err
	}
	byName := map[string]*float64{}
	in := Instrument{Quote: Quote{Symbol: symbol}, AvgVolume20d: avg}
	for _, v := range values {
		val := v.Value
		byName[v.Labels["__name__"]] = &val
		in.Exchange, in.Sector = v.Labels["exchange"], v.Labels["sector"]
	}
	in.Price = byName["stock_last_price"]
	if in.Price == nil {
		return notFound("no price recorded for " + symbol), nil
	}
	in.ChangePercent = byName["stock_change_percent"]
	in.Volume = byName["stock_day_volume_shares"]
	in.Open, in.High, in.Low, in.PrevClose = byName["stock_open_price"], byName["stock_high_price"], byName["stock_low_price"], byName["stock_previous_close_price"]
	if in.PrevClose != nil && *in.PrevClose > 0 {
		c := *in.Price - *in.PrevClose
		in.Change = &c
	}
	in.ExploreURL = symbolExploreURL(in.Exchange, symbol)
	in.ChartURL = exploreURL(fmt.Sprintf("max by (symbol) (stock_last_price{exchange=%q, symbol=%q})", in.Exchange, symbol), "now-30d")
	return in, nil
}
