// Package cache provides a small TTL cache that coalesces concurrent refreshes.
package cache

import (
	"sync"
	"time"
)

// Result is a cached value and when it was produced.
type Result struct {
	Value     any
	UpdatedAt time.Time
	Stale     bool
}

type entry struct {
	val      any
	at       time.Time
	err      error
	inflight chan struct{}
}

// Cache maps keys to values loaded on demand.
type Cache struct {
	mu      sync.Mutex
	entries map[string]*entry
	now     func() time.Time
}

// New returns an empty cache.
func New() *Cache {
	return &Cache{entries: map[string]*entry{}, now: time.Now}
}

// Get returns the value for key, calling load when it is older than ttl. Concurrent callers share one load.
// If load fails, the previous value is returned as stale while it is younger than maxStale.
func (c *Cache) Get(key string, ttl, maxStale time.Duration, load func() (any, error)) (Result, error) {
	c.mu.Lock()
	e, ok := c.entries[key]
	if !ok {
		e = &entry{}
		c.entries[key] = e
	}
	if e.val != nil && e.err == nil && c.now().Sub(e.at) < ttl {
		r := Result{Value: e.val, UpdatedAt: e.at}
		c.mu.Unlock()
		return r, nil
	}
	if e.inflight == nil {
		e.inflight = make(chan struct{})
		go c.refresh(e, load)
	}
	done := e.inflight
	c.mu.Unlock()

	<-done

	c.mu.Lock()
	defer c.mu.Unlock()
	if e.err == nil {
		return Result{Value: e.val, UpdatedAt: e.at}, nil
	}
	if e.val != nil && c.now().Sub(e.at) < maxStale {
		return Result{Value: e.val, UpdatedAt: e.at, Stale: true}, nil
	}
	return Result{}, e.err
}

func (c *Cache) refresh(e *entry, load func() (any, error)) {
	v, err := load()
	c.mu.Lock()
	defer c.mu.Unlock()
	if err == nil {
		e.val, e.at = v, c.now()
	}
	e.err = err
	close(e.inflight)
	e.inflight = nil
}
