package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/seenimo/stock_home/internal/source"
)

// Alertmanager has no acknowledge state: an acknowledgement is a silence whose comment carries this prefix.
const (
	ackPrefix     = "Acknowledged via stock_home"
	silencePrefix = "Silenced via stock_home"
	silenceAuthor = "stock_home"
)

// AlertRow is one alert on the Alerts page: active from Alertmanager or resolved in the last 24h.
type AlertRow struct {
	Key           string     `json:"key"`
	Fingerprint   string     `json:"fingerprint,omitempty"`
	Name          string     `json:"name"`
	Severity      string     `json:"severity"`
	Category      string     `json:"category,omitempty"`
	Summary       string     `json:"summary,omitempty"`
	Description   string     `json:"description,omitempty"`
	Symbol        string     `json:"symbol,omitempty"`
	Exchange      string     `json:"exchange,omitempty"`
	Value         string     `json:"value,omitempty"`
	Threshold     string     `json:"threshold,omitempty"`
	StartsAt      time.Time  `json:"startsAt"`
	EndsAt        *time.Time `json:"endsAt,omitempty"`
	Status        string     `json:"status"` // firing, acknowledged, silenced, inhibited or resolved.
	SilencedUntil *time.Time `json:"silencedUntil,omitempty"`
	Source        string     `json:"source"`
	SourceURL     string     `json:"sourceUrl,omitempty"`
	ExploreURL    string     `json:"exploreUrl,omitempty"`
}

// AlertList is the Alerts page table plus the 24h firing history used for its activity and counts panels.
type AlertList struct {
	Items     []AlertRow     `json:"items"`
	Timeline  []TimelineItem `json:"timeline"`
	From      time.Time      `json:"from"`
	Truncated bool           `json:"truncated"`
}

func (s *Server) alertItems(ctx context.Context) (any, error) {
	alerts, err := s.am.Alerts(ctx)
	if err != nil {
		return nil, err
	}
	silences, err := s.am.Silences(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[string]source.Silence{}
	for _, sl := range silences {
		byID[sl.ID] = sl
	}
	out := AlertList{Items: []AlertRow{}, Timeline: []TimelineItem{}}
	for _, a := range alerts {
		out.Items = append(out.Items, alertRow(a, byID))
	}
	// History is best effort: the table still works when Prometheus is down.
	res, err := s.cache.Get("timeline", 60*time.Second, 11*time.Minute, func() (any, error) { return s.timeline(ctx) })
	if err != nil {
		s.logger.Warn("Alert items: timeline unavailable", "err", err)
	} else {
		tl := res.Value.(Timeline)
		out.Timeline, out.From, out.Truncated = tl.Items, tl.From, tl.Truncated
		for _, it := range tl.Items {
			if it.End == nil {
				continue
			}
			out.Items = append(out.Items, AlertRow{
				Key: fmt.Sprintf("resolved:%s:%s:%d", it.Name, it.Subject, it.Start.Unix()), Name: it.Name, Severity: it.Severity,
				Category: it.Category, Symbol: it.Symbol, StartsAt: it.Start, EndsAt: it.End, Status: "resolved", Source: "Prometheus",
				ExploreURL: symbolURL(it.Symbol),
			})
		}
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].StartsAt.After(out.Items[j].StartsAt) })
	return out, nil
}

func alertRow(a source.Alert, silences map[string]source.Silence) AlertRow {
	r := AlertRow{
		Key: a.Fingerprint, Fingerprint: a.Fingerprint, Name: a.Labels["alertname"], Severity: a.Labels["severity"],
		Category: a.Labels["category"], Summary: a.Annotations["summary"], Description: a.Annotations["description"],
		Symbol: a.Labels["symbol"], Exchange: a.Labels["exchange"], Value: a.Annotations["value"], Threshold: a.Annotations["threshold"],
		StartsAt: a.StartsAt, Status: "firing", Source: alertSource(a.GeneratorURL), SourceURL: a.GeneratorURL,
	}
	if r.Symbol != "" {
		ex := r.Exchange
		if ex == "" {
			ex = "NSE"
		}
		r.ExploreURL = symbolExploreURL(ex, r.Symbol)
	}
	switch {
	case len(a.Status.SilencedBy) > 0:
		r.Status = "silenced"
		for _, id := range a.Status.SilencedBy {
			sl, ok := silences[id]
			if !ok {
				continue
			}
			if strings.HasPrefix(sl.Comment, ackPrefix) {
				r.Status = "acknowledged"
			}
			if until := sl.EndsAt; r.SilencedUntil == nil || until.After(*r.SilencedUntil) {
				r.SilencedUntil = &until
			}
		}
	case len(a.Status.InhibitedBy) > 0:
		r.Status = "inhibited"
	}
	return r
}

func symbolURL(symbol string) string {
	if symbol == "" {
		return ""
	}
	return symbolExploreURL("NSE", symbol)
}

// alertSource names the alert's origin from its generator URL.
func alertSource(generatorURL string) string {
	u, err := url.Parse(generatorURL)
	switch {
	case generatorURL == "" || err != nil:
		return "Alertmanager"
	case strings.Contains(u.Path, "/grafana"):
		return "Grafana"
	default:
		return "Prometheus"
	}
}

