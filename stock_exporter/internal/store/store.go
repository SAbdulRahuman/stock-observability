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

// Package store holds the latest market state shared between feeds (writers)
// and collectors (readers). Scrapes never call upstream APIs directly.
package store

import (
	"sort"
	"sync"
	"time"
)

// Instrument types used in the instrument_type label.
const (
	TypeEquity = "EQ"
	TypeIndex  = "INDEX"
	TypeFuture = "FUT"
	TypeCall   = "CE"
	TypePut    = "PE"
)

// Tiers control scrape frequency: core is the curated watchlist, broad the exchange-wide universe.
const (
	TierCore  = "core"
	TierBroad = "broad"
)

// Instrument is a resolved watchlist entry.
type Instrument struct {
	Token    uint32
	Exchange string
	Symbol   string
	Sector   string
	Type     string
	Tier     string
}

// DepthLevel is a single order book level.
type DepthLevel struct {
	Price    float64
	Quantity float64
	Orders   float64
}

// Depth holds the top 5 bid and ask levels.
type Depth struct {
	Bids [5]DepthLevel
	Asks [5]DepthLevel
}

// Quote is the latest tick for an instrument.
type Quote struct {
	LastPrice     float64
	Open          float64
	High          float64
	Low           float64
	PrevClose     float64
	AveragePrice  float64
	Volume        float64
	LastQuantity  float64
	BuyQuantity   float64
	SellQuantity  float64
	OpenInterest  float64
	LastTradeTime time.Time
	ExchangeTime  time.Time
	ReceivedAt    time.Time
	Depth         Depth
	HasDepth      bool
}

// Circuit holds daily price band limits.
type Circuit struct {
	Upper     float64
	Lower     float64
	UpdatedAt time.Time
}

// Entry is a consistent view of one instrument.
type Entry struct {
	Instrument Instrument
	Quote      Quote
	HasQuote   bool
	Circuit    Circuit
	HasCircuit bool
}

// Store is safe for concurrent use.
type Store struct {
	mu          sync.RWMutex
	instruments map[uint32]Instrument
	quotes      map[uint32]Quote
	circuits    map[uint32]Circuit
	portfolio   Portfolio
	hasPortf    bool
	createdAt   time.Time
}

// New returns an empty store.
func New() *Store {
	return &Store{
		instruments: map[uint32]Instrument{},
		quotes:      map[uint32]Quote{},
		circuits:    map[uint32]Circuit{},
		createdAt:   time.Now(),
	}
}

// CreatedAt is the time the store was created.
func (s *Store) CreatedAt() time.Time { return s.createdAt }

// SetInstruments replaces the tracked universe and drops state of removed instruments.
func (s *Store) SetInstruments(ins []Instrument) {
	next := make(map[uint32]Instrument, len(ins))
	for _, i := range ins {
		next[i.Token] = i
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instruments = next
	for t := range s.quotes {
		if _, ok := next[t]; !ok {
			delete(s.quotes, t)
		}
	}
	for t := range s.circuits {
		if _, ok := next[t]; !ok {
			delete(s.circuits, t)
		}
	}
}

// Instruments returns the tracked universe sorted by exchange and symbol.
func (s *Store) Instruments() []Instrument {
	s.mu.RLock()
	out := make([]Instrument, 0, len(s.instruments))
	for _, i := range s.instruments {
		out = append(out, i)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(a, b int) bool {
		if out[a].Exchange != out[b].Exchange {
			return out[a].Exchange < out[b].Exchange
		}
		return out[a].Symbol < out[b].Symbol
	})
	return out
}

// Tokens returns the tokens of all tracked instruments.
func (s *Store) Tokens() []uint32 {
	ins := s.Instruments()
	out := make([]uint32, len(ins))
	for i, in := range ins {
		out[i] = in.Token
	}
	return out
}

// UpdateQuote stores a tick. Ticks for unknown tokens are ignored.
func (s *Store) UpdateQuote(token uint32, q Quote) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.instruments[token]; !ok {
		return false
	}
	s.quotes[token] = q
	return true
}

// UpdateCircuit stores circuit limits. Unknown tokens are ignored.
func (s *Store) UpdateCircuit(token uint32, c Circuit) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.instruments[token]; !ok {
		return false
	}
	s.circuits[token] = c
	return true
}

// Snapshot returns all instruments with their latest state, sorted by exchange and symbol.
func (s *Store) Snapshot() []Entry {
	s.mu.RLock()
	out := make([]Entry, 0, len(s.instruments))
	for t, i := range s.instruments {
		q, hq := s.quotes[t]
		c, hc := s.circuits[t]
		out = append(out, Entry{Instrument: i, Quote: q, HasQuote: hq, Circuit: c, HasCircuit: hc})
	}
	s.mu.RUnlock()
	sort.Slice(out, func(a, b int) bool {
		if out[a].Instrument.Exchange != out[b].Instrument.Exchange {
			return out[a].Instrument.Exchange < out[b].Instrument.Exchange
		}
		return out[a].Instrument.Symbol < out[b].Instrument.Symbol
	})
	return out
}

// Holding is a long-term (delivery) holding.
type Holding struct {
	Exchange     string
	Symbol       string
	Quantity     float64
	AveragePrice float64
	LastPrice    float64
	PnL          float64
	DayChange    float64
}

// Position is an open intraday/F&O position.
type Position struct {
	Exchange     string
	Symbol       string
	Product      string
	Quantity     float64
	AveragePrice float64
	LastPrice    float64
	PnL          float64
	M2M          float64
}

// Margin is the margin state for a segment (equity or commodity).
type Margin struct {
	Segment  string
	Net      float64 // free to use
	Total    float64
	Utilised float64
}

// Portfolio is the account state.
type Portfolio struct {
	Holdings  []Holding
	Positions []Position
	Margins   []Margin
	UpdatedAt time.Time
}

// SetPortfolio replaces the portfolio.
func (s *Store) SetPortfolio(p Portfolio) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.portfolio = p
	s.hasPortf = true
}

// Portfolio returns the portfolio and whether it was ever set.
func (s *Store) Portfolio() (Portfolio, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.portfolio, s.hasPortf
}
