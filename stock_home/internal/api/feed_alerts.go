package api

import (
	"context"
	"sort"
	"time"

	"github.com/seenimo/stock_home/internal/source"
)

// Feed is section 1: market data feed health.
type Feed struct {
	Source             string             `json:"source"`
	Mode               string             `json:"mode"`
	SessionValid       bool               `json:"sessionValid"`
	SessionExpiresAt   *time.Time         `json:"sessionExpiresAt"`
	Connections        *float64           `json:"connections"`
	Subscribed         *float64           `json:"subscribed"`
	LastTickAgeSeconds *float64           `json:"lastTickAgeSeconds"`
	TicksPerSecond     *float64           `json:"ticksPerSecond"`
	Reconnects1h       *float64           `json:"reconnects1h"`
	Errors1h           *float64           `json:"errors1h"`
	StaleRatio         *float64           `json:"staleRatio"`
	APIOutcomes10m     map[string]float64 `json:"apiOutcomes10m"`
	TickRateSpark      []source.Point     `json:"tickRateSpark"`
	ExploreURL         string             `json:"exploreUrl"`
}

func (s *Server) feed(ctx context.Context) (any, error) {
	q := s.querier(ctx)
	info := q.vector(`max by (source, mode) (stock_feed_info)`)
	f := Feed{
		SessionValid:       isOne(q.scalar(`max(stock_kite_session_valid)`)),
		SessionExpiresAt:   unixTime(q.scalar(`max(stock_kite_session_expiry_timestamp_seconds)`)),
		Connections:        q.scalar(`max(stock_feed_connections)`),
		Subscribed:         q.scalar(`max(stock_feed_subscribed_instruments)`),
		LastTickAgeSeconds: q.scalar(`time() - max(stock_feed_last_tick_timestamp_seconds > 0)`),
		TicksPerSecond:     q.scalar(`max(rate(stock_feed_ticks_total[1m]))`),
		Reconnects1h:       q.scalar(`max(increase(stock_feed_reconnects_total[1h]))`),
		Errors1h:           q.scalar(`max(increase(stock_feed_errors_total[1h]))`),
		StaleRatio:         q.scalar(`max(market:stale_core_instruments:ratio)`),
		APIOutcomes10m:     map[string]float64{},
		ExploreURL:         exploreURL(`max(rate(stock_feed_ticks_total[1m]))`, "now-6h"),
	}
	for _, smp := range q.vector(`sum by (outcome) (increase(stock_kite_api_requests_total[10m]))`) {
		f.APIOutcomes10m[smp.Labels["outcome"]] = smp.Value
	}
	end := s.now()
	if series := q.rangeQuery(`max(rate(stock_feed_ticks_total[1m]))`, end.Add(-time.Hour), end, time.Minute); len(series) > 0 {
		f.TickRateSpark = series[0].Points
	}
	if q.err != nil {
		return nil, q.err
	}
	if len(info) > 0 {
		f.Source, f.Mode = info[0].Labels["source"], info[0].Labels["mode"]
	}
	return f, nil
}

// AlertItem is one active alert.
type AlertItem struct {
	Name        string    `json:"name"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	Summary     string    `json:"summary"`
	Symbol      string    `json:"symbol,omitempty"`
	Exchange    string    `json:"exchange,omitempty"`
	StartsAt    time.Time `json:"startsAt"`
	SourceURL   string    `json:"sourceUrl,omitempty"`
	Fingerprint string    `json:"fingerprint"`
	// Value is the rule's optional "value" annotation.
	Value string `json:"value,omitempty"`
}

// Alerts is section 2: active alerts from Alertmanager.
type Alerts struct {
	BySeverity     map[string]int `json:"bySeverity"`
	ByCategory     map[string]int `json:"byCategory"`
	Silenced       int            `json:"silenced"`
	Inhibited      int            `json:"inhibited"`
	ActiveSilences int            `json:"activeSilences"`
	Items          []AlertItem    `json:"items"`
	Truncated      bool           `json:"truncated"`
}

const maxAlertItems = 50

var severityRank = map[string]int{"critical": 0, "warning": 1, "info": 2}

func (s *Server) alerts(ctx context.Context) (any, error) {
	list, err := s.am.Alerts(ctx)
	if err != nil {
		return nil, err
	}
	silences, err := s.am.Silences(ctx)
	if err != nil {
		return nil, err
	}
	out := Alerts{BySeverity: map[string]int{"critical": 0, "warning": 0, "info": 0}, ByCategory: map[string]int{}, Items: []AlertItem{}}
	for _, sl := range silences {
		if sl.Status.State == "active" {
			out.ActiveSilences++
		}
	}
	for _, a := range list {
		switch {
		case len(a.Status.SilencedBy) > 0:
			out.Silenced++
			continue
		case len(a.Status.InhibitedBy) > 0:
			out.Inhibited++
			continue
		}
		sev := a.Labels["severity"]
		out.BySeverity[sev]++
		if c := a.Labels["category"]; c != "" {
			out.ByCategory[c]++
		}
		summary := a.Annotations["summary"]
		if summary == "" {
			summary = a.Annotations["description"]
		}
		out.Items = append(out.Items, AlertItem{
			Name: a.Labels["alertname"], Severity: sev, Category: a.Labels["category"], Summary: summary,
			Symbol: a.Labels["symbol"], Exchange: a.Labels["exchange"], StartsAt: a.StartsAt,
			SourceURL: a.GeneratorURL, Fingerprint: a.Fingerprint, Value: a.Annotations["value"],
		})
	}
	sort.Slice(out.Items, func(i, j int) bool {
		ri, rj := rank(out.Items[i].Severity), rank(out.Items[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return out.Items[i].StartsAt.After(out.Items[j].StartsAt)
	})
	if len(out.Items) > maxAlertItems {
		out.Items, out.Truncated = out.Items[:maxAlertItems], true
	}
	return out, nil
}

func rank(sev string) int {
	if r, ok := severityRank[sev]; ok {
		return r
	}
	return len(severityRank)
}
