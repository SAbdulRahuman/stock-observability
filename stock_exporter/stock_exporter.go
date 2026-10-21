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

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/user"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	promcollectors "github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/bootstrap"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"

	"github.com/seenimo/stock_exporter/collector"
	"github.com/seenimo/stock_exporter/internal/kite"
	"github.com/seenimo/stock_exporter/internal/market"
	"github.com/seenimo/stock_exporter/internal/questdb"
	"github.com/seenimo/stock_exporter/internal/sim"
	"github.com/seenimo/stock_exporter/internal/store"
	"github.com/seenimo/stock_exporter/internal/watchlist"
)

// handler wraps an unfiltered http.Handler but uses a filtered handler,
// created on the fly, if collector or tier filtering is requested.
type handler struct {
	unfilteredHandler http.Handler
	// enabledCollectors list is used for logging and filtering
	enabledCollectors []string
	// exporterMetricsRegistry is a separate registry for the metrics about
	// the exporter itself.
	exporterMetricsRegistry *prometheus.Registry
	includeExporterMetrics  bool
	maxRequests             int
	deps                    collector.Deps
	logger                  *slog.Logger
}

func newHandler(deps collector.Deps, includeExporterMetrics bool, maxRequests int, logger *slog.Logger) (*handler, error) {
	h := &handler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		includeExporterMetrics:  includeExporterMetrics,
		maxRequests:             maxRequests,
		deps:                    deps,
		logger:                  logger,
	}
	if h.includeExporterMetrics {
		h.exporterMetricsRegistry.MustRegister(
			promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}),
			promcollectors.NewGoCollector(),
		)
	}
	innerHandler, err := h.innerHandler(collector.Scope{})
	if err != nil {
		return nil, fmt.Errorf("couldn't create metrics handler: %w", err)
	}
	h.unfilteredHandler = innerHandler
	return h, nil
}

// ServeHTTP implements http.Handler. Supported query parameters are
// collect[] / exclude[] (collector names) and tier (core or broad).
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	collects := r.URL.Query()["collect[]"]
	excludes := r.URL.Query()["exclude[]"]
	tiers := r.URL.Query()["tier"]
	h.logger.Debug("metrics query", "collects", collects, "excludes", excludes, "tiers", tiers)

	if len(collects) == 0 && len(excludes) == 0 && len(tiers) == 0 {
		h.unfilteredHandler.ServeHTTP(w, r)
		return
	}

	if len(collects) > 0 && len(excludes) > 0 {
		h.logger.Debug("rejecting combined collect and exclude queries")
		http.Error(w, "Combined collect and exclude queries are not allowed.", http.StatusBadRequest)
		return
	}

	scope, err := collector.NewScope(tiers...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filters := collects
	if len(excludes) > 0 {
		filters = []string{}
		for _, c := range h.enabledCollectors {
			if !slices.Contains(excludes, c) {
				filters = append(filters, c)
			}
		}
	}

	filteredHandler, err := h.innerHandler(scope, filters...)
	if err != nil {
		h.logger.Warn("Couldn't create filtered metrics handler:", "err", err)
		http.Error(w, fmt.Sprintf("Couldn't create filtered metrics handler: %s", err), http.StatusBadRequest)
		return
	}
	filteredHandler.ServeHTTP(w, r)
}

