// Command stock_home serves the stock observability home page and its JSON API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/seenimo/stock_home/internal/api"
	"github.com/seenimo/stock_home/internal/holidays"
	"github.com/seenimo/stock_home/internal/openbb"
	"github.com/seenimo/stock_home/internal/platform"
	"github.com/seenimo/stock_home/internal/source"
	"github.com/seenimo/stock_home/ui"
)

var version = "dev"

func main() {
	var (
		listen       = flag.String("web.listen-address", ":8080", "Address to listen on.")
		promURL      = flag.String("prometheus.url", "http://prometheus:9090/prometheus", "Prometheus base URL including route prefix.")
		amURL        = flag.String("alertmanager.url", "http://alertmanager:9093/alertmanager", "Alertmanager base URL including route prefix.")
		grafanaURL   = flag.String("grafana.url", "http://grafana:3000/grafana", "Grafana base URL including sub path.")
		indices      = flag.String("market.indices", "NIFTY 50,NIFTY BANK,NIFTY IT,SENSEX,INDIA VIX", "Comma-separated index symbols for Market overview; the first one gets a sparkline.")
		holidaysFile = flag.String("market.holidays-file", "", "Exchange holidays YAML (stock_exporter format).")
		tz           = flag.String("market.timezone", "Asia/Kolkata", "Exchange timezone.")
		timeout      = flag.Duration("query.timeout", 10*time.Second, "Timeout for one section refresh.")
		openbbURL    = flag.String("openbb.url", "", "OpenBB Platform API base URL; /api/v2 commands without a native port are forwarded there. Empty disables. Credentials: env STOCK_HOME_OPENBB_USERNAME, STOCK_HOME_OPENBB_PASSWORD.")
		openbbTO     = flag.Duration("openbb.timeout", 30*time.Second, "Timeout for one command forwarded to OpenBB.")
		showVersion  = flag.Bool("version", false, "Print version and exit.")
	)
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	ob := openbb.Config{
		URL:      *openbbURL,
		Username: os.Getenv("STOCK_HOME_OPENBB_USERNAME"),
		Password: os.Getenv("STOCK_HOME_OPENBB_PASSWORD"),
		Timeout:  *openbbTO,
	}
	if err := run(logger, *listen, *promURL, *amURL, *grafanaURL, *indices, *holidaysFile, *tz, *timeout, ob); err != nil {
		logger.Error("Exiting", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, listen, promURL, amURL, grafanaURL, indices, holidaysFile, tz string, timeout time.Duration, ob openbb.Config) error {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}
	var hol []holidays.Holiday
	if holidaysFile != "" {
		if hol, err = holidays.Load(holidaysFile, loc); err != nil {
			return err
		}
	}
	prom, err := source.NewPrometheus(trim(promURL))
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: timeout}

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "stock_home_build_info", Help: "Build information.", ConstLabels: prometheus.Labels{"version": version},
	}, func() float64 { return 1 }))

	srv := api.New(api.Config{
		PrometheusURL:   trim(promURL),
		AlertmanagerURL: trim(amURL),
		GrafanaURL:      trim(grafanaURL),
		Indices:         splitList(indices),
		Location:        loc,
		Holidays:        hol,
		QueryTimeout:    timeout,
	}, prom, source.NewAlertmanager(trim(amURL), client), client, logger, reg)

	mux := http.NewServeMux()
	srv.Register(mux)
	plat := platform.NewServer(platform.NewRegistry(), logger, timeout, reg)
	writeTimeout := 2 * timeout
	if ob.URL != "" {
		proxy, err := openbb.New(ob, logger, reg)
		if err != nil {
			return err
		}
		plat.SetFallback(proxy)
		writeTimeout = max(writeTimeout, ob.Timeout+5*time.Second)
	}
	plat.Register(mux)
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "ok") })
	mux.Handle("GET /", ui.Handler())

	hs := &http.Server{
		Addr:              listen,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       2 * time.Minute,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- hs.ListenAndServe() }()
	logger.Info("Listening", "address", listen, "version", version, "prometheus", promURL, "alertmanager", amURL, "openbb", ob.URL, "holidays", len(hol))

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return hs.Shutdown(shutdownCtx)
	}
}

// Next.js inlines its bootstrap scripts and styles, hence 'unsafe-inline'.
const csp = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func trim(u string) string { return strings.TrimRight(u, "/") }

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
