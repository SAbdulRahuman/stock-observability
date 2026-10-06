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

//go:build !nomarket

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_exporter/internal/market"
)

type marketCollector struct {
	deps                                     Deps
	open, phase, tradingDay, toClose, toOpen typedDesc
}

func init() {
	registerCollector("market", defaultEnabled, NewMarketCollector)
}

// NewMarketCollector exposes the exchange session state used to gate alerts.
func NewMarketCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	return &marketCollector{
		deps:       deps,
		open:       gaugeDesc("market", "open", "1 during the normal trading session, else 0."),
		phase:      gaugeDesc("market", "phase", "Current session phase (1 for the active phase).", "phase"),
		tradingDay: gaugeDesc("market", "trading_day", "1 if today is a trading day (weekday, not a holiday)."),
		toClose:    gaugeDesc("market", "seconds_to_close", "Seconds until the normal session closes, 0 when closed."),
		toOpen:     gaugeDesc("market", "seconds_to_open", "Seconds until the next session opens, 0 when open."),
	}, nil
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func (c *marketCollector) Update(ch chan<- prometheus.Metric, _ Scope) error {
	now := c.deps.Now()
	cal := c.deps.Calendar
	phase := cal.Phase(now)
	ch <- c.open.mustNewConstMetric(b2f(phase == market.PhaseOpen))
	for _, p := range market.Phases {
		ch <- c.phase.mustNewConstMetric(b2f(p == phase), p)
	}
	ch <- c.tradingDay.mustNewConstMetric(b2f(cal.IsTradingDay(now)))
	ch <- c.toClose.mustNewConstMetric(cal.SecondsToClose(now))
	ch <- c.toOpen.mustNewConstMetric(cal.SecondsToOpen(now))
	return nil
}
