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

// Package watchlist loads the set of instruments to track and resolves them to feed tokens.
package watchlist

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/seenimo/stock_exporter/internal/store"
)

// MaxInstruments is the Kite limit of 3 WebSocket connections x 3000 instruments.
const MaxInstruments = 9000

var validTypes = map[string]bool{
	store.TypeEquity: true, store.TypeIndex: true, store.TypeFuture: true, store.TypeCall: true, store.TypePut: true,
}

// Entry is one watchlist line.
type Entry struct {
	Exchange string `yaml:"exchange"`
	Symbol   string `yaml:"symbol"`
	Sector   string `yaml:"sector,omitempty"`
	Type     string `yaml:"type,omitempty"`
}

// Key is the Kite "EXCHANGE:SYMBOL" identifier.
func (e Entry) Key() string { return e.Exchange + ":" + e.Symbol }

// Universe adds every instrument of an exchange matching the filters to the broad tier.
type Universe struct {
	Exchange        string   `yaml:"exchange"`
	Segment         string   `yaml:"segment,omitempty"`
	InstrumentTypes []string `yaml:"instrument_types,omitempty"`
	// Hyphenated NSE symbols are non-EQ series (BE, SM, GB, bonds...).
	ExcludeHyphenated bool `yaml:"exclude_hyphenated,omitempty"`
}

func (u Universe) match(r Ref) bool {
	if !strings.EqualFold(r.Exchange, u.Exchange) {
		return false
	}
	if u.Segment != "" && !strings.EqualFold(r.Segment, u.Segment) {
		return false
	}
	if u.ExcludeHyphenated && strings.Contains(r.Symbol, "-") {
		return false
	}
	if len(u.InstrumentTypes) == 0 {
		return true
	}
	for _, t := range u.InstrumentTypes {
		if strings.EqualFold(t, r.InstrumentType) {
			return true
		}
	}
	return false
}

// Watchlist is the parsed watchlist file.
type Watchlist struct {
	Instruments []Entry    `yaml:"instruments,omitempty"`
	Universes   []Universe `yaml:"universes,omitempty"`
}

// UnknownSector is used when no sector is configured.
const UnknownSector = "unknown"

// Load reads and validates a watchlist file.
func Load(path string) (*Watchlist, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	w, err := Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return w, nil
}

// Parse validates and normalises a watchlist.
func Parse(b []byte) (*Watchlist, error) {
	var w Watchlist
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&w); err != nil {
		return nil, fmt.Errorf("parsing watchlist: %w", err)
	}
	if len(w.Instruments) == 0 && len(w.Universes) == 0 {
		return nil, errors.New("watchlist has no instruments or universes")
	}
	if len(w.Instruments) > MaxInstruments {
		return nil, fmt.Errorf("watchlist has %d instruments, maximum is %d", len(w.Instruments), MaxInstruments)
	}
	seen := make(map[string]bool, len(w.Instruments))
	for i := range w.Instruments {
		e := &w.Instruments[i]
		e.Exchange = strings.ToUpper(strings.TrimSpace(e.Exchange))
		e.Symbol = strings.ToUpper(strings.TrimSpace(e.Symbol))
		e.Type = strings.ToUpper(strings.TrimSpace(e.Type))
		e.Sector = strings.TrimSpace(e.Sector)
		if e.Sector == "" {
			e.Sector = UnknownSector
		}
		if e.Exchange == "" || e.Symbol == "" {
			return nil, fmt.Errorf("instrument %d: exchange and symbol are required", i)
		}
		if e.Type != "" && !validTypes[e.Type] {
			return nil, fmt.Errorf("instrument %s: unknown type %q", e.Key(), e.Type)
		}
		if seen[e.Key()] {
			return nil, fmt.Errorf("instrument %s listed more than once", e.Key())
		}
		seen[e.Key()] = true
	}
	for i := range w.Universes {
		u := &w.Universes[i]
		u.Exchange = strings.ToUpper(strings.TrimSpace(u.Exchange))
		if u.Exchange == "" {
			return nil, fmt.Errorf("universe %d: exchange is required", i)
		}
	}
	return &w, nil
}

// Exchanges returns the distinct exchanges referenced by instruments and universes.
func (w *Watchlist) Exchanges() []string {
	var out []string
	seen := map[string]bool{}
	add := func(ex string) {
		if !seen[ex] {
			seen[ex] = true
			out = append(out, ex)
		}
	}
	for _, e := range w.Instruments {
		add(e.Exchange)
	}
	for _, u := range w.Universes {
		add(u.Exchange)
	}
	return out
}

// Ref is an instrument from the broker's instrument master.
type Ref struct {
	Token          uint32
	Exchange       string
	Symbol         string
	Segment        string
	InstrumentType string
}

// Resolve maps watchlist entries (core tier) and universes (broad tier) to
// instrument master tokens, capped at MaxInstruments, and returns the keys of
// entries that could not be resolved.
func (w *Watchlist) Resolve(refs []Ref) ([]store.Instrument, []string) {
	idx := make(map[string]Ref, len(refs))
	for _, r := range refs {
		idx[strings.ToUpper(r.Exchange)+":"+strings.ToUpper(r.Symbol)] = r
	}
	var (
		out     []store.Instrument
		missing []string
		taken   = map[uint32]bool{}
	)
	for _, e := range w.Instruments {
		r, ok := idx[e.Key()]
		if !ok {
			missing = append(missing, e.Key())
			continue
		}
		typ := e.Type
		if typ == "" {
			typ = refType(r)
		}
		taken[r.Token] = true
		out = append(out, store.Instrument{Token: r.Token, Exchange: e.Exchange, Symbol: e.Symbol, Sector: e.Sector, Type: typ, Tier: store.TierCore})
	}
	for _, u := range w.Universes {
		for _, r := range refs {
			if len(out) >= MaxInstruments {
				return out, missing
			}
			if taken[r.Token] || !u.match(r) {
				continue
			}
			taken[r.Token] = true
			out = append(out, store.Instrument{
				Token: r.Token, Exchange: strings.ToUpper(r.Exchange), Symbol: strings.ToUpper(r.Symbol),
				Sector: UnknownSector, Type: refType(r), Tier: store.TierBroad,
			})
		}
	}
	return out, missing
}

func refType(r Ref) string {
	if r.Segment == "INDICES" {
		return store.TypeIndex
	}
	if t := strings.ToUpper(r.InstrumentType); validTypes[t] {
		return t
	}
	return store.TypeEquity
}

// Synthetic assigns sequential tokens for feeds that have no instrument master
// (simulation) and adds broadSize generated broad-tier symbols when universes are configured.
func (w *Watchlist) Synthetic(broadSize int) []store.Instrument {
	out := make([]store.Instrument, 0, len(w.Instruments)+broadSize)
	for i, e := range w.Instruments {
		typ := e.Type
		if typ == "" {
			typ = store.TypeEquity
		}
		out = append(out, store.Instrument{Token: uint32(i + 1), Exchange: e.Exchange, Symbol: e.Symbol, Sector: e.Sector, Type: typ, Tier: store.TierCore})
	}
	if len(w.Universes) == 0 {
		return out
	}
	broadSize = min(broadSize, MaxInstruments-len(out))
	for i := range broadSize {
		out = append(out, store.Instrument{
			Token: uint32(len(w.Instruments) + i + 1), Exchange: w.Universes[0].Exchange, Symbol: fmt.Sprintf("SIM%04d", i+1),
			Sector: UnknownSector, Type: store.TypeEquity, Tier: store.TierBroad,
		})
	}
	return out
}
