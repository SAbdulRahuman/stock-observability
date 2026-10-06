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
	"encoding/binary"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zerodha/gokiteconnect/v4/models"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"

	"github.com/seenimo/stock_exporter/internal/store"
)

func TestShard(t *testing.T) {
	tokens := make([]uint32, 7000)
	sh, err := Shard(tokens)
	if err != nil {
		t.Fatal(err)
	}
	if len(sh) != 3 || len(sh[0]) != 3000 || len(sh[2]) != 1000 {
		t.Fatalf("unexpected shards %d", len(sh))
	}
	if _, err := Shard(make([]uint32, 9001)); err == nil {
		t.Fatal("expected limit error")
	}
}

func TestParseMode(t *testing.T) {
	for _, m := range []string{"full", "quote", "ltp"} {
		if _, err := ParseMode(m); err != nil {
			t.Error(err)
		}
	}
	if _, err := ParseMode("depth"); err == nil {
		t.Error("expected error")
	}
}

func TestConvertTick(t *testing.T) {
	now := time.Now()
	tk := models.Tick{
		Mode: "full", InstrumentToken: 1, LastPrice: 100, VolumeTraded: 500, AverageTradePrice: 99.5,
		TotalBuyQuantity: 10, TotalSellQuantity: 20, LastTradedQuantity: 3, OI: 7,
		OHLC:  models.OHLC{Open: 98, High: 101, Low: 97, Close: 95},
		Depth: models.Depth{Buy: [5]models.DepthItem{{Price: 99.95, Quantity: 4, Orders: 1}}},
	}
	q := ConvertTick(tk, now)
	if q.LastPrice != 100 || q.PrevClose != 95 || q.Volume != 500 || q.OpenInterest != 7 || !q.HasDepth || q.Depth.Bids[0].Price != 99.95 || q.ReceivedAt != now {
		t.Fatalf("unexpected %+v", q)
	}
	tk.IsIndex = true
	if ConvertTick(tk, now).HasDepth {
		t.Fatal("index ticks have no depth")
	}
}

// ltpFrame builds a Kite binary message with one 8-byte LTP packet; prices are in paise.
func ltpFrame(token uint32, paise uint32) []byte {
	b := make([]byte, 2+2+8)
	binary.BigEndian.PutUint16(b[0:2], 1)
	binary.BigEndian.PutUint16(b[2:4], 8)
	binary.BigEndian.PutUint32(b[4:8], token)
	binary.BigEndian.PutUint32(b[8:12], paise)
	return b
}

func TestRunFeedAgainstFakeTicker(t *testing.T) {
	// Token 408065 is on NSE (segment 1), so prices are divided by 100.
	const token = 408065
	subscribed := make(chan []uint32, 1)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "key" || r.URL.Query().Get("access_token") != "tok" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			var m struct {
				A string          `json:"a"`
				V json.RawMessage `json:"v"`
			}
			if json.Unmarshal(msg, &m) == nil && m.A == "subscribe" {
				var toks []uint32
				_ = json.Unmarshal(m.V, &toks)
				subscribed <- toks
				if err := c.WriteMessage(websocket.BinaryMessage, ltpFrame(token, 155025)); err != nil {
					return
				}
			}
		}
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	st := store.New()
	st.SetInstruments([]store.Instrument{{Token: token, Exchange: "NSE", Symbol: "INFY", Type: store.TypeEquity, Tier: store.TierCore}})
	h := store.NewHealth("kite", "ltp")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = RunFeed(ctx, FeedConfig{APIKey: "key", Mode: kiteticker.ModeLTP, RootURL: u}, "tok", st.Tokens(), st, h, slog.New(slog.DiscardHandler))
		close(done)
	}()

	select {
	case toks := <-subscribed:
		if len(toks) != 1 || toks[0] != token {
			t.Fatalf("subscribed %v", toks)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no subscription received")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		e := st.Snapshot()[0]
		if e.HasQuote {
			if e.Quote.LastPrice != 1550.25 {
				t.Fatalf("last price %v", e.Quote.LastPrice)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no tick stored")
		}
		time.Sleep(10 * time.Millisecond)
	}
	snap := h.Snapshot()
	if snap.Connected != 1 || snap.Ticks != 1 || snap.Subscribed != 1 {
		t.Fatalf("health %+v", snap)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("feed did not stop")
	}
	if h.Snapshot().Connected != 0 {
		t.Fatal("connection still reported up after stop")
	}
}