// innerHandler creates the unfiltered handler (no filters, empty scope) and
// the filtered handlers created on the fly.
func (h *handler) innerHandler(scope collector.Scope, filters ...string) (http.Handler, error) {
	sc, err := collector.NewStockCollector(h.logger, h.deps, scope, filters...)
	if err != nil {
		return nil, fmt.Errorf("couldn't create collector: %w", err)
	}

	// Only log the creation of an unfiltered handler, which happens once at startup.
	if len(filters) == 0 && len(scope.Tiers) == 0 {
		for n := range sc.Collectors {
			h.enabledCollectors = append(h.enabledCollectors, n)
		}
		sort.Strings(h.enabledCollectors)
		h.logger.Info("Enabled collectors", "collectors", strings.Join(h.enabledCollectors, ","))
	}

	r := prometheus.NewRegistry()
	r.MustRegister(versioncollector.NewCollector("stock_exporter"))
	if err := r.Register(sc); err != nil {
		return nil, fmt.Errorf("couldn't register stock collector: %w", err)
	}

	opts := promhttp.HandlerOpts{
		ErrorLog:            slog.NewLogLogger(h.logger.Handler(), slog.LevelError),
		ErrorHandling:       promhttp.ContinueOnError,
		MaxRequestsInFlight: h.maxRequests,
	}
	if !h.includeExporterMetrics {
		return promhttp.HandlerFor(r, opts), nil
	}
	// Use h.exporterMetricsRegistry so all expositions share the promhttp metrics.
	opts.Registry = h.exporterMetricsRegistry
	return promhttp.InstrumentMetricHandler(
		h.exporterMetricsRegistry,
		promhttp.HandlerFor(prometheus.Gatherers{h.exporterMetricsRegistry, r}, opts),
	), nil
}

type options struct {
	source        string
	watchlistFile string
	holidaysFile  string
	timezone      string
	forceOpen     bool
	questdbURL    string

	kiteAPIKey         string
	kiteAPISecretFile  string
	kiteCredsFile      string
	kitePostLogin      string
	kiteTokenFile      string
	kiteMode           string
	kiteCircuitEvery   time.Duration
	kiteReferenceEvery time.Duration
	kitePortfolioEvery time.Duration
	kiteBaseURI        string

	simSeed        int64
	simInterval    time.Duration
	simEventProb   float64
	simHoldings    int
	simBroadSize   int
	simDerivatives bool
}

func readSecretFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("--kite.api-secret-file is required")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return s, nil
}

// kiteCredentials resolves credentials from flags or the credentials file; both unset is allowed.
func kiteCredentials(o options) (kite.Credentials, string, error) {
	flags := o.kiteAPIKey != "" || o.kiteAPISecretFile != ""
	switch {
	case flags && o.kiteCredsFile != "":
		return kite.Credentials{}, "", errors.New("--kite.credentials-file cannot be combined with --kite.api-key or --kite.api-secret-file")
	case flags:
		if o.kiteAPIKey == "" {
			return kite.Credentials{}, "", errors.New("--kite.api-key is required with --kite.api-secret-file")
		}
		secret, err := readSecretFile(o.kiteAPISecretFile)
		if err != nil {
			return kite.Credentials{}, "", fmt.Errorf("reading Kite API secret: %w", err)
		}
		return kite.Credentials{APIKey: o.kiteAPIKey, APISecret: secret}, kite.SourceFlags, nil
	case o.kiteCredsFile != "":
		c, err := kite.LoadCredentials(o.kiteCredsFile)
		if err != nil {
			return kite.Credentials{}, "", err
		}
		if c.APIKey == "" {
			return c, kite.SourceNone, nil
		}
		return c, kite.SourceFile, nil
	}
	return kite.Credentials{}, kite.SourceNone, nil
}

