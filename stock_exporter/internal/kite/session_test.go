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

	kiteconnect "github.com/zerodha/gokiteconnect/v4"

	"github.com/seenimo/stock_exporter/internal/store"
)

var ist, _ = time.LoadLocation("Asia/Kolkata")

type fakeAuth struct {
	gotToken, gotSecret string
	err                 error
}

func (f *fakeAuth) GenerateSession(rt, secret string) (kiteconnect.UserSession, error) {
	f.gotToken, f.gotSecret = rt, secret
	if f.err != nil {
		return kiteconnect.UserSession{}, f.err
	}
	return kiteconnect.UserSession{UserID: "AB1234", AccessToken: "acc3ss"}, nil
}

func (f *fakeAuth) GetLoginURLWithparams(p url.Values) string {
	return "https://kite.zerodha.com/connect/login?api_key=k&v=3&redirect_params=" + url.QueryEscape(p.Encode())
}

func newTestSession(t *testing.T, auth Authenticator, now time.Time) (*Session, string, *store.Health) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sub", "token.json")
	h := store.NewHealth("kite", "full")
	s := NewSession(auth, "s3cret", path, ist, h, slog.New(slog.DiscardHandler))
	s.now = func() time.Time { return now }
	return s, path, h
}

func TestExpiryAfter(t *testing.T) {
	cases := []struct{ login, want time.Time }{
		{time.Date(2026, 10, 6, 8, 30, 0, 0, ist), time.Date(2026, 10, 7, 6, 0, 0, 0, ist)},
		{time.Date(2026, 10, 6, 5, 59, 0, 0, ist), time.Date(2026, 10, 6, 6, 0, 0, 0, ist)},
		{time.Date(2026, 10, 6, 6, 0, 0, 0, ist), time.Date(2026, 10, 7, 6, 0, 0, 0, ist)},
		{time.Date(2026, 10, 6, 3, 0, 0, 0, time.UTC), time.Date(2026, 10, 7, 6, 0, 0, 0, ist)}, // 08:30 IST
	}
	for _, c := range cases {
		if got := ExpiryAfter(c.login, ist); !got.Equal(c.want) {
			t.Errorf("ExpiryAfter(%s) = %s, want %s", c.login, got, c.want)
		}
	}
}

func login(t *testing.T, s *Session) string {
	t.Helper()
	rec := httptest.NewRecorder()
	s.LoginHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("login status %d", rec.Code)
	}
	loc, _ := url.Parse(rec.Header().Get("Location"))
	rp, _ := url.ParseQuery(loc.Query().Get("redirect_params"))
	if rp.Get("state") == "" {
		t.Fatal("no state in redirect_params")
	}
	return rp.Get("state")
}

func callback(s *Session, q string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.CallbackHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/kite/callback?"+q, nil))
	return rec
}

func TestLoginFlow(t *testing.T) {
	now := time.Date(2026, 10, 6, 8, 30, 0, 0, ist)
	auth := &fakeAuth{}
	s, path, h := newTestSession(t, auth, now)
	if _, ok := s.Token(); ok {
		t.Fatal("unexpected token before login")
	}

	state := login(t, s)
	rec := callback(s, "status=success&request_token=abc123&state="+state)
	if rec.Code != http.StatusOK {
		t.Fatalf("callback status %d: %s", rec.Code, rec.Body)
	}
	if auth.gotToken != "abc123" || auth.gotSecret != "s3cret" {
		t.Fatalf("exchange got %q/%q", auth.gotToken, auth.gotSecret)
	}
	if strings.Contains(rec.Body.String(), "acc3ss") {
		t.Fatal("access token leaked in response")
	}
	if tok, ok := s.Token(); !ok || tok != "acc3ss" {
		t.Fatalf("token = %q, %v", tok, ok)
	}
	select {
	case <-s.Changed():
	default:
		t.Fatal("Changed not signalled")
	}
	if snap := h.Snapshot(); !snap.SessionValid || !snap.SessionExpiry.Equal(time.Date(2026, 10, 7, 6, 0, 0, 0, ist)) {
		t.Fatalf("health %+v", snap)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("token file mode %v", fi.Mode().Perm())
	}

	// State is single use.
	if rec := callback(s, "status=success&request_token=abc123&state="+state); rec.Code != http.StatusBadRequest {
		t.Fatalf("replayed state accepted: %d", rec.Code)
	}

	// A new session loads the persisted token.
	s2 := NewSession(auth, "s3cret", path, ist, store.NewHealth("kite", "full"), slog.New(slog.DiscardHandler))
	if tok, ok := s2.Token(); !ok || tok != "acc3ss" {
		t.Fatalf("persisted token not loaded: %q %v", tok, ok)
	}
}

