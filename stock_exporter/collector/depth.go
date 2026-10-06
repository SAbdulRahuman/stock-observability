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

//go:build !nodepth

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_exporter/internal/store"
)

type depthCollector struct {
	deps                    Deps
	price, quantity, orders typedDesc
}

func init() {
	// Disabled by default: 30 series per instrument.
	registerCollector("depth", defaultDisabled, NewDepthCollector)
}

// NewDepthCollector exposes the top 5 levels of the order book.
func NewDepthCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	l := append(append([]string{}, instrumentLabels...), "side", "level")
	return &depthCollector{
		deps:     deps,
		price:    gaugeDesc("depth", "price", "Order book price at level.", l...),
		quantity: gaugeDesc("depth", "quantity", "Order book quantity at level.", l...),
		orders:   gaugeDesc("depth", "orders", "Number of orders at level.", l...),
	}, nil
}

var levelNames = [5]string{"1", "2", "3", "4", "5"}

func (c *depthCollector) Update(ch chan<- prometheus.Metric, scope Scope) error {
	for _, e := range c.deps.Store.Snapshot() {
		if !e.HasQuote || !e.Quote.HasDepth || !scope.Includes(e.Instrument.Tier) {
			continue
		}
		base := instrumentLabelValues(e.Instrument)
		for _, s := range []struct {
			name   string
			levels [5]store.DepthLevel
		}{{"bid", e.Quote.Depth.Bids}, {"ask", e.Quote.Depth.Asks}} {
			for i, lvl := range s.levels {
				lv := append(append([]string{}, base...), s.name, levelNames[i])
				ch <- c.price.mustNewConstMetric(lvl.Price, lv...)
				ch <- c.quantity.mustNewConstMetric(lvl.Quantity, lv...)
				ch <- c.orders.mustNewConstMetric(lvl.Orders, lv...)
			}
		}
	}
	return nil
}
