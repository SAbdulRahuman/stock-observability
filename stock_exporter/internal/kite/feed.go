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
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zerodha/gokiteconnect/v4/models"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"

	"github.com/seenimo/stock_exporter/internal/store"
)

// Kite WebSocket limits per API key.
const (
	MaxTokensPerConnection = 3000
	MaxConnections         = 3
)

const shardRestartDelay = 10 * time.Second

// ParseMode validates a ticker mode name.
func ParseMode(m string) (kiteticker.Mode, error) {
	switch kiteticker.Mode(m) {
	case kiteticker.ModeFull, kiteticker.ModeQuote, kiteticker.ModeLTP:
		return kiteticker.Mode(m), nil
	}
	return "", fmt.Errorf("unknown ticker mode %q (want full, quote or ltp)", m)
}

// Shard splits tokens across WebSocket connections within Kite's limits.
func Shard(tokens []uint32) ([][]uint32, error) {
	if len(tokens) > MaxTokensPerConnection*MaxConnections {
		return nil, fmt.Errorf("%d instruments exceed Kite limit of %d", len(tokens), MaxTokensPerConnection*MaxConnections)
	}
	var out [][]uint32
	for len(tokens) > 0 {
		n := min(len(tokens), MaxTokensPerConnection)
		out = append(out, tokens[:n])
		tokens = tokens[n:]
	}
	return out, nil
}

// ConvertTick maps a Kite tick to the store representation.
func ConvertTick(t models.Tick, received time.Time) store.Quote {
	q := store.Quote{
		LastPrice:     t.LastPrice,
		Open:          t.OHLC.Open,
		High:          t.OHLC.High,
		Low:           t.OHLC.Low,
		PrevClose:     t.OHLC.Close,
		AveragePrice:  t.AverageTradePrice,
		Volume:        float64(t.VolumeTraded),
		LastQuantity:  float64(t.LastTradedQuantity),
		BuyQuantity:   float64(t.TotalBuyQuantity),
		SellQuantity:  float64(t.TotalSellQuantity),
		OpenInterest:  float64(t.OI),
		LastTradeTime: t.LastTradeTime.Time,
		ExchangeTime:  t.Timestamp.Time,
		ReceivedAt:    received,
	}
	if t.Mode == string(kiteticker.ModeFull) && !t.IsIndex {
		q.HasDepth = true
		for i := range 5 {
			b, a := t.Depth.Buy[i], t.Depth.Sell[i]
			q.Depth.Bids[i] = store.DepthLevel{Price: b.Price, Quantity: float64(b.Quantity), Orders: float64(b.Orders)}
			q.Depth.Asks[i] = store.DepthLevel{Price: a.Price, Quantity: float64(a.Quantity), Orders: float64(a.Orders)}
		}
	}
	return q
}

// FeedConfig configures the WebSocket feed.
type FeedConfig struct {
	APIKey  string
	Mode    kiteticker.Mode
	RootURL *url.URL
}

// RunFeed streams ticks for tokens into the store until ctx is cancelled.
func RunFeed(ctx context.Context, cfg FeedConfig, accessToken string, tokens []uint32, st *store.Store, health *store.Health, logger *slog.Logger) error {
	shards, err := Shard(tokens)
	if err != nil {
		return err
	}
	health.SetSubscribed(len(tokens))
	var wg sync.WaitGroup
	for i, sh := range shards {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runShard(ctx, i, cfg, accessToken, sh, st, health, logger.With("connection", i))
		}()
	}
	wg.Wait()
	return nil
}

func runShard(ctx context.Context, idx int, cfg FeedConfig, accessToken string, tokens []uint32, st *store.Store, health *store.Health, logger *slog.Logger) {
	defer health.SetConnected(idx, false)
	for ctx.Err() == nil {
		t := kiteticker.New(cfg.APIKey, accessToken)
		if cfg.RootURL != nil {
			t.SetRootURL(*cfg.RootURL)
		}
		// The ticker only checks ctx between reads; closing the socket unblocks a pending read.
		var (
			connMu sync.Mutex
			conn   *websocket.Conn
		)
		stop := context.AfterFunc(ctx, func() {
			connMu.Lock()
			defer connMu.Unlock()
			if conn != nil {
				conn.Close()
			}
		})
		t.OnConnect(func() {
			connMu.Lock()
			conn = t.Conn
			connMu.Unlock()
			health.SetConnected(idx, true)
			logger.Info("Kite WebSocket connected", "instruments", len(tokens))
			if err := t.Subscribe(tokens); err != nil {
				logger.Error("Subscribe failed", "err", err)
				health.IncErrors()
				return
			}
			if err := t.SetMode(cfg.Mode, tokens); err != nil {
				logger.Error("Set mode failed", "err", err)
				health.IncErrors()
			}
		})
		t.OnTick(func(tick models.Tick) {
			now := time.Now()
			if st.UpdateQuote(tick.InstrumentToken, ConvertTick(tick, now)) {
				health.AddTicks(1, now)
			}
		})
		t.OnError(func(err error) {
			health.IncErrors()
			logger.Warn("Kite WebSocket error", "err", err)
		})
		t.OnClose(func(code int, reason string) {
			health.SetConnected(idx, false)
			logger.Warn("Kite WebSocket closed", "code", code, "reason", reason)
		})
		t.OnReconnect(func(attempt int, delay time.Duration) {
			health.SetConnected(idx, false)
			health.IncReconnects()
			logger.Info("Kite WebSocket reconnecting", "attempt", attempt, "delay", delay)
		})
		t.OnNoReconnect(func(attempt int) {
			logger.Error("Kite WebSocket gave up reconnecting", "attempts", attempt)
		})
		t.ServeWithContext(ctx)
		stop()
		health.SetConnected(idx, false)
		select {
		case <-ctx.Done():
		case <-time.After(shardRestartDelay):
		}
	}
}
