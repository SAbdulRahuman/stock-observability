// Copyright 2015 The Prometheus Authors
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

// Package collector includes all individual collectors that expose market metrics.
package collector

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/seenimo/stock_exporter/internal/market"
	"github.com/seenimo/stock_exporter/internal/store"
)

// Namespace defines the common namespace to be used by all metrics.
const namespace = "stock"

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"stock_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"stock_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

const (
	defaultEnabled  = true
	defaultDisabled = false
)

// Deps are the shared runtime dependencies handed to collector factories.
type Deps struct {
	Store    *store.Store
	Health   *store.Health
	Calendar *market.Calendar
	Now      func() time.Time
}

// Scope restricts a scrape to a subset of instrument tiers. An empty scope includes everything.
type Scope struct {
	Tiers map[string]bool
}

// Includes reports whether instruments of the given tier are in scope.
func (s Scope) Includes(tier string) bool {
	return len(s.Tiers) == 0 || s.Tiers[tier]
}

// NewScope builds a scope from tier names and validates them.
func NewScope(tiers ...string) (Scope, error) {
	if len(tiers) == 0 {
		return Scope{}, nil
	}
	s := Scope{Tiers: map[string]bool{}}
	for _, t := range tiers {
		if t != store.TierCore && t != store.TierBroad {
			return Scope{}, fmt.Errorf("unknown tier %q", t)
		}
		s.Tiers[t] = true
	}
	return s, nil
}

type factory func(logger *slog.Logger, deps Deps) (Collector, error)

var (
	factories              = make(map[string]factory)
	initiatedCollectorsMtx = sync.Mutex{}
	initiatedCollectors    = make(map[string]Collector)
	collectorState         = make(map[string]*bool)
	forcedCollectors       = map[string]bool{} // collectors which have been explicitly enabled or disabled
)

func registerCollector(collector string, isDefaultEnabled bool, f factory) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}

	flagName := fmt.Sprintf("collector.%s", collector)
	flagHelp := fmt.Sprintf("Enable the %s collector (default: %s).", collector, helpDefaultState)
	defaultValue := fmt.Sprintf("%v", isDefaultEnabled)

	flag := kingpin.Flag(flagName, flagHelp).Default(defaultValue).Action(collectorFlagAction(collector)).Bool()
	collectorState[collector] = flag

	factories[collector] = f
}

// Enabled reports whether the named collector is enabled.
func Enabled(name string) bool {
	s, ok := collectorState[name]
	return ok && *s
}

// StockCollector implements the prometheus.Collector interface.
type StockCollector struct {
	Collectors map[string]Collector
	scope      Scope
	logger     *slog.Logger
}

// DisableDefaultCollectors sets the collector state to false for all collectors which
// have not been explicitly enabled on the command line.
func DisableDefaultCollectors() {
	for c := range collectorState {
		if _, ok := forcedCollectors[c]; !ok {
			*collectorState[c] = false
		}
	}
}

// collectorFlagAction generates a new action function for the given collector
// to track whether it has been explicitly enabled or disabled from the command line.
// See: https://github.com/alecthomas/kingpin/issues/294
func collectorFlagAction(collector string) func(ctx *kingpin.ParseContext) error {
	return func(_ *kingpin.ParseContext) error {
		forcedCollectors[collector] = true
		return nil
	}
}

// NewStockCollector creates a StockCollector for the enabled collectors matching filters.
func NewStockCollector(logger *slog.Logger, deps Deps, scope Scope, filters ...string) (*StockCollector, error) {
	f := make(map[string]bool)
	for _, filter := range filters {
		enabled, exist := collectorState[filter]
		if !exist {
			return nil, fmt.Errorf("missing collector: %s", filter)
		}
		if !*enabled {
			return nil, fmt.Errorf("disabled collector: %s", filter)
		}
		f[filter] = true
	}
	collectors := make(map[string]Collector)
	initiatedCollectorsMtx.Lock()
	defer initiatedCollectorsMtx.Unlock()
	for key, enabled := range collectorState {
		if !*enabled || (len(f) > 0 && !f[key]) {
			continue
		}
		if collector, ok := initiatedCollectors[key]; ok {
			collectors[key] = collector
		} else {
			collector, err := factories[key](logger.With("collector", key), deps)
			if err != nil {
				return nil, err
			}
			collectors[key] = collector
			initiatedCollectors[key] = collector
		}
	}
	return &StockCollector{Collectors: collectors, scope: scope, logger: logger}, nil
}

// Describe implements the prometheus.Collector interface.
func (n StockCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n StockCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	for name, c := range n.Collectors {
		wg.Go(func() { execute(name, c, n.scope, ch, n.logger) })
	}
	wg.Wait()
}

func execute(name string, c Collector, scope Scope, ch chan<- prometheus.Metric, logger *slog.Logger) {
	begin := time.Now()
	err := c.Update(ch, scope)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		if IsNoDataError(err) {
			logger.Debug("collector returned no data", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		} else {
			logger.Error("collector failed", "name", name, "duration_seconds", duration.Seconds(), "err", err)
		}
		success = 0
	} else {
		logger.Debug("collector succeeded", "name", name, "duration_seconds", duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Update sends metrics for the given scope to ch.
	Update(ch chan<- prometheus.Metric, scope Scope) error
}

type typedDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

func (d *typedDesc) mustNewConstMetric(value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d.desc, d.valueType, value, labels...)
}

func gaugeDesc(subsystem, name, help string, labels ...string) typedDesc {
	return typedDesc{prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, name), help, labels, nil), prometheus.GaugeValue}
}

func counterDesc(subsystem, name, help string, labels ...string) typedDesc {
	return typedDesc{prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, name), help, labels, nil), prometheus.CounterValue}
}

// ErrNoData indicates the collector found no data to collect, but had no other error.
var ErrNoData = errors.New("collector returned no data")

// IsNoDataError reports whether err is ErrNoData.
func IsNoDataError(err error) bool {
	return errors.Is(err, ErrNoData)
}

func unixSeconds(t time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return float64(t.UnixNano()) / 1e9
}
