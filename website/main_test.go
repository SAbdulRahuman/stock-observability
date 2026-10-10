package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDemoURL(t *testing.T) {
	for _, tc := range []struct {
		name, public, domain, listen, host, want string
	}{
		{"explicit", "https://x.example/", "demo.example", ":8080", "a", "https://x.example/"},
		{"domain", "", "demo.iseeticks.com", "", "iseeticks.com", "https://demo.iseeticks.com/"},
		{"ip and port", "", "", ":8080", "203.0.113.5", "http://203.0.113.5:8080/"},
		{"strips visitor port", "", "", "0.0.0.0:8080", "203.0.113.5:80", "http://203.0.113.5:8080/"},
		{"ipv6", "", "", ":8080", "[2001:db8::1]", "http://[2001:db8::1]:8080/"},
		{"hostile host", "", "", ":8080", `evil"><script>`, ""},
		{"disabled", "", "", "", "a", ""},
	} {
		r := httptest.NewRequest("GET", "/api/site/config", nil)
		r.Host = tc.host
		if got := demoURLFunc(tc.public, tc.domain, tc.listen)(r); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSiteRoutes(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "handled "+r.Method) })
	h := siteHandler(siteDeps{ticker: ok, waitlist: ok, demoURL: func(*http.Request) string { return "http://d/" }, version: "1", hsts: true})
	for _, tc := range []struct {
		method, path string
		code         int
		body         string
	}{
		{"GET", "/healthz", 200, "ok"},
		{"GET", "/api/site/config", 200, `"demoUrl":"http://d/"`},
		{"GET", "/api/site/ticker", 200, "handled GET"},
		{"POST", "/api/site/waitlist", 200, "handled POST"},
		{"GET", "/api/site/nope", 404, "not found"},
		{"DELETE", "/api/site/waitlist", 405, ""},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
			t.Errorf("%s %s: %d %q", tc.method, tc.path, rec.Code, rec.Body)
		}
		if rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("Strict-Transport-Security") == "" {
			t.Errorf("%s: missing security headers", tc.path)
		}
	}
}

func TestHostRouter(t *testing.T) {
	named := func(n string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, n) })
	}
	h := hostRouter("demo.iseeticks.com", named("site"), named("demo"))
	for host, want := range map[string]string{"iseeticks.com": "site", "DEMO.iseeticks.com:443": "demo", "other": "site"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = host
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Body.String() != want {
			t.Errorf("%s: got %q", host, rec.Body)
		}
	}
}
