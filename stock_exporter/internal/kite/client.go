// Copyright 2026 The Stock Exporter Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kite

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"

	"github.com/seenimo/stock_exporter/internal/daily"
	"github.com/seenimo/stock_exporter/internal/store"
	"github.com/seenimo/stock_exporter/internal/watchlist"
)

// Kite REST limits: quote endpoints 1 req/s with up to 500 instruments, historical 3 req/s, others 10 req/s.
const (
	QuoteBatchSize = 500
	quoteRate      = time.Second
	historicalRate = 350 * time.Millisecond
	defaultRate    = 100 * time.Millisecond
)

// Endpoint names used in the endpoint label.
const (
	EndpointQuote       = "quote"
	EndpointHoldings    = "holdings"
	EndpointPositions   = "positions"
	EndpointMargins     = "margins"
	EndpointInstruments = "instruments"
	EndpointHistorical  = "historical"
)

// API is the subset of the Kite REST client used by the exporter.
type API interface {
	GetQuote(instruments ...string) (kiteconnect.Quote, error)
	GetHoldings() (kiteconnect.Holdings, error)
	GetPositions() (kiteconnect.Positions, error)
	GetUserMargins() (kiteconnect.AllMargins, error)
	GetInstrumentsByExchange(exchange string) (kiteconnect.Instruments, error)
	GetHistoricalData(instrumentToken int, interval string, fromDate, toDate time.Time, continuous, oi bool) ([]kiteconnect.HistoricalData, error)
}

// limiter enforces a minimum interval between calls.
type limiter struct {
	mu    sync.Mutex
	every time.Duration
	next  time.Time
}

func (l *limiter) wait(ctx context.Context) error {
	l.mu.Lock()
	now := time.Now()
	at := l.next
	if at.Before(now) {
		at = now
	}
	l.next = at.Add(l.every)
	l.mu.Unlock()
	if d := time.Until(at); d > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
	return nil
}

// Client wraps API with rate limiting, outcome accounting and token-error handling.
type Client struct {
	api          API
	health       *store.Health
	onTokenError func()
	quote        *limiter
	historical   *limiter
	other        *limiter
}

// NewClient wraps api. onTokenError is called when Kite rejects the access token.
func NewClient(api API, health *store.Health, onTokenError func()) *Client {
	return &Client{api: api, health: health, onTokenError: onTokenError,
		quote: &limiter{every: quoteRate}, historical: &limiter{every: historicalRate}, other: &limiter{every: defaultRate}}
}

// IsTokenError reports whether err is a Kite TokenException.
func IsTokenError(err error) bool {
	var ke kiteconnect.Error
	return errors.As(err, &ke) && ke.ErrorType == kiteconnect.TokenError
}

func (c *Client) observe(endpoint string, err error) error {
	outcome := store.OutcomeSuccess
	var ke kiteconnect.Error
	switch {
	case err == nil:
	case errors.As(err, &ke) && ke.ErrorType == kiteconnect.TokenError:
		outcome = store.OutcomeTokenError
		if c.onTokenError != nil {
			c.onTokenError()
		}
	case errors.As(err, &ke) && ke.Code == http.StatusTooManyRequests:
		outcome = store.OutcomeRateLimited
	default:
		outcome = store.OutcomeError
	}
	c.health.ObserveAPI(endpoint, outcome)
	return err
}

// Instruments resolves the watchlist against the instrument master of each referenced exchange.
func (c *Client) Instruments(ctx context.Context, wl *watchlist.Watchlist) ([]store.Instrument, []string, error) {
	var refs []watchlist.Ref
	for _, ex := range wl.Exchanges() {
		if err := c.other.wait(ctx); err != nil {
			return nil, nil, err
		}
		ins, err := c.api.GetInstrumentsByExchange(ex)
		if err := c.observe(EndpointInstruments, err); err != nil {
			return nil, nil, err
		}
		for _, i := range ins {
			refs = append(refs, watchlist.Ref{Token: uint32(i.InstrumentToken), Exchange: i.Exchange, Symbol: i.Tradingsymbol, Segment: i.Segment, InstrumentType: i.InstrumentType,
				Name: i.Name, Expiry: i.Expiry.Time, Strike: i.StrikePrice})
		}
	}
	out, missing := wl.Resolve(refs)
	return out, missing, nil
}

