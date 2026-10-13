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

// Package sim generates a deterministic synthetic market feed for development,
// tests, load testing and demos outside market hours.
package sim

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/seenimo/stock_exporter/internal/daily"
	"github.com/seenimo/stock_exporter/internal/store"
)

// Config controls the simulator.
type Config struct {
	Seed             int64
	Interval         time.Duration
	EventProbability float64
	Holdings         int
	// IsTradingDay selects the dates of the synthetic daily history; nil means weekdays.
	IsTradingDay func(time.Time) bool
}

type state struct {
	in        store.Instrument
	spot      uint32 // underlying spot token of a derivative
	oi        float64
	day       string
	last      float64
	open      float64
	high      float64
	low       float64
	prevClose float64
	volume    float64
	turnover  float64
	bias      float64
	sigma     float64
}

type holding struct {
	token uint32
	qty   float64
	avg   float64
}

// Simulator writes synthetic ticks into a store.
type Simulator struct {
	cfg      Config
	st       *store.Store
	health   *store.Health
	loc      *time.Location
	rng      *rand.Rand
	states   map[uint32]*state
	order    []uint32
	holdings []holding
}

// New creates a simulator for the instruments currently in the store.
func New(cfg Config, st *store.Store, health *store.Health, loc *time.Location) *Simulator {
	s := &Simulator{cfg: cfg, st: st, health: health, loc: loc, rng: rand.New(rand.NewSource(cfg.Seed)), states: map[uint32]*state{}}
	if s.cfg.IsTradingDay == nil {
		s.cfg.IsTradingDay = func(t time.Time) bool { return t.Weekday() != time.Saturday && t.Weekday() != time.Sunday }
	}
	spots := map[string]uint32{}
	var derivatives []uint32
	for _, in := range st.Instruments() {
		base := basePrice(in)
		x := &state{in: in, last: base, prevClose: base, sigma: sigmaPerTick(in, cfg.Interval)}
		s.states[in.Token] = x
		if in.IsDerivative() {
			x.oi = float64(100_000 + hash(in.Symbol)%900_000)
			derivatives = append(derivatives, in.Token)
			continue
		}
		spots[underlyingOf(in.Symbol)] = in.Token
		s.order = append(s.order, in.Token)
		if in.Type == store.TypeEquity && in.Tier == store.TierCore && len(s.holdings) < cfg.Holdings {
			s.holdings = append(s.holdings, holding{token: in.Token, qty: float64(10 * (1 + s.rng.Intn(10))), avg: round(base * (0.9 + 0.2*s.rng.Float64()))})
		}
	}
	// Derivatives tick after their underlying.
	for _, tok := range derivatives {
		x := s.states[tok]
		if spot, ok := spots[x.in.Underlying]; ok {
			x.spot = spot
			x.last = derivativePrice(x.in, s.states[spot].last, time.Now())
			x.prevClose = x.last
			s.order = append(s.order, tok)
		}
	}
	health.SetSubscribed(len(s.order))
	return s
}

// underlyingOf maps a spot symbol to the NFO underlying name.
func underlyingOf(symbol string) string {
	switch symbol {
	case "NIFTY 50":
		return "NIFTY"
	case "NIFTY BANK":
		return "BANKNIFTY"
	}
	return symbol
}

// Derivatives returns a monthly future and a 7-strike option chain (calls and puts) for NIFTY 50,
// expiring on the last Tuesday of the month, with tokens after the highest existing token.
func Derivatives(ins []store.Instrument, now time.Time) []store.Instrument {
	var next uint32
	var spot *store.Instrument
	for i := range ins {
		next = max(next, ins[i].Token)
		if ins[i].Symbol == "NIFTY 50" {
			spot = &ins[i]
		}
	}
	if spot == nil {
		return nil
	}
	expiry := lastTuesday(now.Year(), now.Month())
	if now.After(expiry.Add(16 * time.Hour)) {
		expiry = lastTuesday(now.Year(), now.Month()+1)
	}
	prefix := "NIFTY" + expiry.Format("06Jan")
	newIn := func(symbol, typ string, strike float64) store.Instrument {
		next++
		return store.Instrument{Token: next, Exchange: "NFO", Symbol: strings.ToUpper(symbol), Sector: "Index", Type: typ, Tier: store.TierCore,
			Underlying: "NIFTY", Expiry: expiry, Strike: strike}
	}
	out := []store.Instrument{newIn(prefix+"FUT", store.TypeFuture, 0)}
	atm := math.Round(basePrice(*spot)/50) * 50
	for k := -3; k <= 3; k++ {
		strike := atm + float64(k)*50
		out = append(out, newIn(fmt.Sprintf("%s%.0fCE", prefix, strike), store.TypeCall, strike), newIn(fmt.Sprintf("%s%.0fPE", prefix, strike), store.TypePut, strike))
	}
	return out
}

