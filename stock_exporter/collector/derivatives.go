//go:build !noderivatives

package collector

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type derivativesCollector struct {
	deps         Deps
	info, expiry typedDesc
}

func init() {
	registerCollector("derivatives", defaultEnabled, NewDerivativesCollector)
}

// NewDerivativesCollector exposes contract metadata of futures and options for joins in rules:
// `x * on(exchange, symbol) group_left(underlying, expiry, strike) stock_instrument_info`.
func NewDerivativesCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	return &derivativesCollector{
		deps:   deps,
		info:   gaugeDesc("instrument", "info", "Contract metadata of a future or option (value 1).", append(instrumentLabels, "underlying", "expiry", "strike")...),
		expiry: gaugeDesc("instrument", "expiry_timestamp_seconds", "Expiry of a future or option (15:30 IST on the expiry date).", instrumentLabels...),
	}, nil
}

func (c *derivativesCollector) Update(ch chan<- prometheus.Metric, scope Scope) error {
	for _, e := range c.deps.Store.Snapshot() {
		in := e.Instrument
		if !in.IsDerivative() || !scope.Includes(in.Tier) {
			continue
		}
		lv := instrumentLabelValues(in)
		strike := ""
		if in.Strike > 0 {
			strike = strconv.FormatFloat(in.Strike, 'f', -1, 64)
		}
		ch <- c.info.mustNewConstMetric(1, append(lv, in.Underlying, in.Expiry.Format("2006-01-02"), strike)...)
		if !in.Expiry.IsZero() {
			// The instrument master carries the expiry date; contracts settle at the 15:30 IST (10:00 UTC) close.
			y, m, d := in.Expiry.Date()
			ch <- c.expiry.mustNewConstMetric(float64(time.Date(y, m, d, 10, 0, 0, 0, time.UTC).Unix()), lv...)
		}
	}
	return nil
}
