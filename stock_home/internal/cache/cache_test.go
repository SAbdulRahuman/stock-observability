package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetCachesWithinTTL(t *testing.T) {
	c := New()
	var calls atomic.Int32
	load := func() (any, error) { calls.Add(1); return "v", nil }

	for range 3 {
		r, err := c.Get("k", time.Minute, time.Hour, load)
		if err != nil || r.Value != "v" || r.Stale {
			t.Fatalf("got %+v, %v", r, err)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("load called %d times, want 1", n)
	}
}

func TestGetCoalescesConcurrentLoads(t *testing.T) {
	c := New()
	var calls atomic.Int32
	release := make(chan struct{})
	load := func() (any, error) { calls.Add(1); <-release; return 1, nil }

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Get("k", time.Minute, time.Hour, load); err != nil {
				t.Error(err)
			}
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	if n := calls.Load(); n != 1 {
		t.Fatalf("load called %d times, want 1", n)
	}
}

func TestGetServesStaleOnError(t *testing.T) {
	c := New()
	now := time.Unix(1000, 0)
	c.now = func() time.Time { return now }

	if _, err := c.Get("k", time.Second, time.Minute, func() (any, error) { return "old", nil }); err != nil {
		t.Fatal(err)
	}
	now = now.Add(10 * time.Second)
	r, err := c.Get("k", time.Second, time.Minute, func() (any, error) { return nil, errors.New("down") })
	if err != nil || r.Value != "old" || !r.Stale {
		t.Fatalf("got %+v, %v; want stale old value", r, err)
	}

	now = now.Add(time.Hour)
	if _, err := c.Get("k", time.Second, time.Minute, func() (any, error) { return nil, errors.New("down") }); err == nil {
		t.Fatal("want error once maxStale has passed")
	}
}