// start loads configuration, starts the feed in the background and returns collector dependencies.
func start(ctx context.Context, o options, b *bootstrap.Bootstrap) (collector.Deps, error) {
	loc, err := time.LoadLocation(o.timezone)
	if err != nil {
		return collector.Deps{}, fmt.Errorf("loading timezone: %w", err)
	}
	var holidays map[string]string
	if o.holidaysFile != "" {
		if holidays, err = market.LoadHolidays(o.holidaysFile); err != nil {
			return collector.Deps{}, err
		}
	}
	wl, err := watchlist.Load(o.watchlistFile)
	if err != nil {
		return collector.Deps{}, err
	}
	cal := market.NewCalendar(loc, holidays, o.forceOpen)
	st := store.New()
	deps := collector.Deps{Store: st, Calendar: cal, Now: time.Now}
	if o.questdbURL != "" {
		writer, err := questdb.New(o.questdbURL, b.Logger)
		if err != nil {
			return collector.Deps{}, err
		}
		st.SetObserver(writer.Observe)
		go writer.Run(ctx)
	}

	switch o.source {
	case "sim":
		deps.Health = store.NewHealth("sim", "full")
		ins := wl.Synthetic(o.simBroadSize)
		if o.simDerivatives {
			ins = append(ins, sim.Derivatives(ins, time.Now().In(loc))...)
		}
		st.SetInstruments(ins)
		deps.Health.SetSession(true, time.Time{})
		s := sim.New(sim.Config{Seed: o.simSeed, Interval: o.simInterval, EventProbability: o.simEventProb, Holdings: o.simHoldings,
			IsTradingDay: deps.Calendar.IsTradingDay}, st, deps.Health, loc)
		go s.Run(ctx)
		b.Logger.Warn("Running with SIMULATED market data", "instruments", len(st.Instruments()))
	case "kite":
		mode, err := kite.ParseMode(o.kiteMode)
		if err != nil {
			return collector.Deps{}, err
		}
		creds, source, err := kiteCredentials(o)
		if err != nil {
			return collector.Deps{}, err
		}
		deps.Health = store.NewHealth("kite", string(mode))
		if mode != kiteticker.ModeFull && (collector.Enabled("depth") || collector.Enabled("orderbook")) {
			b.Logger.Warn("Order book depth is only streamed in full mode; depth and orderbook metrics will be empty", "mode", mode)
		}
		sup, err := kite.NewSupervisor(kite.SupervisorConfig{
			Runner: kite.RunnerConfig{
				Feed:              kite.FeedConfig{Mode: mode},
				BaseURI:           o.kiteBaseURI,
				CircuitInterval:   o.kiteCircuitEvery,
				ReferenceInterval: o.kiteReferenceEvery,
				PortfolioPoll:     collector.Enabled("portfolio"),
				PortfolioEvery:    o.kitePortfolioEvery,
			},
			TokenPath:         o.kiteTokenFile,
			CredentialsPath:   o.kiteCredsFile,
			PostLoginRedirect: o.kitePostLogin,
			Location:          loc,
		}, creds, source, wl, st, deps.Health, b.Logger)
		if err != nil {
			return collector.Deps{}, err
		}
		b.Handle("/login", sup.LoginHandler())
		b.Handle("/kite/callback", sup.CallbackHandler())
		b.Handle("/kite/config", sup.ConfigHandler())
		sup.Start(ctx)
	default:
		return collector.Deps{}, fmt.Errorf("unknown feed source %q", o.source)
	}
	return deps, nil
}

