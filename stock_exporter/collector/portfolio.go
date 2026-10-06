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

//go:build !noportfolio

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

type portfolioCollector struct {
	deps Deps

	hQty, hAvg, hLast, hPnL, hDay typedDesc
	pQty, pAvg, pPnL, pM2M        typedDesc
	mNet, mTotal, mUsed, updated  typedDesc
}

func init() {
	// Disabled by default: exposes account data.
	registerCollector("portfolio", defaultDisabled, NewPortfolioCollector)
}

// NewPortfolioCollector exposes holdings, positions and margins. Amounts are in INR.
func NewPortfolioCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	h := []string{"exchange", "symbol"}
	p := []string{"exchange", "symbol", "product"}
	return &portfolioCollector{
		deps:    deps,
		hQty:    gaugeDesc("portfolio", "holding_quantity", "Quantity held (including T1).", h...),
		hAvg:    gaugeDesc("portfolio", "holding_average_price", "Average buy price of the holding.", h...),
		hLast:   gaugeDesc("portfolio", "holding_last_price", "Last price reported for the holding.", h...),
		hPnL:    gaugeDesc("portfolio", "holding_pnl", "Unrealised profit and loss of the holding.", h...),
		hDay:    gaugeDesc("portfolio", "holding_day_change", "Change in holding value today.", h...),
		pQty:    gaugeDesc("portfolio", "position_quantity", "Net position quantity (negative for short).", p...),
		pAvg:    gaugeDesc("portfolio", "position_average_price", "Average price of the position.", p...),
		pPnL:    gaugeDesc("portfolio", "position_pnl", "Profit and loss of the position.", p...),
		pM2M:    gaugeDesc("portfolio", "position_m2m", "Mark-to-market of the position.", p...),
		mNet:    gaugeDesc("portfolio", "margin_net", "Free margin available for trading.", "segment"),
		mTotal:  gaugeDesc("portfolio", "margin_total", "Total margin (free plus utilised).", "segment"),
		mUsed:   gaugeDesc("portfolio", "margin_utilised", "Margin utilised (debits).", "segment"),
		updated: gaugeDesc("portfolio", "last_update_timestamp_seconds", "Time the portfolio was last refreshed."),
	}, nil
}

func (c *portfolioCollector) Update(ch chan<- prometheus.Metric, _ Scope) error {
	p, ok := c.deps.Store.Portfolio()
	if !ok {
		return ErrNoData
	}
	seen := map[string]bool{}
	for _, h := range p.Holdings {
		k := h.Exchange + ":" + h.Symbol
		if seen[k] {
			continue
		}
		seen[k] = true
		ch <- c.hQty.mustNewConstMetric(h.Quantity, h.Exchange, h.Symbol)
		ch <- c.hAvg.mustNewConstMetric(h.AveragePrice, h.Exchange, h.Symbol)
		ch <- c.hLast.mustNewConstMetric(h.LastPrice, h.Exchange, h.Symbol)
		ch <- c.hPnL.mustNewConstMetric(h.PnL, h.Exchange, h.Symbol)
		ch <- c.hDay.mustNewConstMetric(h.DayChange, h.Exchange, h.Symbol)
	}
	clear(seen)
	for _, x := range p.Positions {
		k := x.Exchange + ":" + x.Symbol + ":" + x.Product
		if seen[k] {
			continue
		}
		seen[k] = true
		ch <- c.pQty.mustNewConstMetric(x.Quantity, x.Exchange, x.Symbol, x.Product)
		ch <- c.pAvg.mustNewConstMetric(x.AveragePrice, x.Exchange, x.Symbol, x.Product)
		ch <- c.pPnL.mustNewConstMetric(x.PnL, x.Exchange, x.Symbol, x.Product)
		ch <- c.pM2M.mustNewConstMetric(x.M2M, x.Exchange, x.Symbol, x.Product)
	}
	for _, m := range p.Margins {
		ch <- c.mNet.mustNewConstMetric(m.Net, m.Segment)
		ch <- c.mTotal.mustNewConstMetric(m.Total, m.Segment)
		ch <- c.mUsed.mustNewConstMetric(m.Utilised, m.Segment)
	}
	ch <- c.updated.mustNewConstMetric(unixSeconds(p.UpdatedAt))
	return nil
}