func lastTuesday(y int, m time.Month) time.Time {
	d := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC)
	for d.Weekday() != time.Tuesday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// derivativePrice prices a future at 7% carry and an option as intrinsic value plus a
// Gaussian-shaped time value (15% volatility), both from the underlying spot.
func derivativePrice(in store.Instrument, spot float64, now time.Time) float64 {
	t := math.Max(in.Expiry.Add(10*time.Hour).Sub(now).Hours()/24/365, 1.0/365)
	if in.Type == store.TypeFuture {
		return round(spot * (1 + 0.07*t))
	}
	sd := spot * 0.15 * math.Sqrt(t)
	tv := 0.4 * sd * math.Exp(-math.Pow((spot-in.Strike)/sd, 2)/2)
	if in.Type == store.TypeCall {
		return round(math.Max(spot-in.Strike, 0) + tv)
	}
	return round(math.Max(in.Strike-spot, 0) + tv)
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func basePrice(in store.Instrument) float64 {
	switch in.Symbol {
	case "NIFTY 50":
		return 25000
	case "NIFTY BANK":
		return 55000
	case "SENSEX":
		return 82000
	case "INDIA VIX":
		return 14
	}
	h := float64(hash(in.Symbol))
	if in.Type == store.TypeIndex {
		return 10000 + math.Mod(h, 40000)
	}
	return round(50 + math.Mod(h, 3000))
}

// Annualised volatility scaled to one tick and amplified so movements are visible in demos.
func sigmaPerTick(in store.Instrument, interval time.Duration) float64 {
	annual := 0.30
	switch {
	case in.Symbol == "INDIA VIX":
		annual = 0.80
	case in.Type == store.TypeIndex:
		annual = 0.15
	}
	const tradingSecondsPerYear = 252 * 6.25 * 3600
	const amplify = 5
	return amplify * annual * math.Sqrt(interval.Seconds()/tradingSecondsPerYear)
}

func round(p float64) float64 { return math.Round(p*20) / 20 }

// Run ticks until ctx is cancelled.
func (s *Simulator) Run(ctx context.Context) {
	s.health.SetConnected(0, true)
	defer s.health.SetConnected(0, false)
	t := time.NewTicker(s.cfg.Interval)
	defer t.Stop()
	s.Step(time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.Step(now)
		}
	}
}

// Step generates one tick for every instrument at the given time.
func (s *Simulator) Step(now time.Time) {
	day := now.In(s.loc).Format("2006-01-02")
	for _, tok := range s.order {
		x := s.states[tok]
		if x.day != day {
			s.rollover(x, day, now)
		}
		if x.spot != 0 {
			x.last = derivativePrice(x.in, s.states[x.spot].last, now)
			x.oi = math.Round(x.oi * math.Exp(0.002*s.rng.NormFloat64()))
		} else {
			ret := x.bias + x.sigma*s.rng.NormFloat64()
			if s.cfg.EventProbability > 0 && s.rng.Float64() < s.cfg.EventProbability {
				ret += math.Copysign(0.02+0.03*s.rng.Float64(), s.rng.Float64()-0.5)
			}
			x.last = x.last * math.Exp(ret)
			if x.in.Type == store.TypeIndex {
				x.last = math.Round(x.last*100) / 100
			} else {
				x.last = math.Min(math.Max(round(x.last), round(x.prevClose*0.9)), round(x.prevClose*1.1))
			}
		}
		x.high = math.Max(x.high, x.last)
		x.low = math.Min(x.low, x.last)

		q := store.Quote{
			LastPrice: x.last, Open: x.open, High: x.high, Low: x.low, PrevClose: x.prevClose, OpenInterest: x.oi,
			LastTradeTime: now, ExchangeTime: now, ReceivedAt: now,
		}
		if x.in.Type != store.TypeIndex {
			lastQty := float64(1 + s.rng.Intn(500))
			x.volume += lastQty * float64(1+hash(x.in.Symbol)%10)
			x.turnover += lastQty * float64(1+hash(x.in.Symbol)%10) * x.last
			q.Volume = x.volume
			q.LastQuantity = lastQty
			q.AveragePrice = math.Round(x.turnover/x.volume*100) / 100
			imbalance := 0.5 + s.rng.Float64()
			q.BuyQuantity = math.Round(100000 * imbalance)
			q.SellQuantity = math.Round(100000 / imbalance)
			q.Depth, q.HasDepth = s.depth(x.last), true
		}
		s.st.UpdateQuote(tok, q)
	}
	s.health.AddTicks(len(s.order), now)
	s.updatePortfolio(now)
}

