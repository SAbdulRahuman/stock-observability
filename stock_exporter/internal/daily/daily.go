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

// Package daily computes completed-session indicators and events from daily candles
// (stock forecast PRD §7-§8). All windows count trading sessions, not calendar days.
package daily

import (
	"math"
	"time"
)

// Warm-up: smoothed indicators (RSI, MACD, ADX) and their events need this many closes.
const Warmup = 100

// Signal names used in the signal label.
const (
	SignalSMACross       = "sma_20d_50d_cross"
	SignalRSIDivergence  = "rsi_divergence"
	SignalOversoldExit   = "rsi_oversold_exit"
	SignalOverboughtExit = "rsi_overbought_exit"
	SignalMACDCross      = "macd_cross"
	SignalEngulfing      = "engulfing"
)

// Candle is one completed session, Date at 00:00 exchange time.
type Candle struct {
	Date                           time.Time
	Open, High, Low, Close, Volume float64
}

// Event is the latest occurrence of a signal; Age is in sessions (0 = latest session).
type Event struct {
	Signal  string
	Bullish bool
	Age     int
}

// Indicators describe the latest completed session. Unavailable values are NaN.
type Indicators struct {
	SessionDate time.Time
	History     int
	Valid       bool

	Close, SMA20, SMA50                           float64
	Return1D, Return5D, Return20D, Return126x21   float64
	RSI14, RelVolume20                            float64
	Vol5, Vol10, Vol20, Vol60, EWMAVol            float64
	MACDHist, ADX, DIPlus, DIMinus                float64
	Slope20, R2x20, ZScore20, Squeeze, OBVSlope20 float64
	StreakBullish, StreakBearish                  int
	Events                                        []Event
}

var nan = math.NaN()

// Compute returns the indicators of the latest candle. candles must be oldest first;
// volume indicators are computed only when equity is true (indices carry no volume).
func Compute(candles []Candle, equity bool) Indicators {
	n := len(candles)
	ind := Indicators{
		History: n, Close: nan, SMA20: nan, SMA50: nan,
		Return1D: nan, Return5D: nan, Return20D: nan, Return126x21: nan,
		RSI14: nan, RelVolume20: nan, Vol5: nan, Vol10: nan, Vol20: nan, Vol60: nan, EWMAVol: nan,
		MACDHist: nan, ADX: nan, DIPlus: nan, DIMinus: nan,
		Slope20: nan, R2x20: nan, ZScore20: nan, Squeeze: nan, OBVSlope20: nan,
	}
	if n == 0 {
		return ind
	}
	ind.SessionDate = candles[n-1].Date
	if !valid(candles) {
		return ind
	}
	ind.Valid = true
	c := make([]float64, n)
	for i, k := range candles {
		c[i] = k.Close
	}
	last := n - 1
	ind.Close = c[last]
	ret := func(from, to int) float64 {
		if from < 0 {
			return nan
		}
		return 100 * (c[to]/c[from] - 1)
	}
	ind.Return1D, ind.Return5D, ind.Return20D = ret(last-1, last), ret(last-5, last), ret(last-20, last)
	if n >= 127 {
		ind.Return126x21 = ret(last-126, last-21)
	}

	sma20, sma50 := smaSeries(c, 20), smaSeries(c, 50)
	ind.SMA20, ind.SMA50 = sma20[last], sma50[last]
	ind.StreakBullish, ind.StreakBearish = streaks(c, sma20, sma50)

	rsi := rsiSeries(c, 14)
	if n >= Warmup {
		ind.RSI14 = rsi[last]
	}

	logret := make([]float64, 0, n)
	for i := 1; i < n; i++ {
		logret = append(logret, math.Log(c[i]/c[i-1]))
	}
	ind.Vol5, ind.Vol10, ind.Vol20, ind.Vol60 = realizedVol(logret, 5), realizedVol(logret, 10), realizedVol(logret, 20), realizedVol(logret, 60)
	ind.EWMAVol = ewmaVol(logret, 0.94, 20)

	hist := macdHist(c)
	if n >= Warmup {
		ind.MACDHist = hist[last]
	}
	adx, dip, dim := adxSeries(candles, 14)
	if n >= Warmup {
		ind.ADX, ind.DIPlus, ind.DIMinus = adx[last], dip[last], dim[last]
	}

	if n >= 20 {
		w := make([]float64, 20)
		for i := range w {
			w[i] = math.Log(c[n-20+i])
		}
		slope, r2 := ols(w)
		ind.Slope20, ind.R2x20 = slope*252*100, r2
		if sd := pstdev(c[n-20:]); sd > 0 {
			ind.ZScore20 = (c[last] - sma20[last]) / sd
		}
	}
	if n >= 21 {
		atr := 0.0
		for i := n - 20; i < n; i++ {
			atr += trueRange(candles[i], candles[i-1].Close)
		}
		atr /= 20
		if atr > 0 {
			ind.Squeeze = 4 * pstdev(c[n-20:]) / (3 * atr)
		}
	}

	if equity && n >= 21 {
		base := 0.0
		for _, k := range candles[n-21 : n-1] {
			base += k.Volume
		}
		if base > 0 {
			ind.RelVolume20 = candles[last].Volume / (base / 20)
		}
		obv := make([]float64, 20)
		acc, vol := 0.0, 0.0
		for i := n - 20; i < n; i++ {
			switch {
			case c[i] > c[i-1]:
				acc += candles[i].Volume
			case c[i] < c[i-1]:
				acc -= candles[i].Volume
			}
			obv[i-(n-20)] = acc
			vol += candles[i].Volume
		}
		if vol > 0 {
			slope, _ := ols(obv)
			ind.OBVSlope20 = slope / (vol / 20)
		}
	}

	ind.Events = events(candles, c, sma20, sma50, rsi, hist)
	return ind
}

