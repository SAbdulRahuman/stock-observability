// Package openbb forwards /api/v2 commands without a native port to an OpenBB Platform API server.
package openbb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_home/internal/platform"
)

const maxBodyBytes = 32 << 20

// Same command path shape as OpenBB routers; also rules out "..", "//" and encoded characters.
var validPath = regexp.MustCompile(`^[a-z0-9_]+(/[a-z0-9_]+)*$`)

// Config configures the upstream OpenBB server.
type Config struct {
	// URL is the server base URL, e.g. http://openbb:6900.
	URL      string
	Username string
	Password string
	Timeout  time.Duration
}

// Proxy is an http.Handler serving GET /api/v2/<path> from OpenBB's GET /api/v1/<path>.
type Proxy struct {
	base     *url.URL
	username string
	password string
	timeout  time.Duration
	client   *http.Client
	logger   *slog.Logger
	requests *prometheus.CounterVec
}

// New validates cfg and returns a Proxy.
func New(cfg Config, logger *slog.Logger, reg prometheus.Registerer) (*Proxy, error) {
	u, err := url.Parse(strings.TrimRight(cfg.URL, "/"))
	if err != nil {
		return nil, fmt.Errorf("openbb url: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" {
		return nil, fmt.Errorf("openbb url %q: want http(s)://host[:port][/path]", cfg.URL)
	}
	if cfg.Timeout <= 0 {
		return nil, errors.New("openbb timeout must be positive")
	}
	p := &Proxy{
		base:     u,
		username: cfg.Username,
		password: cfg.Password,
		timeout:  cfg.Timeout,
		client: &http.Client{
			// Redirects could leave the configured host.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		logger: logger,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "stock_home_openbb_requests_total",
			Help: "Calls of /api/v2 commands forwarded to OpenBB by HTTP status code.",
		}, []string{"code"}),
	}
	reg.MustRegister(p.requests)
	return p, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := p.serve(w, r)
	p.requests.WithLabelValues(strconv.Itoa(code)).Inc()
}

func (p *Proxy) serve(w http.ResponseWriter, r *http.Request) int {
	path := strings.TrimPrefix(r.URL.EscapedPath(), platform.Prefix)
	if !validPath.MatchString(path) {
		return writeError(w, http.StatusNotFound, "unknown command")
	}
	ctx, cancel := context.WithTimeout(r.Context(), p.timeout)
	defer cancel()

	target := *p.base
	target.Path = p.base.Path + "/api/v1/" + path
	target.RawPath = ""
	target.RawQuery = r.URL.RawQuery
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return writeError(w, http.StatusBadRequest, "invalid request")
	}
	req.Header.Set("Accept", "application/json")
	if p.username != "" {
		req.SetBasicAuth(p.username, p.password)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			p.logger.Warn("OpenBB command timed out", "command", path, "timeout", p.timeout)
			return writeError(w, http.StatusGatewayTimeout, "command timed out")
		}
		if r.Context().Err() != nil {
			return 499
		}
		p.logger.Warn("OpenBB unavailable", "command", path, "err", err)
		return writeError(w, http.StatusBadGateway, "upstream openbb unavailable")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		// Never relay WWW-Authenticate: browsers would prompt for OpenBB's credentials.
		p.logger.Warn("OpenBB rejected stock_home credentials", "command", path, "status", resp.StatusCode)
		return writeError(w, http.StatusBadGateway, "upstream openbb rejected credentials")
	case http.StatusNoContent:
		// OpenBB's answer when the provider has no data.
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
		return http.StatusNoContent
	}
	if mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); mt != "application/json" {
		p.logger.Warn("OpenBB returned non-JSON", "command", path, "status", resp.StatusCode, "content_type", mt)
		return writeError(w, http.StatusBadGateway, "unexpected response from upstream openbb")
	}
	if resp.StatusCode >= 500 {
		p.logger.Warn("OpenBB command failed", "command", path, "status", resp.StatusCode)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, io.LimitReader(resp.Body, maxBodyBytes)); err != nil {
		p.logger.Warn("Copying OpenBB response", "command", path, "err", err)
	}
	return resp.StatusCode
}

func writeError(w http.ResponseWriter, code int, msg string) int {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
	return code
}