func (s *Simulator) rollover(x *state, day string, now time.Time) {
	if x.day != "" {
		x.prevClose = x.last
	}
	x.day, x.open, x.high, x.low, x.volume, x.turnover = day, x.last, x.last, x.last, 0, 0
	// Small per-day drift so some instruments trend and trigger crossover/drawdown rules.
	x.bias = x.sigma * 0.05 * s.rng.NormFloat64()
	if x.in.Type != store.TypeIndex {
		s.st.UpdateCircuit(x.in.Token, store.Circuit{Upper: round(x.prevClose * 1.1), Lower: round(x.prevClose * 0.9), UpdatedAt: now})
	}
	if x.in.Tier == store.TierCore && !x.in.IsDerivative() {
		// Hash-seeded history (not s.rng) so the tick sequence of existing seeds is unchanged.
		date, _ := time.ParseInLocation(time.DateOnly, day, s.loc)
		candles := History(x.in, date, x.prevClose, s.cfg.IsTradingDay)
		highs, lows, closes := make([]float64, len(candles)), make([]float64, len(candles)), make([]float64, len(candles))
		for i, k := range candles {
			highs[i], lows[i], closes[i] = k.High, k.Low, k.Close
		}
		if r, ok := store.NewReference(highs, lows, closes, now); ok {
			s.st.UpdateReference(x.in.Token, r)
		}
		s.st.UpdateDaily(x.in.Token, daily.Compute(candles, x.in.Type == store.TypeEquity))
	}
}

func (s *Simulator) depth(last float64) store.Depth {
	var d store.Depth
	for i := range 5 {
		off := 0.05 * float64(i+1)
		d.Bids[i] = store.DepthLevel{Price: round(last - off), Quantity: float64(100 + s.rng.Intn(5000)), Orders: float64(1 + s.rng.Intn(50))}
		d.Asks[i] = store.DepthLevel{Price: round(last + off), Quantity: float64(100 + s.rng.Intn(5000)), Orders: float64(1 + s.rng.Intn(50))}
	}
	return d
}

func (s *Simulator) updatePortfolio(now time.Time) {
	if len(s.holdings) == 0 {
		return
	}
	p := store.Portfolio{UpdatedAt: now}
	var invested float64
	for i, h := range s.holdings {
		x := s.states[h.token]
		p.Holdings = append(p.Holdings, store.Holding{
			Exchange: x.in.Exchange, Symbol: x.in.Symbol, Quantity: h.qty, AveragePrice: h.avg, LastPrice: x.last,
			PnL: (x.last - h.avg) * h.qty, DayChange: (x.last - x.prevClose) * h.qty,
		})
		if i == 0 {
			p.Positions = append(p.Positions, store.Position{
				Exchange: x.in.Exchange, Symbol: x.in.Symbol, Product: "MIS", Quantity: h.qty, AveragePrice: x.open, LastPrice: x.last,
				PnL: (x.last - x.open) * h.qty, M2M: (x.last - x.open) * h.qty,
			})
			invested = x.open * h.qty / 5
		}
	}
	const net = 500000.0
	p.Margins = []store.Margin{{Segment: "equity", Net: net - invested, Total: net, Utilised: invested}}
	s.st.SetPortfolio(p)
}
