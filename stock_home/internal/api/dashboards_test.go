package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seenimo/stock_home/internal/source"
)

func TestDashboards(t *testing.T) {
	var auth string
	graf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" || r.URL.Query().Get("type") != "dash-db" {
			http.NotFound(w, r)
			return
		}
		auth = r.Header.Get("Authorization")
		io.WriteString(w, `[
 {"uid":"nse-r2","title":"NSE Market Intelligence (R2)","url":"/grafana/d/nse-r2/x","tags":["nse","kpi"],"folderTitle":"NSE"},
 {"uid":"nse-r1","title":"NSE Market Observability (R1)","url":"javascript:alert(1)"},
 {"uid":"../../admin","title":"bad uid"}
]`)
	}))
	defer graf.Close()

	s := newTestServer(t, graf.URL, graf.URL)
	var env struct {
		Data Dashboards `json:"data"`
	}
	if code := get(t, s, "/api/v1/dashboards", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	items := env.Data.Items
	if len(items) != 2 || items[0].UID != "nse-r2" || items[0].Folder != "NSE" || len(items[0].Tags) != 2 || items[1].Tags == nil {
		t.Fatalf("items = %+v", items)
	}
	for _, d := range items {
		if !strings.HasPrefix(d.Href, "/grafana/d/") || strings.Contains(d.Href, "javascript") {
			t.Errorf("href %q must be built from the uid", d.Href)
		}
	}
	if auth != "" {
		t.Errorf("anonymous by default, got Authorization %q", auth)
	}

	s2 := newTestServer(t, graf.URL, graf.URL)
	s2.graf = source.NewGrafana(graf.URL, http.DefaultClient, "secret")
	var raw struct {
		Data any `json:"data"`
	}
	get(t, s2, "/api/v1/dashboards", &raw)
	if auth != "Bearer secret" {
		t.Errorf("token not sent: %q", auth)
	}
}