func valid(candles []Candle) bool {
	ok := func(v float64) bool { return v > 0 && !math.IsInf(v, 0) && !math.IsNaN(v) }
	for i, k := range candles {
		if !ok(k.Open) || !ok(k.High) || !ok(k.Low) || !ok(k.Close) || k.Volume < 0 || math.IsNaN(k.Volume) {
			return false
		}
		if k.High < k.Low || k.High < max(k.Open, k.Close) || k.Low > min(k.Open, k.Close) {
			return false
		}
		if i > 0 && !k.Date.After(candles[i-1].Date) {
			return false
		}
	}
	return true
}

func smaSeries(c []float64, n int) []float64 {
	out := make([]float64, len(c))
	sum := 0.0
	for i, v := range c {
		sum += v
		if i >= n {
			sum -= c[i-n]
		}
		if i >= n-1 {
			out[i] = sum / float64(n)
		} else {
			out[i] = nan
		}
	}
	return out
}

func streaks(c, sma20, sma50 []float64) (bull, bear int) {
	for i := len(c) - 1; i >= 0 && !math.IsNaN(sma50[i]) && c[i] > sma20[i] && sma20[i] > sma50[i]; i-- {
		bull++
	}
	for i := len(c) - 1; i >= 0 && !math.IsNaN(sma50[i]) && c[i] < sma20[i] && sma20[i] < sma50[i]; i-- {
		bear++
	}
	return bull, bear
}

// rsiSeries is Wilder RSI: seeded with the mean gain/loss of the first n changes.
func rsiSeries(c []float64, n int) []float64 {
	out := make([]float64, len(c))
	for i := range out {
		out[i] = nan
	}
	if len(c) <= n {
		return out
	}
	var gain, loss float64
	for i := 1; i <= n; i++ {
		d := c[i] - c[i-1]
		gain += max(d, 0)
		loss += max(-d, 0)
	}
	gain, loss = gain/float64(n), loss/float64(n)
	rs := func() float64 {
		if loss == 0 {
			if gain == 0 {
				return 50
			}
			return 100
		}
		return 100 - 100/(1+gain/loss)
	}
	out[n] = rs()
	for i := n + 1; i < len(c); i++ {
		d := c[i] - c[i-1]
		gain = (gain*float64(n-1) + max(d, 0)) / float64(n)
		loss = (loss*float64(n-1) + max(-d, 0)) / float64(n)
		out[i] = rs()
	}
	return out
}

