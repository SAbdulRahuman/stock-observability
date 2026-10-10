package ratelimit

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestAllowPerKey(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	l := New(1, 2)
	l.now = func() time.Time { return now }
	for i, want := range []bool{true, true, false} {
		if got := l.Allow("a"); got != want {
			t.Fatalf("request %d: got %v, want %v", i, got, want)
		}
	}
	if !l.Allow("b") {
		t.Fatal("other key must have its own bucket")
	}
	now = now.Add(time.Second)
	if !l.Allow("a") {
		t.Fatal("bucket must refill")
	}
}

func TestPrune(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	l := New(1, 1)
	l.now = func() time.Time { return now }
	l.Allow("old")
	now = now.Add(idleAfter + time.Second)
	l.prune(now)
	if _, ok := l.clients["old"]; ok {
		t.Fatal("idle client not pruned")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.7:5555"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := ClientIP(r); got != "203.0.113.7" {
		t.Fatalf("got %q", got)
	}
}
