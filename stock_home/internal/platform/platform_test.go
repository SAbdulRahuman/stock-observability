package platform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func testServer(t *testing.T, cmds ...Command) *httptest.Server {
	t.Helper()
	reg := NewRegistry()
	for _, c := range cmds {
		reg.Register(c)
	}
	mux := http.NewServeMux()
	NewServer(reg, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second, prometheus.NewRegistry()).Register(mux)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ui") })
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, url string, v any) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

var echo = Command{
	Path:        "equity/price/historical",
	Description: "Historical prices.",
	Model:       "EquityHistorical",
	Providers:   []string{"alpha", "beta"},
	Params: []Param{
		{Name: "symbol", Type: String, Required: true, Multiple: true},
		{Name: "start_date", Type: Date, Default: "2024-01-02"},
		{Name: "limit", Type: Integer, Default: int64(10)},
		{Name: "interval", Type: String, Default: "1d", Choices: []string{"1d", "1W"}},
		{Name: "adjusted", Type: Boolean, Providers: []string{"beta"}},
	},
	Handler: func(_ context.Context, req *Request) (*Result, error) {
		return &Result{Results: []map[string]any{{
			"symbols":  req.Params.Strings("symbol"),
			"start":    req.Params.Date("start_date").Format(DateLayout),
			"limit":    req.Params.Int("limit"),
			"interval": req.Params.String("interval"),
			"adjusted": req.Params.Bool("adjusted"),
		}}}, nil
	},
}

func TestCommandSuccess(t *testing.T) {
	ts := testServer(t, echo)
	var out struct {
		OBBject
		Results []map[string]any `json:"results"`
	}
	code := get(t, ts.URL+"/api/v2/equity/price/historical?symbol=AAPL,MSFT&limit=5&interval=1W&adjusted=true&provider=beta", &out)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	r := out.Results[0]
	if got := r["symbols"].([]any); len(got) != 2 || got[1] != "MSFT" {
		t.Errorf("symbols %v", got)
	}
	if r["start"] != "2024-01-02" || r["limit"] != 5.0 || r["interval"] != "1W" || r["adjusted"] != true {
		t.Errorf("results %v", r)
	}
	if out.Provider == nil || *out.Provider != "beta" {
		t.Errorf("provider %v", out.Provider)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("warnings %v", out.Warnings)
	}
	md := out.Extra["metadata"].(map[string]any)
	if md["route"] != "/equity/price/historical" {
		t.Errorf("metadata %v", md)
	}
}

func TestCommandDefaultsAndWarnings(t *testing.T) {
	ts := testServer(t, echo)
	var out struct {
		OBBject
		Results []map[string]any `json:"results"`
	}
	if code := get(t, ts.URL+"/api/v2/equity/price/historical?symbol=AAPL&adjusted=true&bogus=1", &out); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if *out.Provider != "alpha" || out.Results[0]["limit"] != 10.0 || out.Results[0]["adjusted"] != false {
		t.Errorf("defaults not applied: %v %v", *out.Provider, out.Results[0])
	}
	if len(out.Warnings) != 2 || !strings.Contains(out.Warnings[0].Message, "adjusted") || !strings.Contains(out.Warnings[1].Message, "bogus") {
		t.Errorf("warnings %v", out.Warnings)
	}
}

