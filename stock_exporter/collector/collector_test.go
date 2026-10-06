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

package collector

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"

	"github.com/seenimo/stock_exporter/internal/market"
	"github.com/seenimo/stock_exporter/internal/store"
)

type adapter struct {
	c     Collector
	scope Scope
}

func (a adapter) Describe(ch chan<- *prometheus.Desc) { prometheus.DescribeByCollect(a, ch) }
func (a adapter) Collect(ch chan<- prometheus.Metric) { _ = a.c.Update(ch, a.scope) }

var (
	ist, _  = time.LoadLocation("Asia/Kolkata")
	testNow = time.Date(2026, 10, 6, 15, 0, 0, 0, ist)
)

func testDeps(t *testing.T) Deps {
	t.Helper()
	st := store.New()
	st.SetInstruments([]store.Instrument{
		{Token: 1, Exchange: "NSE", Symbol: "INFY", Sector: "IT", Type: store.TypeEquity, Tier: store.TierCore},
		{Token: 2, Exchange: "NSE", Symbol: "NIFTY 50", Sector: "unknown", Type: store.TypeIndex, Tier: store.TierCore},
		{Token: 3, Exchange: "NSE", Symbol: "ZZZ", Sector: "unknown", Type: store.TypeEquity, Tier: store.TierBroad},
	})
	st.UpdateQuote(1, store.Quote{
		LastPrice: 1550, Open: 1500, High: 1560, Low: 1490, PrevClose: 1500, AveragePrice: 1525.5,
		Volume: 1e6, LastQuantity: 10, BuyQuantity: 3000, SellQuantity: 1000,
		LastTradeTime: testNow.Add(-2 * time.Second), ReceivedAt: testNow.Add(-1 * time.Second),
		HasDepth: true, Depth: store.Depth{Bids: [5]store.DepthLevel{{Price: 1549.95, Quantity: 5, Orders: 1}}, Asks: [5]store.DepthLevel{{Price: 1550.05, Quantity: 7, Orders: 2}}},
	})
	st.UpdateQuote(2, store.Quote{LastPrice: 24750, Open: 25000, High: 25100, Low: 24700, PrevClose: 25000, ReceivedAt: testNow})
	st.UpdateCircuit(1, store.Circuit{Upper: 1650, Lower: 1350})
	h := store.NewHealth("kite", "full")
	h.SetConnected(0, true)
	h.AddTicks(42, testNow)
	h.IncReconnects()
	h.SetSubscribed(3)
	h.SetSession(true, time.Date(2026, 10, 7, 6, 0, 0, 0, ist))
	h.ObserveAPI("quote", store.OutcomeSuccess)
	h.ObserveAPI("quote", store.OutcomeRateLimited)
	return Deps{Store: st, Health: h, Calendar: market.NewCalendar(ist, nil, false), Now: func() time.Time { return testNow }}
}