func main() {
	var (
		o                        options
		disableDefaultCollectors = kingpin.Flag("collector.disable-defaults", "Set all collectors to disabled by default.").Default("false").Bool()
		maxProcs                 = kingpin.Flag("runtime.gomaxprocs", "The target number of CPUs Go will run on (GOMAXPROCS).").Default("2").Int()
	)
	kingpin.Flag("feed.source", "Market data source: kite (live Zerodha Kite Connect) or sim (synthetic).").Default("kite").EnumVar(&o.source, "kite", "sim")
	kingpin.Flag("stock.watchlist-file", "YAML file listing instruments (core tier) and universes (broad tier).").Required().StringVar(&o.watchlistFile)
	kingpin.Flag("market.holidays-file", "YAML file with exchange holidays.").Default("").StringVar(&o.holidaysFile)
	kingpin.Flag("market.timezone", "Exchange timezone.").Default("Asia/Kolkata").StringVar(&o.timezone)
	kingpin.Flag("market.force-open", "Report the market as always open (testing and demos only).").Default("false").BoolVar(&o.forceOpen)
	kingpin.Flag("questdb.url", "QuestDB HTTP base URL for parallel market-state storage; empty disables QuestDB writes.").Default("").StringVar(&o.questdbURL)
	kingpin.Flag("kite.api-key", "Kite Connect API key.").Default("").StringVar(&o.kiteAPIKey)
	kingpin.Flag("kite.api-secret-file", "File containing the Kite Connect API secret.").Default("").StringVar(&o.kiteAPISecretFile)
	kingpin.Flag("kite.credentials-file", "JSON file with api_key and api_secret, editable at runtime via /kite/config (mode 0600). Excludes --kite.api-key.").Default("").StringVar(&o.kiteCredsFile)
	kingpin.Flag("kite.post-login-redirect", "Site-relative path the browser returns to after the Kite login callback, e.g. /settings?tab=data-sources&source=kite.").Default("").StringVar(&o.kitePostLogin)
	kingpin.Flag("kite.token-file", "File where the daily Kite access token is persisted (mode 0600).").Default("data/kite_session.json").StringVar(&o.kiteTokenFile)
	kingpin.Flag("kite.ticker-mode", "Kite WebSocket mode: full, quote or ltp.").Default("full").StringVar(&o.kiteMode)
	kingpin.Flag("kite.circuit-refresh-interval", "How often to refresh circuit limits via the quote API.").Default("5m").DurationVar(&o.kiteCircuitEvery)
	kingpin.Flag("kite.reference-refresh-interval", "How often to refresh 52-week and daily SMA levels of core instruments from historical day candles (needs the Kite historical data add-on; 0 disables).").Default("6h").DurationVar(&o.kiteReferenceEvery)
	kingpin.Flag("kite.portfolio-refresh-interval", "How often to refresh holdings, positions and margins.").Default("30s").DurationVar(&o.kitePortfolioEvery)
	kingpin.Flag("kite.base-uri", "Override the Kite REST API base URI (testing).").Default("").Hidden().StringVar(&o.kiteBaseURI)
	kingpin.Flag("sim.seed", "Random seed for the simulator.").Default("1").Int64Var(&o.simSeed)
	kingpin.Flag("sim.tick-interval", "Interval between simulated ticks.").Default("1s").DurationVar(&o.simInterval)
	kingpin.Flag("sim.event-probability", "Per-tick probability of a 2-5% price jump.").Default("0.0005").Float64Var(&o.simEventProb)
	kingpin.Flag("sim.holdings", "Number of simulated portfolio holdings.").Default("5").IntVar(&o.simHoldings)
	kingpin.Flag("sim.broad-size", "Number of synthetic broad-tier instruments when universes are configured.").Default("2000").IntVar(&o.simBroadSize)
	kingpin.Flag("sim.derivatives", "Add a NIFTY monthly future and a 7-strike option chain (core tier, NFO).").Default("false").BoolVar(&o.simDerivatives)

	kingpin.CommandLine.UsageWriter(os.Stdout)
	runner := bootstrap.New(bootstrap.Config{
		App:            kingpin.CommandLine,
		Name:           "stock_exporter",
		Description:    "Prometheus exporter for stock market data (Zerodha Kite Connect)",
		DefaultAddress: ":9101",
		MetricsHandlerFactory: func(b *bootstrap.Bootstrap) (http.Handler, error) {
			if *disableDefaultCollectors {
				collector.DisableDefaultCollectors()
			}
			if u, err := user.Current(); err == nil && u.Uid == "0" {
				b.Logger.Warn("Stock Exporter is running as root user. This exporter is designed to run as unprivileged user, root is not required.")
			}
			runtime.GOMAXPROCS(*maxProcs)
			b.Logger.Debug("Go MAXPROCS", "procs", runtime.GOMAXPROCS(0))
			// The process lifetime bounds background feeds; there is no graceful shutdown hook.
			deps, err := start(context.Background(), o, b)
			if err != nil {
				return nil, err
			}
			b.Handle("/debug/pprof/", http.DefaultServeMux)
			b.HandleFunc("/-/healthy", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "OK") })
			return newHandler(deps, !b.DisableExporterMetrics, b.MaxRequests, b.Logger)
		},
	})

	if err := runner.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
