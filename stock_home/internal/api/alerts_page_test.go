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

const amAlerts = `[
 {"fingerprint":"f1","labels":{"alertname":"PriceAboveR1","severity":"critical","symbol":"RELIANCE","category":"price"},
  "annotations":{"summary":"Price > Resistance (R1)","value":"2,842.50","threshold":"2840"},"startsAt":"2026-10-08T04:11:00Z",
  "generatorURL":"http://prometheus:9090/prometheus/graph?g0.expr=x","status":{"state":"active"}},
 {"fingerprint":"f2","labels":{"alertname":"VolumeSpike","severity":"warning","symbol":"INFY"},"startsAt":"2026-10-08T04:08:00Z",
  "generatorURL":"http://grafana:3000/grafana/alerting/x","status":{"state":"suppressed","silencedBy":["ack1"]}},
 {"fingerprint":"f3","labels":{"alertname":"Rsi","severity":"info"},"startsAt":"2026-10-08T04:05:00Z","status":{"state":"suppressed","silencedBy":["s2"]}},
 {"fingerprint":"f4","labels":{"alertname":"Inh","severity":"info"},"startsAt":"2026-10-08T04:00:00Z","status":{"state":"suppressed","inhibitedBy":["x"]}}
]`

const amSilences = `[
 {"id":"ack1","comment":"Acknowledged via stock_home: looking","endsAt":"2026-10-08T05:00:00Z","status":{"state":"active"}},
 {"id":"s2","comment":"maintenance","endsAt":"2026-10-08T06:00:00Z","status":{"state":"active"}}
]`

type fakeAM struct {
	*httptest.Server
	mu    sync.Mutex
	posts []source.NewSilence
}