func TestCommandErrors(t *testing.T) {
	failing := Command{Path: "fail/upstream", Handler: func(context.Context, *Request) (*Result, error) {
		return nil, Upstream("sec", errors.New("secret detail"))
	}}
	internal := Command{Path: "fail/internal", Handler: func(context.Context, *Request) (*Result, error) {
		return nil, errors.New("secret detail")
	}}
	creds := Command{Path: "fail/creds", Handler: func(context.Context, *Request) (*Result, error) {
		return nil, MissingCredential("fred", "api_key")
	}}
	slow := Command{Path: "fail/slow", Timeout: 10 * time.Millisecond, Handler: func(ctx context.Context, _ *Request) (*Result, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	ts := testServer(t, echo, failing, internal, creds, slow)
	for _, tc := range []struct {
		path string
		code int
		msg  string
	}{
		{"equity/price/historical", 400, `invalid parameter "symbol": required`},
		{"equity/price/historical?symbol=A&limit=x", 400, "not an integer"},
		{"equity/price/historical?symbol=A&start_date=2024-13-01", 400, "not a date"},
		{"equity/price/historical?symbol=A&interval=5m", 400, "must be one of 1d, 1W"},
		{"equity/price/historical?symbol=A&provider=gamma", 400, "must be one of alpha, beta"},
		{"equity/price/historical?symbol=,", 400, "empty list"},
		{"fail/upstream", 502, "upstream sec unavailable"},
		{"fail/internal", 500, "internal error"},
		{"fail/creds", 401, "missing credential: fred.api_key"},
		{"fail/creds?provider=x", 400, "command has no providers"},
		{"fail/slow", 504, "command timed out"},
		{"no/such/command", 404, "unknown command"},
	} {
		var out map[string]string
		code := get(t, ts.URL+Prefix+tc.path, &out)
		if code != tc.code || !strings.Contains(out["error"], tc.msg) || strings.Contains(out["error"], "secret") {
			t.Errorf("%s: got %d %q, want %d %q", tc.path, code, out["error"], tc.code, tc.msg)
		}
	}
}

func TestPostBody(t *testing.T) {
	post := Command{Path: "technical/sma", Method: "POST", Handler: func(_ context.Context, req *Request) (*Result, error) {
		var in struct{ Data []float64 }
		if err := json.Unmarshal(req.Body, &in); err != nil {
			return nil, InvalidParam("data", "bad json")
		}
		return &Result{Results: in.Data}, nil
	}}
	ts := testServer(t, post)
	resp, err := http.Post(ts.URL+"/api/v2/technical/sma", "application/json", strings.NewReader(`{"data":[1,2]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct{ Results []float64 }
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != 200 || len(out.Results) != 2 {
		t.Fatalf("status %d results %v", resp.StatusCode, out.Results)
	}
	resp2, err := http.Post(ts.URL+"/api/v2/technical/sma", "application/json", strings.NewReader(strings.Repeat(" ", maxBodyBytes+1)))
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body: status %d", resp2.StatusCode)
	}
}

func TestListCommands(t *testing.T) {
	ts := testServer(t, echo)
	var out []CommandInfo
	if code := get(t, ts.URL+"/api/v2/commands", &out); code != 200 {
		t.Fatalf("status %d", code)
	}
	if len(out) != 1 || out[0].Path != echo.Path || out[0].Method != "GET" || len(out[0].Params) != 5 || out[0].Providers[0] != "alpha" {
		t.Errorf("commands %+v", out)
	}
}

func TestFallback(t *testing.T) {
	reg := NewRegistry()
	reg.Register(echo)
	s := NewServer(reg, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second, prometheus.NewRegistry())
	s.SetFallback(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"fallback": r.URL.Path})
	}))
	mux := http.NewServeMux()
	s.Register(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	var native struct{ Results []map[string]any }
	if code := get(t, ts.URL+"/api/v2/equity/price/historical?symbol=A", &native); code != 200 || len(native.Results) != 1 {
		t.Errorf("native command: %d %v", code, native)
	}
	var cmds []CommandInfo
	if code := get(t, ts.URL+"/api/v2/commands", &cmds); code != 200 || len(cmds) != 1 {
		t.Errorf("commands: %d %v", code, cmds)
	}
	var out map[string]string
	if code := get(t, ts.URL+"/api/v2/equity/price/quote", &out); code != 200 || out["fallback"] != "/api/v2/equity/price/quote" {
		t.Errorf("fallback: %d %v", code, out)
	}
	resp, err := http.Post(ts.URL+"/api/v2/equity/price/quote", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST to unknown command: status %d", resp.StatusCode)
	}
}

func TestRegisterValidation(t *testing.T) {
	h := func(context.Context, *Request) (*Result, error) { return nil, nil }
	for name, c := range map[string]Command{
		"bad path":         {Path: "Equity/Quote", Handler: h},
		"nil handler":      {Path: "a"},
		"bad method":       {Path: "a", Method: "PUT", Handler: h},
		"reserved param":   {Path: "a", Handler: h, Params: []Param{{Name: "provider", Type: String}}},
		"dup param":        {Path: "a", Handler: h, Params: []Param{{Name: "x", Type: String}, {Name: "x", Type: String}}},
		"bad type":         {Path: "a", Handler: h, Params: []Param{{Name: "x", Type: "list"}}},
		"int default":      {Path: "a", Handler: h, Params: []Param{{Name: "x", Type: Integer, Default: 5}}},
		"bad date default": {Path: "a", Handler: h, Params: []Param{{Name: "x", Type: Date, Default: "yesterday"}}},
		"choice default":   {Path: "a", Handler: h, Params: []Param{{Name: "x", Type: String, Default: "z", Choices: []string{"y"}}}},
		"unknown provider": {Path: "a", Handler: h, Params: []Param{{Name: "x", Type: String, Providers: []string{"p"}}}},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("expected panic")
				}
			}()
			NewRegistry().Register(c)
		})
	}
	reg := NewRegistry()
	reg.Register(Command{Path: "a", Handler: h})
	defer func() {
		if recover() == nil {
			t.Error("expected duplicate panic")
		}
	}()
	reg.Register(Command{Path: "a", Handler: h})
}
