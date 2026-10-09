package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/seenimo/stock_home/internal/source"
)

// Read-only Prometheus explorer for the Prometheus page. Only the fixed upstream paths below are reachable;
// user input is limited to PromQL text and time parameters, which are bounded and forwarded as query values.

const (
	maxPromQL        = 4000
	maxPromBody      = 16 << 20
	maxRangePoints   = 11000
	maxRange         = 31 * 24 * time.Hour
	promConcurrency  = 4
	promMetricNames  = "/api/v1/label/__name__/values"
	promQueryPath    = "/api/v1/query"
	promQueryRange   = "/api/v1/query_range"
	promTargetsPath  = "/api/v1/targets"
	promRulesPath    = "/api/v1/rules"
	promAlertsPath   = "/api/v1/alerts"
	promBuildPath    = "/api/v1/status/buildinfo"
	promRuntimePath  = "/api/v1/status/runtimeinfo"
	promReadyPath    = "/-/ready"
	promHealthyPath  = "/-/healthy"
	promDefaultLimit = 10000
)

// promError is an upstream API error that is safe to show (PromQL parse/execution errors).
type promError struct {
	status int
	msg    string
}

func (e *promError) Error() string { return e.msg }

// promGet calls one allowlisted Prometheus API path and returns its "data" and "warnings".
func (s *Server) promGet(ctx context.Context, path string, params url.Values) (json.RawMessage, []string, error) {
	u := s.cfg.PrometheusURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPromBody+1))
	if err != nil {
		return nil, nil, err
	}
	if len(body) > maxPromBody {
		return nil, nil, &promError{http.StatusRequestEntityTooLarge, "result too large; narrow the query or range"}
	}
	var env struct {
		Status    string          `json:"status"`
		Data      json.RawMessage `json:"data"`
		ErrorType string          `json:"errorType"`
		Error     string          `json:"error"`
		Warnings  []string        `json:"warnings"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, nil, fmt.Errorf("decode %s: status %d", path, resp.StatusCode)
	}
	if env.Status != "success" {
		// bad_data and execution errors describe the user's query; other failures stay generic.
		if env.ErrorType == "bad_data" || env.ErrorType == "execution" || env.ErrorType == "timeout" || env.ErrorType == "canceled" {
			return nil, nil, &promError{http.StatusBadRequest, env.ErrorType + ": " + env.Error}
		}
		return nil, nil, fmt.Errorf("%s: %s %s", path, env.ErrorType, env.Error)
	}
	return env.Data, env.Warnings, nil
}

type promResult struct {
	Data       json.RawMessage `json:"data"`
	Warnings   []string        `json:"warnings,omitempty"`
	DurationMs float64         `json:"durationMs"`
}

func (s *Server) promReply(w http.ResponseWriter, r *http.Request, section, path string, params url.Values) {
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.QueryTimeout)
	defer cancel()
	start := time.Now()
	data, warnings, err := s.promGet(ctx, path, params)
	if err != nil {
		var pe *promError
		if errors.As(err, &pe) {
			writeJSON(w, pe.status, envelope{Error: pe.msg, UpdatedAt: s.now()})
			return
		}
		s.errors.WithLabelValues(section).Inc()
		s.logger.Warn("Prometheus API call failed", "path", path, "err", err)
		writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: promResult{Data: data, Warnings: warnings, DurationMs: float64(time.Since(start).Microseconds()) / 1000}, UpdatedAt: s.now()})
}

func (s *Server) registerProm(mux *http.ServeMux) {
	slots := make(chan struct{}, promConcurrency)
	limited := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
				h(w, r)
			default:
				writeJSON(w, http.StatusTooManyRequests, envelope{Error: "too many concurrent queries; retry shortly", UpdatedAt: s.now()})
			}
		}
	}
	fixed := func(section, path string, params url.Values) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { s.promReply(w, r, section, path, params) }
	}
	mux.HandleFunc("GET /api/v1/prom/query", limited(s.promQuery))
	mux.HandleFunc("GET /api/v1/prom/query_range", limited(s.promQueryRange))
	mux.HandleFunc("GET /api/v1/prom/metric-names", limited(fixed("prom-names", promMetricNames, url.Values{"limit": {strconv.Itoa(promDefaultLimit)}})))
	mux.HandleFunc("GET /api/v1/prom/targets", s.promTargets)
	mux.HandleFunc("GET /api/v1/prom/rules", fixed("prom-rules", promRulesPath, nil))
	mux.HandleFunc("GET /api/v1/prom/alerts", fixed("prom-alerts", promAlertsPath, nil))
	mux.HandleFunc("GET /api/v1/prom/status", s.promStatus)
}

// PromTarget is an active scrape target without its discovered (pre-relabelling) labels, which can carry
// service-discovery metadata the page does not need.
type PromTarget struct {
	Labels             map[string]string `json:"labels"`
	ScrapePool         string            `json:"scrapePool"`
	ScrapeURL          string            `json:"scrapeUrl"`
	Health             string            `json:"health"`
	LastError          string            `json:"lastError"`
	LastScrape         time.Time         `json:"lastScrape"`
	LastScrapeDuration float64           `json:"lastScrapeDuration"`
	ScrapeInterval     string            `json:"scrapeInterval"`
	ScrapeTimeout      string            `json:"scrapeTimeout"`
}

func (s *Server) promTargets(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.QueryTimeout)
	defer cancel()
	start := time.Now()
	raw, _, err := s.promGet(ctx, promTargetsPath, url.Values{"state": {"active"}})
	var data struct {
		ActiveTargets []PromTarget `json:"activeTargets"`
	}
	if err == nil {
		err = json.Unmarshal(raw, &data)
	}
	if err != nil {
		s.errors.WithLabelValues("prom-targets").Inc()
		s.logger.Warn("Prometheus targets failed", "err", err)
		writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
		return
	}
	if data.ActiveTargets == nil {
		data.ActiveTargets = []PromTarget{}
	}
	b, _ := json.Marshal(data)
	writeJSON(w, http.StatusOK, envelope{Data: promResult{Data: b, DurationMs: float64(time.Since(start).Microseconds()) / 1000}, UpdatedAt: s.now()})
}

func promQueryText(r *http.Request) (string, error) {
	q := strings.TrimSpace(r.URL.Query().Get("query"))
	switch {
	case q == "":
		return "", errors.New("query is required")
	case len(q) > maxPromQL || !utf8.ValidString(q):
		return "", fmt.Errorf("query must be valid UTF-8 and at most %d bytes", maxPromQL)
	}
	return q, nil
}

func parsePromTime(v string) (time.Time, error) {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
		return time.Time{}, errors.New("times must be Unix seconds")
	}
	sec, frac := math.Modf(f)
	return time.Unix(int64(sec), int64(frac*1e9)), nil
}

func (s *Server) promQuery(w http.ResponseWriter, r *http.Request) {
	q, err := promQueryText(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{Error: err.Error(), UpdatedAt: s.now()})
		return
	}
	params := url.Values{"query": {q}, "timeout": {s.cfg.QueryTimeout.String()}, "limit": {strconv.Itoa(promDefaultLimit)}}
	if t := r.URL.Query().Get("time"); t != "" {
		ts, err := parsePromTime(t)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, envelope{Error: err.Error(), UpdatedAt: s.now()})
			return
		}
		params.Set("time", strconv.FormatFloat(float64(ts.UnixMilli())/1000, 'f', 3, 64))
	}
	s.promReply(w, r, "prom-query", promQueryPath, params)
}

func (s *Server) promQueryRange(w http.ResponseWriter, r *http.Request) {
	bad := func(msg string) { writeJSON(w, http.StatusBadRequest, envelope{Error: msg, UpdatedAt: s.now()}) }
	q, err := promQueryText(r)
	if err != nil {
		bad(err.Error())
		return
	}
	v := r.URL.Query()
	start, err1 := parsePromTime(v.Get("start"))
	end, err2 := parsePromTime(v.Get("end"))
	step, err3 := strconv.ParseFloat(v.Get("step"), 64)
	switch {
	case err1 != nil || err2 != nil:
		bad("start and end must be Unix seconds")
		return
	case err3 != nil || step < 1 || math.IsInf(step, 0):
		bad("step must be at least 1 second")
		return
	case !end.After(start) || end.Sub(start) > maxRange:
		bad("range must be positive and at most 31 days")
		return
	case end.Sub(start).Seconds()/step > maxRangePoints:
		bad(fmt.Sprintf("range/step yields more than %d points; increase the step", maxRangePoints))
		return
	}
	params := url.Values{
		"query": {q}, "timeout": {s.cfg.QueryTimeout.String()},
		"start": {strconv.FormatInt(start.Unix(), 10)}, "end": {strconv.FormatInt(end.Unix(), 10)},
		"step": {strconv.FormatFloat(step, 'f', -1, 64)},
	}
	s.promReply(w, r, "prom-query-range", promQueryRange, params)
}

// PromStatus is the subset of Prometheus server status safe to show; configuration and flags are not exposed.
type PromStatus struct {
	Ready   bool              `json:"ready"`
	Healthy bool              `json:"healthy"`
	Build   map[string]string `json:"build"`
	Runtime map[string]any    `json:"runtime"`
}

var runtimeFields = []string{"startTime", "reloadConfigSuccess", "lastConfigTime", "goroutineCount", "storageRetention", "corruptionCount"}

func (s *Server) promStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.QueryTimeout)
	defer cancel()
	st := PromStatus{Build: map[string]string{}, Runtime: map[string]any{}}
	st.Ready, _ = source.Probe(ctx, s.http, s.cfg.PrometheusURL+promReadyPath)
	st.Healthy, _ = source.Probe(ctx, s.http, s.cfg.PrometheusURL+promHealthyPath)
	if raw, _, err := s.promGet(ctx, promBuildPath, nil); err == nil {
		var b map[string]string
		if json.Unmarshal(raw, &b) == nil {
			for _, k := range []string{"version", "revision", "branch", "buildDate", "goVersion"} {
				if v, ok := b[k]; ok {
					st.Build[k] = v
				}
			}
		}
	}
	if raw, _, err := s.promGet(ctx, promRuntimePath, nil); err == nil {
		var rt map[string]any
		if json.Unmarshal(raw, &rt) == nil {
			for _, k := range runtimeFields {
				if v, ok := rt[k]; ok {
					st.Runtime[k] = v
				}
			}
		}
	}
	if !st.Ready && !st.Healthy && len(st.Build) == 0 {
		writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: st, UpdatedAt: s.now()})
}
