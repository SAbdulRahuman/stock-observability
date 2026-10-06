package api

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/seenimo/stock_home/internal/source"
)

// Component is the health of one stack component.
type Component struct {
	Name      string  `json:"name"`
	Healthy   bool    `json:"healthy"`
	LatencyMs float64 `json:"latencyMs"`
	Href      string  `json:"href"`
}

// Target is the scrape health of one Prometheus job.
type Target struct {
	Job             string   `json:"job"`
	Up              bool     `json:"up"`
	DurationSeconds *float64 `json:"durationSeconds"`
	TimeoutSeconds  *float64 `json:"timeoutSeconds"`
	SampleUsage     *float64 `json:"sampleUsage"` // Fraction of sample_limit used.
}

// System is section 9: stack and scrape health.
type System struct {
	Components []Component     `json:"components"`
	Targets    []Target        `json:"targets"`
	Collectors map[string]bool `json:"collectors"`
	HeadSeries *float64        `json:"headSeries"`
}

func (s *Server) system(ctx context.Context) (any, error) {
	sys := System{Components: s.probeComponents(ctx), Targets: []Target{}, Collectors: map[string]bool{}}

	q := s.querier(ctx)
	up := q.vector(`min by (job) (up)`)
	dur := byLabel(q.vector(`max by (job) (scrape_duration_seconds)`), "job")
	timeout := byLabel(q.vector(`max by (job) (scrape_timeout_seconds)`), "job")
	usage := byLabel(q.vector(`max by (job) (scrape_samples_post_metric_relabeling / (scrape_sample_limit > 0))`), "job")
	for _, smp := range q.vector(`min by (collector) (stock_scrape_collector_success)`) {
		sys.Collectors[smp.Labels["collector"]] = smp.Value == 1
	}
	sys.HeadSeries = q.scalar(`max(prometheus_tsdb_head_series)`)
	if q.err != nil {
		// Component probes are still useful when Prometheus itself is down.
		s.logger.Warn("System: Prometheus query failed", "err", q.err)
		return sys, nil
	}
	for _, smp := range up {
		job := smp.Labels["job"]
		sys.Targets = append(sys.Targets, Target{
			Job: job, Up: smp.Value == 1,
			DurationSeconds: ptr(dur, job), TimeoutSeconds: ptr(timeout, job), SampleUsage: ptr(usage, job),
		})
	}
	sort.Slice(sys.Targets, func(i, j int) bool { return sys.Targets[i].Job < sys.Targets[j].Job })
	return sys, nil
}

func (s *Server) probeComponents(ctx context.Context) []Component {
	checks := []struct{ name, url, href string }{
		{"Prometheus", s.cfg.PrometheusURL + "/-/ready", "/prometheus/"},
		{"Alertmanager", s.cfg.AlertmanagerURL + "/-/ready", "/alertmanager/"},
		{"Grafana", s.cfg.GrafanaURL + "/api/health", "/grafana/"},
	}
	out := make([]Component, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, lat := source.Probe(ctx, s.http, c.url)
			out[i] = Component{Name: c.name, Healthy: ok, LatencyMs: float64(lat.Microseconds()) / 1000, Href: c.href}
		}()
	}
	wg.Wait()
	return out
}

// TimelineItem is one firing period of an alert.
type TimelineItem struct {
	Name     string     `json:"name"`
	Severity string     `json:"severity"`
	Category string     `json:"category"`
	Subject  string     `json:"subject,omitempty"`
	Start    time.Time  `json:"start"`
	End      *time.Time `json:"end"` // nil while still firing.
}

// Timeline is section 10: alert firing history.
type Timeline struct {
	From       time.Time      `json:"from"`
	Items      []TimelineItem `json:"items"`
	BySeverity map[string]int `json:"bySeverity"`
	Truncated  bool           `json:"truncated"`
}

const (
	timelineWindow = 24 * time.Hour
	timelineStep   = time.Minute
	maxTimeline    = 100
)

func (s *Server) timeline(ctx context.Context) (any, error) {
	end := s.now()
	start := end.Add(-timelineWindow)
	series, err := s.prom.Range(ctx, `ALERTS{alertstate="firing"}`, start, end, timelineStep)
	if err != nil {
		return nil, err
	}
	t := buildTimeline(series, start, end, timelineStep)
	return t, nil
}

func buildTimeline(series []source.Series, start, end time.Time, step time.Duration) Timeline {
	t := Timeline{From: start, Items: []TimelineItem{}, BySeverity: map[string]int{}}
	for _, s := range series {
		var runStart, prev int64 = -1, -1
		flush := func(ongoing bool) {
			if runStart < 0 {
				return
			}
			it := TimelineItem{
				Name: s.Labels["alertname"], Severity: s.Labels["severity"], Category: s.Labels["category"],
				Subject: subject(s.Labels), Start: time.UnixMilli(runStart).UTC(),
			}
			if !ongoing {
				e := time.UnixMilli(prev).Add(step).UTC()
				it.End = &e
			}
			t.Items = append(t.Items, it)
			t.BySeverity[it.Severity]++
		}
		for _, p := range s.Points {
			if prev >= 0 && p.T-prev > step.Milliseconds() {
				flush(false)
				runStart = -1
			}
			if runStart < 0 {
				runStart = p.T
			}
			prev = p.T
		}
		flush(prev >= 0 && end.UnixMilli()-prev <= 2*step.Milliseconds())
	}
	sort.Slice(t.Items, func(i, j int) bool { return t.Items[i].Start.After(t.Items[j].Start) })
	if len(t.Items) > maxTimeline {
		t.Items, t.Truncated = t.Items[:maxTimeline], true
	}
	return t
}

func subject(l map[string]string) string {
	var parts []string
	for _, k := range []string{"symbol", "sector", "segment", "job"} {
		if v := l[k]; v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " · ")
}

func byLabel(v []source.Sample, label string) map[string]float64 {
	out := make(map[string]float64, len(v))
	for _, s := range v {
		out[s.Labels[label]] = s.Value
	}
	return out
}

func ptr(m map[string]float64, k string) *float64 {
	if v, ok := m[k]; ok {
		return &v
	}
	return nil
}
