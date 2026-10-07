package openbb

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newProxy(t *testing.T, upstream http.HandlerFunc, cfg Config) (*Proxy, *httptest.Server) {
	t.Helper()
	up := httptest.NewServer(upstream)
	t.Cleanup(up.Close)
	if cfg.URL == "" {
		cfg.URL = up.URL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = time.Second
	}
	p, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), prometheus.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	return p, up
}

func do(p http.Handler, target string, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	return rec
}

func errorOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct{ Error string }
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Error
}

func TestForwardsCommand(t *testing.T) {
	var got *http.Request
	p, _ := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "upstream=1")
		_, _ = io.WriteString(w, `{"results":[{"close":1.5}],"provider":"yfinance","warnings":null,"chart":null,"extra":{}}`)
	}, Config{Username: "openbb", Password: "s3cret"})

	rec := do(p, "/api/v2/equity/price/historical?symbol=RELIANCE.NS&provider=yfinance", http.Header{
		"Authorization": {"Bearer browser-token"},
		"Cookie":        {"session=browser"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if got.URL.Path != "/api/v1/equity/price/historical" || got.URL.RawQuery != "symbol=RELIANCE.NS&provider=yfinance" {
		t.Errorf("upstream request %s", got.URL)
	}
	if u, pw, ok := got.BasicAuth(); !ok || u != "openbb" || pw != "s3cret" {
		t.Errorf("basic auth %q %q %v", u, pw, ok)
	}
	if got.Header.Get("Cookie") != "" {
		t.Errorf("browser cookie forwarded")
	}
	if rec.Header().Get("Set-Cookie") != "" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("response headers %v", rec.Header())
	}
	if !strings.Contains(rec.Body.String(), `"provider":"yfinance"`) {
		t.Errorf("body %s", rec.Body)
	}
	if n := testutil.ToFloat64(p.requests.WithLabelValues("200")); n != 1 {
		t.Errorf("requests_total{code=200} = %v", n)
	}
}

func TestKeepsBasePath(t *testing.T) {
	var path string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(up.Close)
	p, err := New(Config{URL: up.URL + "/openbb/", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), prometheus.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	do(p, "/api/v2/news/company", nil)
	if path != "/openbb/api/v1/news/company" {
		t.Errorf("path %q", path)
	}
}

func TestRelaysUpstreamErrors(t *testing.T) {
	p, _ := newProxy(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"detail":"symbol is required"}`)
	}, Config{})
	rec := do(p, "/api/v2/equity/price/quote", nil)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "symbol is required") {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
}

func TestRejectsInvalidPaths(t *testing.T) {
	called := false
	p, _ := newProxy(t, func(http.ResponseWriter, *http.Request) { called = true }, Config{})
	for _, target := range []string{
		"/api/v2/",
		"/api/v2/equity/../../admin",
		"/api/v2/equity//quote",
		"/api/v2/equity/%2e%2e/quote",
		"/api/v2/Equity/Quote",
		"/api/v2/equity/quote/",
	} {
		rec := do(p, target, nil)
		if rec.Code != http.StatusNotFound || errorOf(t, rec) != "unknown command" {
			t.Errorf("%s: %d", target, rec.Code)
		}
	}
	if called {
		t.Error("upstream called for an invalid path")
	}
}

func TestHidesUpstreamAuthChallenge(t *testing.T) {
	p, _ := newProxy(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="openbb"`)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
	}, Config{})
	rec := do(p, "/api/v2/equity/price/quote", nil)
	if rec.Code != http.StatusBadGateway || rec.Header().Get("WWW-Authenticate") != "" {
		t.Errorf("%d %v", rec.Code, rec.Header())
	}
	if msg := errorOf(t, rec); msg != "upstream openbb rejected credentials" {
		t.Errorf("error %q", msg)
	}
}

func TestPassesNoContent(t *testing.T) {
	p, _ := newProxy(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, Config{})
	rec := do(p, "/api/v2/equity/profile?symbol=X", nil)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Errorf("%d %q", rec.Code, rec.Body)
	}
}

func TestRejectsNonJSON(t *testing.T) {
	p, _ := newProxy(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<script>alert(1)</script>")
	}, Config{})
	rec := do(p, "/api/v2/equity/price/quote", nil)
	if rec.Code != http.StatusBadGateway || strings.Contains(rec.Body.String(), "script") {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
}

func TestDoesNotFollowRedirects(t *testing.T) {
	p, _ := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data", http.StatusFound)
	}, Config{})
	if rec := do(p, "/api/v2/equity/price/quote", nil); rec.Code != http.StatusBadGateway {
		t.Errorf("status %d", rec.Code)
	}
}

func TestTimeout(t *testing.T) {
	release := make(chan struct{})
	p, _ := newProxy(t, func(http.ResponseWriter, *http.Request) { <-release }, Config{Timeout: 50 * time.Millisecond})
	defer close(release)
	rec := do(p, "/api/v2/equity/price/quote", nil)
	if rec.Code != http.StatusGatewayTimeout || errorOf(t, rec) != "command timed out" {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
}

func TestUnavailable(t *testing.T) {
	p, up := newProxy(t, func(http.ResponseWriter, *http.Request) {}, Config{})
	up.Close()
	rec := do(p, "/api/v2/equity/price/quote", nil)
	if rec.Code != http.StatusBadGateway || errorOf(t, rec) != "upstream openbb unavailable" {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
}

func TestNewValidatesConfig(t *testing.T) {
	for _, cfg := range []Config{
		{URL: "openbb:6900", Timeout: time.Second},
		{URL: "ftp://openbb", Timeout: time.Second},
		{URL: "http://user:pw@openbb:6900", Timeout: time.Second},
		{URL: "http://openbb:6900?x=1", Timeout: time.Second},
		{URL: "http://openbb:6900", Timeout: 0},
	} {
		if _, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), prometheus.NewRegistry()); err == nil {
			t.Errorf("%+v: want error", cfg)
		}
	}
}
