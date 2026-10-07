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

//go:build !noorderbook

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_exporter/internal/store"
)

type orderbookCollector struct {
	deps Deps

	spread, spreadBps, mid, microprice, topImbalance, imbalance    typedDesc
	quantity, orders, notional, weightedPrice, avgOrderQty, levels typedDesc
}

func init() {
	registerCollector("orderbook", defaultEnabled, NewOrderbookCollector)
}

// NewOrderbookCollector exposes KPIs derived from the top 5 levels of the order book.
func NewOrderbookCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	l := instrumentLabels
	ls := append(append([]string{}, instrumentLabels...), "side")
	return &orderbookCollector{
		deps:          deps,
		spread:        gaugeDesc("orderbook", "spread_price", "Best ask minus best bid; zero or negative means a locked or crossed book.", l...),
		spreadBps:     gaugeDesc("orderbook", "spread_bps", "Bid-ask spread relative to the mid price, in basis points.", l...),
		mid:           gaugeDesc("orderbook", "mid_price", "Midpoint of best bid and best ask.", l...),
		microprice:    gaugeDesc("orderbook", "microprice", "Best bid and ask weighted by the opposite side's level 1 quantity.", l...),
		topImbalance:  gaugeDesc("orderbook", "top_imbalance_ratio", "Level 1 (bid - ask) / (bid + ask) quantity, from -1 (all asks) to 1 (all bids).", l...),
		imbalance:     gaugeDesc("orderbook", "imbalance_ratio", "Top 5 levels (bid - ask) / (bid + ask) quantity, from -1 (all asks) to 1 (all bids).", l...),
		quantity:      gaugeDesc("orderbook", "quantity", "Quantity across the top 5 levels of a side.", ls...),
		orders:        gaugeDesc("orderbook", "orders", "Number of orders across the top 5 levels of a side.", ls...),
		notional:      gaugeDesc("orderbook", "notional", "Price times quantity across the top 5 levels of a side.", ls...),
		weightedPrice: gaugeDesc("orderbook", "weighted_average_price", "Quantity weighted average price of the top 5 levels of a side.", ls...),
		avgOrderQty:   gaugeDesc("orderbook", "average_order_quantity", "Average quantity per order across the top 5 levels of a side.", ls...),
		levels:        gaugeDesc("orderbook", "levels", "Number of non-empty levels (0-5) of a side.", ls...),
	}, nil
}

type bookSide struct {
	quantity, orders, notional, levels float64
}

func summarize(levels [5]store.DepthLevel) bookSide {
	var s bookSide
	for _, l := range levels {
		if l.Empty() {
			continue
		}
		s.quantity += l.Quantity
		s.orders += l.Orders
		s.notional += l.Price * l.Quantity
		s.levels++
	}
	return s
}

func (c *orderbookCollector) Update(ch chan<- prometheus.Metric, scope Scope) error {
	for _, e := range c.deps.Store.Snapshot() {
		if !e.HasQuote || !e.Quote.HasDepth || !scope.Includes(e.Instrument.Tier) {
			continue
		}
		lv := instrumentLabelValues(e.Instrument)
		d := e.Quote.Depth
		bids, asks := summarize(d.Bids), summarize(d.Asks)
		for _, s := range []struct {
			name string
			bookSide
		}{{"bid", bids}, {"ask", asks}} {
			slv := append(append([]string{}, lv...), s.name)
			ch <- c.quantity.mustNewConstMetric(s.quantity, slv...)
			ch <- c.orders.mustNewConstMetric(s.orders, slv...)
			ch <- c.notional.mustNewConstMetric(s.notional, slv...)
			ch <- c.levels.mustNewConstMetric(s.levels, slv...)
			if s.quantity > 0 {
				ch <- c.weightedPrice.mustNewConstMetric(s.notional/s.quantity, slv...)
			}
			if s.orders > 0 {
				ch <- c.avgOrderQty.mustNewConstMetric(s.quantity/s.orders, slv...)
			}
		}

		bid, ask := d.Bids[0], d.Asks[0]
		if bid.Empty() || ask.Empty() {
			continue
		}
		mid := (bid.Price + ask.Price) / 2
		ch <- c.spread.mustNewConstMetric(ask.Price-bid.Price, lv...)
		ch <- c.mid.mustNewConstMetric(mid, lv...)
		if mid > 0 {
			ch <- c.spreadBps.mustNewConstMetric((ask.Price-bid.Price)/mid*1e4, lv...)
		}
		if top := bid.Quantity + ask.Quantity; top > 0 {
			ch <- c.microprice.mustNewConstMetric((bid.Price*ask.Quantity+ask.Price*bid.Quantity)/top, lv...)
			ch <- c.topImbalance.mustNewConstMetric((bid.Quantity-ask.Quantity)/top, lv...)
		}
		if total := bids.quantity + asks.quantity; total > 0 {
			ch <- c.imbalance.mustNewConstMetric((bids.quantity-asks.quantity)/total, lv...)
		}
	}
	return nil
}