// RefreshCircuits fetches circuit limits for all non-index instruments in batches.
func (c *Client) RefreshCircuits(ctx context.Context, st *store.Store) error {
	var keys []string
	for _, in := range st.Instruments() {
		if in.Type != store.TypeIndex {
			keys = append(keys, in.Exchange+":"+in.Symbol)
		}
	}
	for len(keys) > 0 {
		n := min(len(keys), QuoteBatchSize)
		batch := keys[:n]
		keys = keys[n:]
		if err := c.quote.wait(ctx); err != nil {
			return err
		}
		q, err := c.api.GetQuote(batch...)
		if err := c.observe(EndpointQuote, err); err != nil {
			return err
		}
		now := time.Now()
		for _, v := range q {
			if v.UpperCircuitLimit > 0 {
				st.UpdateCircuit(uint32(v.InstrumentToken), store.Circuit{Upper: v.UpperCircuitLimit, Lower: v.LowerCircuitLimit, UpdatedAt: now})
			}
		}
	}
	return nil
}

// RefreshReference fetches one year of daily candles up to yesterday for each core-tier instrument
// and stores its 52-week range, daily SMA 50/200 and SMA cross (needs the Kite historical data
// add-on). A failing instrument is skipped; a token error aborts the refresh.
func (c *Client) RefreshReference(ctx context.Context, st *store.Store, now time.Time) error {
	var errs []error
	to := now.AddDate(0, 0, -1)
	for _, in := range st.Instruments() {
		if in.Tier != store.TierCore {
			continue
		}
		if err := c.historical.wait(ctx); err != nil {
			return err
		}
		candles, err := c.api.GetHistoricalData(int(in.Token), "day", to.AddDate(-1, 0, 0), to, false, false)
		if err := c.observe(EndpointHistorical, err); err != nil {
			if IsTokenError(err) {
				return err
			}
			errs = append(errs, fmt.Errorf("%s:%s: %w", in.Exchange, in.Symbol, err))
			continue
		}
		highs, lows, closes := make([]float64, len(candles)), make([]float64, len(candles)), make([]float64, len(candles))
		days := make([]daily.Candle, len(candles))
		for i, k := range candles {
			highs[i], lows[i], closes[i] = k.High, k.Low, k.Close
			y, m, d := k.Date.Date()
			days[i] = daily.Candle{Date: time.Date(y, m, d, 0, 0, 0, 0, k.Date.Location()), Open: k.Open, High: k.High, Low: k.Low, Close: k.Close, Volume: float64(k.Volume)}
		}
		if r, ok := store.NewReference(highs, lows, closes, now); ok {
			st.UpdateReference(in.Token, r)
		}
		if len(days) > 0 {
			st.UpdateDaily(in.Token, daily.Compute(days, in.Type == store.TypeEquity))
		}
	}
	return errors.Join(errs...)
}

// RefreshPortfolio fetches holdings, positions and margins.
func (c *Client) RefreshPortfolio(ctx context.Context, st *store.Store) error {
	if err := c.other.wait(ctx); err != nil {
		return err
	}
	hs, err := c.api.GetHoldings()
	if err := c.observe(EndpointHoldings, err); err != nil {
		return err
	}
	if err := c.other.wait(ctx); err != nil {
		return err
	}
	ps, err := c.api.GetPositions()
	if err := c.observe(EndpointPositions, err); err != nil {
		return err
	}
	if err := c.other.wait(ctx); err != nil {
		return err
	}
	ms, err := c.api.GetUserMargins()
	if err := c.observe(EndpointMargins, err); err != nil {
		return err
	}
	p := store.Portfolio{UpdatedAt: time.Now()}
	for _, h := range hs {
		p.Holdings = append(p.Holdings, store.Holding{
			Exchange: h.Exchange, Symbol: h.Tradingsymbol, Quantity: float64(h.Quantity + h.T1Quantity), AveragePrice: h.AveragePrice,
			LastPrice: h.LastPrice, PnL: h.PnL, DayChange: h.DayChange * float64(h.Quantity+h.T1Quantity),
		})
	}
	for _, x := range ps.Net {
		p.Positions = append(p.Positions, store.Position{
			Exchange: x.Exchange, Symbol: x.Tradingsymbol, Product: x.Product, Quantity: float64(x.Quantity), AveragePrice: x.AveragePrice,
			LastPrice: x.LastPrice, PnL: x.PnL, M2M: x.M2M,
		})
	}
	for _, seg := range []struct {
		name string
		m    kiteconnect.Margins
	}{{"equity", ms.Equity}, {"commodity", ms.Commodity}} {
		if !seg.m.Enabled {
			continue
		}
		p.Margins = append(p.Margins, store.Margin{Segment: seg.name, Net: seg.m.Net, Total: seg.m.Net + seg.m.Used.Debits, Utilised: seg.m.Used.Debits})
	}
	st.SetPortfolio(p)
	return nil
}

// Poll runs fn immediately and then every interval until ctx is cancelled.
func Poll(ctx context.Context, interval time.Duration, name string, logger *slog.Logger, fn func(context.Context) error) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if err := fn(ctx); err != nil && ctx.Err() == nil {
			logger.Warn("Kite poll failed", "poll", name, "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
