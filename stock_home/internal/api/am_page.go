package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"go.yaml.in/yaml/v3"

	"github.com/seenimo/stock_home/internal/source"
)

// AlertManager page: alert groups, silences (create/expire behind --alertmanager.allow-silences) and a
// read-only, redacted view of routes and receivers.

// AMAlert is one alert instance inside a group.
type AMAlert struct {
	Fingerprint  string            `json:"fingerprint"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	State        string            `json:"state"`
	SilencedBy   []string          `json:"silencedBy"`
	InhibitedBy  []string          `json:"inhibitedBy"`
	GeneratorURL string            `json:"generatorUrl,omitempty"`
}

// AMGroup is one Alertmanager alert group; a group holds one or more alert instances.
type AMGroup struct {
	Labels   map[string]string `json:"labels"`
	Receiver string            `json:"receiver"`
	Alerts   []AMAlert         `json:"alerts"`
}

func (s *Server) amGroups(ctx context.Context) (any, error) {
	groups, err := s.am.Groups(ctx)
	if err != nil {
		return nil, err
	}
	out := struct {
		Groups []AMGroup `json:"groups"`
	}{Groups: []AMGroup{}}
	for _, g := range groups {
		ag := AMGroup{Labels: nonNil(g.Labels), Receiver: g.Receiver.Name, Alerts: []AMAlert{}}
		for _, a := range g.Alerts {
			al := AMAlert{
				Fingerprint: a.Fingerprint, Labels: nonNil(a.Labels), Annotations: nonNil(a.Annotations), StartsAt: a.StartsAt, EndsAt: a.EndsAt,
				State: a.Status.State, SilencedBy: nonNilSlice(a.Status.SilencedBy), InhibitedBy: nonNilSlice(a.Status.InhibitedBy),
			}
			if u, err := url.Parse(a.GeneratorURL); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
				al.GeneratorURL = a.GeneratorURL
			}
			ag.Alerts = append(ag.Alerts, al)
		}
		out.Groups = append(out.Groups, ag)
	}
	return out, nil
}

// AMSilence is a silence as listed on the AlertManager page.
type AMSilence struct {
	ID        string           `json:"id"`
	Matchers  []source.Matcher `json:"matchers"`
	CreatedBy string           `json:"createdBy"`
	Comment   string           `json:"comment"`
	StartsAt  time.Time        `json:"startsAt"`
	EndsAt    time.Time        `json:"endsAt"`
	State     string           `json:"state"`
}

var silenceStateOrder = map[string]int{"active": 0, "pending": 1, "expired": 2}

func (s *Server) amSilences(ctx context.Context) (any, error) {
	list, err := s.am.Silences(ctx)
	if err != nil {
		return nil, err
	}
	out := struct {
		Silences []AMSilence `json:"silences"`
	}{Silences: []AMSilence{}}
	for _, sl := range list {
		m := sl.Matchers
		if m == nil {
			m = []source.Matcher{}
		}
		out.Silences = append(out.Silences, AMSilence{ID: sl.ID, Matchers: m, CreatedBy: sl.CreatedBy, Comment: sl.Comment, StartsAt: sl.StartsAt, EndsAt: sl.EndsAt, State: sl.Status.State})
	}
	sort.SliceStable(out.Silences, func(i, j int) bool {
		a, b := out.Silences[i], out.Silences[j]
		if silenceStateOrder[a.State] != silenceStateOrder[b.State] {
			return silenceStateOrder[a.State] < silenceStateOrder[b.State]
		}
		return a.StartsAt.After(b.StartsAt)
	})
	return out, nil
}

// MatcherInput is one matcher as entered on the Create Silence form.
type MatcherInput struct {
	Name  string `json:"name"`
	Op    string `json:"op"` // =, !=, =~ or !~.
	Value string `json:"value"`
}

// SilenceForm creates (or, with DryRun, previews) a silence from user-defined matchers.
type SilenceForm struct {
	Matchers []MatcherInput `json:"matchers"`
	StartsAt string         `json:"startsAt"` // RFC 3339; empty means now.
	EndsAt   string         `json:"endsAt"`
	Comment  string         `json:"comment"`
	DryRun   bool           `json:"dryRun"`
}

// SilencePreview reports which active alerts the matchers select.
type SilencePreview struct {
	SilenceID string   `json:"silenceId,omitempty"`
	Matches   int      `json:"matches"`
	Sample    []string `json:"sample"`
	Broad     bool     `json:"broad"`
	StartsAt  string   `json:"startsAt"`
	EndsAt    string   `json:"endsAt"`
}

var (
	labelName      = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,63}$`)
	silenceIDRe    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	maxSilenceSpan = 30 * 24 * time.Hour
)

