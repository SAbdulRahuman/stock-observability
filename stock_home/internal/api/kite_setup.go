package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	kiteConsoleURL   = "https://developers.kite.trade/apps"
	kiteLoginPath    = "/exporter/login"
	kiteCallbackPath = "/exporter/kite/callback"
	maxKiteBody      = 4 << 10
)

// Reasons the Kite credentials cannot be edited from this browser.
const (
	readOnlyRemote      = "remote"      // Request is not from this computer.
	readOnlyManaged     = "managed"     // stock_exporter takes its credentials from flags (e.g. a Kubernetes Secret).
	readOnlyUnavailable = "unavailable" // No --exporter.url, or stock_exporter is unreachable.
)

// KiteSetup is the Kite Connect state shown in Settings. It never holds secrets.
type KiteSetup struct {
	Available        bool       `json:"available"`
	Configured       bool       `json:"configured"`
	Source           string     `json:"source,omitempty"`
	Editable         bool       `json:"editable"`
	ReadOnlyReason   string     `json:"readOnlyReason,omitempty"`
	APIKeyHint       string     `json:"apiKeyHint,omitempty"`
	SessionValid     bool       `json:"sessionValid"`
	UserID           string     `json:"userId,omitempty"`
	SessionExpiresAt *time.Time `json:"sessionExpiresAt,omitempty"`
	RedirectURL      string     `json:"redirectUrl"`
	LoginURL         string     `json:"loginUrl"`
	ConsoleURL       string     `json:"consoleUrl"`
}

// exporterStatus mirrors stock_exporter's /kite/config response.
type exporterStatus struct {
	Configured       bool       `json:"configured"`
	Source           string     `json:"source"`
	Editable         bool       `json:"editable"`
	APIKeyHint       string     `json:"api_key_hint"`
	SessionValid     bool       `json:"session_valid"`
	UserID           string     `json:"user_id"`
	SessionExpiresAt *time.Time `json:"session_expires_at"`
}

type kiteCredentialsBody struct {
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
}

func (s *Server) registerKiteSetup(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/kite/setup", s.kiteSetupGet)
	mux.HandleFunc("PUT /api/v1/kite/setup", s.kiteSetupWrite)
	mux.HandleFunc("DELETE /api/v1/kite/setup", s.kiteSetupWrite)
}

func (s *Server) kiteSetupGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	st, err := s.exporterKite(r.Context(), http.MethodGet, nil)
	writeJSON(w, http.StatusOK, envelope{Data: s.kiteSetup(r, st, err), UpdatedAt: s.now()})
}

func (s *Server) kiteSetupWrite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	fail := func(code int, msg string) { writeJSON(w, code, envelope{Error: msg, UpdatedAt: s.now()}) }
	if !sameOrigin(r) {
		fail(http.StatusForbidden, "cross-origin request rejected")
		return
	}
	if !localRequest(r) {
		fail(http.StatusForbidden, "Kite credentials can only be changed from the computer running iSee Ticks")
		return
	}
	if s.cfg.ExporterURL == "" {
		fail(http.StatusServiceUnavailable, "stock_exporter is not configured (--exporter.url)")
		return
	}
	var body []byte
	if r.Method == http.MethodPut {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			fail(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return
		}
		var in kiteCredentialsBody
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxKiteBody))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			fail(http.StatusBadRequest, "invalid JSON body")
			return
		}
		body, _ = json.Marshal(map[string]string{"api_key": strings.TrimSpace(in.APIKey), "api_secret": strings.TrimSpace(in.APISecret)})
	}
	st, err := s.exporterKite(r.Context(), r.Method, body)
	var ue *upstreamError
	switch {
	case errors.As(err, &ue) && (ue.code == http.StatusBadRequest || ue.code == http.StatusForbidden):
		fail(ue.code, ue.msg)
		return
	case err != nil:
		s.logger.Warn("Updating Kite credentials failed", "err", err)
		fail(http.StatusBadGateway, "stock_exporter did not accept the change")
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: s.kiteSetup(r, st, nil), UpdatedAt: s.now()})
}

func (s *Server) kiteSetup(r *http.Request, st *exporterStatus, err error) KiteSetup {
	k := KiteSetup{RedirectURL: kiteRedirectURL(r), LoginURL: kiteLoginPath, ConsoleURL: kiteConsoleURL}
	if err != nil || st == nil {
		if err != nil && s.cfg.ExporterURL != "" {
			s.logger.Warn("Reading Kite setup from stock_exporter failed", "err", err)
		}
		k.ReadOnlyReason = readOnlyUnavailable
		return k
	}
	k.Available = true
	k.Configured, k.Source, k.APIKeyHint = st.Configured, st.Source, st.APIKeyHint
	k.SessionValid, k.UserID, k.SessionExpiresAt = st.SessionValid, st.UserID, st.SessionExpiresAt
	switch {
	case !st.Editable:
		k.ReadOnlyReason = readOnlyManaged
	case !localRequest(r):
		k.ReadOnlyReason = readOnlyRemote
	default:
		k.Editable = true
	}
	return k
}

type upstreamError struct {
	code int
	msg  string
}

func (e *upstreamError) Error() string { return e.msg }

func (s *Server) exporterKite(ctx context.Context, method string, body []byte) (*exporterStatus, error) {
	if s.cfg.ExporterURL == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, s.cfg.QueryTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, s.cfg.ExporterURL+"/kite/config", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var e struct{ Error string }
		_ = json.Unmarshal(b, &e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return nil, &upstreamError{code: resp.StatusCode, msg: e.Error}
	}
	var st exporterStatus
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// localRequest reports whether the browser runs on this computer. X-Forwarded-For is only trusted from a
// loopback proxy (the portal), and the Host must be a loopback name so DNS rebinding cannot pass.
func localRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !isLoopback(host) {
		return false
	}
	if xff := r.Header.Values("X-Forwarded-For"); len(xff) > 0 {
		hops := strings.Split(strings.Join(xff, ","), ",")
		if !isLoopback(strings.TrimSpace(hops[len(hops)-1])) {
			return false
		}
	}
	return isLoopback(hostname(r.Host))
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func hostname(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return strings.Trim(hostport, "[]")
}

// kiteRedirectURL is the URL to register in the Kite app. Kite needs an exact match, so localhost becomes 127.0.0.1.
func kiteRedirectURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if h := hostname(host); isLoopback(h) {
		if _, port, err := net.SplitHostPort(host); err == nil {
			host = net.JoinHostPort("127.0.0.1", port)
		} else {
			host = "127.0.0.1"
		}
	}
	return scheme + "://" + host + kiteCallbackPath
}
