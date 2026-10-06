// Copyright 2026 The Stock Exporter Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !nocircuit

package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

type circuitCollector struct {
	deps         Deps
	upper, lower typedDesc
}

func init() {
	registerCollector("circuit", defaultEnabled, NewCircuitCollector)
}

// NewCircuitCollector exposes daily circuit (price band) limits.
func NewCircuitCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	return &circuitCollector{
		deps:  deps,
		upper: gaugeDesc("circuit", "upper_limit_price", "Upper circuit (price band) limit for the day.", instrumentLabels...),
		lower: gaugeDesc("circuit", "lower_limit_price", "Lower circuit (price band) limit for the day.", instrumentLabels...),
	}, nil
}

func (c *circuitCollector) Update(ch chan<- prometheus.Metric, scope Scope) error {
	for _, e := range c.deps.Store.Snapshot() {
		if !e.HasCircuit || !scope.Includes(e.Instrument.Tier) {
			continue
		}
		lv := instrumentLabelValues(e.Instrument)
		ch <- c.upper.mustNewConstMetric(e.Circuit.Upper, lv...)
		ch <- c.lower.mustNewConstMetric(e.Circuit.Lower, lv...)
	}
	return nil
}
