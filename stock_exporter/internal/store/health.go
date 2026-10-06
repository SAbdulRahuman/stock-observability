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
	"sort"
	"sync"
	"time"
)

// API call outcomes used in the outcome label.
const (
	OutcomeSuccess     = "success"
	OutcomeError       = "error"
	OutcomeTokenError  = "token_error"
	OutcomeRateLimited = "rate_limited"
)

// APICount is the number of upstream API calls for an endpoint and outcome.
type APICount struct {
	Endpoint string
	Outcome  string
	Count    float64
}

type apiKey struct{ endpoint, outcome string }

// Health tracks feed and upstream session health.
type Health struct {
	mu            sync.Mutex
	source        string
	mode          string
	connections   map[int]bool
	ticks         uint64
	reconnects    uint64
	errors        uint64
	subscribed    int
	lastTick      time.Time
	sessionValid  bool
	sessionExpiry time.Time
	api           map[apiKey]uint64
}

// HealthSnapshot is a point-in-time copy of Health.
type HealthSnapshot struct {
	Source        string
	Mode          string
	Connected     int
	Ticks         uint64
	Reconnects    uint64
	Errors        uint64
	Subscribed    int
	LastTick      time.Time
	SessionValid  bool
	SessionExpiry time.Time
	API           []APICount
}

// NewHealth returns health for the given feed source (kite or sim) and mode.
func NewHealth(source, mode string) *Health {
	return &Health{source: source, mode: mode, connections: map[int]bool{}, api: map[apiKey]uint64{}}
}

// SetConnected records the state of WebSocket connection n.
func (h *Health) SetConnected(n int, up bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[n] = up
}

// ResetConnections forgets all connections, e.g. when the session rotates.
func (h *Health) ResetConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections = map[int]bool{}
}

// AddTicks records received ticks.
func (h *Health) AddTicks(n int, at time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ticks += uint64(n)
	h.lastTick = at
}

// IncReconnects records a reconnect attempt.
func (h *Health) IncReconnects() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reconnects++
}

// IncErrors records a feed error.
func (h *Health) IncErrors() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors++
}

// SetSubscribed records the number of subscribed instruments.
func (h *Health) SetSubscribed(n int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribed = n
}

// SetSession records session validity and expiry.
func (h *Health) SetSession(valid bool, expiry time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessionValid = valid
	h.sessionExpiry = expiry
}

// ObserveAPI records an upstream API call.
func (h *Health) ObserveAPI(endpoint, outcome string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.api[apiKey{endpoint, outcome}]++
}

// Snapshot returns a copy of the current health.
func (h *Health) Snapshot() HealthSnapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := HealthSnapshot{
		Source:        h.source,
		Mode:          h.mode,
		Ticks:         h.ticks,
		Reconnects:    h.reconnects,
		Errors:        h.errors,
		Subscribed:    h.subscribed,
		LastTick:      h.lastTick,
		SessionValid:  h.sessionValid,
		SessionExpiry: h.sessionExpiry,
	}
	for _, up := range h.connections {
		if up {
			s.Connected++
		}
	}
	for k, v := range h.api {
		s.API = append(s.API, APICount{Endpoint: k.endpoint, Outcome: k.outcome, Count: float64(v)})
	}
	sort.Slice(s.API, func(a, b int) bool {
		if s.API[a].Endpoint != s.API[b].Endpoint {
			return s.API[a].Endpoint < s.API[b].Endpoint
		}
		return s.API[a].Outcome < s.API[b].Outcome
	})
	return s
}
