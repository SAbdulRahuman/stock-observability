package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seenimo/stock_home/internal/source"
)

func postTest(t *testing.T, s *Server, id string, header map[string]string) (int, IntegrationTest) {
	t.Helper()
	mux := http.NewServeMux()
	s.Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+id+"/test", nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var env struct{ Data IntegrationTest }
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec.Code, env.Data
}

func TestIntegrationsList(t *testing.T) {
	s := newTestServer(t, "http://prometheus:9090/prometheus", "http://alertmanager:9093")
	s.cfg.OpenBBURL = "http://openbb:6900"
	s.cfg.OpenBBUsername, s.cfg.OpenBBPassword = "openbb", "s3cret"
	s.cfg.GrafanaURL = "http://admin:pw@grafana:3000/grafana?token=abc"
	s.cfg.GrafanaToken = "glsa_secret"

	mux := http.NewServeMux()
	s.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil))
	body := rec.Body.String()
	for _, secret := range []string{"s3cret", "glsa_secret", "admin", "pw@", "token=abc"} {
		if strings.Contains(body, secret) {
			t.Errorf("integrations leaks %q: %s", secret, body)
		}
	}
	var env struct{ Data []Integration }
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data) != 5 || env.Data[0].ID != "kite" {
		t.Fatalf("integrations = %+v", env.Data)
	}
	byID := map[string]Integration{}
	for _, i := range env.Data {
		byID[i.ID] = i
	}
	if g := byID["grafana"]; g.Endpoint != "http://grafana:3000/grafana" || !strings.Contains(g.Auth, "token configured") {
		t.Errorf("grafana = %+v", g)
	}
	if o := byID["openbb"]; !o.Configured || !strings.Contains(o.Auth, "basic auth") {
		t.Errorf("openbb = %+v", o)
	}
}

func TestIntegrationTest(t *testing.T) {
	prom := fakeProm(t, map[string]string{
		"stock_kite_session_valid": vec("1", nil),
		"stock_feed_info":          vec("1", map[string]string{"source": "kite"}),
	})
	defer prom.Close()
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer down.Close()
	var gotAuth string
	openbb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer openbb.Close()
	grafana := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/search" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"database":"ok"}`))
	}))
	defer grafana.Close()

	s := newTestServer(t, prom.URL, down.URL)
	same := map[string]string{"Sec-Fetch-Site": "same-origin"}

	if code, _ := postTest(t, s, "prometheus", map[string]string{"Origin": "http://evil.example"}); code != http.StatusForbidden {
		t.Errorf("cross-origin code = %d", code)
	}
	if code, _ := postTest(t, s, "http%3A%2F%2Fevil", same); code != http.StatusNotFound {
		t.Errorf("unknown id code = %d", code)
	}
	if _, r := postTest(t, s, "prometheus", same); r.Status != "connected" || r.CheckedAt.IsZero() {
		t.Errorf("prometheus = %+v", r)
	}
	if _, r := postTest(t, s, "alertmanager", same); r.Status != "error" {
		t.Errorf("alertmanager = %+v", r)
	}
	if _, r := postTest(t, s, "kite", same); r.Status != "connected" {
		t.Errorf("kite = %+v", r)
	}
	if _, r := postTest(t, s, "openbb", same); r.Status != "not_configured" {
		t.Errorf("openbb unset = %+v", r)
	}

	s = newTestServer(t, prom.URL, down.URL)
	s.cfg.OpenBBURL, s.cfg.OpenBBUsername, s.cfg.OpenBBPassword = openbb.URL, "openbb", "s3cret"
	if _, r := postTest(t, s, "openbb", same); r.Status != "error" || !strings.Contains(r.Message, "credentials") || strings.Contains(r.Message, "s3cret") {
		t.Errorf("openbb = %+v", r)
	}
	if gotAuth == "" {
		t.Error("openbb test did not send the configured credentials")
	}

	s = newTestServer(t, prom.URL, down.URL)
	s.cfg.GrafanaURL = grafana.URL
	if _, r := postTest(t, s, "grafana", same); r.Status != "connected" {
		t.Errorf("grafana anonymous = %+v", r)
	}
	s = newTestServer(t, prom.URL, down.URL)
	s.cfg.GrafanaURL, s.cfg.GrafanaToken = grafana.URL, "glsa_bad"
	s.graf = source.NewGrafana(grafana.URL, http.DefaultClient, "glsa_bad")
	if _, r := postTest(t, s, "grafana", same); r.Status != "error" || !strings.Contains(r.Message, "token") {
		t.Errorf("grafana token = %+v", r)
	}
}
