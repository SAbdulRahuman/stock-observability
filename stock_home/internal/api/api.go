// Package api serves the curated JSON consumed by the home page UI.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_home/internal/cache"
	"github.com/seenimo/stock_home/internal/holidays"
	"github.com/seenimo/stock_home/internal/source"
)

// Config holds the settings the API needs beyond its clients.
type Config struct {
	PrometheusURL   string
	AlertmanagerURL string
	GrafanaURL      string
	GrafanaToken    string // Optional service-account token for the dashboard list; never sent to the browser.
	Indices         []string
	SectorIndices   []string
	Location        *time.Location
	Holidays        []holidays.Holiday
	QueryTimeout    time.Duration
	Version         string
	Exchange        string
	SessionOpen     string // HH:MM in Location.
	SessionClose    string
	// SilencesAllowed enables POST /api/v1/silences (Acknowledge/Silence on the Alerts page).
	SilencesAllowed bool
}

// Server implements the /api/v1 endpoints.
type Server struct {
	cfg    Config
	prom   *source.Prometheus
	am     *source.Alertmanager
	graf   *source.Grafana
	http   *http.Client
	cache  *cache.Cache
	logger *slog.Logger
	now    func() time.Time
	errors *prometheus.CounterVec
}

// New returns a Server. reg receives the server's own metrics.
func New(cfg Config, prom *source.Prometheus, am *source.Alertmanager, client *http.Client, logger *slog.Logger, reg prometheus.Registerer) *Server {
	s := &Server{
		cfg:    cfg,
		prom:   prom,
		am:     am,
		graf:   source.NewGrafana(cfg.GrafanaURL, client, cfg.GrafanaToken),
		http:   client,
		cache:  cache.New(),
		logger: logger,
		now:    time.Now,
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "stock_home_section_errors_total",
			Help: "Section refreshes that failed because an upstream was unavailable.",
		}, []string{"section"}),
	}
	reg.MustRegister(s.errors)
	return s
}

// Register adds the API routes to mux.
func (s *Server) Register(mux *http.ServeMux) {
	sections := []struct {
		name string
		ttl  time.Duration
		fn   func(context.Context) (any, error)
	}{
		{"status", 5 * time.Second, s.status},
		{"feed", 5 * time.Second, s.feed},
		{"alerts", 5 * time.Second, s.alerts},
		{"market", 10 * time.Second, s.market},
		{"movers", 10 * time.Second, s.movers},
		{"sectors", 10 * time.Second, s.sectorIndices},
		{"circuits", 10 * time.Second, s.circuits},
		{"portfolio", 30 * time.Second, s.portfolio},
		{"system", 15 * time.Second, s.system},
		{"timeline", 60 * time.Second, s.timeline},
		{"dashboards", 60 * time.Second, s.dashboards},
	}
	for _, sec := range sections {
		mux.HandleFunc("GET /api/v1/"+sec.name, s.section(sec.name, sec.ttl, sec.fn))
	}
	mux.HandleFunc("GET /api/v1/links", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, envelope{Data: quickLinks, UpdatedAt: s.now()})
	})
	mux.HandleFunc("GET /api/v1/info", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, envelope{Data: s.info(), UpdatedAt: s.now()})
	})
	mux.HandleFunc("GET /api/v1/analyst", s.analystHandler)
	mux.HandleFunc("GET /api/v1/chart", s.chartHandler)
	mux.HandleFunc("GET /api/v1/quotes", s.quotesHandler)
	mux.HandleFunc("GET /api/v1/instrument", s.instrumentHandler)
	mux.HandleFunc("GET /api/v1/movers/scan", s.moversScanHandler)
	mux.HandleFunc("GET /api/v1/stocks", s.stocksHandler)
	mux.HandleFunc("GET /api/v1/alerts/items", s.section("alert-items", 5*time.Second, s.alertItems))
	mux.HandleFunc("POST /api/v1/silences", s.createSilences)
}