func TestCallbackRejects(t *testing.T) {
	now := time.Date(2026, 10, 6, 8, 30, 0, 0, ist)
	cases := map[string]func(state string) string{
		"no state":      func(string) string { return "status=success&request_token=abc" },
		"wrong state":   func(string) string { return "status=success&request_token=abc&state=deadbeef" },
		"not success":   func(s string) string { return "status=cancelled&request_token=abc&state=" + s },
		"bad token":     func(s string) string { return "status=success&request_token=%3Cscript%3E&state=" + s },
		"missing token": func(s string) string { return "status=success&state=" + s },
	}
	for name, q := range cases {
		s, _, _ := newTestSession(t, &fakeAuth{}, now)
		state := login(t, s)
		if rec := callback(s, q(state)); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d", name, rec.Code)
		}
		if _, ok := s.Token(); ok {
			t.Errorf("%s: token set", name)
		}
	}

	s, _, _ := newTestSession(t, &fakeAuth{err: errors.New("boom")}, now)
	if rec := callback(s, "status=success&request_token=abc&state="+login(t, s)); rec.Code != http.StatusBadGateway {
		t.Errorf("exchange failure: status %d", rec.Code)
	}
}

func TestStateExpiresAndIsBounded(t *testing.T) {
	now := time.Date(2026, 10, 6, 8, 30, 0, 0, ist)
	s, _, _ := newTestSession(t, &fakeAuth{}, now)
	state := login(t, s)
	s.now = func() time.Time { return now.Add(stateTTL + time.Second) }
	if rec := callback(s, "status=success&request_token=abc&state="+state); rec.Code != http.StatusBadRequest {
		t.Fatalf("expired state accepted: %d", rec.Code)
	}
	s.now = func() time.Time { return now }
	for range maxStates {
		login(t, s)
	}
	rec := httptest.NewRecorder()
	s.LoginHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected pending-login limit, got %d", rec.Code)
	}
}

func TestLoadPlainAndExpiredTokens(t *testing.T) {
	dir := t.TempDir()
	h := store.NewHealth("kite", "full")
	logger := slog.New(slog.DiscardHandler)

	plain := filepath.Join(dir, "plain")
	if err := os.WriteFile(plain, []byte("tok123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if tok, ok := NewSession(&fakeAuth{}, "x", plain, ist, h, logger).Token(); !ok || tok != "tok123" {
		t.Fatalf("plain token not loaded: %q %v", tok, ok)
	}

	expired := filepath.Join(dir, "expired.json")
	if err := os.WriteFile(expired, []byte(`{"access_token":"old","expires_at":"2020-01-01T06:00:00+05:30"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := NewSession(&fakeAuth{}, "x", expired, ist, h, logger).Token(); ok {
		t.Fatal("expired token loaded")
	}
}

func TestInvalidate(t *testing.T) {
	s, _, h := newTestSession(t, &fakeAuth{}, time.Date(2026, 10, 6, 8, 30, 0, 0, ist))
	callback(s, "status=success&request_token=abc&state="+login(t, s))
	<-s.Changed()
	s.Invalidate("test")
	if _, ok := s.Token(); ok {
		t.Fatal("token still valid")
	}
	if h.Snapshot().SessionValid {
		t.Fatal("health still valid")
	}
	select {
	case <-s.Changed():
	default:
		t.Fatal("Changed not signalled on invalidate")
	}
}