const (
	maxMatchers      = 20
	maxMatcherValue  = 200
	broadMatchCount  = 10
	previewSampleLen = 20
)

type compiledMatcher struct {
	in source.Matcher
	re *regexp.Regexp
}

func (m compiledMatcher) matches(labels map[string]string) bool {
	v := labels[m.in.Name]
	ok := v == m.in.Value
	if m.re != nil {
		ok = m.re.MatchString(v)
	}
	return ok == m.in.IsEqual
}

func compileMatchers(in []MatcherInput) ([]compiledMatcher, error) {
	if len(in) == 0 || len(in) > maxMatchers {
		return nil, fmt.Errorf("1 to %d matchers required", maxMatchers)
	}
	var out []compiledMatcher
	selective := false
	for _, m := range in {
		if !labelName.MatchString(m.Name) {
			return nil, fmt.Errorf("invalid label name %q", m.Name)
		}
		if len(m.Value) > maxMatcherValue || strings.IndexFunc(m.Value, unicode.IsControl) >= 0 {
			return nil, fmt.Errorf("value for %s must be at most %d printable characters", m.Name, maxMatcherValue)
		}
		cm := compiledMatcher{in: source.Matcher{Name: m.Name, Value: m.Value}}
		switch m.Op {
		case "=":
			cm.in.IsEqual = true
		case "!=":
		case "=~", "!~":
			re, err := regexp.Compile("^(?:" + m.Value + ")$")
			if err != nil {
				return nil, fmt.Errorf("invalid regular expression for %s", m.Name)
			}
			cm.re, cm.in.IsRegex, cm.in.IsEqual = re, true, m.Op == "=~"
		default:
			return nil, fmt.Errorf("operator must be one of =, !=, =~, !~")
		}
		// Alertmanager rejects silences whose matchers all match the empty label value.
		if !cm.matches(map[string]string{}) {
			selective = true
		}
		out = append(out, cm)
	}
	if !selective {
		return nil, fmt.Errorf("at least one matcher must not match an empty label value")
	}
	return out, nil
}

