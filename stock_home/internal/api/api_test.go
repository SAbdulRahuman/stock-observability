package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_home/internal/source"
)

// fakeProm answers instant queries from a map of query substring -> vector JSON.
func fakeProm(t *testing.T, answers map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		q := r.Form.Get("query")
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/query_range") {
			io.WriteString(w, `{"status":"success","data":{"resultType":"matrix","result":[]}}`)
			return
		}
		result := "[]"
		for k, v := range answers {
			if strings.Contains(q, k) {
				result = v
				break
			}
		}
		io.WriteString(w, `{"status":"success","data":{"resultType":"vector","result":`+result+`}}`)
	}))
}

func vec(value string, labels map[string]string) string {
	b, _ := json.Marshal(labels)
	return `[{"metric":` + string(b) + `,"value":[1700000000,"` + value + `"]}]`
}

func newTestServer(t *testing.T, promURL, amURL string) *Server {
	t.Helper()
	prom, err := source.NewPrometheus(promURL)
	if err != nil {
		t.Fatal(err)
	}
	ist, _ := time.LoadLocation("Asia/Kolkata")
	cfg := Config{PrometheusURL: promURL, AlertmanagerURL: amURL, GrafanaURL: amURL, Indices: []string{"NIFTY 50"}, Location: ist, QueryTimeout: 5 * time.Second}
	return New(cfg, prom, source.NewAlertmanager(amURL, http.DefaultClient), http.DefaultClient, slog.New(slog.DiscardHandler), prometheus.NewRegistry())
}

func get(t *testing.T, s *Server, path string, v any) int {
	t.Helper()
	mux := http.NewServeMux()
	s.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if v != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
			t.Fatalf("%s: decode %q: %v", path, rec.Body.String(), err)
		}
	}
	return rec.Code
}

func TestStatusBanners(t *testing.T) {
	prom := fakeProm(t, map[string]string{
		"stock_market_phase":       vec("1", map[string]string{"phase": "open"}),
		"stock_market_open":        vec("1", nil),
		"stock_market_trading_day": vec("1", nil),
		`up{job=~`:                 vec("1", nil),
		"stock_kite_session_valid": vec("0", nil),
		"stock_feed_info":          vec("1", map[string]string{"source": "kite"}),
		"stale_core_instruments":   vec("0.5", nil),
	})
	defer prom.Close()
	am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[{"labels":{"alertname":"KiteSessionInvalid","severity":"critical"},"status":{"state":"active"}}]`)
	}))
	defer am.Close()

	var env struct {
		Data Status `json:"data"`
	}
	if code := get(t, newTestServer(t, prom.URL, am.URL), "/api/v1/status", &env); code != http.StatusOK {
		t.Fatalf("status code %d", code)
	}
	if env.Data.Market.Phase != "open" || !env.Data.Market.Open {
		t.Errorf("market = %+v", env.Data.Market)
	}
	titles := map[string]bool{}
	for _, b := range env.Data.Banners {
		titles[b.Title] = true
	}
	for _, want := range []string{"Kite session expired", "Market data stale", "1 critical alert firing"} {
		if !titles[want] {
			t.Errorf("missing banner %q in %+v", want, env.Data.Banners)
		}
	}
}

func TestStatusSurvivesUpstreamOutage(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer down.Close()

	var env struct {
		Data Status `json:"data"`
	}
	if code := get(t, newTestServer(t, down.URL, down.URL), "/api/v1/status", &env); code != http.StatusOK {
		t.Fatalf("status code %d", code)
	}
	if len(env.Data.Banners) != 2 {
		t.Fatalf("want Prometheus and Alertmanager banners, got %+v", env.Data.Banners)
	}
}

func TestSectionErrorIsGeneric(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "secret internal detail", http.StatusServiceUnavailable)
	}))
	defer down.Close()

	var env envelope
	if code := get(t, newTestServer(t, down.URL, down.URL), "/api/v1/alerts", &env); code != http.StatusBadGateway {
		t.Fatalf("status code %d", code)
	}
	if env.Error != "upstream unavailable" {
		t.Errorf("error = %q", env.Error)
	}
}

func TestAlertsShaping(t *testing.T) {
	am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/silences") {
			io.WriteString(w, `[{"id":"1","status":{"state":"active"}},{"id":"2","status":{"state":"expired"}}]`)
			return
		}
		io.WriteString(w, `[
 {"labels":{"alertname":"A","severity":"info","category":"price"},"startsAt":"2026-10-06T09:00:00Z","status":{"state":"active"}},
 {"labels":{"alertname":"B","severity":"critical","category":"infra"},"annotations":{"summary":"down"},"startsAt":"2026-10-06T08:00:00Z","status":{"state":"active"}},
 {"labels":{"alertname":"C","severity":"warning"},"status":{"state":"suppressed","silencedBy":["1"]}},
 {"labels":{"alertname":"D","severity":"warning"},"status":{"state":"suppressed","inhibitedBy":["x"]}}
]`)
	}))
	defer am.Close()

	var env struct {
		Data Alerts `json:"data"`
	}
	get(t, newTestServer(t, am.URL, am.URL), "/api/v1/alerts", &env)
	a := env.Data
	if a.BySeverity["critical"] != 1 || a.BySeverity["info"] != 1 || a.Silenced != 1 || a.Inhibited != 1 || a.ActiveSilences != 1 {
		t.Errorf("counts = %+v", a)
	}
	if len(a.Items) != 2 || a.Items[0].Name != "B" || a.Items[0].Summary != "down" {
		t.Errorf("items = %+v", a.Items)
	}
}

func TestBuildTimeline(t *testing.T) {
	step := time.Minute
	end := time.Unix(10_000, 0)
	start := end.Add(-time.Hour)
	ms := func(min int) int64 { return start.Add(time.Duration(min) * time.Minute).UnixMilli() }
	series := []source.Series{{
		Labels: map[string]string{"alertname": "X", "severity": "warning", "symbol": "INFY"},
		Points: []source.Point{{T: ms(1)}, {T: ms(2)}, {T: ms(3)}, {T: ms(10)}, {T: ms(59)}, {T: ms(60)}},
	}}
	tl := buildTimeline(series, start, end, step)
	if len(tl.Items) != 3 {
		t.Fatalf("want 3 runs, got %+v", tl.Items)
	}
	if tl.Items[0].End != nil {
		t.Errorf("latest run should be ongoing: %+v", tl.Items[0])
	}
	if tl.Items[2].End == nil || !tl.Items[2].End.Equal(time.UnixMilli(ms(4)).UTC()) {
		t.Errorf("first run end = %v", tl.Items[2].End)
	}
	if tl.BySeverity["warning"] != 3 || tl.Items[0].Subject != "INFY" {
		t.Errorf("timeline = %+v", tl)
	}
}

func TestExploreURLEscapesQuery(t *testing.T) {
	u := symbolExploreURL("NSE", `M&M"x`)
	if !strings.HasPrefix(u, "/grafana/explore?") || strings.Contains(u, `"`) || strings.Contains(u, " ") {
		t.Errorf("unexpected URL %s", u)
	}
}
