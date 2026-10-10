package demo

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/seenimo/stock_website/internal/ratelimit"
)

func upstream(t *testing.T, name string) *url.URL {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, name+" "+r.Method+" "+r.URL.Path+" host="+r.Host)
	}))
	t.Cleanup(s.Close)
	u, _ := url.Parse(s.URL)
	return u
}

func TestRouting(t *testing.T) {
	h := New(Config{
		HomeURL:    upstream(t, "home"),
		GrafanaURL: upstream(t, "grafana"),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	for _, tc := range []struct {
		method, path string
		code         int
		body         string
	}{
		{"GET", "/", 200, "home GET / host=demo.test"},
		{"GET", "/api/v1/market", 200, "home GET /api/v1/market"},
		{"HEAD", "/watchlist", 200, ""},
		{"POST", "/api/v1/am/silences", 405, "read-only"},
		{"DELETE", "/api/v1/am/silences/x", 405, "read-only"},
		{"GET", "/metrics", 404, ""},
		{"GET", "/grafana/", 200, "grafana GET /grafana/"},
		{"GET", "/grafana/d/abc/overview", 200, "grafana GET /grafana/d/abc/overview"},
		{"POST", "/grafana/api/ds/query", 200, "grafana POST /grafana/api/ds/query"},
		{"POST", "/grafana/api/dashboards/db", 405, ""},
		{"GET", "/grafana/login", 403, ""},
		{"GET", "/grafana/LOGIN", 403, ""},
		{"GET", "/grafana/api/admin/settings", 403, ""},
		{"GET", "/grafana/api/user/", 403, ""},
		{"GET", "/grafana/public/../api/admin/settings", 403, ""},
		{"GET", "/grafana/%2e%2e/api/v1/market", 200, "home GET /api/v1/market"},
	} {
		r := httptest.NewRequest(tc.method, "http://demo.test"+tc.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
			t.Errorf("%s %s: %d %q, want %d %q", tc.method, tc.path, rec.Code, rec.Body, tc.code, tc.body)
		}
		if rec.Header().Get("X-Robots-Tag") != "noindex" {
			t.Errorf("%s: missing X-Robots-Tag", tc.path)
		}
	}
}

func TestNoGrafana(t *testing.T) {
	h := New(Config{HomeURL: upstream(t, "home"), Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/grafana/", nil))
	if rec.Code != 404 {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpstreamDownAndLimit(t *testing.T) {
	dead, _ := url.Parse("http://127.0.0.1:1")
	h := New(Config{HomeURL: dead, Limit: ratelimit.New(0, 1), Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("code %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("code %d, want 429", rec.Code)
	}
}
