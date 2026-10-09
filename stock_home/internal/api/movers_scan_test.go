package api

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/seenimo/stock_home/internal/source"
)

func f(v float64) *float64 { return &v }

func TestBuildScan(t *testing.T) {
	rows := []moverRow{
		{exchange: "NSE", symbol: "SBIN", sector: "Banks", price: f(621.35), change: f(18.75), pct: f(3.11), volume: f(25e6), relVol: f(2.5), hi: f(650), low: f(500)},
		{exchange: "NSE", symbol: "HDFCBANK", sector: "Banks", price: f(1672.1), pct: f(-0.5), volume: f(8e6), hi: f(1700), low: f(1400)},
		{exchange: "NSE", symbol: "INFY", sector: "IT", price: f(1487.3), pct: f(-6), volume: f(9e6), relVol: f(3), hi: f(2000), low: f(1480)},
		{exchange: "NSE", symbol: "TCS", sector: "IT", price: f(3412.2), pct: f(0)},
		{exchange: "NSE", symbol: "IDEA", sector: "Telecom", price: f(7.25), pct: f(5)},
	}
	sc := buildScan(rows, "core", "", "all", priceBands["all"])
	if sc.Total != 5 || sc.Advancers != 2 || sc.Decliners != 2 || sc.Unchanged != 1 {
		t.Fatalf("counts = %+v", sc)
	}
	if len(sc.Gainers) != 2 || sc.Gainers[0].Symbol != "IDEA" || len(sc.Losers) != 2 || sc.Losers[0].Symbol != "INFY" {
		t.Errorf("gainers %+v losers %+v", sc.Gainers, sc.Losers)
	}
	if sc.MostActive[0].Symbol != "SBIN" || len(sc.MostActive) != 3 || sc.VolumeChange[0].Symbol != "INFY" || sc.VolumeChange[0].RelativeVolume != 3 {
		t.Errorf("active %+v volume %+v", sc.MostActive, sc.VolumeChange)
	}
	if sc.High52[0].Symbol != "HDFCBANK" || math.Abs(sc.High52[0].DistancePercent-(1672.1/1700-1)*100) > 1e-9 {
		t.Errorf("52w high %+v", sc.High52)
	}
	if sc.Low52[0].Symbol != "INFY" || sc.Low52[0].Position < 0 || sc.Low52[0].Position > 0.02 {
		t.Errorf("52w low %+v", sc.Low52)
	}
	counts := map[string]int{}
	for _, b := range sc.Distribution {
		counts[b.Label] = b.Count
	}
	if len(sc.Distribution) != 13 || counts["< -5%"] != 1 || counts["-1% to 0%"] != 1 || counts["0%"] != 1 || counts["3% to 4%"] != 1 || counts["≥ 5%"] != 1 {
		t.Errorf("distribution %+v", sc.Distribution)
	}
	if sc.Sectors[0].Name != "Telecom" || sc.Sectors[len(sc.Sectors)-1].Name != "IT" {
		t.Errorf("sectors %+v", sc.Sectors)
	}
	for _, s := range sc.Sectors {
		if s.Name == "Banks" && (s.Gainer.Symbol != "SBIN" || s.Loser.Symbol != "HDFCBANK" || s.Count != 2 || s.Advancers != 1 || s.Decliners != 1) {
			t.Errorf("banks %+v", s)
		}
		if s.Name == "IT" && s.Gainer != nil {
			t.Errorf("IT has no gainer: %+v", s)
		}
	}
	if strings.Join(sc.SectorNames, ",") != "Banks,IT,Telecom" {
		t.Errorf("sector names %v", sc.SectorNames)
	}

	filtered := buildScan(rows, "core", "Banks", "500-2000", priceBands["500-2000"])
	if filtered.Total != 2 || filtered.Gainers[0].Symbol != "SBIN" || len(filtered.SectorNames) != 3 {
		t.Errorf("filtered %+v", filtered)
	}
	if buildScan(rows, "core", "Banks", "gt2000", priceBands["gt2000"]).Total != 0 {
		t.Error("price band not applied")
	}
}

func TestMoversScanHandler(t *testing.T) {
	m := func(sym, sector, v string) string {
		return fmt.Sprintf(`{"metric":{"exchange":"NSE","symbol":%q,"sector":%q},"value":[1,%q]}`, sym, sector, v)
	}
	prom, queries := fakePromSeq(t, []answer{
		{"stock_previous_close_price", `[` + m("SBIN", "", "18.75") + `]`},
		{"stock_change_percent", `[` + m("SBIN", "Banks", "3.11") + `,` + m("INFY", "IT", "-1.2") + `]`},
		{"stock_last_price", `[` + m("SBIN", "", "621.35") + `,` + m("INFY", "", "1487.3") + `]`},
	}, []answer{
		{`symbol=~"SBIN|INFY"`, matrix(`{"symbol":"SBIN"}`, source.Point{T: 1, V: 600}, source.Point{T: 2, V: 621.35})},
	})
	s := newTestServer(t, prom.URL, prom.URL)
	var env struct {
		Data MoversScan `json:"data"`
	}
	if code := get(t, s, "/api/v1/movers/scan?universe=all", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	sc := env.Data
	if sc.Total != 2 || sc.Gainers[0].Symbol != "SBIN" || *sc.Gainers[0].Change != 18.75 || len(sc.Gainers[0].Spark) != 2 {
		t.Errorf("scan = %+v", sc)
	}
	for _, q := range queries() {
		if strings.Contains(q, "stock_change_percent{") && strings.Contains(q, "tier=") {
			t.Errorf("universe=all must not filter on tier: %s", q)
		}
	}
	for _, bad := range []string{"universe=nifty", "price=cheap", "sector=" + strings.Repeat("x", 50), `sector=a%22%7D`} {
		if code := get(t, s, "/api/v1/movers/scan?"+bad, nil); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", bad, code)
		}
	}
}
