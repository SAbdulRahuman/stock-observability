package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeExporter mimics stock_exporter's /kite/config and records what stock_home forwarded.
type fakeExporter struct {
	mu       sync.Mutex
	editable bool
	status   int
	gotBody  string
	gotMeth  string
}

func (f *fakeExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, _ := io.ReadAll(r.Body)
	f.gotMeth, f.gotBody = r.Method, string(b)
	w.Header().Set("Content-Type", "application/json")
	if f.status != 0 {
		w.WriteHeader(f.status)
		io.WriteString(w, `{"error":"api_key must be 6-64 letters or digits"}`)
		return
	}
	source := "flags"
	if f.editable {
		source = "file"
	}
	io.WriteString(w, `{"configured":true,"source":"`+source+`","editable":`+map[bool]string{true: "true", false: "false"}[f.editable]+
		`,"api_key_hint":"••••3456","session_valid":true,"user_id":"AB1234","session_expires_at":"2026-10-12T06:00:00+05:30"}`)
}

func kiteRequest(t *testing.T, s *Server, method, body string, mod func(*http.Request)) (*httptest.ResponseRecorder, KiteSetup, string) {
	t.Helper()
	mux := http.NewServeMux()
	s.Register(mux)
	req := httptest.NewRequest(method, "http://localhost:8080/api/v1/kite/setup", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:50000"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if mod != nil {
		mod(req)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var env struct {
		Data  KiteSetup
		Error string
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec, env.Data, env.Error
}

func newKiteServer(t *testing.T, f *fakeExporter) *Server {
	t.Helper()
	s := newTestServer(t, "http://prometheus:9090", "http://alertmanager:9093")
	if f != nil {
		exp := httptest.NewServer(f)
		t.Cleanup(exp.Close)
		s.cfg.ExporterURL = exp.URL
	}
	return s
}

func TestKiteSetupGet(t *testing.T) {
	s := newKiteServer(t, &fakeExporter{editable: true})
	rec, k, _ := kiteRequest(t, s, http.MethodGet, "", nil)
	if rec.Code != http.StatusOK || !k.Available || !k.Configured || !k.Editable || k.ReadOnlyReason != "" {
		t.Fatalf("local GET %d %+v", rec.Code, k)
	}
	if k.APIKeyHint != "••••3456" || k.UserID != "AB1234" || !k.SessionValid || k.SessionExpiresAt == nil {
		t.Fatalf("status fields %+v", k)
	}
	if k.RedirectURL != "http://127.0.0.1:8080/exporter/kite/callback" || k.LoginURL != "/exporter/login" {
		t.Fatalf("urls %+v", k)
	}

	_, k, _ = kiteRequest(t, s, http.MethodGet, "", func(r *http.Request) { r.Header.Set("X-Forwarded-For", "192.168.1.20") })
	if k.Editable || k.ReadOnlyReason != readOnlyRemote {
		t.Fatalf("remote GET %+v", k)
	}

	_, k, _ = kiteRequest(t, newKiteServer(t, &fakeExporter{}), http.MethodGet, "", nil)
	if k.Editable || k.ReadOnlyReason != readOnlyManaged || !k.Configured {
		t.Fatalf("managed GET %+v", k)
	}

	rec, k, _ = kiteRequest(t, newKiteServer(t, nil), http.MethodGet, "", nil)
	if rec.Code != http.StatusOK || k.Available || k.ReadOnlyReason != readOnlyUnavailable || k.RedirectURL == "" {
		t.Fatalf("no exporter GET %d %+v", rec.Code, k)
	}
}

func TestKiteSetupPut(t *testing.T) {
	f := &fakeExporter{editable: true}
	s := newKiteServer(t, f)
	rec, k, _ := kiteRequest(t, s, http.MethodPut, `{"apiKey":" abcdef123456 ","apiSecret":"s3cretvalue42"}`, func(r *http.Request) {
		r.Header.Set("Origin", "http://localhost:8080")
		r.Header.Set("Sec-Fetch-Site", "same-origin")
	})
	if rec.Code != http.StatusOK || !k.Configured {
		t.Fatalf("PUT %d %s", rec.Code, rec.Body)
	}
	if f.gotMeth != http.MethodPut || f.gotBody != `{"api_key":"abcdef123456","api_secret":"s3cretvalue42"}` {
		t.Fatalf("forwarded %s %s", f.gotMeth, f.gotBody)
	}
	if strings.Contains(rec.Body.String(), "s3cretvalue42") {
		t.Fatal("secret echoed to the browser")
	}

	if rec, _, _ := kiteRequest(t, s, http.MethodDelete, "", nil); rec.Code != http.StatusOK || f.gotMeth != http.MethodDelete {
		t.Fatalf("DELETE %d %s", rec.Code, f.gotMeth)
	}

	f.status = http.StatusBadRequest
	rec, _, msg := kiteRequest(t, s, http.MethodPut, `{"apiKey":"x","apiSecret":"y"}`, nil)
	if rec.Code != http.StatusBadRequest || !strings.Contains(msg, "api_key") {
		t.Fatalf("validation error passthrough %d %q", rec.Code, msg)
	}
}

func TestStatusBannerWithoutKiteCredentials(t *testing.T) {
	prom := fakeProm(t, map[string]string{
		`up{job=~`:                          vec("1", nil),
		"stock_kite_session_valid":          vec("0", nil),
		"stock_kite_credentials_configured": vec("0", nil),
		"stock_feed_info":                   vec("1", map[string]string{"source": "kite"}),
	})
	defer prom.Close()
	am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `[]`) }))
	defer am.Close()
	s := newTestServer(t, prom.URL, am.URL)

	var status struct{ Data Status }
	get(t, s, "/api/v1/status", &status)
	if len(status.Data.Banners) != 1 || status.Data.Banners[0].Action == nil || status.Data.Banners[0].Action.Href != "/settings?tab=data-sources&source=kite" {
		t.Fatalf("banners %+v", status.Data.Banners)
	}
	var feed struct{ Data Feed }
	get(t, s, "/api/v1/feed", &feed)
	if !feed.Data.CredentialsMissing {
		t.Fatal("feed does not report missing credentials")
	}
}

