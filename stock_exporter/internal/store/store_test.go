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

package store

import (
	"testing"
	"time"

	"github.com/seenimo/stock_exporter/internal/daily"
)

func TestNewReference(t *testing.T) {
	if _, ok := NewReference(nil, nil, nil, time.Time{}); ok {
		t.Fatal("no candles must give no reference")
	}
	if _, ok := NewReference([]float64{1}, []float64{1, 2}, []float64{1}, time.Time{}); ok {
		t.Fatal("mismatched slices must give no reference")
	}
	var highs, lows, closes []float64
	for i := range 250 {
		c := float64(100 + i%50)
		highs, lows, closes = append(highs, c+2), append(lows, c-3), append(closes, c)
	}
	r, ok := NewReference(highs, lows, closes, time.Time{})
	if !ok || r.High52W != 151 || r.Low52W != 97 {
		t.Fatalf("52-week range %+v", r)
	}
	// Last 50 closes are 100..149, last 200 are four such cycles.
	if r.SMA50D != 124.5 || r.SMA200D != 124.5 || r.Cross != 0 {
		t.Fatalf("SMA %+v", r)
	}

	// 200 flat closes then a jump: SMA50 rises above SMA200 on the last candle only.
	flat := make([]float64, 201)
	for i := range flat {
		flat[i] = 100
	}
	flat[200] = 150
	if g, _ := NewReference(flat, flat, flat, time.Time{}); g.Cross != 1 {
		t.Fatalf("golden cross not detected: %+v", g)
	}
	flat[200] = 50
	if d, _ := NewReference(flat, flat, flat, time.Time{}); d.Cross != -1 {
		t.Fatalf("death cross not detected: %+v", d)
	}

	st := New()
	st.SetInstruments([]Instrument{{Token: 1}})
	if !st.UpdateReference(1, r) || st.UpdateReference(2, r) {
		t.Fatal("UpdateReference must accept known tokens only")
	}
	if e := st.Snapshot()[0]; !e.HasReference || e.Reference != r {
		t.Fatalf("snapshot %+v", e)
	}
	st.SetInstruments(nil)
	st.SetInstruments([]Instrument{{Token: 1}})
	if st.Snapshot()[0].HasReference {
		t.Fatal("reference of a removed instrument must be dropped")
	}
}

func TestObserverReceivesMarketStateUpdates(t *testing.T) {
	st := New()
	instrument := Instrument{Token: 1, Exchange: "NSE", Symbol: "INFY"}
	st.SetInstruments([]Instrument{instrument})
	var changes []Change
	st.SetObserver(func(change Change) {
		changes = append(changes, change)
	})

	if !st.UpdateQuote(1, Quote{LastPrice: 100}) ||
		!st.UpdateCircuit(1, Circuit{Upper: 110}) ||
		!st.UpdateReference(1, Reference{High52W: 120}) ||
		!st.UpdateDaily(1, daily.Indicators{Valid: true}) {
		t.Fatal("failed to update known instrument")
	}
	st.SetPortfolio(Portfolio{UpdatedAt: time.Now()})

	want := []ChangeKind{ChangeQuote, ChangeCircuit, ChangeReference, ChangeDaily, ChangePortfolio}
	if len(changes) != len(want) {
		t.Fatalf("observer received %d updates, want %d", len(changes), len(want))
	}
	for i, kind := range want {
		if changes[i].Kind != kind {
			t.Errorf("update %d kind = %q, want %q", i, changes[i].Kind, kind)
		}
	}
	if changes[0].Instrument != instrument {
		t.Errorf("quote update instrument = %+v, want %+v", changes[0].Instrument, instrument)
	}
}
