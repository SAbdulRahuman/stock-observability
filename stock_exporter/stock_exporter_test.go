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

package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"

	"github.com/seenimo/stock_exporter/collector"
	"github.com/seenimo/stock_exporter/internal/market"
	"github.com/seenimo/stock_exporter/internal/sim"
	"github.com/seenimo/stock_exporter/internal/store"
	"github.com/seenimo/stock_exporter/internal/watchlist"
)

func TestMain(m *testing.M) {
	// Apply collector flag defaults.
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func simHandler(t testing.TB, broad int) *handler {
	t.Helper()
	wl, err := watchlist.Load("examples/watchlist.yml")
	if err != nil {
		t.Fatal(err)
	}
	holidays, err := market.LoadHolidays("examples/holidays-nse-2026.yml")
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Asia/Kolkata")
	now := time.Date(2026, 10, 6, 11, 0, 0, 0, loc)
	st := store.New()
	st.SetInstruments(wl.Synthetic(broad))
	h := store.NewHealth("sim", "full")
	s := sim.New(sim.Config{Seed: 1, Interval: time.Second, Holdings: 5}, st, h, loc)
	for i := range 60 {
		s.Step(now.Add(time.Duration(i-60) * time.Second))
	}
	deps := collector.Deps{Store: st, Health: h, Calendar: market.NewCalendar(loc, holidays, false), Now: func() time.Time { return now }}
	hd, err := newHandler(deps, false, 0, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return hd
}

func scrape(t testing.TB, h http.Handler, query string) (int, map[string]int) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics?"+query, nil))
	if rec.Code != http.StatusOK {
		return rec.Code, nil
	}
	p := expfmt.NewTextParser(model.UTF8Validation)
	fams, err := p.TextToMetricFamilies(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for n, f := range fams {
		counts[n] = len(f.Metric)
	}
	return rec.Code, counts
}

func TestHandlerTiersAndFilters(t *testing.T) {
	h := simHandler(t, 100)

	_, all := scrape(t, h, "")
	_, core := scrape(t, h, "tier=core")
	_, broad := scrape(t, h, "tier=broad&collect[]=quote&collect[]=circuit")

	if all["stock_last_price"] != core["stock_last_price"]+broad["stock_last_price"] {
		t.Fatalf("tiers do not partition instruments: all=%d core=%d broad=%d", all["stock_last_price"], core["stock_last_price"], broad["stock_last_price"])
	}
	if broad["stock_last_price"] != 100 {
		t.Fatalf("broad tier has %d instruments, want 100", broad["stock_last_price"])
	}
	if _, ok := broad["stock_market_open"]; ok {
		t.Fatal("collect[] filter not applied to broad scrape")
	}
	if all["stock_market_open"] != 1 || all["stock_feed_connections"] != 1 {
		t.Fatalf("market/feed metrics missing: %v", all)
	}
	if _, ok := all["stock_portfolio_holding_pnl"]; ok {
		t.Fatal("portfolio collector must be disabled by default")
	}

	for _, q := range []string{"tier=bogus", "collect[]=quote&exclude[]=market", "collect[]=nope", "collect[]=depth"} {
		if code, _ := scrape(t, h, q); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", q, code)
		}
	}
}

// TestCoreTierSampleBudget documents the expected ingestion rate of the default watchlist.
func TestCoreTierSampleBudget(t *testing.T) {
	h := simHandler(t, 0)
	_, core := scrape(t, h, "tier=core")
	total := 0
	for _, n := range core {
		total += n
	}
	t.Logf("core tier: %d samples per scrape", total)
	if total < 2500 || total > 3500 {
		t.Fatalf("core tier produces %d samples per scrape, expected ~3000 (1s scrape budget)", total)
	}
}

func BenchmarkScrapeAllNSE(b *testing.B) {
	h := simHandler(b, 2400)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	b.ResetTimer()
	for range b.N {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatal(rec.Code)
		}
		_, _ = io.Copy(io.Discard, rec.Body)
	}
}
