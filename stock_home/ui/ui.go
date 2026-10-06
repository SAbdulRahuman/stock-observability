// Package ui serves the statically exported Next.js app.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// dist is filled from web/out by `make web`; it only holds .gitkeep in a fresh checkout.
//
//go:embed all:dist
var dist embed.FS

// Handler serves the UI. Hashed assets are cached for a year, HTML is always revalidated.
func Handler() http.Handler {
	root, _ := fs.Sub(dist, "dist")
	files := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		if _, err := fs.Stat(root, "index.html"); err != nil {
			http.Error(w, "UI not built: run `make web`.", http.StatusServiceUnavailable)
			return
		}
		if strings.HasPrefix(p, "/_next/static/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		if p != "/" {
			if _, err := fs.Stat(root, strings.TrimPrefix(p, "/")); err != nil {
				if _, err := fs.Stat(root, strings.TrimPrefix(p, "/")+".html"); err == nil {
					r.URL.Path = p + ".html"
				} else {
					w.Header().Set("Cache-Control", "no-cache")
					serve404(w, root)
					return
				}
			}
		}
		files.ServeHTTP(w, r)
	})
}

func serve404(w http.ResponseWriter, root fs.FS) {
	b, err := fs.ReadFile(root, "404.html")
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(b)
}
