package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandlerRoutes(t *testing.T) {
	root := fstest.MapFS{
		"index.html":              {Data: []byte("home")},
		"404.html":                {Data: []byte("missing")},
		"openbb.html":             {Data: []byte("openbb page")},
		"openbb/__next._tree.txt": {Data: []byte("rsc")},
		"_next/static/app.js":     {Data: []byte("js")},
		"docs/index.html":         {Data: []byte("docs index")},
	}
	h := handler(root)
	for _, tc := range []struct {
		path, body string
		code       int
	}{
		{"/", "home", 200},
		{"/openbb", "openbb page", 200},
		{"/openbb/", "openbb page", 200},
		{"/openbb/__next._tree.txt", "rsc", 200},
		{"/docs/", "docs index", 200},
		{"/_next/static/app.js", "js", 200},
		{"/nope", "missing", 404},
		{"/_next/static/", "missing", 404},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
			t.Errorf("%s: %d %q, want %d %q", tc.path, rec.Code, rec.Body, tc.code, tc.body)
		}
	}
}
