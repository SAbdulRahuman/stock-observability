package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/seenimo/stock_home/internal/source"
)

// Integration is one external service as configured on this server; credentials are reduced to a description.
type Integration struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
	Endpoint   string `json:"endpoint,omitempty"` // Without user info, query or fragment.
	Auth       string `json:"auth"`
	Note       string `json:"note,omitempty"`
}

// IntegrationTest is the outcome of one connectivity check.
type IntegrationTest struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"` // connected, error or not_configured.
	Message   string    `json:"message"`
	LatencyMs float64   `json:"latencyMs"`
	CheckedAt time.Time `json:"checkedAt"`
}

var integrationIDs = []string{"kite", "openbb", "prometheus", "alertmanager", "grafana"}

// integrationTestTTL throttles repeated Test clicks to one upstream check per service.
const integrationTestTTL = 5 * time.Second

func (s *Server) registerIntegrations(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/integrations", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, envelope{Data: s.integrations(), UpdatedAt: s.now()})
	})
	mux.HandleFunc("POST /api/v1/integrations/{id}/test", s.integrationTest)
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	return u.String()
}

func (s *Server) integrations() []Integration {
	ob := Integration{ID: "openbb", Name: "OpenBB", Configured: s.cfg.OpenBBURL != "", Endpoint: redactURL(s.cfg.OpenBBURL), Auth: "none",
		Note: "Commands without a native implementation are forwarded here (--openbb.url)."}
	if s.cfg.OpenBBUsername != "" {
		ob.Auth = "basic auth configured (masked)"
	}
	graf := Integration{ID: "grafana", Name: "Grafana", Configured: s.cfg.GrafanaURL != "", Endpoint: redactURL(s.cfg.GrafanaURL), Auth: "anonymous"}
	if s.cfg.GrafanaToken != "" {
		graf.Auth = "service-account token configured (masked)"
	}
	return []Integration{
		{ID: "kite", Name: "Zerodha Kite API", Configured: true, Auth: "credentials are held by stock_exporter",
			Note: "Set the API key and secret in the Kite Connect card. The status comes from stock_exporter metrics in Prometheus."},
		ob,
		{ID: "prometheus", Name: "Prometheus", Configured: s.cfg.PrometheusURL != "", Endpoint: redactURL(s.cfg.PrometheusURL), Auth: "none"},
		{ID: "alertmanager", Name: "Alertmanager", Configured: s.cfg.AlertmanagerURL != "", Endpoint: redactURL(s.cfg.AlertmanagerURL), Auth: "none"},
		graf,
	}
}

func (s *Server) integrationTest(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, envelope{Error: "cross-origin request rejected", UpdatedAt: s.now()})
		return
	}
	id := r.PathValue("id")
	if !slices.Contains(integrationIDs, id) {
		writeJSON(w, http.StatusNotFound, envelope{Error: "unknown integration", UpdatedAt: s.now()})
		return
	}
	res, err := s.cache.Get("integration-test:"+id, integrationTestTTL, integrationTestTTL, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.QueryTimeout)
		defer cancel()
		return s.testIntegration(ctx, id), nil
	})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: res.Value, UpdatedAt: res.UpdatedAt})
}

func (s *Server) testIntegration(ctx context.Context, id string) IntegrationTest {
	start := time.Now()
	t := IntegrationTest{ID: id}
	set := func(status, msg string) IntegrationTest {
		t.Status, t.Message = status, msg
		t.LatencyMs = float64(time.Since(start).Microseconds()) / 1000
		t.CheckedAt = s.now().UTC()
		return t
	}
	ready := func(name, u string) IntegrationTest {
		if ok, _ := source.Probe(ctx, s.http, u); ok {
			return set("connected", name+" is ready")
		}
		return set("error", name+" is unreachable or not ready")
	}
	switch id {
	case "prometheus":
		return ready("Prometheus", s.cfg.PrometheusURL+"/-/ready")
	case "alertmanager":
		return ready("Alertmanager", s.cfg.AlertmanagerURL+"/-/ready")
	case "grafana":
		if ok, _ := source.Probe(ctx, s.http, s.cfg.GrafanaURL+"/api/health"); !ok {
			return set("error", "Grafana is unreachable or unhealthy")
		}
		if s.cfg.GrafanaToken == "" {
			return set("connected", "Grafana is healthy (anonymous access)")
		}
		if _, err := s.graf.Dashboards(ctx); err != nil {
			return set("error", "Grafana is healthy but the API token was rejected")
		}
		return set("connected", "Grafana is healthy and the API token is accepted")
	case "openbb":
		if s.cfg.OpenBBURL == "" {
			return set("not_configured", "No OpenBB server is configured (--openbb.url)")
		}
		code, err := s.openbbStatus(ctx)
		switch {
		case err != nil:
			return set("error", "OpenBB is unreachable")
		case code == http.StatusUnauthorized || code == http.StatusForbidden:
			return set("error", "OpenBB rejected the configured credentials")
		case code >= 500:
			return set("error", fmt.Sprintf("OpenBB answered HTTP %d", code))
		}
		return set("connected", fmt.Sprintf("OpenBB is reachable (HTTP %d)", code))
	default: // kite
		q := s.querier(ctx)
		valid := q.scalar(`max(stock_kite_session_valid)`)
		feed := q.vector(`stock_feed_info`)
		if q.err != nil {
			return set("error", "Prometheus query failed; the Kite status is unknown")
		}
		src := ""
		if len(feed) > 0 {
			src = feed[0].Labels["source"]
		}
		switch {
		case valid == nil && src != "" && src != "kite":
			return set("not_configured", "The market feed source is "+src+"; Kite is not in use")
		case valid == nil:
			return set("not_configured", "No Kite session metrics in Prometheus")
		case *valid == 1:
			return set("connected", "Kite session is valid")
		}
		return set("error", "Kite session expired; log in to Kite")
	}
}

func (s *Server) openbbStatus(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.OpenBBURL+"/", nil)
	if err != nil {
		return 0, err
	}
	if s.cfg.OpenBBUsername != "" {
		req.SetBasicAuth(s.cfg.OpenBBUsername, s.cfg.OpenBBPassword)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	resp.Body.Close()
	return resp.StatusCode, nil
}
