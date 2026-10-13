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

package sim

import (
	"math"
	"math/rand"
	"time"

	"github.com/seenimo/stock_exporter/internal/daily"
	"github.com/seenimo/stock_exporter/internal/store"
)

// HistorySessions is the number of synthetic daily candles per core instrument (about one year).
const HistorySessions = 250

// Scenarios shape the last sessions of the synthetic history so every forecast rule
// (stock forecast PRD §15.3) fires on some core instruments.
const (
	scenarioRandom = iota
	scenarioTrendUp
	scenarioTrendDown
	scenarioSideways
	scenarioBreakout
	scenarioBreakdown
	scenarioVolExpansion
	scenarioSqueeze
	scenarioOversoldRebound
	scenarioOverboughtPullback
	scenarioEngulfing
	scenarioCount
)

// Scenario returns the history scenario of an instrument; indices follow a random walk.
func Scenario(in store.Instrument) int {
	if in.Type == store.TypeIndex {
		return scenarioRandom
	}
	return int(hash(in.Symbol+"scenario") % scenarioCount)
}

// History returns HistorySessions deterministic daily candles for the trading days before
// day (oldest first), ending at lastClose. Equal inputs give equal candles.
func History(in store.Instrument, day time.Time, lastClose float64, isTradingDay func(time.Time) bool) []daily.Candle {
	n := HistorySessions
	dates := make([]time.Time, 0, n)
	for d := day.AddDate(0, 0, -1); len(dates) < n; d = d.AddDate(0, 0, -1) {
		if isTradingDay(d.Add(12 * time.Hour)) {
			dates = append(dates, d)
		}
	}
	rng := rand.New(rand.NewSource(int64(hash(in.Symbol + day.Format(time.DateOnly)))))
	sigma, mu := 0.018, 0.0004*rng.NormFloat64()
	if in.Type == store.TypeIndex {
		sigma = 0.009
	}
	ret := make([]float64, n)
	volMult := make([]float64, n)
	for i := range ret {
		ret[i] = mu + sigma*rng.NormFloat64()
		volMult[i] = math.Exp(0.3 * rng.NormFloat64())
	}
	last := n - 1
	tail := func(k int, m, s float64) {
		for i := n - k; i < n; i++ {
			ret[i] = m + s*rng.NormFloat64()
		}
	}
	switch Scenario(in) {
	case scenarioTrendUp:
		tail(60, 0.004, 0.008)
	case scenarioTrendDown:
		tail(60, -0.004, 0.008)
	case scenarioSideways:
		tail(40, 0, 0.006)
	case scenarioBreakout:
		tail(20, 0, 0.008)
		ret[last], volMult[last] = 0.04, 3
	case scenarioBreakdown:
		tail(20, 0, 0.008)
		ret[last], volMult[last] = -0.04, 3
	case scenarioVolExpansion:
		tail(10, 0, 0.055)
	case scenarioSqueeze:
		tail(20, 0, 0.003)
	case scenarioOversoldRebound:
		tail(15, -0.02, 0.004)
		ret[last-1], ret[last] = 0.02, 0.025
	case scenarioOverboughtPullback:
		tail(15, 0.02, 0.004)
		ret[last-1], ret[last] = -0.02, -0.025
	case scenarioEngulfing:
		tail(30, -0.004, 0.006)
		ret[last-1], ret[last], volMult[last] = -0.01, 0.025, 2
	}

	closes := make([]float64, n)
	closes[last] = lastClose
	for i := last; i > 0; i-- {
		closes[i-1] = closes[i] / math.Exp(ret[i])
	}
	px := func(p float64) float64 {
		if in.Type == store.TypeIndex {
			return math.Round(p*100) / 100
		}
		return math.Max(round(p), 0.05)
	}
	baseVol := float64(100_000 + hash(in.Symbol)%900_000)
	out := make([]daily.Candle, n)
	for i := range out {
		c := px(closes[i])
		o := c
		if i > 0 {
			o = px(closes[i-1] * math.Exp(0.003*rng.NormFloat64()))
		}
		if Scenario(in) == scenarioEngulfing && i == last {
			// Bullish engulfing: open below the previous (red) close, close above its open.
			o = px(math.Min(out[i-1].Close, out[i-1].Open) * 0.995)
			c = math.Max(c, px(math.Max(out[i-1].Open, out[i-1].Close)*1.005))
		}
		hi := math.Max(px(math.Max(o, c)*(1+math.Abs(0.006*rng.NormFloat64()))), math.Max(o, c))
		lo := math.Min(px(math.Min(o, c)*(1-math.Abs(0.006*rng.NormFloat64()))), math.Min(o, c))
		vol := 0.0
		if in.Type != store.TypeIndex {
			vol = math.Round(baseVol * volMult[i])
		}
		out[i] = daily.Candle{Date: dates[n-1-i], Open: o, High: hi, Low: lo, Close: c, Volume: vol}
	}
	return out
}