type envelope struct {
	Data      any       `json:"data,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
	Stale     bool      `json:"stale"`
	Error     string    `json:"error,omitempty"`
}

func (s *Server) section(name string, ttl time.Duration, fn func(context.Context) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		res, err := s.cache.Get(name, ttl, 10*ttl+time.Minute, func() (any, error) {
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.QueryTimeout)
			defer cancel()
			return fn(ctx)
		})
		if err != nil {
			s.errors.WithLabelValues(name).Inc()
			s.logger.Warn("Section refresh failed", "section", name, "err", err)
			writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
			return
		}
		writeJSON(w, http.StatusOK, envelope{Data: res.Value, UpdatedAt: res.UpdatedAt, Stale: res.Stale})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// querier runs several PromQL queries and keeps the first error.
type querier struct {
	ctx  context.Context
	prom *source.Prometheus
	err  error
}

func (s *Server) querier(ctx context.Context) *querier { return &querier{ctx: ctx, prom: s.prom} }

// scalar returns nil when the result is empty or a previous query failed.
func (q *querier) scalar(expr string) *float64 {
	if q.err != nil {
		return nil
	}
	v, ok, err := q.prom.Scalar(q.ctx, expr)
	if err != nil {
		q.err = err
		return nil
	}
	if !ok {
		return nil
	}
	return &v
}

func (q *querier) vector(expr string) []source.Sample {
	if q.err != nil {
		return nil
	}
	v, err := q.prom.Vector(q.ctx, expr)
	if err != nil {
		q.err = err
	}
	return v
}

func (q *querier) rangeQuery(expr string, start, end time.Time, step time.Duration) []source.Series {
	if q.err != nil {
		return nil
	}
	v, err := q.prom.Range(q.ctx, expr, start, end, step)
	if err != nil {
		q.err = err
	}
	return v
}

// Link is a labelled URL on the portal.
type Link struct {
	Label       string `json:"label"`
	Href        string `json:"href"`
	Description string `json:"description,omitempty"`
	ID          string `json:"id,omitempty"`
}

var quickLinks = []Link{
	{Label: "Kite login", Href: "/exporter/login", Description: "Required once per trading day", ID: "kite-login"},
	{Label: "Grafana", Href: "/grafana/", Description: "Dashboards and Explore", ID: "grafana"},
	{Label: "Grafana Alerting", Href: "/grafana/alerting/list", Description: "Rules, alerts and silences", ID: "grafana-alerting"},
	{Label: "Prometheus targets", Href: "/prometheus/targets", Description: "Scrape health", ID: "prometheus-targets"},
	{Label: "Prometheus rules", Href: "/prometheus/rules", Description: "Recording and alerting rules", ID: "prometheus-rules"},
	{Label: "Alertmanager", Href: "/alertmanager/", Description: "Alerts, silences and routing", ID: "alertmanager"},
	{Label: "Prometheus", Href: "/prometheus/", Description: "Metrics and rules", ID: "prometheus"},
	{Label: "OpenBB", Href: "/openbb", Description: "Fundamental and technical analysis", ID: "openbb"},
}

// Info is static build and exchange configuration for the app shell.
type Info struct {
	Version      string `json:"version"`
	Exchange     string `json:"exchange"`
	SessionOpen  string `json:"sessionOpen"`
	SessionClose string `json:"sessionClose"`
	Timezone     string `json:"timezone"`
	// QuoteSymbolsMax bounds /api/v1/quotes and therefore one watchlist group.
	QuoteSymbolsMax int  `json:"quoteSymbolsMax"`
	SilencesAllowed bool `json:"silencesAllowed"`
}

func (s *Server) info() Info {
	return Info{Version: s.cfg.Version, Exchange: s.cfg.Exchange, SessionOpen: s.cfg.SessionOpen, SessionClose: s.cfg.SessionClose,
		Timezone: s.cfg.Location.String(), QuoteSymbolsMax: maxQuoteSymbols, SilencesAllowed: s.cfg.SilencesAllowed}
}

// exploreURL links to Grafana Explore with a PromQL query on the provisioned Prometheus data source.
func exploreURL(expr, from string) string {
	pane := map[string]any{
		"a": map[string]any{
			"datasource": "prometheus",
			"queries": []map[string]any{{
				"refId":      "A",
				"expr":       expr,
				"datasource": map[string]string{"type": "prometheus", "uid": "prometheus"},
			}},
			"range": map[string]string{"from": from, "to": "now"},
		},
	}
	b, _ := json.Marshal(pane)
	return "/grafana/explore?schemaVersion=1&orgId=1&panes=" + url.QueryEscape(string(b))
}

func symbolExploreURL(exchange, symbol string) string {
	return exploreURL(fmt.Sprintf("max by (symbol) (stock_last_price{exchange=%q, symbol=%q})", exchange, symbol), "now-6h")
}

func unixTime(f *float64) *time.Time {
	if f == nil || *f <= 0 {
		return nil
	}
	t := time.Unix(int64(*f), 0).UTC()
	return &t
}