func (s *Server) amCreateSilence(w http.ResponseWriter, r *http.Request) {
	fail := func(code int, msg string) { writeJSON(w, code, envelope{Error: msg, UpdatedAt: s.now()}) }
	if !sameOrigin(r) {
		fail(http.StatusForbidden, "cross-origin request rejected")
		return
	}
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
		fail(http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}
	var f SilenceForm
	dec := json.NewDecoder(io.LimitReader(r.Body, maxSilenceBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		fail(http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !f.DryRun && !s.cfg.SilencesAllowed {
		fail(http.StatusForbidden, "silences are disabled on this stock_home")
		return
	}
	matchers, err := compileMatchers(f.Matchers)
	if err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	now := s.now()
	start := now
	if f.StartsAt != "" {
		if start, err = time.Parse(time.RFC3339, f.StartsAt); err != nil || start.After(now.Add(maxSilenceSpan)) {
			fail(http.StatusBadRequest, "startsAt must be RFC 3339 and within 30 days")
			return
		}
		if start.Before(now) {
			start = now
		}
	}
	end, err := time.Parse(time.RFC3339, f.EndsAt)
	if err != nil || !end.After(start) || end.Sub(start) > maxSilenceSpan {
		fail(http.StatusBadRequest, "endsAt must be RFC 3339, after the start and at most 30 days later")
		return
	}
	comment := strings.TrimSpace(f.Comment)
	if comment == "" || len([]rune(comment)) > maxSilenceComment || strings.IndexFunc(comment, unicode.IsControl) >= 0 {
		fail(http.StatusBadRequest, fmt.Sprintf("a comment of 1 to %d printable characters is required", maxSilenceComment))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.QueryTimeout)
	defer cancel()
	alerts, err := s.am.Alerts(ctx)
	if err != nil {
		s.logger.Warn("Silence preview: Alertmanager unavailable", "err", err)
		fail(http.StatusBadGateway, "upstream unavailable")
		return
	}
	p := SilencePreview{Sample: []string{}, StartsAt: start.UTC().Format(time.RFC3339), EndsAt: end.UTC().Format(time.RFC3339)}
	hasAlertname := false
	for _, m := range matchers {
		hasAlertname = hasAlertname || (m.in.Name == "alertname" && m.in.IsEqual && !m.in.IsRegex)
	}
	for _, a := range alerts {
		if matchAll(matchers, a.Labels) {
			p.Matches++
			if len(p.Sample) < previewSampleLen {
				p.Sample = append(p.Sample, describeAlert(a.Labels))
			}
		}
	}
	p.Broad = p.Matches > broadMatchCount || !hasAlertname
	if f.DryRun {
		writeJSON(w, http.StatusOK, envelope{Data: p, UpdatedAt: s.now()})
		return
	}
	ms := make([]source.Matcher, len(matchers))
	for i, m := range matchers {
		ms[i] = m.in
	}
	id, err := s.am.CreateSilence(ctx, source.NewSilence{Matchers: ms, StartsAt: start, EndsAt: end, CreatedBy: silenceAuthor, Comment: comment})
	if err != nil {
		s.logger.Warn("Silence: create failed", "err", err)
		fail(http.StatusBadGateway, "Alertmanager rejected the silence")
		return
	}
	s.logger.Info("Silence created", "silence", id, "matchers", len(ms), "starts", start, "ends", end)
	p.SilenceID = id
	s.forgetAlertCaches()
	writeJSON(w, http.StatusOK, envelope{Data: p, UpdatedAt: s.now()})
}

func (s *Server) amExpireSilence(w http.ResponseWriter, r *http.Request) {
	fail := func(code int, msg string) { writeJSON(w, code, envelope{Error: msg, UpdatedAt: s.now()}) }
	id := r.PathValue("id")
	switch {
	case !s.cfg.SilencesAllowed:
		fail(http.StatusForbidden, "silences are disabled on this stock_home")
		return
	case !sameOrigin(r):
		fail(http.StatusForbidden, "cross-origin request rejected")
		return
	case !silenceIDRe.MatchString(id):
		fail(http.StatusBadRequest, "invalid silence id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.QueryTimeout)
	defer cancel()
	if err := s.am.ExpireSilence(ctx, id); err != nil {
		s.logger.Warn("Silence: expire failed", "silence", id, "err", err)
		fail(http.StatusBadGateway, "Alertmanager could not expire the silence")
		return
	}
	s.logger.Info("Silence expired", "silence", id)
	s.forgetAlertCaches()
	writeJSON(w, http.StatusOK, envelope{Data: map[string]string{"silenceId": id}, UpdatedAt: s.now()})
}

func (s *Server) forgetAlertCaches() {
	for _, key := range []string{"alerts", "alert-items", "status", "am-groups", "am-silences"} {
		s.cache.Forget(key)
	}
}

func matchAll(ms []compiledMatcher, labels map[string]string) bool {
	for _, m := range ms {
		if !m.matches(labels) {
			return false
		}
	}
	return true
}

func describeAlert(l map[string]string) string {
	d := l["alertname"]
	for _, k := range []string{"symbol", "job", "severity"} {
		if v := l[k]; v != "" {
			d += " · " + v
		}
	}
	return d
}

// RouteNode is one node of the routing tree with matchers and timings only.
type RouteNode struct {
	Receiver       string      `json:"receiver"`
	Matchers       []string    `json:"matchers"`
	GroupBy        []string    `json:"groupBy"`
	GroupWait      string      `json:"groupWait,omitempty"`
	GroupInterval  string      `json:"groupInterval,omitempty"`
	RepeatInterval string      `json:"repeatInterval,omitempty"`
	Continue       bool        `json:"continue"`
	Routes         []RouteNode `json:"routes"`
}

// ReceiverInfo names a receiver and its integration types; destinations and credentials are never included.
type ReceiverInfo struct {
	Name         string           `json:"name"`
	Integrations []IntegrationRef `json:"integrations"`
}

type IntegrationRef struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// AMStatusView is the AlertManager status tab payload.
type AMStatusView struct {
	Ready           bool              `json:"ready"`
	Version         map[string]string `json:"version"`
	Uptime          *time.Time        `json:"uptime"`
	Cluster         string            `json:"cluster"`
	Peers           int               `json:"peers"`
	ConfigAvailable bool              `json:"configAvailable"`
	Route           *RouteNode        `json:"route"`
	Receivers       []ReceiverInfo    `json:"receivers"`
}

type amRouteYAML struct {
	Receiver       string            `yaml:"receiver"`
	GroupBy        []string          `yaml:"group_by"`
	GroupWait      string            `yaml:"group_wait"`
	GroupInterval  string            `yaml:"group_interval"`
	RepeatInterval string            `yaml:"repeat_interval"`
	Matchers       []string          `yaml:"matchers"`
	Match          map[string]string `yaml:"match"`
	MatchRE        map[string]string `yaml:"match_re"`
	Continue       bool              `yaml:"continue"`
	Routes         []amRouteYAML     `yaml:"routes"`
}

func (s *Server) amStatus(ctx context.Context) (any, error) {
	st, err := s.am.Status(ctx)
	if err != nil {
		return nil, err
	}
	v := AMStatusView{Version: map[string]string{}, Cluster: st.Cluster.Status, Peers: len(st.Cluster.Peers), Receivers: []ReceiverInfo{}}
	v.Ready, _ = source.Probe(ctx, s.http, s.cfg.AlertmanagerURL+"/-/ready")
	for _, k := range []string{"version", "revision", "branch", "buildDate", "goVersion"} {
		if x, ok := st.VersionInfo[k]; ok {
			v.Version[k] = x
		}
	}
	if !st.Uptime.IsZero() {
		v.Uptime = &st.Uptime
	}
	route, receivers, err := redactConfig(st.Config.Original)
	if err != nil {
		s.logger.Warn("AlertManager: configuration not parsed", "err", err)
		return v, nil
	}
	v.ConfigAvailable, v.Route, v.Receivers = true, route, receivers
	return v, nil
}

// redactConfig keeps the routing tree and receiver integration types; every other value is dropped.
func redactConfig(original string) (*RouteNode, []ReceiverInfo, error) {
	if strings.TrimSpace(original) == "" {
		return nil, nil, fmt.Errorf("empty configuration")
	}
	var cfg struct {
		Route     amRouteYAML      `yaml:"route"`
		Receivers []map[string]any `yaml:"receivers"`
	}
	if err := yaml.Unmarshal([]byte(original), &cfg); err != nil {
		return nil, nil, err
	}
	root := toRouteNode(cfg.Route)
	receivers := []ReceiverInfo{}
	for _, rc := range cfg.Receivers {
		name, _ := rc["name"].(string)
		info := ReceiverInfo{Name: name, Integrations: []IntegrationRef{}}
		for k, val := range rc {
			if t, ok := strings.CutSuffix(k, "_configs"); ok {
				n := 0
				if list, ok := val.([]any); ok {
					n = len(list)
				}
				info.Integrations = append(info.Integrations, IntegrationRef{Type: t, Count: n})
			}
		}
		sort.Slice(info.Integrations, func(i, j int) bool { return info.Integrations[i].Type < info.Integrations[j].Type })
		receivers = append(receivers, info)
	}
	return &root, receivers, nil
}

func toRouteNode(r amRouteYAML) RouteNode {
	n := RouteNode{
		Receiver: r.Receiver, Matchers: append([]string{}, r.Matchers...), GroupBy: nonNilSlice(r.GroupBy),
		GroupWait: r.GroupWait, GroupInterval: r.GroupInterval, RepeatInterval: r.RepeatInterval, Continue: r.Continue, Routes: []RouteNode{},
	}
	for _, k := range sortedKeys(r.Match) {
		n.Matchers = append(n.Matchers, fmt.Sprintf("%s=%q", k, r.Match[k]))
	}
	for _, k := range sortedKeys(r.MatchRE) {
		n.Matchers = append(n.Matchers, fmt.Sprintf("%s=~%q", k, r.MatchRE[k]))
	}
	for _, c := range r.Routes {
		n.Routes = append(n.Routes, toRouteNode(c))
	}
	return n
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func nonNil(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

func nonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (s *Server) registerAM(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/am/groups", s.section("am-groups", 5*time.Second, s.amGroups))
	mux.HandleFunc("GET /api/v1/am/silences", s.section("am-silences", 5*time.Second, s.amSilences))
	mux.HandleFunc("GET /api/v1/am/status", s.section("am-status", 30*time.Second, s.amStatus))
	mux.HandleFunc("POST /api/v1/am/silences", s.amCreateSilence)
	mux.HandleFunc("DELETE /api/v1/am/silences/{id}", s.amExpireSilence)
}
