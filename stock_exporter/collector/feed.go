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

//go:build !nofeed

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_exporter/internal/store"
)

type feedCollector struct {
	deps                                         Deps
	info, connections, ticks, reconnects, errors typedDesc
	subscribed, lastTick, instruments            typedDesc
	sessionValid, sessionExpiry, apiRequests     typedDesc
}

func init() {
	registerCollector("feed", defaultEnabled, NewFeedCollector)
}

// NewFeedCollector exposes the health of the market data feed and Kite session.
func NewFeedCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	return &feedCollector{
		deps:          deps,
		info:          gaugeDesc("feed", "info", "Market data feed source and mode.", "source", "mode"),
		connections:   gaugeDesc("feed", "connections", "Number of connected market data WebSocket connections."),
		ticks:         counterDesc("feed", "ticks_total", "Ticks received for tracked instruments."),
		reconnects:    counterDesc("feed", "reconnects_total", "WebSocket reconnect attempts."),
		errors:        counterDesc("feed", "errors_total", "WebSocket errors."),
		subscribed:    gaugeDesc("feed", "subscribed_instruments", "Instruments subscribed on the feed."),
		lastTick:      gaugeDesc("feed", "last_tick_timestamp_seconds", "Time the most recent tick was received."),
		instruments:   gaugeDesc("feed", "instruments", "Resolved instruments per tier.", "tier"),
		sessionValid:  gaugeDesc("kite", "session_valid", "1 if a non-expired Kite access token is held."),
		sessionExpiry: gaugeDesc("kite", "session_expiry_timestamp_seconds", "Expiry time of the Kite access token."),
		apiRequests:   counterDesc("kite", "api_requests_total", "Kite REST API requests by endpoint and outcome.", "endpoint", "outcome"),
	}, nil
}

func (c *feedCollector) Update(ch chan<- prometheus.Metric, _ Scope) error {
	h := c.deps.Health.Snapshot()
	ch <- c.info.mustNewConstMetric(1, h.Source, h.Mode)
	ch <- c.connections.mustNewConstMetric(float64(h.Connected))
	ch <- c.ticks.mustNewConstMetric(float64(h.Ticks))
	ch <- c.reconnects.mustNewConstMetric(float64(h.Reconnects))
	ch <- c.errors.mustNewConstMetric(float64(h.Errors))
	ch <- c.subscribed.mustNewConstMetric(float64(h.Subscribed))
	if !h.LastTick.IsZero() {
		ch <- c.lastTick.mustNewConstMetric(unixSeconds(h.LastTick))
	}
	tiers := map[string]float64{store.TierCore: 0, store.TierBroad: 0}
	for _, in := range c.deps.Store.Instruments() {
		tiers[in.Tier]++
	}
	for t, n := range tiers {
		ch <- c.instruments.mustNewConstMetric(n, t)
	}
	ch <- c.sessionValid.mustNewConstMetric(b2f(h.SessionValid))
	if !h.SessionExpiry.IsZero() {
		ch <- c.sessionExpiry.mustNewConstMetric(unixSeconds(h.SessionExpiry))
	}
	for _, a := range h.API {
		ch <- c.apiRequests.mustNewConstMetric(a.Count, a.Endpoint, a.Outcome)
	}
	return nil
}