func emaSeries(x []float64, start, n int) []float64 {
	out := make([]float64, len(x))
	for i := range out {
		out[i] = nan
	}
	if len(x)-start < n {
		return out
	}
	sum := 0.0
	for i := start; i < start+n; i++ {
		sum += x[i]
	}
	k := 2 / float64(n+1)
	out[start+n-1] = sum / float64(n)
	for i := start + n; i < len(x); i++ {
		out[i] = x[i]*k + out[i-1]*(1-k)
	}
	return out
}

// macdHist is MACD(12,26) minus its 9-period signal line; EMAs seeded with an SMA.
func macdHist(c []float64) []float64 {
	e12, e26 := emaSeries(c, 0, 12), emaSeries(c, 0, 26)
	line := make([]float64, len(c))
	for i := range c {
		line[i] = e12[i] - e26[i]
	}
	signal := emaSeries(line, 25, 9)
	out := make([]float64, len(c))
	for i := range c {
		out[i] = line[i] - signal[i]
	}
	return out
}

func trueRange(k Candle, prevClose float64) float64 {
	return max(k.High, prevClose) - min(k.Low, prevClose)
}

// adxSeries is Wilder's directional movement system.
func adxSeries(k []Candle, n int) (adx, dip, dim []float64) {
	adx, dip, dim = make([]float64, len(k)), make([]float64, len(k)), make([]float64, len(k))
	for i := range k {
		adx[i], dip[i], dim[i] = nan, nan, nan
	}
	if len(k) <= 2*n {
		return
	}
	var str, sp, sm float64
	dx := make([]float64, len(k))
	for i := 1; i < len(k); i++ {
		up, down := k[i].High-k[i-1].High, k[i-1].Low-k[i].Low
		pdm, mdm := 0.0, 0.0
		if up > down && up > 0 {
			pdm = up
		}
		if down > up && down > 0 {
			mdm = down
		}
		tr := trueRange(k[i], k[i-1].Close)
		if i <= n {
			str, sp, sm = str+tr, sp+pdm, sm+mdm
			if i < n {
				continue
			}
		} else {
			str, sp, sm = str-str/float64(n)+tr, sp-sp/float64(n)+pdm, sm-sm/float64(n)+mdm
		}
		if str == 0 {
			continue
		}
		dip[i], dim[i] = 100*sp/str, 100*sm/str
		if s := dip[i] + dim[i]; s > 0 {
			dx[i] = 100 * math.Abs(dip[i]-dim[i]) / s
		}
		switch {
		case i == 2*n-1:
			sum := 0.0
			for j := n; j <= i; j++ {
				sum += dx[j]
			}
			adx[i] = sum / float64(n)
		case i >= 2*n:
			adx[i] = (adx[i-1]*float64(n-1) + dx[i]) / float64(n)
		}
	}
	return
}

func pstdev(x []float64) float64 {
	mean := 0.0
	for _, v := range x {
		mean += v
	}
	mean /= float64(len(x))
	ss := 0.0
	for _, v := range x {
		ss += (v - mean) * (v - mean)
	}
	return math.Sqrt(ss / float64(len(x)))
}

// realizedVol is the annualised population standard deviation of the last n log returns, in percent.
func realizedVol(logret []float64, n int) float64 {
	if len(logret) < n {
		return nan
	}
	return pstdev(logret[len(logret)-n:]) * math.Sqrt(252) * 100
}

// ewmaVol is the annualised one-step EWMA volatility forecast, seeded with the variance of the first seed returns.
func ewmaVol(logret []float64, lambda float64, seed int) float64 {
	if len(logret) <= seed {
		return nan
	}
	sd := pstdev(logret[:seed])
	v := sd * sd
	for _, r := range logret[seed:] {
		v = lambda*v + (1-lambda)*r*r
	}
	return math.Sqrt(v*252) * 100
}