func mustCollector(t *testing.T, f factory, deps Deps) Collector {
	t.Helper()
	c, err := f(slog.New(slog.DiscardHandler), deps)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const labels = `exchange="NSE",instrument_type="EQ",sector="IT",symbol="INFY",tier="core"`

func TestQuoteCollector(t *testing.T) {
	deps := testDeps(t)
	c := mustCollector(t, NewQuoteCollector, deps)
	want := `
# HELP stock_change_percent Change of last price versus previous close, in percent.
# TYPE stock_change_percent gauge
stock_change_percent{` + labels + `} 3.3333333333333335
stock_change_percent{exchange="NSE",instrument_type="INDEX",sector="unknown",symbol="NIFTY 50",tier="core"} -1
# HELP stock_day_volume_shares Shares traded today; resets every session.
# TYPE stock_day_volume_shares gauge
stock_day_volume_shares{` + labels + `} 1e+06
# HELP stock_last_price Last traded price (index value for indices).
# TYPE stock_last_price gauge
stock_last_price{` + labels + `} 1550
stock_last_price{exchange="NSE",instrument_type="INDEX",sector="unknown",symbol="NIFTY 50",tier="core"} 24750
# HELP stock_tick_age_seconds Seconds since the last tick was received (since exporter start if none yet).
# TYPE stock_tick_age_seconds gauge
stock_tick_age_seconds{` + labels + `} 1
stock_tick_age_seconds{exchange="NSE",instrument_type="INDEX",sector="unknown",symbol="NIFTY 50",tier="core"} 0
# HELP stock_total_buy_quantity Total pending buy quantity in the order book.
# TYPE stock_total_buy_quantity gauge
stock_total_buy_quantity{` + labels + `} 3000
`
	core, _ := NewScope(store.TierCore)
	if err := testutil.CollectAndCompare(adapter{c, core}, strings.NewReader(want),
		"stock_change_percent", "stock_day_volume_shares", "stock_last_price", "stock_tick_age_seconds", "stock_total_buy_quantity"); err != nil {
		t.Fatal(err)
	}

	// 13 series for the equity, 7 for the index, 1 (tick age only) for the broad instrument without quotes.
	if n := testutil.CollectAndCount(adapter{c, Scope{}}); n != 21 {
		t.Fatalf("all tiers: got %d series, want 21", n)
	}
	broad, _ := NewScope(store.TierBroad)
	if n := testutil.CollectAndCount(adapter{c, broad}); n != 1 {
		t.Fatalf("broad tier: got %d series, want 1", n)
	}
}

func TestCircuitCollector(t *testing.T) {
	c := mustCollector(t, NewCircuitCollector, testDeps(t))
	want := `
# HELP stock_circuit_lower_limit_price Lower circuit (price band) limit for the day.
# TYPE stock_circuit_lower_limit_price gauge
stock_circuit_lower_limit_price{` + labels + `} 1350
# HELP stock_circuit_upper_limit_price Upper circuit (price band) limit for the day.
# TYPE stock_circuit_upper_limit_price gauge
stock_circuit_upper_limit_price{` + labels + `} 1650
`
	if err := testutil.CollectAndCompare(adapter{c, Scope{}}, strings.NewReader(want)); err != nil {
		t.Fatal(err)
	}
}

func TestDepthCollector(t *testing.T) {
	c := mustCollector(t, NewDepthCollector, testDeps(t))
	if n := testutil.CollectAndCount(adapter{c, Scope{}}); n != 30 {
		t.Fatalf("got %d series, want 30", n)
	}
	want := `
# HELP stock_depth_price Order book price at level.
# TYPE stock_depth_price gauge
stock_depth_price{exchange="NSE",instrument_type="EQ",level="1",sector="IT",side="ask",symbol="INFY",tier="core"} 1550.05
stock_depth_price{exchange="NSE",instrument_type="EQ",level="1",sector="IT",side="bid",symbol="INFY",tier="core"} 1549.95
`
	// Compare only level 1 by filtering the other levels out of the collected output.
	got, err := testutil.CollectAndFormat(adapter{c, Scope{}}, expfmt.TypeTextPlain, "stock_depth_price")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(want), "\n") {
		if !strings.Contains(string(got), line) {
			t.Errorf("missing %q", line)
		}
	}
}

func TestMarketCollector(t *testing.T) {
	c := mustCollector(t, NewMarketCollector, testDeps(t))
	want := `
# HELP stock_market_open 1 during the normal trading session, else 0.
# TYPE stock_market_open gauge
stock_market_open 1
# HELP stock_market_phase Current session phase (1 for the active phase).
# TYPE stock_market_phase gauge
stock_market_phase{phase="closed"} 0
stock_market_phase{phase="open"} 1
stock_market_phase{phase="post_close"} 0
stock_market_phase{phase="pre_open"} 0
# HELP stock_market_seconds_to_close Seconds until the normal session closes, 0 when closed.
# TYPE stock_market_seconds_to_close gauge
stock_market_seconds_to_close 1800
# HELP stock_market_seconds_to_open Seconds until the next session opens, 0 when open.
# TYPE stock_market_seconds_to_open gauge
stock_market_seconds_to_open 0
# HELP stock_market_trading_day 1 if today is a trading day (weekday, not a holiday).
# TYPE stock_market_trading_day gauge
stock_market_trading_day 1
`
	if err := testutil.CollectAndCompare(adapter{c, Scope{}}, strings.NewReader(want)); err != nil {
		t.Fatal(err)
	}
}