func newFakeAM(t *testing.T) *fakeAM {
	t.Helper()
	f := &fakeAM{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/silences"):
			var s source.NewSilence
			_ = json.NewDecoder(r.Body).Decode(&s)
			f.mu.Lock()
			f.posts = append(f.posts, s)
			f.mu.Unlock()
			io.WriteString(w, `{"silenceID":"new1"}`)
		case strings.HasSuffix(r.URL.Path, "/silences"):
			io.WriteString(w, amSilences)
		default:
			io.WriteString(w, amAlerts)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func TestAlertItems(t *testing.T) {
	am := newFakeAM(t)
	end := time.Date(2026, 10, 8, 4, 20, 0, 0, time.UTC)
	prom, _ := fakePromSeq(t, nil, []answer{
		{`ALERTS{alertstate="firing"}`, matrix(`{"alertname":"HighVolume","severity":"info","symbol":"ITC"}`,
			source.Point{T: end.Add(-90 * time.Minute).UnixMilli()}, source.Point{T: end.Add(-89 * time.Minute).UnixMilli()})},
	})
	s := newTestServer(t, prom.URL, am.URL)
	s.now = func() time.Time { return end }

	var env struct {
		Data AlertList `json:"data"`
	}
	if code := get(t, s, "/api/v1/alerts/items", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	got := map[string]AlertRow{}
	for _, r := range env.Data.Items {
		got[r.Key] = r
	}
	if r := got["f1"]; r.Status != "firing" || r.Source != "Prometheus" || r.Threshold != "2840" || r.Value != "2,842.50" || !strings.HasPrefix(r.ExploreURL, "/grafana/explore") {
		t.Errorf("f1 = %+v", r)
	}
	if r := got["f2"]; r.Status != "acknowledged" || r.Source != "Grafana" || r.SilencedUntil == nil {
		t.Errorf("f2 = %+v", r)
	}
	if got["f3"].Status != "silenced" || got["f3"].Source != "Alertmanager" || got["f4"].Status != "inhibited" {
		t.Errorf("f3/f4 = %+v %+v", got["f3"], got["f4"])
	}
	if len(env.Data.Items) != 5 || env.Data.Items[0].Key != "f1" || env.Data.Items[4].Status != "resolved" || env.Data.Items[4].Symbol != "ITC" {
		t.Errorf("items = %+v", env.Data.Items)
	}
	if len(env.Data.Timeline) != 1 {
		t.Errorf("timeline = %+v", env.Data.Timeline)
	}
}

func post(t *testing.T, s *Server, body string, hdr map[string]string) (int, envelope) {
	t.Helper()
	mux := http.NewServeMux()
	s.Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/silences", strings.NewReader(body))
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

func TestCreateSilences(t *testing.T) {
	am := newFakeAM(t)
	s := newTestServer(t, am.URL, am.URL)
	now := time.Date(2026, 10, 8, 4, 20, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	ok := `{"fingerprints":["f1","f2","nope","f1"],"kind":"acknowledge","duration":"1h","comment":"on it"}`

	if code, _ := post(t, s, ok, nil); code != http.StatusForbidden {
		t.Fatalf("disabled: status %d", code)
	}
	s.cfg.SilencesAllowed = true
	for name, tc := range map[string]struct {
		body string
		hdr  map[string]string
		code int
	}{
		"cross-site":    {ok, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		"other origin":  {ok, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		"content type":  {ok, map[string]string{"Content-Type": "text/plain"}, http.StatusUnsupportedMediaType},
		"duration":      {`{"fingerprints":["f1"],"kind":"silence","duration":"7d"}`, nil, http.StatusBadRequest},
		"kind":          {`{"fingerprints":["f1"],"kind":"resolve","duration":"1h"}`, nil, http.StatusBadRequest},
		"unknown field": {`{"fingerprints":["f1"],"kind":"silence","duration":"1h","matchers":[]}`, nil, http.StatusBadRequest},
		"control chars": {`{"fingerprints":["f1"],"kind":"silence","duration":"1h","comment":"a\u0007"}`, nil, http.StatusBadRequest},
		"empty":         {`{"fingerprints":[],"kind":"silence","duration":"1h"}`, nil, http.StatusBadRequest},
	} {
		if code, _ := post(t, s, tc.body, tc.hdr); code != tc.code {
			t.Errorf("%s: status %d, want %d", name, code, tc.code)
		}
	}
	if len(am.posts) != 0 {
		t.Fatalf("rejected requests reached Alertmanager: %+v", am.posts)
	}

	code, env := post(t, s, ok, map[string]string{"Origin": "http://example.com", "Sec-Fetch-Site": "same-origin"})
	if code != http.StatusOK {
		t.Fatalf("status %d: %+v", code, env)
	}
	b, _ := json.Marshal(env.Data)
	var res SilenceResult
	_ = json.Unmarshal(b, &res)
	if len(res.Created) != 1 || res.Created[0].Fingerprint != "f1" || len(res.Failed) != 2 {
		t.Fatalf("result = %+v", res)
	}
	if len(am.posts) != 1 {
		t.Fatalf("posts = %+v", am.posts)
	}
	p := am.posts[0]
	if p.Comment != "Acknowledged via stock_home: on it" || p.CreatedBy != "stock_home" || p.EndsAt.Sub(p.StartsAt) != time.Hour {
		t.Errorf("silence = %+v", p)
	}
	want := []source.Matcher{{Name: "alertname", Value: "PriceAboveR1", IsEqual: true}, {Name: "category", Value: "price", IsEqual: true},
		{Name: "severity", Value: "critical", IsEqual: true}, {Name: "symbol", Value: "RELIANCE", IsEqual: true}}
	if len(p.Matchers) != len(want) {
		t.Fatalf("matchers = %+v", p.Matchers)
	}
	for i := range want {
		if p.Matchers[i] != want[i] {
			t.Errorf("matcher %d = %+v, want %+v", i, p.Matchers[i], want[i])
		}
	}

	if code, _ := post(t, s, `{"fingerprints":["f2"],"kind":"silence","duration":"30m"}`, nil); code != http.StatusConflict {
		t.Errorf("already silenced: status %d, want 409", code)
	}
}
