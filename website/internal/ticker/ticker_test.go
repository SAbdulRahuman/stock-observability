package ticker

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newSource(t *testing.T, h http.Handler) (*Source, *time.Time) {
	t.Helper()
	up := httptest.NewServer(h)
	t.Cleanup(up.Close)
	now := time.Date(2026, 10, 10, 9, 30, 0, 0, time.UTC)
	return &Source{
		BaseURL:   up.URL,
		Client:    up.Client(),
		TTL:       5 * time.Second,
		Simulated: true,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:       func() time.Time { return now },
	}, &now
}

func TestSnapshotAndCache(t *testing.T) {
	var calls atomic.Int32
	src, now := newSource(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/api/v1/market":
			io.WriteString(w, `{"data":{"indices":[{"symbol":"NIFTY 50","price":25000.5,"changePercent":0.4,"exploreUrl":"x"},{"symbol":"INDIA VIX","price":null,"changePercent":null}],"advancers":30,"decliners":20},"updatedAt":"x"}`)
		case "/api/v1/movers":
			io.WriteString(w, `{"data":{"gainers":[{"symbol":"TCS","price":4000,"changePercent":2.1}],"losers":[],"highVolume":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	snap := src.Get(context.Background())
	if !snap.Available || !snap.Simulated || len(snap.Indices) != 2 || *snap.Indices[0].Price != 25000.5 || snap.Indices[1].Price != nil {
		t.Fatalf("snapshot %+v", snap)
	}
	if len(snap.Gainers) != 1 || snap.Gainers[0].Symbol != "TCS" || *snap.Advancers != 30 {
		t.Fatalf("movers %+v", snap)
	}
	src.Get(context.Background())
	if calls.Load() != 2 {
		t.Fatalf("calls %d, want 2 (cached)", calls.Load())
	}
	*now = now.Add(6 * time.Second)
	src.Get(context.Background())
	if calls.Load() != 4 {
		t.Fatalf("calls %d, want 4 after TTL", calls.Load())
	}
}

func TestUnavailable(t *testing.T) {
	src, _ := newSource(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	rec := httptest.NewRecorder()
	src.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/site/ticker", nil))
	var snap map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || snap["available"] != false || snap["simulated"] != true {
		t.Fatalf("got %d %v", rec.Code, snap)
	}
	if idx, ok := snap["indices"].([]any); !ok || len(idx) != 0 {
		t.Fatalf("indices must be an empty list, got %v", snap["indices"])
	}
}

func TestMoversOptional(t *testing.T) {
	src, _ := newSource(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/market" {
			io.WriteString(w, `{"data":{"indices":[]}}`)
			return
		}
		http.Error(w, "boom", 500)
	}))
	if snap := src.Get(context.Background()); !snap.Available || snap.Gainers == nil {
		t.Fatalf("snapshot %+v", snap)
	}
}
