// Command stock_website serves the iSee Ticks website, its small JSON API and the read-only public demo proxy.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"

	"github.com/seenimo/stock_website/internal/demo"
	"github.com/seenimo/stock_website/internal/ratelimit"
	"github.com/seenimo/stock_website/internal/ticker"
	"github.com/seenimo/stock_website/internal/waitlist"
)

var version = "dev"

type options struct {
	listen, tlsListen, demoListen, metricsListen string
	homeURL, grafanaURL, demoPublicURL           string
	siteDomain, demoDomain, tlsCacheDir, tlsMail string
	dbPath                                       string
	tickerTTL                                    time.Duration
	simulated                                    bool
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "waitlist" {
		if err := waitlistCmd(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	var o options
	flag.StringVar(&o.listen, "web.listen-address", ":8090", "Website address. With --site.domain it only answers ACME challenges and redirects to HTTPS (use :80).")
	flag.StringVar(&o.tlsListen, "web.tls-listen-address", ":443", "HTTPS address, used when --site.domain is set.")
	flag.StringVar(&o.demoListen, "demo.listen-address", "", "Separate plain-HTTP address for the demo when there is no domain yet (e.g. :8080). Empty disables.")
	flag.StringVar(&o.metricsListen, "web.metrics-address", "127.0.0.1:9112", "Address for /metrics. Empty disables.")
	flag.StringVar(&o.homeURL, "demo.home-url", "http://127.0.0.1:8081", "stock_home base URL proxied by the demo and read by the ticker.")
	flag.StringVar(&o.grafanaURL, "demo.grafana-url", "", "Grafana base URL proxied under /grafana/ (Grafana must serve from sub path /grafana). Empty disables.")
	flag.StringVar(&o.demoPublicURL, "demo.public-url", "", "Public demo URL shown on the site. Empty derives it from --demo.domain or --demo.listen-address.")
	flag.StringVar(&o.siteDomain, "site.domain", "", "Website domain, e.g. iseeticks.com. Enables HTTPS with Let's Encrypt (autocert).")
	flag.StringVar(&o.demoDomain, "demo.domain", "", "Demo domain, e.g. demo.iseeticks.com. Requires --site.domain.")
	flag.StringVar(&o.tlsCacheDir, "tls.cache-dir", "autocert", "Directory for Let's Encrypt account keys and certificates.")
	flag.StringVar(&o.tlsMail, "tls.email", "", "Contact email for Let's Encrypt (optional).")
	flag.StringVar(&o.dbPath, "db.path", "site.db", "SQLite database for waitlist sign-ups.")
	flag.DurationVar(&o.tickerTTL, "ticker.cache", 5*time.Second, "How long a ticker snapshot is reused.")
	flag.BoolVar(&o.simulated, "demo.simulated", true, "Label demo data as simulated.")
	showVersion := flag.Bool("version", false, "Print version and exit.")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(logger, o); err != nil {
		logger.Error("Exiting", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, o options) error {
	if o.demoDomain != "" && o.siteDomain == "" {
		return errors.New("--demo.domain requires --site.domain")
	}
	homeURL, err := parseURL("demo.home-url", o.homeURL)
	if err != nil {
		return err
	}
	var grafanaURL *url.URL
	if o.grafanaURL != "" {
		if grafanaURL, err = parseURL("demo.grafana-url", o.grafanaURL); err != nil {
			return err
		}
	}
	if dir := filepath.Dir(o.dbPath); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	store, err := waitlist.Open(o.dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "stock_website_build_info", Help: "Build information.", ConstLabels: prometheus.Labels{"version": version},
	}, func() float64 { return 1 }))
	signups := prometheus.NewCounter(prometheus.CounterOpts{Name: "stock_website_waitlist_signups_total", Help: "New waitlist emails stored."})
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "stock_website_http_requests_total", Help: "HTTP requests by server and status code."}, []string{"server", "code", "method"})
	reg.MustRegister(signups, requests)
	count := func(server string, h http.Handler) http.Handler {
		return promhttp.InstrumentHandlerCounter(requests.MustCurryWith(prometheus.Labels{"server": server}), h)
	}

	site := siteHandler(siteDeps{
		ticker: &ticker.Source{
			BaseURL:   strings.TrimRight(homeURL.String(), "/"),
			Client:    &http.Client{Timeout: 3 * time.Second},
			TTL:       o.tickerTTL,
			Simulated: o.simulated,
			Logger:    logger,
		},
		waitlist: &waitlist.Handler{
			Store:  store,
			Limit:  ratelimit.New(1.0/60, 5),
			Logger: logger,
			OnAdd:  signups.Inc,
		},
		demoURL: demoURLFunc(o.demoPublicURL, o.demoDomain, o.demoListen),
		version: version,
		hsts:    o.siteDomain != "",
	})
	demoH := demo.New(demo.Config{HomeURL: homeURL, GrafanaURL: grafanaURL, Limit: ratelimit.New(30, 120), Logger: logger})

	var servers []*http.Server
	var serveTLS *http.Server
	if o.siteDomain != "" {
		hosts := []string{o.siteDomain}
		var root http.Handler = count("site", site)
		if o.demoDomain != "" {
			hosts = append(hosts, o.demoDomain)
			root = hostRouter(o.demoDomain, root, count("demo", demoH))
		}
		if err := os.MkdirAll(o.tlsCacheDir, 0o700); err != nil {
			return err
		}
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(o.tlsCacheDir),
			HostPolicy: autocert.HostWhitelist(hosts...),
			Email:      o.tlsMail,
		}
		tlsCfg := m.TLSConfig()
		tlsCfg.MinVersion = tls.VersionTLS12
		serveTLS = newServer(o.tlsListen, root, 0)
		serveTLS.TLSConfig = tlsCfg
		// nil fallback: plain HTTP requests are redirected to HTTPS.
		servers = append(servers, newServer(o.listen, m.HTTPHandler(nil), 30*time.Second))
	} else {
		servers = append(servers, newServer(o.listen, count("site", site), 30*time.Second))
	}
	if o.demoListen != "" {
		// No write timeout: Grafana Live keeps websockets open.
		servers = append(servers, newServer(o.demoListen, count("demo", demoH), 0))
	}
	if o.metricsListen != "" {
		mm := http.NewServeMux()
		mm.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		servers = append(servers, newServer(o.metricsListen, mm, 30*time.Second))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errc := make(chan error, len(servers)+1)
	for _, s := range servers {
		go func() { errc <- s.ListenAndServe() }()
	}
	if serveTLS != nil {
		go func() { errc <- serveTLS.ListenAndServeTLS("", "") }()
		servers = append(servers, serveTLS)
	}
	logger.Info("Listening", "version", version, "site", o.listen, "tls", o.siteDomain != "", "demo", o.demoListen,
		"demo_domain", o.demoDomain, "home", o.homeURL, "grafana", o.grafanaURL, "db", o.dbPath)

	var runErr error
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, s := range servers {
		_ = s.Shutdown(shutdownCtx)
	}
	return runErr
}

func newServer(addr string, h http.Handler, writeTimeout time.Duration) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
	}
}

func parseURL(flagName, raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("--%s: want an http(s) URL, got %q", flagName, raw)
	}
	return u, nil
}

func waitlistCmd(args []string) error {
	if len(args) == 0 || args[0] != "export" {
		return errors.New("usage: stock_website waitlist export [--db.path site.db] [--format csv|json]")
	}
	fs := flag.NewFlagSet("waitlist export", flag.ContinueOnError)
	db := fs.String("db.path", "site.db", "SQLite database for waitlist sign-ups.")
	format := fs.String("format", "csv", "Output format: csv or json.")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if _, err := os.Stat(*db); err != nil {
		return err
	}
	store, err := waitlist.Open(*db)
	if err != nil {
		return err
	}
	defer store.Close()
	entries, err := store.List(context.Background())
	if err != nil {
		return err
	}
	return waitlist.Export(os.Stdout, entries, *format)
}
