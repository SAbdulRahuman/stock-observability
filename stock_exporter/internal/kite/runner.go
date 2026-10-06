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
	"log/slog"
	"sync"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"

	"github.com/seenimo/stock_exporter/internal/store"
	"github.com/seenimo/stock_exporter/internal/watchlist"
)

const resolveRetryDelay = 30 * time.Second

// RunnerConfig configures the Kite pipeline.
type RunnerConfig struct {
	Feed            FeedConfig
	BaseURI         string
	CircuitInterval time.Duration
	PortfolioPoll   bool
	PortfolioEvery  time.Duration
}

// Run starts the feed and pollers whenever a valid session exists and restarts
// them when the session changes, until ctx is cancelled.
func Run(ctx context.Context, cfg RunnerConfig, sess *Session, wl *watchlist.Watchlist, st *store.Store, health *store.Health, logger *slog.Logger) {
	for ctx.Err() == nil {
		token, ok := sess.Token()
		if !ok {
			logger.Warn("No valid Kite session; open /login in a browser to authenticate")
			select {
			case <-ctx.Done():
				return
			case <-sess.Changed():
				continue
			}
		}
		runSession(ctx, cfg, token, sess, wl, st, health, logger)
	}
}

func runSession(ctx context.Context, cfg RunnerConfig, token string, sess *Session, wl *watchlist.Watchlist, st *store.Store, health *store.Health, logger *slog.Logger) {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-cctx.Done():
		case <-sess.Changed():
			cancel()
		}
	}()

	api := kiteconnect.New(cfg.Feed.APIKey)
	api.SetAccessToken(token)
	if cfg.BaseURI != "" {
		api.SetBaseURI(cfg.BaseURI)
	}
	client := NewClient(api, health, func() { sess.Invalidate("Kite rejected the access token") })

	for {
		ins, missing, err := client.Instruments(cctx, wl)
		if err == nil {
			if len(missing) > 0 {
				logger.Warn("Watchlist instruments not found in instrument master", "count", len(missing), "instruments", missing)
			}
			st.SetInstruments(ins)
			logger.Info("Resolved watchlist", "instruments", len(ins))
			break
		}
		logger.Error("Resolving watchlist failed", "err", err)
		select {
		case <-cctx.Done():
			return
		case <-time.After(resolveRetryDelay):
		}
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := RunFeed(cctx, cfg.Feed, token, st.Tokens(), st, health, logger); err != nil {
			logger.Error("Kite feed failed", "err", err)
		}
	})
	wg.Go(func() {
		Poll(cctx, cfg.CircuitInterval, EndpointQuote, logger, func(c context.Context) error { return client.RefreshCircuits(c, st) })
	})
	if cfg.PortfolioPoll {
		wg.Go(func() {
			Poll(cctx, cfg.PortfolioEvery, "portfolio", logger, func(c context.Context) error { return client.RefreshPortfolio(c, st) })
		})
	}
	<-cctx.Done()
	wg.Wait()
	health.ResetConnections()
}