func TestFeedCollector(t *testing.T) {
	c := mustCollector(t, NewFeedCollector, testDeps(t))
	want := `
# HELP stock_feed_connections Number of connected market data WebSocket connections.
# TYPE stock_feed_connections gauge
stock_feed_connections 1
# HELP stock_feed_instruments Resolved instruments per tier.
# TYPE stock_feed_instruments gauge
stock_feed_instruments{tier="broad"} 1
stock_feed_instruments{tier="core"} 2
# HELP stock_feed_ticks_total Ticks received for tracked instruments.
# TYPE stock_feed_ticks_total counter
stock_feed_ticks_total 42
# HELP stock_kite_api_requests_total Kite REST API requests by endpoint and outcome.
# TYPE stock_kite_api_requests_total counter
stock_kite_api_requests_total{endpoint="quote",outcome="rate_limited"} 1
stock_kite_api_requests_total{endpoint="quote",outcome="success"} 1
# HELP stock_kite_session_expiry_timestamp_seconds Expiry time of the Kite access token.
# TYPE stock_kite_session_expiry_timestamp_seconds gauge
stock_kite_session_expiry_timestamp_seconds 1.791333e+09
# HELP stock_kite_session_valid 1 if a non-expired Kite access token is held.
# TYPE stock_kite_session_valid gauge
stock_kite_session_valid 1
`
	if err := testutil.CollectAndCompare(adapter{c, Scope{}}, strings.NewReader(want),
		"stock_feed_connections", "stock_feed_instruments", "stock_feed_ticks_total", "stock_kite_api_requests_total",
		"stock_kite_session_expiry_timestamp_seconds", "stock_kite_session_valid"); err != nil {
		t.Fatal(err)
	}
}

func TestPortfolioCollector(t *testing.T) {
	deps := testDeps(t)
	c := mustCollector(t, NewPortfolioCollector, deps)
	if err := c.Update(make(chan prometheus.Metric, 10), Scope{}); !IsNoDataError(err) {
		t.Fatalf("expected ErrNoData before first refresh, got %v", err)
	}
	deps.Store.SetPortfolio(store.Portfolio{
		Holdings:  []store.Holding{{Exchange: "NSE", Symbol: "INFY", Quantity: 10, AveragePrice: 1400, LastPrice: 1550, PnL: 1500}, {Exchange: "NSE", Symbol: "INFY"}},
		Positions: []store.Position{{Exchange: "NSE", Symbol: "INFY", Product: "MIS", Quantity: -5, PnL: -250}},
		Margins:   []store.Margin{{Segment: "equity", Net: 80000, Total: 100000, Utilised: 20000}},
		UpdatedAt: testNow,
	})
	want := `
# HELP stock_portfolio_holding_pnl Unrealised profit and loss of the holding.
# TYPE stock_portfolio_holding_pnl gauge
stock_portfolio_holding_pnl{exchange="NSE",symbol="INFY"} 1500
# HELP stock_portfolio_margin_utilised Margin utilised (debits).
# TYPE stock_portfolio_margin_utilised gauge
stock_portfolio_margin_utilised{segment="equity"} 20000
# HELP stock_portfolio_position_quantity Net position quantity (negative for short).
# TYPE stock_portfolio_position_quantity gauge
stock_portfolio_position_quantity{exchange="NSE",product="MIS",symbol="INFY"} -5
`
	if err := testutil.CollectAndCompare(adapter{c, Scope{}}, strings.NewReader(want),
		"stock_portfolio_holding_pnl", "stock_portfolio_margin_utilised", "stock_portfolio_position_quantity"); err != nil {
		t.Fatal(err)
	}
}

func TestNewScope(t *testing.T) {
	if _, err := NewScope("bogus"); err == nil {
		t.Fatal("expected error for unknown tier")
	}
	s, err := NewScope(store.TierCore)
	if err != nil || !s.Includes(store.TierCore) || s.Includes(store.TierBroad) {
		t.Fatalf("unexpected scope %+v err %v", s, err)
	}
	if !(Scope{}).Includes(store.TierBroad) {
		t.Fatal("empty scope must include everything")
	}
}