// SilenceRequest silences active alerts by fingerprint; matchers are derived server-side from each alert's labels.
type SilenceRequest struct {
	Fingerprints []string `json:"fingerprints"`
	Kind         string   `json:"kind"` // acknowledge or silence.
	Duration     string   `json:"duration"`
	Comment      string   `json:"comment"`
}

// SilenceResult reports per-alert outcomes.
type SilenceResult struct {
	Created []SilenceCreated `json:"created"`
	Failed  []SilenceFailed  `json:"failed"`
}

type SilenceCreated struct {
	Fingerprint string `json:"fingerprint"`
	SilenceID   string `json:"silenceId"`
}

type SilenceFailed struct {
	Fingerprint string `json:"fingerprint"`
	Error       string `json:"error"`
}

var silenceDurations = map[string]time.Duration{
	"30m": 30 * time.Minute, "1h": time.Hour, "2h": 2 * time.Hour, "4h": 4 * time.Hour, "24h": 24 * time.Hour,
}

const (
	maxSilenceAlerts  = 50
	maxSilenceComment = 200
	maxSilenceBody    = 16 << 10
)

func (s *Server) createSilences(w http.ResponseWriter, r *http.Request) {
	fail := func(code int, msg string) { writeJSON(w, code, envelope{Error: msg, UpdatedAt: s.now()}) }
	if !s.cfg.SilencesAllowed {
		fail(http.StatusForbidden, "silences are disabled on this stock_home")
		return
	}
	if !sameOrigin(r) {
		fail(http.StatusForbidden, "cross-origin request rejected")
		return
	}
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
		fail(http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}
	var req SilenceRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, maxSilenceBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		fail(http.StatusBadRequest, "invalid JSON body")
		return
	}
	dur, ok := silenceDurations[req.Duration]
	if !ok {
		fail(http.StatusBadRequest, "duration must be one of 30m, 1h, 2h, 4h, 24h")
		return
	}
	prefix := map[string]string{"acknowledge": ackPrefix, "silence": silencePrefix}[req.Kind]
	if prefix == "" {
		fail(http.StatusBadRequest, "kind must be acknowledge or silence")
		return
	}
	comment := strings.TrimSpace(req.Comment)
	if len([]rune(comment)) > maxSilenceComment || strings.IndexFunc(comment, func(r rune) bool { return unicode.IsControl(r) }) >= 0 {
		fail(http.StatusBadRequest, fmt.Sprintf("comment must be at most %d printable characters", maxSilenceComment))
		return
	}
	if len(req.Fingerprints) == 0 || len(req.Fingerprints) > maxSilenceAlerts {
		fail(http.StatusBadRequest, fmt.Sprintf("fingerprints: 1 to %d required", maxSilenceAlerts))
		return
	}
	if comment != "" {
		prefix += ": " + comment
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.QueryTimeout)
	defer cancel()
	alerts, err := s.am.Alerts(ctx)
	if err != nil {
		s.logger.Warn("Silence: Alertmanager unavailable", "err", err)
		fail(http.StatusBadGateway, "upstream unavailable")
		return
	}
	active := map[string]source.Alert{}
	for _, a := range alerts {
		active[a.Fingerprint] = a
	}
	res := SilenceResult{Created: []SilenceCreated{}, Failed: []SilenceFailed{}}
	now := s.now()
	for _, fp := range dedupe(req.Fingerprints) {
		a, ok := active[fp]
		switch {
		case !ok:
			res.Failed = append(res.Failed, SilenceFailed{fp, "alert is not active"})
			continue
		case len(a.Status.SilencedBy) > 0:
			res.Failed = append(res.Failed, SilenceFailed{fp, "alert is already silenced"})
			continue
		}
		id, err := s.am.CreateSilence(ctx, source.NewSilence{
			Matchers: exactMatchers(a.Labels), StartsAt: now, EndsAt: now.Add(dur), CreatedBy: silenceAuthor, Comment: prefix,
		})
		if err != nil {
			s.logger.Warn("Silence: create failed", "fingerprint", fp, "err", err)
			res.Failed = append(res.Failed, SilenceFailed{fp, "Alertmanager rejected the silence"})
			continue
		}
		s.logger.Info("Silence created", "fingerprint", fp, "silence", id, "kind", req.Kind, "duration", req.Duration)
		res.Created = append(res.Created, SilenceCreated{fp, id})
	}
	for _, key := range []string{"alerts", "alert-items", "status"} {
		s.cache.Forget(key)
	}
	code := http.StatusOK
	if len(res.Created) == 0 {
		code = http.StatusConflict
	}
	writeJSON(w, code, envelope{Data: res, UpdatedAt: s.now()})
}

// sameOrigin rejects browser requests from other sites; non-browser clients send neither header.
func sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host {
			return false
		}
	}
	return true
}

func exactMatchers(labels map[string]string) []source.Matcher {
	out := make([]source.Matcher, 0, len(labels))
	for k, v := range labels {
		out = append(out, source.Matcher{Name: k, Value: v, IsEqual: true})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
