// Package source holds clients for the upstream observability services.
package source

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Sample is one element of an instant vector.
type Sample struct {
	Labels map[string]string
	Value  float64
}

// Point is one timestamped value of a range series.
type Point struct {
	T int64   `json:"t"` // Unix milliseconds.
	V float64 `json:"v"`
}

// Series is one element of a range query result.
type Series struct {
	Labels map[string]string
	Points []Point
}

// Prometheus runs PromQL queries against a Prometheus server.
type Prometheus struct {
	api v1.API
}

// NewPrometheus returns a client for the Prometheus server at baseURL (including any route prefix).
func NewPrometheus(baseURL string) (*Prometheus, error) {
	c, err := api.NewClient(api.Config{Address: baseURL})
	if err != nil {
		return nil, err
	}
	return &Prometheus{api: v1.NewAPI(c)}, nil
}

// Vector runs an instant query. NaN and infinite samples are dropped because JSON cannot carry them.
func (p *Prometheus) Vector(ctx context.Context, query string) ([]Sample, error) {
	v, _, err := p.api.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("query %q: %w", query, err)
	}
	switch r := v.(type) {
	case model.Vector:
		out := make([]Sample, 0, len(r))
		for _, s := range r {
			if f := float64(s.Value); finite(f) {
				out = append(out, Sample{Labels: labels(s.Metric), Value: f})
			}
		}
		return out, nil
	case *model.Scalar:
		if f := float64(r.Value); finite(f) {
			return []Sample{{Labels: map[string]string{}, Value: f}}, nil
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("query %q: unexpected result type %s", query, v.Type())
	}
}

// Scalar returns the first sample of an instant query; ok is false when the result is empty.
func (p *Prometheus) Scalar(ctx context.Context, query string) (value float64, ok bool, err error) {
	v, err := p.Vector(ctx, query)
	if err != nil || len(v) == 0 {
		return 0, false, err
	}
	return v[0].Value, true, nil
}

// Range runs a range query.
func (p *Prometheus) Range(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]Series, error) {
	v, _, err := p.api.QueryRange(ctx, query, v1.Range{Start: start, End: end, Step: step})
	if err != nil {
		return nil, fmt.Errorf("range query %q: %w", query, err)
	}
	m, ok := v.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("range query %q: unexpected result type %s", query, v.Type())
	}
	out := make([]Series, 0, len(m))
	for _, s := range m {
		pts := make([]Point, 0, len(s.Values))
		for _, sp := range s.Values {
			if f := float64(sp.Value); finite(f) {
				pts = append(pts, Point{T: int64(sp.Timestamp), V: f})
			}
		}
		out = append(out, Series{Labels: labels(s.Metric), Points: pts})
	}
	return out, nil
}

func labels(m model.Metric) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[string(k)] = string(v)
	}
	return out
}

func finite(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) }
