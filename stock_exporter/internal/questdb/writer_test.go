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

package questdb

import (
	"context"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/seenimo/stock_exporter/internal/daily"
	"github.com/seenimo/stock_exporter/internal/store"
)

func TestQuoteLineEscapesTagsAndIncludesQuoteAndDepth(t *testing.T) {
	at := time.Date(2026, 10, 21, 10, 0, 0, 123, time.UTC)
	line := quoteLine(
		store.Instrument{
			Token: 42, Exchange: "NSE", Symbol: "A B,C=1", Sector: "IT",
			Type: store.TypeEquity, Tier: store.TierCore,
		},
		store.Quote{
			LastPrice: 10.5, Volume: 100, ReceivedAt: at,
			LastTradeTime: at.Add(-time.Second), ExchangeTime: at.Add(-500 * time.Millisecond), HasDepth: true,
			Depth: store.Depth{Bids: [5]store.DepthLevel{{Price: 10.4, Quantity: 5, Orders: 2}}},
		},
	)
	for _, part := range []string{
		`stock_quotes,exchange=NSE,symbol=A\ B\,C\=1,sector=IT,instrument_type=EQ,tier=core token=42i,last_price=10.5`,
		`,volume=100,`,
		`,has_depth=true,`,
		`,bid1_price=10.4,bid1_quantity=5,bid1_orders=2`,
		",last_trade_time=" + strconv.FormatInt(at.Add(-time.Second).UnixNano(), 10) + "i",
		",exchange_time=" + strconv.FormatInt(at.Add(-500*time.Millisecond).UnixNano(), 10) + "i",
		" " + strconv.FormatInt(at.UnixNano(), 10),
	} {
		if !strings.Contains(line, part) {
			t.Errorf("line %q does not contain %q", line, part)
		}
	}
}

func TestWriterPostsBatchesToQuestDB(t *testing.T) {
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/write" || r.URL.Query().Get("precision") != "ns" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
		}
		requests <- string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	writer, err := New(server.URL, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		writer.Run(ctx)
		close(done)
	}()
	st := store.New()
	st.SetInstruments([]store.Instrument{{Token: 1, Exchange: "NSE", Symbol: "INFY"}})
	st.SetObserver(writer.Observe)
	st.UpdateQuote(1, store.Quote{
		LastPrice: 1500, ReceivedAt: time.Now(),
	})
	select {
	case body := <-requests:
		if !strings.Contains(body, "stock_quotes,exchange=NSE,symbol=INFY") || !strings.Contains(body, "last_price=1500") {
			t.Fatalf("unexpected QuestDB line: %s", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for QuestDB write")
	}
	cancel()
	<-done
}

func TestLinesForMarketStateChanges(t *testing.T) {
	instrument := store.Instrument{Token: 42, Exchange: "NSE", Symbol: "INFY", Sector: "IT", Type: store.TypeEquity}
	at := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	changes := []struct {
		change store.Change
		want   []string
	}{
		{
			change: store.Change{Kind: store.ChangeCircuit, Instrument: instrument, Circuit: store.Circuit{Upper: 110, Lower: 90, UpdatedAt: at}},
			want:   []string{"stock_circuits", "upper=110", "lower=90"},
		},
		{
			change: store.Change{Kind: store.ChangeReference, Instrument: instrument, Reference: store.Reference{High52W: 120, Low52W: 80, SMA50D: 100, Cross: 1, UpdatedAt: at}},
			want:   []string{"stock_references", "high_52w=120", "sma_50d=100", "cross=1i"},
		},
		{
			change: store.Change{Kind: store.ChangeDaily, Instrument: instrument, Daily: daily.Indicators{
				SessionDate: at, History: 40, Valid: true, Close: 100, RSI14: math.NaN(),
				Events: []daily.Event{{Signal: "golden cross", Bullish: true, Age: 2}},
			}},
			want: []string{"stock_daily_indicators", "history=40i", "valid=true", "close=100", "stock_daily_signals", "signal=golden\\ cross", "bullish=true"},
		},
		{
			change: store.Change{Kind: store.ChangePortfolio, Portfolio: store.Portfolio{
				UpdatedAt: at,
				Holdings:  []store.Holding{{Exchange: "NSE", Symbol: "INFY", Quantity: 5, AveragePrice: 99, LastPrice: 100, PnL: 5}},
				Positions: []store.Position{{Exchange: "NFO", Symbol: "NIFTY", Product: "MIS", Quantity: -1, M2M: 2}},
				Margins:   []store.Margin{{Segment: "equity", Net: 500, Total: 600, Utilised: 100}},
			}},
			want: []string{"kind=snapshot", "holding_count=1i", "kind=holding", "quantity=5", "kind=position", "product=MIS", "kind=margin", "segment=equity"},
		},
	}
	for _, tc := range changes {
		lines := linesForChange(tc.change, at)
		joined := strings.Join(lines, "\n")
		for _, want := range tc.want {
			if !strings.Contains(joined, want) {
				t.Errorf("kind %q output %q does not contain %q", tc.change.Kind, joined, want)
			}
		}
	}
}

func TestNewRejectsNonBaseURLs(t *testing.T) {
	for _, endpoint := range []string{"", "localhost:9000", "ftp://questdb:9000", "http://questdb:9000/write"} {
		if _, err := New(endpoint, slog.New(slog.DiscardHandler)); err == nil {
			t.Errorf("New(%q) succeeded, want error", endpoint)
		}
	}
}
