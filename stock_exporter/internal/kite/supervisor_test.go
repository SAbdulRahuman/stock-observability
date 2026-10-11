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

package kite

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/seenimo/stock_exporter/internal/store"
	"github.com/seenimo/stock_exporter/internal/watchlist"
)

const (
	testKey    = "abcdef123456"
	testSecret = "s3cretvalue42"
)

var errExchange = errors.New("exchange failed")

type supFixture struct {
	sup             *Supervisor
	health          *store.Health
	creds, tokenDir string
}

func newSupervisor(t *testing.T, editable bool, initial Credentials, source, postLogin string) supFixture {
	t.Helper()
	dir := t.TempDir()
	kiteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(kiteAPI.Close)
	cfg := SupervisorConfig{
		Runner:            RunnerConfig{BaseURI: kiteAPI.URL},
		TokenPath:         filepath.Join(dir, "data", "kite_session.json"),
		PostLoginRedirect: postLogin,
		Location:          ist,
	}
	if editable {
		cfg.CredentialsPath = filepath.Join(dir, "conf", "kite_credentials.json")
	}
	h := store.NewHealth("kite", "full")
	sup, err := NewSupervisor(cfg, initial, source, &watchlist.Watchlist{}, store.New(), h, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sup.Start(ctx)
	return supFixture{sup: sup, health: h, creds: cfg.CredentialsPath, tokenDir: filepath.Dir(cfg.TokenPath)}
}

func configRequest(t *testing.T, h http.Handler, method, body string, header map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, "/kite/config", strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func TestSupervisorStartsWithoutCredentials(t *testing.T) {
	f := newSupervisor(t, true, Credentials{}, SourceNone, "")
	if st := f.sup.Status(); st.Configured || st.Source != SourceNone || !st.Editable || st.SessionValid {
		t.Fatalf("status %+v", st)
	}
	if f.health.Snapshot().Credentials {
		t.Fatal("health reports credentials")
	}
	rec := httptest.NewRecorder()
	f.sup.LoginHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("login without credentials: %d", rec.Code)
	}

	r := newSupervisor(t, true, Credentials{}, SourceNone, "/settings?tab=data-sources&source=kite")
	rec = httptest.NewRecorder()
	r.sup.CallbackHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/kite/callback", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/settings?kite_error=not_configured&source=kite&tab=data-sources" {
		t.Fatalf("callback without credentials: %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestSupervisorUpdatePersistsAndApplies(t *testing.T) {
	f := newSupervisor(t, true, Credentials{}, SourceNone, "")
	h := f.sup.ConfigHandler()
	rec, out := configRequest(t, h, http.MethodPut, `{"api_key":" `+testKey+` ","api_secret":"`+testSecret+`"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT %d: %s", rec.Code, rec.Body)
	}
	if out["configured"] != true || out["source"] != SourceFile || out["api_key_hint"] != "••••3456" {
		t.Fatalf("status %v", out)
	}
	if strings.Contains(rec.Body.String(), testSecret) || strings.Contains(rec.Body.String(), testKey) {
		t.Fatal("credentials leaked in response")
	}
	fi, err := os.Stat(f.creds)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("credentials file mode %v", fi.Mode().Perm())
	}
	if c, err := LoadCredentials(f.creds); err != nil || c.APIKey != testKey || c.APISecret != testSecret {
		t.Fatalf("persisted %+v, %v", c, err)
	}
	if !f.health.Snapshot().Credentials {
		t.Fatal("health does not report credentials")
	}

	rec = httptest.NewRecorder()
	f.sup.LoginHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	loc, _ := url.Parse(rec.Header().Get("Location"))
	if rec.Code != http.StatusFound || loc.Query().Get("api_key") != testKey {
		t.Fatalf("login redirect %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestSupervisorKeyChangeDropsToken(t *testing.T) {
	f := newSupervisor(t, true, Credentials{APIKey: testKey, APISecret: testSecret}, SourceFile, "")
	token := filepath.Join(f.tokenDir, "kite_session.json")
	writeToken := func() {
		t.Helper()
		if err := os.MkdirAll(f.tokenDir, 0o700); err != nil {
			t.Fatal(err)
		}
		// Expired, so the restarted session never calls Kite.
		if err := os.WriteFile(token, []byte(`{"access_token":"old","expires_at":"2020-01-01T06:00:00+05:30"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	writeToken()
	if err := f.sup.Update(Credentials{APIKey: testKey, APISecret: "othersecret9"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(token); err != nil {
		t.Fatal("token removed although the API key did not change")
	}
	if err := f.sup.Update(Credentials{APIKey: "newkey987654", APISecret: "othersecret9"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(token); !os.IsNotExist(err) {
		t.Fatal("token kept after the API key changed")
	}

	writeToken()
	rec, out := configRequest(t, f.sup.ConfigHandler(), http.MethodDelete, "", nil)
	if rec.Code != http.StatusOK || out["configured"] != false || out["source"] != SourceNone {
		t.Fatalf("DELETE %d %v", rec.Code, out)
	}
	for _, p := range []string{token, f.creds} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s not removed", p)
		}
	}
}

func TestSupervisorReadOnly(t *testing.T) {
	f := newSupervisor(t, false, Credentials{APIKey: testKey, APISecret: testSecret}, SourceFlags, "")
	h := f.sup.ConfigHandler()
	rec, out := configRequest(t, h, http.MethodGet, "", nil)
	if rec.Code != http.StatusOK || out["source"] != SourceFlags || out["editable"] != false || out["configured"] != true {
		t.Fatalf("GET %d %v", rec.Code, out)
	}
	if rec, _ := configRequest(t, h, http.MethodPut, `{"api_key":"`+testKey+`","api_secret":"`+testSecret+`"}`, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("PUT on read-only: %d", rec.Code)
	}
	if rec, _ := configRequest(t, h, http.MethodDelete, "", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("DELETE on read-only: %d", rec.Code)
	}
}

func TestConfigHandlerRejects(t *testing.T) {
	f := newSupervisor(t, true, Credentials{}, SourceNone, "")
	h := f.sup.ConfigHandler()
	valid := `{"api_key":"` + testKey + `","api_secret":"` + testSecret + `"}`
	cases := map[string]struct {
		method, body string
		header       map[string]string
		want         int
	}{
		"bad key":        {http.MethodPut, `{"api_key":"a b","api_secret":"` + testSecret + `"}`, nil, http.StatusBadRequest},
		"short secret":   {http.MethodPut, `{"api_key":"` + testKey + `","api_secret":"x"}`, nil, http.StatusBadRequest},
		"unknown field":  {http.MethodPut, `{"api_key":"` + testKey + `","api_secret":"` + testSecret + `","x":1}`, nil, http.StatusBadRequest},
		"oversized":      {http.MethodPut, `{"api_key":"` + strings.Repeat("a", maxConfigBody) + `"}`, nil, http.StatusBadRequest},
		"not json":       {http.MethodPut, valid, map[string]string{"Content-Type": "text/plain"}, http.StatusUnsupportedMediaType},
		"browser origin": {http.MethodPut, valid, map[string]string{"Origin": "http://localhost:8080"}, http.StatusForbidden},
		"browser fetch":  {http.MethodGet, "", map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusForbidden},
		"method":         {http.MethodPost, valid, nil, http.StatusMethodNotAllowed},
	}
	for name, c := range cases {
		if rec, _ := configRequest(t, h, c.method, c.body, c.header); rec.Code != c.want {
			t.Errorf("%s: status %d, want %d", name, rec.Code, c.want)
		}
	}
	if _, err := os.Stat(f.creds); !os.IsNotExist(err) {
		t.Fatal("rejected request wrote credentials")
	}
}

func TestLoadCredentials(t *testing.T) {
	dir := t.TempDir()
	if c, err := LoadCredentials(filepath.Join(dir, "missing.json")); err != nil || c.APIKey != "" {
		t.Fatalf("missing file: %+v %v", c, err)
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCredentials(bad); err == nil {
		t.Fatal("invalid JSON accepted")
	}
}

func TestSupervisorRejectsOpenRedirect(t *testing.T) {
	for _, p := range []string{"//evil.example", "https://evil.example/", "setup", "/\\evil"} {
		if _, err := NewSupervisor(SupervisorConfig{PostLoginRedirect: p}, Credentials{}, SourceNone, nil, nil, nil, nil); err == nil {
			t.Errorf("accepted %q", p)
		}
	}
}

func TestCallbackPostLoginRedirect(t *testing.T) {
	s, _, _ := newTestSession(t, &fakeAuth{}, time.Date(2026, 10, 6, 8, 30, 0, 0, ist))
	s.postLogin = "/settings?tab=data-sources&source=kite"
	rec := callback(s, "status=success&request_token=abc&state="+login(t, s))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/settings?kite=connected&source=kite&tab=data-sources" {
		t.Fatalf("success redirect %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := callback(s, "status=success&request_token=abc&state=bogus"); rec.Header().Get("Location") != "/settings?kite_error=state&source=kite&tab=data-sources" {
		t.Fatalf("state error redirect %q", rec.Header().Get("Location"))
	}

	f, _, _ := newTestSession(t, &fakeAuth{err: errExchange}, time.Date(2026, 10, 6, 8, 30, 0, 0, ist))
	f.postLogin = "/settings?tab=data-sources&source=kite"
	if rec := callback(f, "status=success&request_token=abc&state="+login(t, f)); rec.Header().Get("Location") != "/settings?kite_error=exchange&source=kite&tab=data-sources" {
		t.Fatalf("exchange error redirect %q", rec.Header().Get("Location"))
	}
}