func TestKiteSetupWriteRejects(t *testing.T) {
	f := &fakeExporter{editable: true}
	s := newKiteServer(t, f)
	valid := `{"apiKey":"abcdef123456","apiSecret":"s3cretvalue42"}`
	cases := map[string]struct {
		body string
		mod  func(*http.Request)
		want int
	}{
		"cross site":        {valid, func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, http.StatusForbidden},
		"other origin":      {valid, func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") }, http.StatusForbidden},
		"remote via portal": {valid, func(r *http.Request) { r.Header.Set("X-Forwarded-For", "10.0.0.7") }, http.StatusForbidden},
		"spoofed forward":   {valid, func(r *http.Request) { r.RemoteAddr = "10.0.0.7:4000"; r.Header.Set("X-Forwarded-For", "127.0.0.1") }, http.StatusForbidden},
		"dns rebinding": {valid, func(r *http.Request) {
			r.Host = "evil.example:8080"
			r.Header.Set("Origin", "http://evil.example:8080")
		}, http.StatusForbidden},
		"unknown field": {`{"apiKey":"a","apiSecret":"b","x":1}`, nil, http.StatusBadRequest},
		"not json":      {valid, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, http.StatusUnsupportedMediaType},
	}
	for name, c := range cases {
		f.gotMeth = ""
		if rec, _, _ := kiteRequest(t, s, http.MethodPut, c.body, c.mod); rec.Code != c.want {
			t.Errorf("%s: status %d, want %d", name, rec.Code, c.want)
		}
		if f.gotMeth == http.MethodPut {
			t.Errorf("%s: forwarded to stock_exporter", name)
		}
	}

	if rec, _, _ := kiteRequest(t, newKiteServer(t, nil), http.MethodPut, valid, nil); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("no exporter PUT %d", rec.Code)
	}
}
