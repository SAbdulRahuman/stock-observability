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

//go:build !noquote

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_exporter/internal/store"
)

var instrumentLabels = []string{"exchange", "symbol", "sector", "instrument_type", "tier"}

func instrumentLabelValues(i store.Instrument) []string {
	return []string{i.Exchange, i.Symbol, i.Sector, i.Type, i.Tier}
}

type quoteCollector struct {
	deps Deps

	last, open, high, low, prevClose, change typedDesc
	volume, vwap, buyQty, sellQty, lastQty   typedDesc
	openInterest, lastTrade, tickAge         typedDesc
}

func init() {
	registerCollector("quote", defaultEnabled, NewQuoteCollector)
}

// NewQuoteCollector exposes the latest tick of every instrument.
func NewQuoteCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	l := instrumentLabels
	return &quoteCollector{
		deps:         deps,
		last:         gaugeDesc("", "last_price", "Last traded price (index value for indices).", l...),
		open:         gaugeDesc("", "open_price", "Day open price.", l...),
		high:         gaugeDesc("", "high_price", "Day high price.", l...),
		low:          gaugeDesc("", "low_price", "Day low price.", l...),
		prevClose:    gaugeDesc("", "previous_close_price", "Previous session close price.", l...),
		change:       gaugeDesc("", "change_percent", "Change of last price versus previous close, in percent.", l...),
		volume:       gaugeDesc("", "day_volume_shares", "Shares traded today; resets every session.", l...),
		vwap:         gaugeDesc("", "average_traded_price", "Volume weighted average traded price today (VWAP).", l...),
		buyQty:       gaugeDesc("", "total_buy_quantity", "Total pending buy quantity in the order book.", l...),
		sellQty:      gaugeDesc("", "total_sell_quantity", "Total pending sell quantity in the order book.", l...),
		lastQty:      gaugeDesc("", "last_traded_quantity", "Quantity of the last trade.", l...),
		openInterest: gaugeDesc("", "open_interest", "Open interest of derivative contracts.", l...),
		lastTrade:    gaugeDesc("", "last_trade_timestamp_seconds", "Exchange time of the last trade.", l...),
		tickAge:      gaugeDesc("", "tick_age_seconds", "Seconds since the last tick was received (since exporter start if none yet).", l...),
	}, nil
}

func (c *quoteCollector) Update(ch chan<- prometheus.Metric, scope Scope) error {
	now := c.deps.Now()
	for _, e := range c.deps.Store.Snapshot() {
		if !scope.Includes(e.Instrument.Tier) {
			continue
		}
		lv := instrumentLabelValues(e.Instrument)
		if !e.HasQuote {
			ch <- c.tickAge.mustNewConstMetric(now.Sub(c.deps.Store.CreatedAt()).Seconds(), lv...)
			continue
		}
		q := e.Quote
		ch <- c.tickAge.mustNewConstMetric(now.Sub(q.ReceivedAt).Seconds(), lv...)
		ch <- c.last.mustNewConstMetric(q.LastPrice, lv...)
		ch <- c.open.mustNewConstMetric(q.Open, lv...)
		ch <- c.high.mustNewConstMetric(q.High, lv...)
		ch <- c.low.mustNewConstMetric(q.Low, lv...)
		ch <- c.prevClose.mustNewConstMetric(q.PrevClose, lv...)
		if q.PrevClose > 0 {
			ch <- c.change.mustNewConstMetric((q.LastPrice-q.PrevClose)/q.PrevClose*100, lv...)
		}
		switch e.Instrument.Type {
		case store.TypeIndex:
			continue
		case store.TypeFuture, store.TypeCall, store.TypePut:
			ch <- c.openInterest.mustNewConstMetric(q.OpenInterest, lv...)
		}
		ch <- c.volume.mustNewConstMetric(q.Volume, lv...)
		ch <- c.vwap.mustNewConstMetric(q.AveragePrice, lv...)
		ch <- c.buyQty.mustNewConstMetric(q.BuyQuantity, lv...)
		ch <- c.sellQty.mustNewConstMetric(q.SellQuantity, lv...)
		ch <- c.lastQty.mustNewConstMetric(q.LastQuantity, lv...)
		if !q.LastTradeTime.IsZero() {
			ch <- c.lastTrade.mustNewConstMetric(unixSeconds(q.LastTradeTime), lv...)
		}
	}
	return nil
}
