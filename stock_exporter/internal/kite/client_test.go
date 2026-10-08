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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"

	"github.com/seenimo/stock_exporter/internal/store"
	"github.com/seenimo/stock_exporter/internal/watchlist"
)

const instrumentsCSV = `instrument_token,exchange_token,tradingsymbol,name,last_price,expiry,strike,tick_size,lot_size,instrument_type,segment,exchange
408065,1594,INFY,INFOSYS,0,,0,0.05,1,EQ,NSE,NSE
2953217,11536,TCS,TATA CONSULTANCY SERV LT,0,,0,0.05,1,EQ,NSE,NSE
256265,0,NIFTY 50,NIFTY 50,0,,0,0,0,EQ,INDICES,NSE
`

func fakeKite(t *testing.T, quoteCalls *atomic.Int32, tokenOK *atomic.Bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	ok := func(w http.ResponseWriter, data string) {
		if !tokenOK.Load() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"status":"error","message":"Incorrect api_key or access_token.","error_type":"TokenException"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"success","data":%s}`, data)
	}
	mux.HandleFunc("/instruments/NSE", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token key:tok" {
			http.Error(w, "unauthorised", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		fmt.Fprint(w, instrumentsCSV)
	})
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		if quoteCalls.Add(1) == 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"status":"error","message":"Too many requests","error_type":"NetworkException"}`)
			return
		}
		var parts []string
		for _, i := range r.URL.Query()["i"] {
			tok := map[string]int{"NSE:INFY": 408065, "NSE:TCS": 2953217}[i]
			parts = append(parts, fmt.Sprintf(`%q:{"instrument_token":%d,"upper_circuit_limit":1700.5,"lower_circuit_limit":1391.3}`, i, tok))
		}
		ok(w, "{"+strings.Join(parts, ",")+"}")
	})
	mux.HandleFunc("/portfolio/holdings", func(w http.ResponseWriter, _ *http.Request) {
		ok(w, `[{"tradingsymbol":"INFY","exchange":"NSE","quantity":10,"t1_quantity":2,"average_price":1400,"last_price":1550,"pnl":1800,"day_change":5}]`)
	})
	mux.HandleFunc("/portfolio/positions", func(w http.ResponseWriter, _ *http.Request) {
		ok(w, `{"net":[{"tradingsymbol":"TCS","exchange":"NSE","product":"MIS","quantity":-5,"average_price":4000,"last_price":3990,"pnl":50,"m2m":50}],"day":[]}`)
	})
	mux.HandleFunc("/user/margins", func(w http.ResponseWriter, _ *http.Request) {
		ok(w, `{"equity":{"enabled":true,"net":80000,"utilised":{"debits":20000}},"commodity":{"enabled":false}}`)
	})
	// 60 daily candles for INFY (close 100..159, high +1, low -1); the index fails.
	mux.HandleFunc("/instruments/historical/408065/day", func(w http.ResponseWriter, _ *http.Request) {
		var rows []string
		for i := range 60 {
			c := 100 + i
			rows = append(rows, fmt.Sprintf(`["2026-01-%02dT00:00:00+0530",%d,%d,%d,%d,1000]`, 1+i%28, c, c+1, c-1, c))
		}
		ok(w, `{"candles":[`+strings.Join(rows, ",")+`]}`)
	})
	mux.HandleFunc("/instruments/historical/256265/day", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"status":"error","message":"Insufficient permission for that call.","error_type":"PermissionException"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newTestClient(t *testing.T) (*Client, *store.Health, *atomic.Int32, *atomic.Bool, *atomic.Int32) {
	t.Helper()
	var quoteCalls, tokenErrors atomic.Int32
	var tokenOK atomic.Bool
	tokenOK.Store(true)
	srv := fakeKite(t, &quoteCalls, &tokenOK)
	api := kiteconnect.New("key")
	api.SetAccessToken("tok")
	api.SetBaseURI(srv.URL)
	h := store.NewHealth("kite", "full")
	c := NewClient(api, h, func() { tokenErrors.Add(1) })
	c.quote.every, c.other.every, c.historical.every = 0, 0, 0
	return c, h, &quoteCalls, &tokenOK, &tokenErrors
}

func apiCount(h *store.Health, endpoint, outcome string) float64 {
	for _, a := range h.Snapshot().API {
		if a.Endpoint == endpoint && a.Outcome == outcome {
			return a.Count
		}
	}
	return 0
}

func TestInstrumentsAndCircuits(t *testing.T) {
	c, h, _, _, _ := newTestClient(t)
	wl, err := watchlist.Parse([]byte("instruments:\n  - {exchange: NSE, symbol: INFY, sector: IT}\n  - {exchange: NSE, symbol: NIFTY 50}\n  - {exchange: NSE, symbol: GONE}\nuniverses:\n  - {exchange: NSE, segment: NSE, instrument_types: [EQ]}\n"))
	if err != nil {
		t.Fatal(err)
	}
	ins, missing, err := c.Instruments(context.Background(), wl)
	if err != nil {
		t.Fatal(err)
	}
	if len(ins) != 3 || len(missing) != 1 || missing[0] != "NSE:GONE" {
		t.Fatalf("instruments %+v missing %v", ins, missing)
	}
	if ins[1].Type != store.TypeIndex || ins[2].Symbol != "TCS" || ins[2].Tier != store.TierBroad {
		t.Fatalf("unexpected resolution %+v", ins)
	}

	st := store.New()
	st.SetInstruments(ins)
	if err := c.RefreshCircuits(context.Background(), st); err != nil {
		t.Fatal(err)
	}
	for _, e := range st.Snapshot() {
		if e.Instrument.Type == store.TypeIndex {
			if e.HasCircuit {
				t.Fatal("index must not be queried for circuits")
			}
			continue
		}
		if !e.HasCircuit || e.Circuit.Upper != 1700.5 || e.Circuit.Lower != 1391.3 {
			t.Fatalf("%s circuit %+v", e.Instrument.Symbol, e.Circuit)
		}
	}
	if apiCount(h, EndpointInstruments, store.OutcomeSuccess) != 1 || apiCount(h, EndpointQuote, store.OutcomeSuccess) != 1 {
		t.Fatalf("api counts %+v", h.Snapshot().API)
	}

	// Second quote call is rate limited by the fake server.
	if err := c.RefreshCircuits(context.Background(), st); err == nil {
		t.Fatal("expected rate limit error")
	}
	if apiCount(h, EndpointQuote, store.OutcomeRateLimited) != 1 {
		t.Fatalf("rate limit not recorded: %+v", h.Snapshot().API)
	}
}

func TestReference(t *testing.T) {
	c, h, _, _, _ := newTestClient(t)
	st := store.New()
	st.SetInstruments([]store.Instrument{
		{Token: 408065, Exchange: "NSE", Symbol: "INFY", Type: store.TypeEquity, Tier: store.TierCore},
		{Token: 256265, Exchange: "NSE", Symbol: "NIFTY 50", Type: store.TypeIndex, Tier: store.TierCore},
		{Token: 2953217, Exchange: "NSE", Symbol: "TCS", Type: store.TypeEquity, Tier: store.TierBroad},
	})
	if err := c.RefreshReference(context.Background(), st, time.Now()); err == nil || !strings.Contains(err.Error(), "NIFTY 50") {
		t.Fatalf("expected the index failure to be reported, got %v", err)
	}
	for _, e := range st.Snapshot() {
		switch e.Instrument.Symbol {
		case "INFY":
			r := e.Reference
			if !e.HasReference || r.High52W != 160 || r.Low52W != 99 || r.SMA50D != 134.5 || r.SMA200D != 0 {
				t.Fatalf("INFY reference %+v", r)
			}
		default:
			if e.HasReference {
				t.Fatalf("%s must have no reference", e.Instrument.Symbol)
			}
		}
	}
	if apiCount(h, EndpointHistorical, store.OutcomeSuccess) != 1 || apiCount(h, EndpointHistorical, store.OutcomeError) != 1 {
		t.Fatalf("api counts %+v", h.Snapshot().API)
	}
}

func TestPortfolio(t *testing.T) {
	c, _, _, _, _ := newTestClient(t)
	st := store.New()
	if err := c.RefreshPortfolio(context.Background(), st); err != nil {
		t.Fatal(err)
	}
	p, ok := st.Portfolio()
	if !ok {
		t.Fatal("portfolio not set")
	}
	if len(p.Holdings) != 1 || p.Holdings[0].Quantity != 12 || p.Holdings[0].DayChange != 60 {
		t.Fatalf("holdings %+v", p.Holdings)
	}
	if len(p.Positions) != 1 || p.Positions[0].Quantity != -5 || p.Positions[0].Product != "MIS" {
		t.Fatalf("positions %+v", p.Positions)
	}
	if len(p.Margins) != 1 || p.Margins[0].Segment != "equity" || p.Margins[0].Total != 100000 || p.Margins[0].Utilised != 20000 {
		t.Fatalf("margins %+v", p.Margins)
	}
}

func TestTokenErrorInvalidatesSession(t *testing.T) {
	c, h, _, tokenOK, tokenErrors := newTestClient(t)
	tokenOK.Store(false)
	err := c.RefreshPortfolio(context.Background(), store.New())
	if !IsTokenError(err) {
		t.Fatalf("expected token error, got %v", err)
	}
	if tokenErrors.Load() != 1 {
		t.Fatal("onTokenError not called")
	}
	if apiCount(h, EndpointHoldings, store.OutcomeTokenError) != 1 {
		t.Fatalf("token error not recorded: %+v", h.Snapshot().API)
	}
}

func TestLimiterRespectsContext(t *testing.T) {
	l := &limiter{every: 1 << 40}
	ctx, cancel := context.WithCancel(context.Background())
	if err := l.wait(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := l.wait(ctx); err == nil {
		t.Fatal("expected context error")
	}
}
