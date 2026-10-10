// Package ticker serves a small, cached market snapshot read from the stock_home demo API.
package ticker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const (
	maxIndices = 6
	maxMovers  = 5
)

// Quote is one instrument on the ticker.
type Quote struct {
	Symbol        string   `json:"symbol"`
	Price         *float64 `json:"price"`
	ChangePercent *float64 `json:"changePercent"`
}

// Snapshot is the ticker payload.
type Snapshot struct {
	Available bool      `json:"available"`
	Simulated bool      `json:"simulated"`
	UpdatedAt time.Time `json:"updatedAt"`
	Indices   []Quote   `json:"indices"`
	Gainers   []Quote   `json:"gainers"`
	Losers    []Quote   `json:"losers"`
	Advancers *float64  `json:"advancers"`
	Decliners *float64  `json:"decliners"`
}

// Source fetches and caches snapshots. Concurrent callers share one upstream fetch.
type Source struct {
	BaseURL   string
	Client    *http.Client
	TTL       time.Duration
	Simulated bool
	Logger    *slog.Logger
	Now       func() time.Time

	mu      sync.Mutex
	cached  Snapshot
	expires time.Time
}

// Get returns the cached snapshot, refreshing it when older than TTL. Failures are cached too.
func (s *Source) Get(ctx context.Context) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if now.Before(s.expires) {
		return s.cached
	}
	snap, err := s.fetch(context.WithoutCancel(ctx))
	if err != nil {
		s.Logger.Warn("Ticker refresh failed", "err", err)
		snap = Snapshot{Indices: []Quote{}, Gainers: []Quote{}, Losers: []Quote{}}
	}
	snap.Simulated = s.Simulated
	snap.UpdatedAt = now
	s.cached, s.expires = snap, now.Add(s.TTL)
	return snap
}

func (s *Source) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s.Get(r.Context()))
}

func (s *Source) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

type upstreamQuote struct {
	Symbol        string   `json:"symbol"`
	Price         *float64 `json:"price"`
	ChangePercent *float64 `json:"changePercent"`
}

type market struct {
	Indices   []upstreamQuote `json:"indices"`
	Advancers *float64        `json:"advancers"`
	Decliners *float64        `json:"decliners"`
}

type movers struct {
	Gainers []upstreamQuote `json:"gainers"`
	Losers  []upstreamQuote `json:"losers"`
}

func (s *Source) fetch(ctx context.Context) (Snapshot, error) {
	var (
		m          market
		mv         movers
		wg         sync.WaitGroup
		errM, errV error
	)
	wg.Go(func() { errM = s.section(ctx, "market", &m) })
	wg.Go(func() { errV = s.section(ctx, "movers", &mv) })
	wg.Wait()
	if errM != nil {
		return Snapshot{}, errM
	}
	snap := Snapshot{
		Available: true,
		Indices:   convert(m.Indices, maxIndices),
		Advancers: m.Advancers,
		Decliners: m.Decliners,
		Gainers:   []Quote{},
		Losers:    []Quote{},
	}
	// Movers are optional decoration; the ticker still works with indices only.
	if errV == nil {
		snap.Gainers, snap.Losers = convert(mv.Gainers, maxMovers), convert(mv.Losers, maxMovers)
	}
	return snap, nil
}

func (s *Source) section(ctx context.Context, name string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+"/api/v1/"+name, nil)
	if err != nil {
		return err
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", name, resp.StatusCode)
	}
	env := struct {
		Data json.RawMessage `json:"data"`
	}{}
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&env); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := json.Unmarshal(env.Data, into); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func convert(in []upstreamQuote, n int) []Quote {
	out := make([]Quote, 0, min(len(in), n))
	for _, q := range in {
		if len(out) == n {
			break
		}
		out = append(out, Quote(q))
	}
	return out
}
