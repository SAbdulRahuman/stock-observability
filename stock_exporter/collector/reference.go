//go:build !noreference

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

type referenceCollector struct {
	deps                            Deps
	high, low, sma50, sma200, cross typedDesc
}

func init() {
	registerCollector("reference", defaultEnabled, NewReferenceCollector)
}

// NewReferenceCollector exposes 52-week and daily SMA levels from historical day candles.
func NewReferenceCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	return &referenceCollector{
		deps:   deps,
		high:   gaugeDesc("52w", "high_price", "Highest daily high of the last 52 weeks.", instrumentLabels...),
		low:    gaugeDesc("52w", "low_price", "Lowest daily low of the last 52 weeks.", instrumentLabels...),
		sma50:  gaugeDesc("sma", "50d_price", "Simple moving average of the last 50 daily closes.", instrumentLabels...),
		sma200: gaugeDesc("sma", "200d_price", "Simple moving average of the last 200 daily closes.", instrumentLabels...),
		cross:  gaugeDesc("sma", "50d_200d_cross", "1 if the latest daily close moved SMA50 above SMA200, -1 below; absent otherwise.", instrumentLabels...),
	}, nil
}

func (c *referenceCollector) Update(ch chan<- prometheus.Metric, scope Scope) error {
	for _, e := range c.deps.Store.Snapshot() {
		if !e.HasReference || !scope.Includes(e.Instrument.Tier) {
			continue
		}
		lv := instrumentLabelValues(e.Instrument)
		r := e.Reference
		ch <- c.high.mustNewConstMetric(r.High52W, lv...)
		ch <- c.low.mustNewConstMetric(r.Low52W, lv...)
		if r.SMA50D > 0 {
			ch <- c.sma50.mustNewConstMetric(r.SMA50D, lv...)
		}
		if r.SMA200D > 0 {
			ch <- c.sma200.mustNewConstMetric(r.SMA200D, lv...)
		}
		if r.Cross != 0 {
			ch <- c.cross.mustNewConstMetric(float64(r.Cross), lv...)
		}
	}
	return nil
}
