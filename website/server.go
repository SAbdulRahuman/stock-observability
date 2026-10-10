package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/seenimo/stock_website/ui"
)

// Next.js inlines its bootstrap scripts and styles, hence 'unsafe-inline'.
const csp = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

type siteDeps struct {
	ticker   http.Handler
	waitlist http.Handler
	demoURL  func(*http.Request) string
	version  string
	hsts     bool
}

func siteHandler(d siteDeps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "ok") })
	mux.HandleFunc("GET /api/site/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{"demoUrl": d.demoURL(r), "version": d.version})
	})
	mux.Handle("GET /api/site/ticker", d.ticker)
	mux.Handle("POST /api/site/waitlist", d.waitlist)
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, `{"error":"not found"}`)
	})
	mux.Handle("GET /", ui.Handler())
	return securityHeaders(mux, d.hsts)
}

func securityHeaders(next http.Handler, hsts bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if hsts {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// hostRouter sends the demo host to demo and everything else to site.
func hostRouter(demoHost string, site, demo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(hostname(r.Host), demoHost) {
			demo.ServeHTTP(w, r)
			return
		}
		site.ServeHTTP(w, r)
	})
}

var hostnameRE = regexp.MustCompile(`^[A-Za-z0-9.-]+$|^\[[0-9A-Fa-f:.]+\]$`)

// demoURLFunc picks the public demo link: explicit URL, demo domain, or the demo port on the host the visitor used.
func demoURLFunc(publicURL, demoDomain, demoListen string) func(*http.Request) string {
	return func(r *http.Request) string {
		switch {
		case publicURL != "":
			return publicURL
		case demoDomain != "":
			return "https://" + demoDomain + "/"
		case demoListen != "":
			_, port, err := net.SplitHostPort(demoListen)
			host := hostname(r.Host)
			if strings.Contains(host, ":") {
				host = "[" + host + "]"
			}
			if err != nil || !hostnameRE.MatchString(host) {
				return ""
			}
			return "http://" + net.JoinHostPort(strings.Trim(host, "[]"), port) + "/"
		}
		return ""
	}
}

func hostname(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return strings.Trim(hostport, "[]")
}
