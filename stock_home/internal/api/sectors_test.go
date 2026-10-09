package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/seenimo/stock_home/internal/source"
)

func TestSectorIndices(t *testing.T) {
	m := func(sym, v string) string {
		return `{"metric":{"exchange":"NSE","symbol":"` + sym + `"},"value":[1,"` + v + `"]}`
	}
	prom, queries := fakePromSeq(t, []answer{
		{"stock_previous_close_price", `[` + m("NIFTY IT", "-120.35") + `]`},
		{"stock_change_percent", `[` + m("NIFTY IT", "-0.32") + `,` + m("NIFTY BANK", "0.77") + `]`},
		{"stock_last_price", `[` + m("NIFTY IT", "37124.2") + `,` + m("NIFTY BANK", "54321.1") + `,` + m("NIFTY 50", "24892.3") + `]`},
	}, []answer{
		{`symbol=~"NIFTY IT|NIFTY BANK"`, matrix(`{"symbol":"NIFTY IT"}`, source.Point{T: 1, V: 37200}, source.Point{T: 2, V: 37124.2})},
	})
	s := newTestServer(t, prom.URL, prom.URL)
	s.cfg.SectorIndices = []string{"NIFTY IT", "NIFTY AUTO", "NIFTY BANK"}

	var env struct {
		Data SectorIndices `json:"data"`
	}
	if code := get(t, s, "/api/v1/sectors", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	ix := env.Data.Indices
	if len(ix) != 2 || ix[0].Symbol != "NIFTY IT" || ix[1].Symbol != "NIFTY BANK" {
		t.Fatalf("indices = %+v (configured order, missing ones skipped)", ix)
	}
	if *ix[0].Change != -120.35 || *ix[0].ChangePercent != -0.32 || len(ix[0].Spark) != 2 || ix[1].Change != nil {
		t.Errorf("NIFTY IT = %+v, NIFTY BANK = %+v", ix[0], ix[1])
	}

	if code := get(t, s, "/api/v1/chart?symbol=NIFTY%20AUTO&range=1W", nil); code != http.StatusOK {
		t.Errorf("sector index chart: status %d", code)
	}
	found := false
	for _, q := range queries() {
		found = found || strings.Contains(q, `symbol="NIFTY AUTO", instrument_type="INDEX"`)
	}
	if !found {
		t.Error("sector index chart should query the INDEX series")
	}
}
