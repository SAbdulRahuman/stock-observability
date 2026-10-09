package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type promFake struct {
	*httptest.Server
	mu    sync.Mutex
	paths []string
	forms []url.Values
	block chan struct{}
}

func newPromFake(t *testing.T) *promFake {
	t.Helper()
	f := &promFake{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		f.paths, f.forms = append(f.paths, r.URL.Path), append(f.forms, r.Form)
		block := f.block
		f.mu.Unlock()
		if block != nil && strings.HasSuffix(r.URL.Path, "/query") {
			<-block
		}
		switch r.URL.Path {
		case "/-/ready", "/-/healthy":
			io.WriteString(w, "ok")
		case "/api/v1/query":
			if strings.Contains(r.Form.Get("query"), "((") {
				w.WriteHeader(http.StatusBadRequest)
				io.WriteString(w, `{"status":"error","errorType":"bad_data","error":"parse error: unexpected left parenthesis"}`)
				return
			}
			io.WriteString(w, `{"status":"success","data":{"resultType":"vector","result":[]},"warnings":["w1"]}`)
		case "/api/v1/query_range":
			io.WriteString(w, `{"status":"success","data":{"resultType":"matrix","result":[]}}`)
		case "/api/v1/targets":
			io.WriteString(w, `{"status":"success","data":{"activeTargets":[{"health":"up","discoveredLabels":{"__meta_kubernetes_secret":"x"},"labels":{"job":"stock_core"}}],"droppedTargets":[]}}`)
		case "/api/v1/rules":
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `{"status":"error","errorType":"internal","error":"secret internal path /etc/x"}`)
		case "/api/v1/status/buildinfo":
			io.WriteString(w, `{"status":"success","data":{"version":"3.5.0","revision":"abc","extra":"x"}}`)
		case "/api/v1/status/runtimeinfo":
			io.WriteString(w, `{"status":"success","data":{"startTime":"2026-10-01T00:00:00Z","CWD":"/prometheus","storageRetention":"15d"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *promFake) last() (string, url.Values) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.paths[len(f.paths)-1], f.forms[len(f.forms)-1]
}

func TestPromProxy(t *testing.T) {
	prom := newPromFake(t)
	s := newTestServer(t, prom.URL, prom.URL)

	var env struct {
		Data  promResult `json:"data"`
		Error string     `json:"error"`
	}
	if code := get(t, s, "/api/v1/prom/query?query="+url.QueryEscape(`up{job="stock_core"}`)+"&time=1759895100.5", &env); code != http.StatusOK {
		t.Fatalf("query status %d: %s", code, env.Error)
	}
	path, form := prom.last()
	if path != "/api/v1/query" || form.Get("query") != `up{job="stock_core"}` || form.Get("time") != "1759895100.500" || form.Get("timeout") == "" {
		t.Errorf("forwarded %s %v", path, form)
	}
	if len(env.Data.Warnings) != 1 || !strings.Contains(string(env.Data.Data), "vector") {
		t.Errorf("result %+v", env.Data)
	}

	if code := get(t, s, "/api/v1/prom/query?query="+url.QueryEscape("sum((up)"), &env); code != http.StatusBadRequest || !strings.Contains(env.Error, "parse error") {
		t.Errorf("bad PromQL: %d %q", code, env.Error)
	}
	for _, bad := range []string{
		"/api/v1/prom/query",
		"/api/v1/prom/query?query=" + strings.Repeat("a", maxPromQL+1),
		"/api/v1/prom/query?query=up&time=yesterday",
		"/api/v1/prom/query_range?query=up&start=0&end=100",
		"/api/v1/prom/query_range?query=up&start=100&end=100&step=1",
		"/api/v1/prom/query_range?query=up&start=0&end=86400&step=1",
		"/api/v1/prom/query_range?query=up&start=0&end=" + "99999999" + "&step=3600",
	} {
		if code := get(t, s, bad, nil); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", bad, code)
		}
	}
	if code := get(t, s, "/api/v1/prom/query_range?query=up&start=1000&end=4600&step=14", nil); code != http.StatusOK {
		t.Errorf("range status %d", code)
	}
	if path, form := prom.last(); path != "/api/v1/query_range" || form.Get("step") != "14" || form.Get("start") != "1000" {
		t.Errorf("forwarded %s %v", path, form)
	}

	if code := get(t, s, "/api/v1/prom/targets", &env); code != http.StatusOK {
		t.Errorf("targets status %d", code)
	}
	if strings.Contains(string(env.Data.Data), "__meta") || !strings.Contains(string(env.Data.Data), `"job":"stock_core"`) {
		t.Errorf("targets must drop discovered labels: %s", env.Data.Data)
	}
	if path, form := prom.last(); path != "/api/v1/targets" || form.Get("state") != "active" {
		t.Errorf("forwarded %s %v", path, form)
	}
	if code := get(t, s, "/api/v1/prom/rules", &env); code != http.StatusBadGateway || strings.Contains(env.Error, "secret") {
		t.Errorf("internal errors must stay generic: %d %q", code, env.Error)
	}

	var st struct {
		Data PromStatus `json:"data"`
	}
	if code := get(t, s, "/api/v1/prom/status", &st); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if !st.Data.Ready || !st.Data.Healthy || st.Data.Build["version"] != "3.5.0" || st.Data.Build["extra"] != "" || st.Data.Runtime["CWD"] != nil || st.Data.Runtime["storageRetention"] != "15d" {
		t.Errorf("status = %+v", st.Data)
	}
}

func TestPromQueryConcurrencyLimit(t *testing.T) {
	prom := newPromFake(t)
	prom.block = make(chan struct{})
	s := newTestServer(t, prom.URL, prom.URL)
	mux := http.NewServeMux()
	s.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var wg sync.WaitGroup
	for range promConcurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if resp, err := http.Get(srv.URL + "/api/v1/prom/query?query=up"); err == nil {
				resp.Body.Close()
			}
		}()
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		prom.mu.Lock()
		n := len(prom.paths)
		prom.mu.Unlock()
		if n >= promConcurrency || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	resp, err := http.Get(srv.URL + "/api/v1/prom/query?query=up")
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	_ = json.NewDecoder(resp.Body).Decode(&env)
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status %d, want 429", resp.StatusCode)
	}
	close(prom.block)
	wg.Wait()
}
