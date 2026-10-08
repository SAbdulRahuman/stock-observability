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
	"testing"
	"time"

	"github.com/seenimo/stock_exporter/internal/store"
)

func newSim(t *testing.T, seed int64) (*Simulator, *store.Store) {
	t.Helper()
	st := store.New()
	st.SetInstruments([]store.Instrument{
		{Token: 1, Exchange: "NSE", Symbol: "INFY", Type: store.TypeEquity, Tier: store.TierCore},
		{Token: 2, Exchange: "NSE", Symbol: "NIFTY 50", Type: store.TypeIndex, Tier: store.TierCore},
	})
	return New(Config{Seed: seed, Interval: time.Second, EventProbability: 0.01, Holdings: 1}, st, store.NewHealth("sim", "full"), time.UTC), st
}

func TestDeterministic(t *testing.T) {
	a, sa := newSim(t, 42)
	b, sb := newSim(t, 42)
	start := time.Date(2026, 10, 6, 4, 0, 0, 0, time.UTC)
	for i := range 100 {
		now := start.Add(time.Duration(i) * time.Second)
		a.Step(now)
		b.Step(now)
	}
	ea, eb := sa.Snapshot(), sb.Snapshot()
	for i := range ea {
		if ea[i].Quote != eb[i].Quote {
			t.Fatalf("same seed produced different quotes: %+v vs %+v", ea[i].Quote, eb[i].Quote)
		}
	}
}

func TestInvariants(t *testing.T) {
	s, st := newSim(t, 7)
	start := time.Date(2026, 10, 6, 4, 0, 0, 0, time.UTC)
	for i := range 2000 {
		s.Step(start.Add(time.Duration(i) * time.Second))
	}
	for _, e := range st.Snapshot() {
		q := e.Quote
		if !e.HasQuote || q.LastPrice <= 0 {
			t.Fatalf("%s: no positive price", e.Instrument.Symbol)
		}
		if q.Low > q.LastPrice || q.High < q.LastPrice || q.Low > q.Open || q.High < q.Open {
			t.Fatalf("%s: OHLC inconsistent %+v", e.Instrument.Symbol, q)
		}
		if e.Instrument.Type == store.TypeEquity {
			if !e.HasCircuit || q.LastPrice > e.Circuit.Upper || q.LastPrice < e.Circuit.Lower {
				t.Fatalf("%s: price %v outside circuit %+v", e.Instrument.Symbol, q.LastPrice, e.Circuit)
			}
			if q.Volume <= 0 || q.AveragePrice <= 0 {
				t.Fatalf("%s: missing volume/vwap", e.Instrument.Symbol)
			}
		} else if e.HasCircuit || q.Volume != 0 {
			t.Fatalf("%s: index must not have circuit or volume", e.Instrument.Symbol)
		}
	}
	if p, ok := st.Portfolio(); !ok || len(p.Holdings) != 1 || len(p.Positions) != 1 {
		t.Fatalf("unexpected portfolio %+v", p)
	}
}

func TestDayRollover(t *testing.T) {
	s, st := newSim(t, 1)
	day1 := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	s.Step(day1)
	s.Step(day1.Add(time.Second))
	last := st.Snapshot()[0].Quote.LastPrice
	s.Step(day1.Add(24 * time.Hour))
	q := st.Snapshot()[0].Quote
	if q.PrevClose != last {
		t.Fatalf("prev close = %v, want %v", q.PrevClose, last)
	}
}

func TestDerivatives(t *testing.T) {
	now := time.Date(2026, 10, 6, 4, 0, 0, 0, time.UTC)
	spot := []store.Instrument{{Token: 2, Exchange: "NSE", Symbol: "NIFTY 50", Type: store.TypeIndex, Tier: store.TierCore}}
	ins := Derivatives(spot, now)
	if len(ins) != 15 || ins[0].Type != store.TypeFuture || ins[0].Symbol != "NIFTY26OCTFUT" || ins[0].Token != 3 {
		t.Fatalf("future %+v of %d contracts", ins[0], len(ins))
	}
	if exp := ins[0].Expiry; exp.Weekday() != time.Tuesday || exp.Day() != 27 {
		t.Fatalf("expiry %v, want the last Tuesday (27 Oct 2026)", exp)
	}
	if ins[1].Symbol != "NIFTY26OCT24850CE" || ins[2].Type != store.TypePut || ins[1].Underlying != "NIFTY" {
		t.Fatalf("options %+v %+v", ins[1], ins[2])
	}
	if Derivatives(nil, now) != nil {
		t.Fatal("no NIFTY 50, no derivatives")
	}

	st := store.New()
	st.SetInstruments(append(spot, ins...))
	s := New(Config{Seed: 1, Interval: time.Second}, st, store.NewHealth("sim", "full"), time.UTC)
	for i := range 100 {
		s.Step(now.Add(time.Duration(i) * time.Second))
	}
	var nifty float64
	for _, e := range st.Snapshot() {
		if e.Instrument.Symbol == "NIFTY 50" {
			nifty = e.Quote.LastPrice
		}
	}
	for _, e := range st.Snapshot() {
		in, q := e.Instrument, e.Quote
		if !in.IsDerivative() {
			continue
		}
		switch {
		case q.LastPrice <= 0 || q.OpenInterest <= 0:
			t.Fatalf("%s: price %v OI %v", in.Symbol, q.LastPrice, q.OpenInterest)
		case in.Type == store.TypeFuture && q.LastPrice <= nifty:
			t.Fatalf("future %v must trade above spot %v (positive carry)", q.LastPrice, nifty)
		case in.Type == store.TypeCall && q.LastPrice < nifty-in.Strike-0.05:
			t.Fatalf("%s below intrinsic value: %v (spot %v)", in.Symbol, q.LastPrice, nifty)
		case in.Type == store.TypePut && q.LastPrice < in.Strike-nifty-0.05:
			t.Fatalf("%s below intrinsic value: %v (spot %v)", in.Symbol, q.LastPrice, nifty)
		}
	}
}
