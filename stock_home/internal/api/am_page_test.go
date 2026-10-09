package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seenimo/stock_home/internal/source"
)

const amConfig = `global:
  smtp_auth_password: hunter2
route:
  receiver: default
  group_by: [alertname, symbol]
  group_wait: 30s
  routes:
    - receiver: critical
      matchers: ['severity="critical"']
      continue: true
    - receiver: slack
      match: {team: market}
receivers:
  - name: default
  - name: critical
    webhook_configs:
      - url: https://hooks.example.com/services/T000/SECRET-TOKEN
  - name: slack
    slack_configs:
      - api_url: https://hooks.slack.com/services/XYZ
        channel: '#alerts'
`

type amPageFake struct {
	*httptest.Server
	mu      sync.Mutex
	created []source.NewSilence
	expired []string
}

func newAMPageFake(t *testing.T) *amPageFake {
	t.Helper()
	f := &amPageFake{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.URL.Path == "/-/ready":
			io.WriteString(w, "OK")
		case r.URL.Path == "/api/v2/alerts/groups":
			io.WriteString(w, `[{"labels":{"alertname":"PriceAboveR1"},"receiver":{"name":"critical"},"alerts":[
 {"fingerprint":"f1","labels":{"alertname":"PriceAboveR1","severity":"critical","symbol":"RELIANCE"},"annotations":{"summary":"R1"},
  "startsAt":"2026-10-08T04:11:00Z","endsAt":"2026-10-08T05:11:00Z","generatorURL":"javascript:alert(1)","status":{"state":"active","silencedBy":[],"inhibitedBy":[]}},
 {"fingerprint":"f2","labels":{"alertname":"PriceAboveR1","severity":"critical","symbol":"INFY"},"generatorURL":"http://prometheus:9090/graph","status":{"state":"suppressed","silencedBy":["s1"]}}]}]`)
		case r.URL.Path == "/api/v2/alerts":
			io.WriteString(w, `[{"fingerprint":"f1","labels":{"alertname":"PriceAboveR1","severity":"critical","symbol":"RELIANCE"},"status":{"state":"active"}},
 {"fingerprint":"f2","labels":{"alertname":"PriceAboveR1","severity":"critical","symbol":"INFY"},"status":{"state":"active"}},
 {"fingerprint":"f3","labels":{"alertname":"VolumeSpike","severity":"warning","symbol":"TCS"},"status":{"state":"active"}}]`)
		case r.URL.Path == "/api/v2/silences" && r.Method == http.MethodPost:
			var s source.NewSilence
			_ = json.NewDecoder(r.Body).Decode(&s)
			f.created = append(f.created, s)
			io.WriteString(w, `{"silenceID":"new"}`)
		case r.URL.Path == "/api/v2/silences":
			io.WriteString(w, `[{"id":"old","matchers":[{"name":"job","value":"x","isEqual":true}],"status":{"state":"expired"},"startsAt":"2026-10-01T00:00:00Z"},
 {"id":"s1","matchers":[{"name":"symbol","value":"INFY","isEqual":true}],"createdBy":"stock_home","comment":"c","status":{"state":"active"},"startsAt":"2026-10-08T00:00:00Z"}]`)
		case strings.HasPrefix(r.URL.Path, "/api/v2/silence/") && r.Method == http.MethodDelete:
			f.expired = append(f.expired, strings.TrimPrefix(r.URL.Path, "/api/v2/silence/"))
		case r.URL.Path == "/api/v2/status":
			b, _ := json.Marshal(map[string]any{
				"cluster": map[string]any{"status": "ready", "peers": []any{map[string]string{"name": "a"}}}, "versionInfo": map[string]string{"version": "0.28.1", "secretish": "x"},
				"uptime": "2026-10-01T00:00:00Z", "config": map[string]string{"original": amConfig},
			})
			w.Write(b)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func TestAMGroupsSilencesStatus(t *testing.T) {
	am := newAMPageFake(t)
	s := newTestServer(t, am.URL, am.URL)

	var groups struct {
		Data struct {
			Groups []AMGroup `json:"groups"`
		} `json:"data"`
	}
	if code := get(t, s, "/api/v1/am/groups", &groups); code != http.StatusOK {
		t.Fatalf("groups %d", code)
	}
	g := groups.Data.Groups
	if len(g) != 1 || g[0].Receiver != "critical" || len(g[0].Alerts) != 2 || g[0].Alerts[0].GeneratorURL != "" || g[0].Alerts[1].GeneratorURL == "" || len(g[0].Alerts[1].SilencedBy) != 1 {
		t.Errorf("groups = %+v", g)
	}

	var sil struct {
		Data struct {
			Silences []AMSilence `json:"silences"`
		} `json:"data"`
	}
	get(t, s, "/api/v1/am/silences", &sil)
	if len(sil.Data.Silences) != 2 || sil.Data.Silences[0].ID != "s1" || sil.Data.Silences[0].Matchers[0].Name != "symbol" {
		t.Errorf("silences = %+v", sil.Data.Silences)
	}

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	s.Register(mux)
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/am/status", nil))
	body := rec.Body.String()
	for _, secret := range []string{"hunter2", "SECRET-TOKEN", "hooks.slack.com", "#alerts", "secretish"} {
		if strings.Contains(body, secret) {
			t.Errorf("status leaks %q: %s", secret, body)
		}
	}
	var st struct {
		Data AMStatusView `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	v := st.Data
	if !v.Ready || v.Version["version"] != "0.28.1" || v.Peers != 1 || !v.ConfigAvailable || v.Route == nil || len(v.Route.Routes) != 2 {
		t.Fatalf("status = %+v", v)
	}
	if v.Route.Routes[0].Matchers[0] != `severity="critical"` || !v.Route.Routes[0].Continue || v.Route.Routes[1].Matchers[0] != `team="market"` || v.Route.GroupWait != "30s" {
		t.Errorf("routes = %+v", v.Route.Routes)
	}
	if len(v.Receivers) != 3 || v.Receivers[1].Integrations[0] != (IntegrationRef{Type: "webhook", Count: 1}) || len(v.Receivers[0].Integrations) != 0 {
		t.Errorf("receivers = %+v", v.Receivers)
	}
}

func postJSON(t *testing.T, s *Server, method, path, body string, hdr map[string]string) (int, envelope) {
	t.Helper()
	mux := http.NewServeMux()
	s.Register(mux)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var env envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec.Code, env
}

func TestAMCreateAndExpireSilence(t *testing.T) {
	am := newAMPageFake(t)
	s := newTestServer(t, am.URL, am.URL)
	now := time.Date(2026, 10, 8, 4, 20, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	ends := now.Add(2 * time.Hour).Format(time.RFC3339)
	form := func(matchers string, extra string) string {
		return `{"matchers":` + matchers + `,"endsAt":"` + ends + `","comment":"maintenance"` + extra + `}`
	}

	code, env := postJSON(t, s, http.MethodPost, "/api/v1/am/silences", form(`[{"name":"alertname","op":"=","value":"PriceAboveR1"}]`, `,"dryRun":true`), nil)
	if code != http.StatusOK {
		t.Fatalf("preview %d %q", code, env.Error)
	}
	var p SilencePreview
	b, _ := json.Marshal(env.Data)
	_ = json.Unmarshal(b, &p)
	if p.Matches != 2 || p.Broad || p.Sample[0] != "PriceAboveR1 · RELIANCE · critical" {
		t.Errorf("preview = %+v", p)
	}
	_, env = postJSON(t, s, http.MethodPost, "/api/v1/am/silences", form(`[{"name":"severity","op":"=~","value":"critical|warning"}]`, `,"dryRun":true`), nil)
	b, _ = json.Marshal(env.Data)
	_ = json.Unmarshal(b, &p)
	if p.Matches != 3 || !p.Broad {
		t.Errorf("broad preview = %+v", p)
	}

	if code, _ := postJSON(t, s, http.MethodPost, "/api/v1/am/silences", form(`[{"name":"alertname","op":"=","value":"PriceAboveR1"}]`, ""), nil); code != http.StatusForbidden {
		t.Errorf("create while disabled: %d", code)
	}
	s.cfg.SilencesAllowed = true
	for name, tc := range map[string]struct {
		body string
		hdr  map[string]string
	}{
		"cross-site":    {form(`[{"name":"alertname","op":"=","value":"X"}]`, ""), map[string]string{"Sec-Fetch-Site": "cross-site"}},
		"no matchers":   {form(`[]`, ""), nil},
		"bad name":      {form(`[{"name":"bad-name","op":"=","value":"X"}]`, ""), nil},
		"bad op":        {form(`[{"name":"a","op":"~","value":"X"}]`, ""), nil},
		"bad regex":     {form(`[{"name":"a","op":"=~","value":"("}]`, ""), nil},
		"matches empty": {form(`[{"name":"a","op":"=~","value":".*"},{"name":"b","op":"!=","value":"x"}]`, ""), nil},
		"no comment":    {`{"matchers":[{"name":"alertname","op":"=","value":"X"}],"endsAt":"` + ends + `","comment":" "}`, nil},
		"too long":      {`{"matchers":[{"name":"alertname","op":"=","value":"X"}],"endsAt":"` + now.Add(31*24*time.Hour).Format(time.RFC3339) + `","comment":"c"}`, nil},
		"ends first":    {`{"matchers":[{"name":"alertname","op":"=","value":"X"}],"endsAt":"` + now.Add(-time.Hour).Format(time.RFC3339) + `","comment":"c"}`, nil},
		"unknown field": {form(`[{"name":"alertname","op":"=","value":"X"}]`, `,"createdBy":"admin"`), nil},
	} {
		want := http.StatusBadRequest
		if name == "cross-site" {
			want = http.StatusForbidden
		}
		if code, _ := postJSON(t, s, http.MethodPost, "/api/v1/am/silences", tc.body, tc.hdr); code != want {
			t.Errorf("%s: status %d, want %d", name, code, want)
		}
	}
	if len(am.created) != 0 {
		t.Fatalf("rejected requests reached Alertmanager: %+v", am.created)
	}

	code, env = postJSON(t, s, http.MethodPost, "/api/v1/am/silences", form(`[{"name":"alertname","op":"=","value":"PriceAboveR1"},{"name":"symbol","op":"!~","value":"TCS|INFY"}]`, ""), nil)
	if code != http.StatusOK {
		t.Fatalf("create %d %q", code, env.Error)
	}
	if len(am.created) != 1 {
		t.Fatalf("created = %+v", am.created)
	}
	c := am.created[0]
	if c.CreatedBy != "stock_home" || c.Comment != "maintenance" || c.EndsAt.Sub(c.StartsAt) != 2*time.Hour {
		t.Errorf("silence = %+v", c)
	}
	if c.Matchers[1] != (source.Matcher{Name: "symbol", Value: "TCS|INFY", IsRegex: true, IsEqual: false}) {
		t.Errorf("matchers = %+v", c.Matchers)
	}

	id := "0f1e2d3c-4b5a-6978-8a9b-0c1d2e3f4a5b"
	if code, _ := postJSON(t, s, http.MethodDelete, "/api/v1/am/silences/not-an-id", "", nil); code != http.StatusBadRequest {
		t.Errorf("bad id: %d", code)
	}
	if code, _ := postJSON(t, s, http.MethodDelete, "/api/v1/am/silences/"+id, "", map[string]string{"Origin": "https://evil.example"}); code != http.StatusForbidden {
		t.Errorf("cross-origin expire: %d", code)
	}
	if code, _ := postJSON(t, s, http.MethodDelete, "/api/v1/am/silences/"+id, "", nil); code != http.StatusOK || len(am.expired) != 1 || am.expired[0] != id {
		t.Errorf("expire: %d %v", code, am.expired)
	}
}
