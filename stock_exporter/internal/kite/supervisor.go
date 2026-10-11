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
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"

	"github.com/seenimo/stock_exporter/internal/store"
	"github.com/seenimo/stock_exporter/internal/watchlist"
)

// Credential sources reported by the config API.
const (
	SourceNone  = "none"
	SourceFile  = "file"
	SourceFlags = "flags"
)

const (
	maxConfigBody   = 4 << 10
	stopWaitTimeout = 10 * time.Second
)

var (
	apiKeyRe    = regexp.MustCompile(`^[A-Za-z0-9]{6,64}$`)
	apiSecretRe = regexp.MustCompile(`^[A-Za-z0-9]{6,128}$`)
)

// ErrReadOnly is returned when credentials come from flags and cannot be changed at runtime.
var ErrReadOnly = errors.New("kite credentials are managed by command-line flags")

// ErrInvalid wraps credential validation failures.
var ErrInvalid = errors.New("invalid credentials")

// Credentials are a Kite Connect app's API key and secret.
type Credentials struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// Validate checks the shape of both values; Kite only verifies them at login.
func (c Credentials) Validate() error {
	if !apiKeyRe.MatchString(c.APIKey) {
		return fmt.Errorf("%w: api_key must be 6-64 letters or digits", ErrInvalid)
	}
	if !apiSecretRe.MatchString(c.APISecret) {
		return fmt.Errorf("%w: api_secret must be 6-128 letters or digits", ErrInvalid)
	}
	return nil
}

func (c Credentials) configured() bool { return c.APIKey != "" && c.APISecret != "" }

// LoadCredentials reads a credentials file. A missing file yields empty credentials.
func LoadCredentials(path string) (Credentials, error) {
	var c Credentials
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return c, nil
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return Credentials{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Credentials{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// SupervisorConfig configures a Supervisor.
type SupervisorConfig struct {
	Runner    RunnerConfig
	TokenPath string
	// CredentialsPath enables runtime changes; empty means the credentials are fixed.
	CredentialsPath string
	// PostLoginRedirect is a site-relative path the browser returns to after the Kite callback.
	PostLoginRedirect string
	Location          *time.Location
}

// Supervisor owns the Kite credentials and (re)starts the session and feed whenever they change.
type Supervisor struct {
	cfg    SupervisorConfig
	wl     *watchlist.Watchlist
	st     *store.Store
	health *store.Health
	logger *slog.Logger

	mu     sync.Mutex
	ctx    context.Context
	creds  Credentials
	source string
	sess   *Session
	cancel context.CancelFunc
	done   chan struct{}
}

// NewSupervisor returns a supervisor for the given initial credentials and source.
func NewSupervisor(cfg SupervisorConfig, creds Credentials, source string, wl *watchlist.Watchlist, st *store.Store, health *store.Health, logger *slog.Logger) (*Supervisor, error) {
	if p := cfg.PostLoginRedirect; p != "" && (!strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") || strings.Contains(p, "\\")) {
		return nil, fmt.Errorf("post-login redirect %q must be a site-relative path", p)
	}
	return &Supervisor{cfg: cfg, creds: creds, source: source, wl: wl, st: st, health: health, logger: logger}, nil
}

// Start runs the pipeline for the current credentials until ctx is cancelled.
func (s *Supervisor) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
	s.applyLocked()
}

// Editable reports whether credentials can be changed at runtime.
func (s *Supervisor) Editable() bool { return s.cfg.CredentialsPath != "" }

// applyLocked stops the running pipeline and starts one for s.creds, if configured.
func (s *Supervisor) applyLocked() {
	if s.cancel != nil {
		s.cancel()
		select {
		case <-s.done:
		case <-time.After(stopWaitTimeout):
			s.logger.Warn("Previous Kite pipeline did not stop in time")
		}
		s.cancel, s.done, s.sess = nil, nil, nil
	}
	s.health.SetCredentials(s.creds.configured())
	if !s.creds.configured() {
		s.health.SetSession(false, time.Time{})
		s.logger.Warn("Kite credentials are not configured; add them in stock_home Settings or pass --kite.api-key")
		return
	}
	auth := kiteconnect.New(s.creds.APIKey)
	if s.cfg.Runner.BaseURI != "" {
		auth.SetBaseURI(s.cfg.Runner.BaseURI)
	}
	sess := NewSession(auth, s.creds.APISecret, s.cfg.TokenPath, s.cfg.Location, s.health, s.logger.With("component", "kite-session"))
	sess.postLogin = s.cfg.PostLoginRedirect
	rc := s.cfg.Runner
	rc.Feed.APIKey = s.creds.APIKey
	ctx, cancel := context.WithCancel(s.ctx)
	done := make(chan struct{})
	go sess.Watch(ctx)
	go func() {
		defer close(done)
		Run(ctx, rc, sess, s.wl, s.st, s.health, s.logger.With("component", "kite"))
	}()
	s.sess, s.cancel, s.done = sess, cancel, done
}

// Update validates, persists and applies new credentials.
func (s *Supervisor) Update(c Credentials) error {
	if !s.Editable() {
		return ErrReadOnly
	}
	if err := c.Validate(); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx == nil {
		return errors.New("supervisor not started")
	}
	// Access tokens belong to one API key.
	if c.APIKey != s.creds.APIKey {
		s.removeToken()
	}
	if err := writePrivateFile(s.cfg.CredentialsPath, b); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}
	s.creds, s.source = c, SourceFile
	s.logger.Info("Kite credentials updated", "api_key", maskKey(c.APIKey))
	s.applyLocked()
	return nil
}

// Clear removes the stored credentials and today's token, stopping the feed.
func (s *Supervisor) Clear() error {
	if !s.Editable() {
		return ErrReadOnly
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx == nil {
		return errors.New("supervisor not started")
	}
	if err := os.Remove(s.cfg.CredentialsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing credentials: %w", err)
	}
	s.removeToken()
	s.creds, s.source = Credentials{}, SourceNone
	s.logger.Info("Kite credentials removed")
	s.applyLocked()
	return nil
}

func (s *Supervisor) removeToken() {
	if err := os.Remove(s.cfg.TokenPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.logger.Warn("Could not remove Kite token file", "path", s.cfg.TokenPath, "err", err)
	}
}

// Status is the credential and session state served by the config API. It never contains secrets.
type Status struct {
	Configured       bool       `json:"configured"`
	Source           string     `json:"source"`
	Editable         bool       `json:"editable"`
	APIKeyHint       string     `json:"api_key_hint,omitempty"`
	SessionValid     bool       `json:"session_valid"`
	UserID           string     `json:"user_id,omitempty"`
	SessionExpiresAt *time.Time `json:"session_expires_at,omitempty"`
}

// Status returns the current state.
func (s *Supervisor) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{Configured: s.creds.configured(), Source: s.source, Editable: s.Editable()}
	if st.Configured {
		st.APIKeyHint = maskKey(s.creds.APIKey)
	}
	if s.sess != nil {
		valid, user, exp := s.sess.Info()
		st.SessionValid, st.UserID = valid, user
		if valid {
			st.SessionExpiresAt = &exp
		}
	}
	return st
}

func (s *Supervisor) session() *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sess
}

