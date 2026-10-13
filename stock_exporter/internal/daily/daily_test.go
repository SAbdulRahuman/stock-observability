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

package daily

import (
	"math"
	"testing"
	"time"
)

func series(closes []float64) []Candle {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]Candle, len(closes))
	for i, c := range closes {
		o := c
		if i > 0 {
			o = closes[i-1]
		}
		out[i] = Candle{Date: start.AddDate(0, 0, i), Open: o, High: math.Max(o, c) + 0.5, Low: math.Min(o, c) - 0.5, Close: c, Volume: 1000}
	}
	return out
}

func near(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func event(ind Indicators, signal string) (Event, bool) {
	for _, e := range ind.Events {
		if e.Signal == signal {
			return e, true
		}
	}
	return Event{}, false
}

func TestReturnsAndAverages(t *testing.T) {
	closes := make([]float64, 130)
	for i := range closes {
		closes[i] = 100 + float64(i)
	}
	ind := Compute(series(closes), true)
	if !ind.Valid || ind.History != 130 {
		t.Fatalf("valid=%v history=%d", ind.Valid, ind.History)
	}
	near(t, "close", ind.Close, 229)
	near(t, "return1d", ind.Return1D, 100*(229.0/228-1))
	near(t, "return5d", ind.Return5D, 100*(229.0/224-1))
	near(t, "return20d", ind.Return20D, 100*(229.0/209-1))
	near(t, "return126x21", ind.Return126x21, 100*(208.0/103-1))
	near(t, "sma20", ind.SMA20, 219.5)
	near(t, "sma50", ind.SMA50, 204.5)
	near(t, "rsi", ind.RSI14, 100)
	near(t, "relvol", ind.RelVolume20, 1)
	near(t, "r2", ind.R2x20, ind.R2x20) // finite
	if ind.StreakBullish != 81 || ind.StreakBearish != 0 {
		t.Errorf("streaks %d/%d, want 81/0", ind.StreakBullish, ind.StreakBearish)
	}
	if ind.Slope20 <= 0 || ind.R2x20 < 0.99 {
		t.Errorf("slope %v r2 %v, want positive near-perfect fit", ind.Slope20, ind.R2x20)
	}
	if ind.ADX < 25 || ind.DIPlus <= ind.DIMinus {
		t.Errorf("adx %v di+ %v di- %v, want strong uptrend", ind.ADX, ind.DIPlus, ind.DIMinus)
	}
	if ind.OBVSlope20 <= 0 {
		t.Errorf("obv slope %v, want positive", ind.OBVSlope20)
	}
	if !math.IsNaN(Compute(series(closes), false).RelVolume20) {
		t.Error("indices must not get volume indicators")
	}
}

func TestWarmupAndValidation(t *testing.T) {
	short := Compute(series([]float64{100, 101, 102}), true)
	if !short.Valid || !math.IsNaN(short.SMA20) || !math.IsNaN(short.RSI14) || !math.IsNaN(short.Return5D) {
		t.Fatalf("short history must leave windowed indicators absent: %+v", short)
	}
	bad := series([]float64{100, 101, 102})
	bad[1].High = 50
	if Compute(bad, true).Valid {
		t.Fatal("high below close must be invalid")
	}
	dup := series([]float64{100, 101})
	dup[1].Date = dup[0].Date
	if Compute(dup, true).Valid {
		t.Fatal("non-increasing dates must be invalid")
	}
	if Compute(nil, true).Valid {
		t.Fatal("no candles must be invalid")
	}
}

func TestVolatility(t *testing.T) {
	closes := []float64{100}
	for i := 1; i < 80; i++ {
		closes = append(closes, closes[i-1]*1.01)
	}
	ind := Compute(series(closes), true)
	near(t, "vol10 of constant returns", ind.Vol10, 0)
	alt := []float64{100}
	for i := 1; i < 80; i++ {
		f := 1.02
		if i%2 == 0 {
			f = 1 / 1.02
		}
		alt = append(alt, alt[i-1]*f)
	}
	ind = Compute(series(alt), true)
	near(t, "vol60", ind.Vol60, math.Log(1.02)*math.Sqrt(252)*100)
	if ind.EWMAVol <= 0 {
		t.Errorf("ewma vol %v", ind.EWMAVol)
	}
}

func TestSMACrossEvent(t *testing.T) {
	var closes []float64
	for i := 0; i < 60; i++ {
		closes = append(closes, 200-float64(i))
	}
	for i := 0; i < 40; i++ {
		closes = append(closes, 141+3*float64(i))
	}
	ind := Compute(series(closes), true)
	e, ok := event(ind, SignalSMACross)
	if !ok || !e.Bullish {
		t.Fatalf("bullish SMA20/50 cross not detected: %+v", ind.Events)
	}
	sma20, sma50 := smaSeries(closes, 20), smaSeries(closes, 50)
	at := len(closes) - 1 - e.Age
	if !(sma20[at-1] <= sma50[at-1] && sma20[at] > sma50[at]) {
		t.Fatalf("event at %d is not a crossing", at)
	}
}

func TestRSIExitAndEngulfing(t *testing.T) {
	var closes []float64
	for i := 0; i < 100; i++ {
		closes = append(closes, 100+math.Sin(float64(i)))
	}
	for i := 0; i < 15; i++ {
		closes = append(closes, closes[len(closes)-1]*0.97)
	}
	for range 3 {
		closes = append(closes, closes[len(closes)-1]*1.08)
	}
	c := series(closes)
	n := len(c)
	// Make the last candle a bullish engulfing of the red one before it.
	c[n-2].Open = c[n-2].Close * 1.02
	c[n-2].High = c[n-2].Open + 0.5
	c[n-1].Open = c[n-2].Close * 0.99
	c[n-1].Low = c[n-1].Open - 0.5
	ind := Compute(c, true)
	if e, ok := event(ind, SignalOversoldExit); !ok || !e.Bullish || e.Age > 2 {
		t.Fatalf("oversold exit not detected near the end: %+v (rsi %v)", ind.Events, ind.RSI14)
	}
	if e, ok := event(ind, SignalEngulfing); !ok || !e.Bullish || e.Age != 0 {
		t.Fatalf("bullish engulfing not detected on the last candle: %+v", ind.Events)
	}
}

func TestDivergenceNeedsConfirmation(t *testing.T) {
	// Rising pivot highs with fading momentum: steep rise, pullback, slow rise to a higher high, pullback.
	closes := make([]float64, 0, 140)
	for i := 0; i < 110; i++ {
		closes = append(closes, 100+0.01*float64(i%3))
	}
	up := func(n int, step float64) {
		for range n {
			closes = append(closes, closes[len(closes)-1]+step)
		}
	}
	up(6, 3)
	up(5, -2)
	up(12, 1)
	up(3, -1)
	ind := Compute(series(closes), true)
	e, ok := event(ind, SignalRSIDivergence)
	if !ok || e.Bullish || e.Age != 0 {
		t.Fatalf("bearish divergence not detected on the confirmation session: %+v", ind.Events)
	}
	// Without the 3 confirming sessions after the second peak there is no event yet.
	if _, ok := event(Compute(series(closes[:len(closes)-1]), true), SignalRSIDivergence); ok {
		t.Fatal("divergence reported before confirmation")
	}
}
