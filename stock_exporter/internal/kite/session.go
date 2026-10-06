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

// Package kite integrates the Zerodha Kite Connect API: login session,
// WebSocket market feed and rate-limited REST pollers.
package kite

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"

	"github.com/seenimo/stock_exporter/internal/store"
)

const (
	stateTTL     = 10 * time.Minute
	maxStates    = 64
	expiryHour   = 6
	checkPeriod  = 30 * time.Second
	tokenFileMod = 0o600
)

var requestTokenRe = regexp.MustCompile(`^[A-Za-z0-9]{1,128}$`)

// Authenticator exchanges a request token for a session. *kiteconnect.Client implements it.
type Authenticator interface {
	GenerateSession(requestToken, apiSecret string) (kiteconnect.UserSession, error)
	GetLoginURLWithparams(p url.Values) string
}

type persistedToken struct {
	AccessToken string    `json:"access_token"`
	UserID      string    `json:"user_id"`
	LoginTime   time.Time `json:"login_time"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Session manages the daily Kite access token. Login is always initiated
// manually in a browser; automating credentials/TOTP violates Kite's terms.
type Session struct {
	mu        sync.Mutex
	auth      Authenticator
	apiSecret string
	tokenPath string
	loc       *time.Location
	health    *store.Health
	logger    *slog.Logger
	now       func() time.Time

	token   string
	userID  string
	expiry  time.Time
	states  map[string]time.Time
	changed chan struct{}
}

// NewSession creates a session and loads a previously persisted token, if still valid.
func NewSession(auth Authenticator, apiSecret, tokenPath string, loc *time.Location, health *store.Health, logger *slog.Logger) *Session {
	s := &Session{
		auth: auth, apiSecret: apiSecret, tokenPath: tokenPath, loc: loc, health: health, logger: logger,
		now: time.Now, states: map[string]time.Time{}, changed: make(chan struct{}, 1),
	}
	if err := s.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Warn("Ignoring persisted Kite token", "path", tokenPath, "err", err)
	}
	s.publishHealth()
	return s
}

// ExpiryAfter returns the first 06:00 exchange-local time after login, when Kite tokens expire.
func ExpiryAfter(login time.Time, loc *time.Location) time.Time {
	l := login.In(loc)
	exp := time.Date(l.Year(), l.Month(), l.Day(), expiryHour, 0, 0, 0, loc)
	if !exp.After(l) {
		exp = exp.AddDate(0, 0, 1)
	}
	return exp
}

func (s *Session) load() error {
	b, err := os.ReadFile(s.tokenPath)
	if err != nil {
		return err
	}
	var p persistedToken
	if jerr := json.Unmarshal(b, &p); jerr != nil {
		// Plain token files are supported for externally managed tokens.
		raw := strings.TrimSpace(string(b))
		if !requestTokenRe.MatchString(raw) {
			return fmt.Errorf("token file is neither JSON nor a plain token: %w", jerr)
		}
		fi, err := os.Stat(s.tokenPath)
		if err != nil {
			return err
		}
		p = persistedToken{AccessToken: raw, LoginTime: fi.ModTime(), ExpiresAt: ExpiryAfter(fi.ModTime(), s.loc)}
	}
	if p.AccessToken == "" {
		return errors.New("token file has no access_token")
	}
	if !s.now().Before(p.ExpiresAt) {
		return fmt.Errorf("token expired at %s", p.ExpiresAt)
	}
	s.mu.Lock()
	s.token, s.userID, s.expiry = p.AccessToken, p.UserID, p.ExpiresAt
	s.mu.Unlock()
	s.logger.Info("Loaded persisted Kite session", "user_id", p.UserID, "expires_at", p.ExpiresAt)
	return nil
}

func (s *Session) persist(p persistedToken) error {
	if err := os.MkdirAll(filepath.Dir(s.tokenPath), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.tokenPath), ".kite-token-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(tokenFileMod); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.tokenPath)
}

// Token returns the access token if one is present and not expired.
func (s *Session) Token() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == "" || !s.now().Before(s.expiry) {
		return "", false
	}
	return s.token, true
}

// Changed is signalled whenever the token is set or invalidated.
func (s *Session) Changed() <-chan struct{} { return s.changed }

func (s *Session) notify() {
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

func (s *Session) publishHealth() {
	s.mu.Lock()
	valid := s.token != "" && s.now().Before(s.expiry)
	exp := s.expiry
	s.mu.Unlock()
	s.health.SetSession(valid, exp)
}

// Invalidate drops the current token, e.g. after a TokenException.
func (s *Session) Invalidate(reason string) {
	s.mu.Lock()
	had := s.token != ""
	s.token = ""
	s.mu.Unlock()
	s.publishHealth()
	if had {
		s.logger.Warn("Kite session invalidated, browser login required", "reason", reason)
		s.notify()
	}
}

// Watch expires the token at its expiry time and keeps health current.
func (s *Session) Watch(ctx context.Context) {
	t := time.NewTicker(checkPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.mu.Lock()
			expired := s.token != "" && !s.now().Before(s.expiry)
			s.mu.Unlock()
			if expired {
				s.Invalidate("token expired")
			}
			s.publishHealth()
		}
	}
}

func (s *Session) newState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	st := hex.EncodeToString(b)
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, exp := range s.states {
		if now.After(exp) {
			delete(s.states, k)
		}
	}
	if len(s.states) >= maxStates {
		return "", errors.New("too many pending logins")
	}
	s.states[st] = now.Add(stateTTL)
	return st, nil
}

func (s *Session) consumeState(st string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.states[st]
	delete(s.states, st)
	return ok && s.now().Before(exp)
}

// LoginHandler redirects the browser to the Kite login page.
func (s *Session) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st, err := s.newState()
		if err != nil {
			http.Error(w, "cannot start login: "+err.Error(), http.StatusTooManyRequests)
			return
		}
		http.Redirect(w, r, s.auth.GetLoginURLWithparams(url.Values{"state": {st}}), http.StatusFound)
	})
}

// CallbackHandler is the Kite redirect URL. It validates state and exchanges the request token.
func (s *Session) CallbackHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		q := r.URL.Query()
		if !s.consumeState(q.Get("state")) {
			http.Error(w, "invalid or expired login state; start again at /login", http.StatusBadRequest)
			return
		}
		if q.Get("status") != "success" {
			http.Error(w, "Kite login was not successful", http.StatusBadRequest)
			return
		}
		rt := q.Get("request_token")
		if !requestTokenRe.MatchString(rt) {
			http.Error(w, "invalid request_token", http.StatusBadRequest)
			return
		}
		sess, err := s.auth.GenerateSession(rt, s.apiSecret)
		if err != nil {
			s.logger.Error("Kite session exchange failed", "err", err)
			http.Error(w, "session exchange with Kite failed; see exporter logs", http.StatusBadGateway)
			return
		}
		login := s.now()
		p := persistedToken{AccessToken: sess.AccessToken, UserID: sess.UserID, LoginTime: login, ExpiresAt: ExpiryAfter(login, s.loc)}
		if err := s.persist(p); err != nil {
			s.logger.Error("Could not persist Kite token; it will be lost on restart", "path", s.tokenPath, "err", err)
		}
		s.mu.Lock()
		s.token, s.userID, s.expiry = p.AccessToken, p.UserID, p.ExpiresAt
		s.mu.Unlock()
		s.publishHealth()
		s.notify()
		s.logger.Info("Kite session established", "user_id", p.UserID, "expires_at", p.ExpiresAt)
		fmt.Fprintf(w, "Kite session established for %s, valid until %s.\n", p.UserID, p.ExpiresAt.Format(time.RFC1123))
	})
}