// ols fits y = a + b*x for x = 0..len(y)-1 and returns b and R².
func ols(y []float64) (slope, r2 float64) {
	n := float64(len(y))
	var sx, sy, sxx, sxy, syy float64
	for i, v := range y {
		x := float64(i)
		sx, sy, sxx, sxy, syy = sx+x, sy+v, sxx+x*x, sxy+x*v, syy+v*v
	}
	vx, vy, cov := sxx-sx*sx/n, syy-sy*sy/n, sxy-sx*sy/n
	if vx == 0 {
		return nan, nan
	}
	slope = cov / vx
	if vy <= 0 {
		return slope, nan
	}
	return slope, cov * cov / (vx * vy)
}

func events(k []Candle, c, sma20, sma50, rsi, hist []float64) []Event {
	last := len(c) - 1
	var out []Event
	add := func(signal string, t int, bullish bool) {
		if t >= 0 {
			out = append(out, Event{Signal: signal, Bullish: bullish, Age: last - t})
		}
	}

	t, bull := latestCross(sma20, sma50, 1)
	add(SignalSMACross, t, bull)
	t, bull = latestSignCross(hist, Warmup)
	add(SignalMACDCross, t, bull)

	exitOver, exitUnder := -1, -1
	for i := max(Warmup, 1); i <= last; i++ {
		if rsi[i-1] < 30 && rsi[i] >= 30 {
			exitUnder = i
		}
		if rsi[i-1] > 70 && rsi[i] <= 70 {
			exitOver = i
		}
	}
	add(SignalOversoldExit, exitUnder, true)
	add(SignalOverboughtExit, exitOver, false)

	t, bull = divergence(c, rsi)
	add(SignalRSIDivergence, t, bull)

	eng, engBull := -1, false
	for i := 1; i <= last; i++ {
		p, q := k[i-1], k[i]
		switch {
		case q.Close > q.Open && p.Close < p.Open && q.Open <= p.Close && q.Close >= p.Open:
			eng, engBull = i, true
		case q.Close < q.Open && p.Close > p.Open && q.Open >= p.Close && q.Close <= p.Open:
			eng, engBull = i, false
		}
	}
	add(SignalEngulfing, eng, engBull)
	return out
}

// latestCross finds the latest t where a moved from <= b to > b (bullish) or from >= b to < b.
func latestCross(a, b []float64, from int) (int, bool) {
	for i := len(a) - 1; i >= max(from, 1); i-- {
		if math.IsNaN(a[i-1]) || math.IsNaN(b[i-1]) {
			break
		}
		switch {
		case a[i-1] <= b[i-1] && a[i] > b[i]:
			return i, true
		case a[i-1] >= b[i-1] && a[i] < b[i]:
			return i, false
		}
	}
	return -1, false
}

func latestSignCross(x []float64, from int) (int, bool) {
	zero := make([]float64, len(x))
	return latestCross(x, zero, from)
}

// divergence replays confirmed price pivots (3 sessions each side, confirmed 3 sessions later)
// and returns the session of the latest still-valid divergence event (PRD §7.9).
func divergence(c, rsi []float64) (int, bool) {
	const k, minGap, maxGap = 3, 5, 40
	event, bullish := -1, false
	lastHigh, lastLow := -1, -1
	for s := 2 * k; s < len(c); s++ {
		p := s - k
		isHigh, isLow := true, true
		for j := 1; j <= k; j++ {
			isHigh = isHigh && c[p] > c[p-j] && c[p] > c[p+j]
			isLow = isLow && c[p] < c[p-j] && c[p] < c[p+j]
		}
		rsiOK := func(i int) bool { return i >= Warmup-1 && !math.IsNaN(rsi[i]) }
		if isHigh {
			if gap := p - lastHigh; lastHigh >= 0 && gap >= minGap && gap <= maxGap && rsiOK(lastHigh) && rsiOK(p) &&
				c[p] > c[lastHigh] && rsi[p] < rsi[lastHigh] {
				event, bullish = s, false
			} else if event >= 0 && !bullish {
				event = -1
			}
			lastHigh = p
		}
		if isLow {
			if gap := p - lastLow; lastLow >= 0 && gap >= minGap && gap <= maxGap && rsiOK(lastLow) && rsiOK(p) &&
				c[p] < c[lastLow] && rsi[p] > rsi[lastLow] {
				event, bullish = s, true
			} else if event >= 0 && bullish {
				event = -1
			}
			lastLow = p
		}
	}
	return event, bullish
}