func maskKey(k string) string {
	if len(k) <= 4 {
		return "••••"
	}
	return "••••" + k[len(k)-4:]
}

// LoginHandler starts the browser login on the current session.
func (s *Supervisor) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sess := s.session(); sess != nil {
			sess.LoginHandler().ServeHTTP(w, r)
			return
		}
		s.notConfigured(w, r)
	})
}

// CallbackHandler completes the browser login on the current session.
func (s *Supervisor) CallbackHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sess := s.session(); sess != nil {
			sess.CallbackHandler().ServeHTTP(w, r)
			return
		}
		s.notConfigured(w, r)
	})
}

func (s *Supervisor) notConfigured(w http.ResponseWriter, r *http.Request) {
	if s.cfg.PostLoginRedirect != "" {
		http.Redirect(w, r, withQuery(s.cfg.PostLoginRedirect, "kite_error", "not_configured"), http.StatusSeeOther)
		return
	}
	http.Error(w, "Kite credentials are not configured", http.StatusServiceUnavailable)
}

// ConfigHandler serves GET, PUT and DELETE /kite/config for stock_home. Browsers are refused:
// the exporter listens on loopback and stock_home performs the user-facing checks.
func (s *Supervisor) ConfigHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Header.Get("Origin") != "" || r.Header.Get("Sec-Fetch-Site") != "" {
			writeConfigError(w, http.StatusForbidden, "browser requests are not accepted")
			return
		}
		var err error
		switch r.Method {
		case http.MethodGet:
		case http.MethodPut:
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				writeConfigError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
			var c Credentials
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxConfigBody))
			dec.DisallowUnknownFields()
			if derr := dec.Decode(&c); derr != nil {
				writeConfigError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			c.APIKey, c.APISecret = strings.TrimSpace(c.APIKey), strings.TrimSpace(c.APISecret)
			err = s.Update(c)
		case http.MethodDelete:
			err = s.Clear()
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			writeConfigError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		switch {
		case errors.Is(err, ErrReadOnly):
			writeConfigError(w, http.StatusForbidden, err.Error())
			return
		case errors.Is(err, ErrInvalid):
			writeConfigError(w, http.StatusBadRequest, err.Error())
			return
		case err != nil:
			s.logger.Error("Updating Kite credentials failed", "err", err)
			writeConfigError(w, http.StatusInternalServerError, "could not update credentials; see exporter logs")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.Status())
	})
}

func writeConfigError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
