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

package watchlist

import (
	"fmt"
	"strings"
	"testing"

	"github.com/seenimo/stock_exporter/internal/store"
)

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"empty":         "instruments: []",
		"no exchange":   "universes:\n  - {segment: NSE}\n",
		"missing field": "instruments:\n  - exchange: NSE\n",
		"duplicate":     "instruments:\n  - {exchange: NSE, symbol: INFY}\n  - {exchange: nse, symbol: infy}\n",
		"bad type":      "instruments:\n  - {exchange: NSE, symbol: INFY, type: BOND}\n",
		"unknown field": "instruments:\n  - {exchange: NSE, symbol: INFY, sectr: IT}\n",
	}
	for name, in := range cases {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestParseTooMany(t *testing.T) {
	var b strings.Builder
	b.WriteString("instruments:\n")
	for i := 0; i <= MaxInstruments; i++ {
		fmt.Fprintf(&b, "  - {exchange: NSE, symbol: S%d}\n", i)
	}
	if _, err := Parse([]byte(b.String())); err == nil {
		t.Fatal("expected limit error")
	}
}

func TestResolve(t *testing.T) {
	w, err := Parse([]byte(`
instruments:
  - {exchange: nse, symbol: " reliance ", sector: Energy}
  - {exchange: NSE, symbol: NIFTY 50}
  - {exchange: NSE, symbol: NIFTY26OCTFUT, sector: Index}
  - {exchange: NSE, symbol: UNKNOWN}
`))
	if err != nil {
		t.Fatal(err)
	}
	got, missing := w.Resolve([]Ref{
		{Token: 738561, Exchange: "NSE", Symbol: "RELIANCE", Segment: "NSE", InstrumentType: "EQ"},
		{Token: 256265, Exchange: "NSE", Symbol: "NIFTY 50", Segment: "INDICES", InstrumentType: "EQ"},
		{Token: 9999, Exchange: "NFO", Symbol: "NIFTY26OCTFUT", Segment: "NFO-FUT", InstrumentType: "FUT"},
	})
	want := []store.Instrument{
		{Token: 738561, Exchange: "NSE", Symbol: "RELIANCE", Sector: "Energy", Type: store.TypeEquity, Tier: store.TierCore},
		{Token: 256265, Exchange: "NSE", Symbol: "NIFTY 50", Sector: UnknownSector, Type: store.TypeIndex, Tier: store.TierCore},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %+v, want %+v", got[i], want[i])
		}
	}
	if len(missing) != 2 || missing[0] != "NSE:NIFTY26OCTFUT" || missing[1] != "NSE:UNKNOWN" {
		t.Errorf("missing = %v", missing)
	}
	if ex := w.Exchanges(); len(ex) != 1 || ex[0] != "NSE" {
		t.Errorf("exchanges = %v", ex)
	}
}

func TestResolveUniverse(t *testing.T) {
	w, err := Parse([]byte(`
instruments:
  - {exchange: NSE, symbol: INFY, sector: IT}
universes:
  - {exchange: nse, segment: NSE, instrument_types: [EQ], exclude_hyphenated: true}
`))
	if err != nil {
		t.Fatal(err)
	}
	got, missing := w.Resolve([]Ref{
		{Token: 1, Exchange: "NSE", Symbol: "INFY", Segment: "NSE", InstrumentType: "EQ"},
		{Token: 2, Exchange: "NSE", Symbol: "TCS", Segment: "NSE", InstrumentType: "EQ"},
		{Token: 3, Exchange: "NSE", Symbol: "ABC-BE", Segment: "NSE", InstrumentType: "EQ"},
		{Token: 4, Exchange: "NSE", Symbol: "NIFTY BANK", Segment: "INDICES", InstrumentType: "EQ"},
		{Token: 5, Exchange: "BSE", Symbol: "TCS", Segment: "BSE", InstrumentType: "EQ"},
	})
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	want := []store.Instrument{
		{Token: 1, Exchange: "NSE", Symbol: "INFY", Sector: "IT", Type: store.TypeEquity, Tier: store.TierCore},
		{Token: 2, Exchange: "NSE", Symbol: "TCS", Sector: UnknownSector, Type: store.TypeEquity, Tier: store.TierBroad},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %+v, want %+v", got[i], want[i])
		}
	}
}

func TestSynthetic(t *testing.T) {
	w, err := Parse([]byte("instruments:\n  - {exchange: NSE, symbol: INFY}\n  - {exchange: NSE, symbol: INDIA VIX, type: index}\nuniverses:\n  - {exchange: NSE}\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := w.Synthetic(3)
	if len(s) != 5 {
		t.Fatalf("len = %d", len(s))
	}
	if s[0].Token != 1 || s[0].Type != store.TypeEquity || s[1].Token != 2 || s[1].Type != store.TypeIndex {
		t.Fatalf("unexpected %+v", s)
	}
	if s[4].Symbol != "SIM0003" || s[4].Tier != store.TierBroad || s[4].Token != 5 {
		t.Fatalf("unexpected broad %+v", s[4])
	}
}
