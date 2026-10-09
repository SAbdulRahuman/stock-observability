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
	cfg := Config{PrometheusURL: promURL, AlertmanagerURL: amURL, GrafanaURL: amURL, Indices: []string{"NIFTY 50"}, Location: ist, QueryTimeout: 5 * time.Second,
		Version: "0.1.0", Exchange: "NSE", SessionOpen: "09:15", SessionClose: "15:30"}
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

func TestShellInfoAndLinks(t *testing.T) {
	s := newTestServer(t, "http://127.0.0.1:0", "http://127.0.0.1:0")
	var info struct {
		Data Info `json:"data"`
	}
	if code := get(t, s, "/api/v1/info", &info); code != http.StatusOK {
		t.Fatalf("info status %d", code)
	}
	if want := (Info{Version: "0.1.0", Exchange: "NSE", SessionOpen: "09:15", SessionClose: "15:30", Timezone: "Asia/Kolkata", QuoteSymbolsMax: 20}); info.Data != want {
		t.Errorf("info = %+v", info.Data)
	}
	var links struct {
		Data []Link `json:"data"`
	}
	get(t, s, "/api/v1/links", &links)
	ids := map[string]string{}
	for _, l := range links.Data {
		ids[l.ID] = l.Href
	}
	for _, id := range []string{"grafana", "prometheus", "alertmanager", "openbb"} {
		if !strings.HasPrefix(ids[id], "/") {
			t.Errorf("link %q missing or not same-origin: %q", id, ids[id])
		}
	}
}

func TestExploreURLEscapesQuery(t *testing.T) {
	u := symbolExploreURL("NSE", `M&M"x`)
	if !strings.HasPrefix(u, "/grafana/explore?") || strings.Contains(u, `"`) || strings.Contains(u, " ") {
		t.Errorf("unexpected URL %s", u)
	}
}

func TestAnalyst(t *testing.T) {
	infy := `"exchange":"NSE","symbol":"INFY"`
	prom := fakeProm(t, map[string]string{
		`__name__=~"stock_change_percent`: `[` +
			`{"metric":{"__name__":"stock_change_percent",` + infy + `},"value":[1,"2.4"]},` +
			`{"metric":{"__name__":"stock:relative_strength_nifty:percent",` + infy + `},"value":[1,"1.1"]},` +
			`{"metric":{"__name__":"stock:rsi_wilder_14m:index",` + infy + `},"value":[1,"74"]},` +
			`{"metric":{"__name__":"stock:volume_delta_1h:ratio",` + infy + `},"value":[1,"-0.45"]},` +
			`{"metric":{"__name__":"stock:abnormal_return_15m:percent",` + infy + `},"value":[1,"NaN"]}]`,
		"stock_kpi_info": `[{"metric":{"kpi":"stock:volume_delta_1h:ratio","quality":"approximation","method":"tick-rule"},"value":[1,"1"]}]`,
		"orderflow)_regime": `[{"metric":{"__name__":"stock:momentum_regime:info","regime":"UPTREND"},"value":[1,"1"]},` +
			`{"metric":{"__name__":"stock:volatility_regime:info","regime":"HIGH"},"value":[1,"1"]},` +
			`{"metric":{"__name__":"stock:orderflow_regime:info","regime":"DISTRIBUTION"},"value":[1,"1"]}]`,
		"market:regime:info": vec("1", map[string]string{"regime": "TRENDING_BULL"}),
		"ALERTS{": `[{"metric":{"alertname":"StockMomentumExhaustion","direction":"bearish"},"value":[1,"1"]},` +
			`{"metric":{"alertname":"StockOrderBookWall","side":"bid","level":"3"},"value":[1,"1"]}]`,
		"alert:hit_rate_5d": vec("0.6", map[string]string{"alertname": "StockMomentumExhaustion", "direction": "bearish"}),
	})
	defer prom.Close()
	s := newTestServer(t, prom.URL, prom.URL)

	var env struct {
		Data  Analysis `json:"data"`
		Error string   `json:"error"`
	}
	if code := get(t, s, "/api/v1/analyst?symbol=infy", &env); code != http.StatusOK {
		t.Fatalf("status %d: %s", code, env.Error)
	}
	a := env.Data
	if len(a.Evidence) != 4 || a.Evidence[3].KPI != "stock:volume_delta_1h:ratio" || a.Evidence[3].Quality != "approximation" {
		t.Fatalf("evidence %+v (NaN must be dropped, catalogue quality attached)", a.Evidence)
	}
	want := []string{
		"INFY is up 2.40% today, 1.10 pp stronger than NIFTY 50.",
		"Momentum is stretched: Wilder RSI 74.",
		"Tick-rule volume delta over the last hour is -0.45 (approximation): seller-initiated flow dominates.",
		"Stock regimes: momentum uptrend, volatility high, order flow distribution.",
		"Market regime: trending bull.",
		"Active condition: StockMomentumExhaustion (bearish), 60% 15m hit rate over the last 5 days.",
		"Active condition: StockOrderBookWall, no effectiveness history yet.",
	}
	if strings.Join(a.Summary, "\n") != strings.Join(want, "\n") {
		t.Fatalf("summary:\n%s\nwant:\n%s", strings.Join(a.Summary, "\n"), strings.Join(want, "\n"))
	}
	if a.Disclaimer == "" {
		t.Fatal("missing disclaimer")
	}

	for _, bad := range []string{"", `INFY"}`, "INFY%7Bjob%3D%22x%22%7D", strings.Repeat("A", 40)} {
		if code := get(t, s, "/api/v1/analyst?symbol="+bad, nil); code != http.StatusBadRequest {
			t.Errorf("symbol %q: status %d, want 400", bad, code)
		}
	}
	empty := fakeProm(t, map[string]string{})
	defer empty.Close()
	if code := get(t, newTestServer(t, empty.URL, empty.URL), "/api/v1/analyst?symbol=TCS", nil); code != http.StatusNotFound {
		t.Fatalf("unknown symbol: status %d, want 404", code)
	}
}
