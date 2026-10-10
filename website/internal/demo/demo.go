// Package demo is the read-only public reverse proxy in front of stock_home and Grafana.
package demo

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/seenimo/stock_website/internal/ratelimit"
)

// Config selects the upstreams. GrafanaURL may be nil; Grafana must use serve_from_sub_path with /grafana.
type Config struct {
	HomeURL    *url.URL
	GrafanaURL *url.URL
	Limit      *ratelimit.Limiter
	Logger     *slog.Logger
}

// Paths that would let anonymous visitors log in or change Grafana; compared lower-cased.
var grafanaBlocked = []string{
	"/grafana/login", "/grafana/admin", "/grafana/profile", "/grafana/org",
	"/grafana/api/admin", "/grafana/api/user", "/grafana/api/users", "/grafana/api/org",
	"/grafana/api/orgs", "/grafana/api/serviceaccounts", "/grafana/api/auth", "/grafana/api/teams",
}

// Grafana panels query data with POST; nothing else may write.
var grafanaPOST = map[string]bool{
	"/grafana/api/ds/query":         true,
	"/grafana/api/frontend-metrics": true,
}

type proxy struct {
	home, grafana http.Handler
	limit         *ratelimit.Limiter
}

// New returns the demo handler.
func New(cfg Config) http.Handler {
	p := &proxy{home: reverse(cfg.HomeURL, cfg.Logger), limit: cfg.Limit}
	if cfg.GrafanaURL != nil {
		p.grafana = reverse(cfg.GrafanaURL, cfg.Logger)
	}
	return p
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex")
	if p.limit != nil && !p.limit.Allow(ratelimit.ClientIP(r)) {
		w.Header().Set("Retry-After", "5")
		http.Error(w, "Too many requests.", http.StatusTooManyRequests)
		return
	}
	clean := path.Clean("/" + r.URL.Path)
	if strings.HasSuffix(r.URL.Path, "/") && clean != "/" {
		clean += "/"
	}
	lower := strings.ToLower(clean)
	r = r.Clone(r.Context())
	r.URL.Path, r.URL.RawPath = clean, ""

	if lower == "/grafana" || strings.HasPrefix(lower, "/grafana/") {
		switch {
		case p.grafana == nil:
			http.NotFound(w, r)
		case blocked(lower):
			http.Error(w, "Not available in the public demo.", http.StatusForbidden)
		case !readOnly(r.Method) && !(r.Method == http.MethodPost && grafanaPOST[strings.TrimSuffix(lower, "/")]):
			methodNotAllowed(w)
		default:
			p.grafana.ServeHTTP(w, r)
		}
		return
	}
	switch {
	case lower == "/metrics":
		http.NotFound(w, r)
	case !readOnly(r.Method):
		methodNotAllowed(w)
	default:
		p.home.ServeHTTP(w, r)
	}
}

func blocked(p string) bool {
	for _, b := range grafanaBlocked {
		if p == b || strings.HasPrefix(p, b+"/") {
			return true
		}
	}
	return false
}

func readOnly(m string) bool { return m == http.MethodGet || m == http.MethodHead }

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "The public demo is read-only.", http.StatusMethodNotAllowed)
}

func reverse(target *url.URL, logger *slog.Logger) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
			// Grafana's CSRF check compares Origin with Host, so keep the public host.
			pr.Out.Host = pr.In.Host
		},
		Transport: &http.Transport{
			Proxy:                 nil,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConnsPerHost:   16,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Warn("Demo upstream failed", "upstream", target.Host, "path", r.URL.Path, "err", err)
			http.Error(w, "The demo is unavailable right now. Please try again shortly.", http.StatusBadGateway)
		},
	}
}
