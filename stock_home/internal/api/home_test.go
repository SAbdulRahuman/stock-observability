package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seenimo/stock_home/internal/source"
)

type answer struct{ match, result string }

// fakePromSeq answers instant and range queries with the first answer whose match is a substring of the query.
func fakePromSeq(t *testing.T, instant, ranges []answer) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		q := r.Form.Get("query")
		mu.Lock()
		seen = append(seen, q)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		kind, list := "vector", instant
		if strings.HasSuffix(r.URL.Path, "/query_range") {
			kind, list = "matrix", ranges
		}
		result := "[]"
		for _, a := range list {
			if strings.Contains(q, a.match) {
				result = a.result
				break
			}
		}
		io.WriteString(w, `{"status":"success","data":{"resultType":"`+kind+`","result":`+result+`}}`)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func matrix(labels string, pts ...source.Point) string {
	vals := make([]string, len(pts))
	for i, p := range pts {
		vals[i] = fmt.Sprintf(`[%d.%03d,"%g"]`, p.T/1000, p.T%1000, p.V)
	}
	return `[{"metric":` + labels + `,"values":[` + strings.Join(vals, ",") + `]}]`
}

func ist(t *testing.T, s string) int64 {
	t.Helper()
	loc, _ := time.LoadLocation("Asia/Kolkata")
	ts, err := time.ParseInLocation("2006-01-02 15:04", s, loc)
	if err != nil {
		t.Fatal(err)
	}
	return ts.UnixMilli()
}

func TestChart(t *testing.T) {
	prom, queries := fakePromSeq(t, nil, []answer{
		{`stock_last_price{symbol="NIFTY 50"`, matrix(`{}`,
			source.Point{T: ist(t, "2026-10-07 15:00"), V: 24700},
			source.Point{T: ist(t, "2026-10-08 09:20"), V: 24800},
			source.Point{T: ist(t, "2026-10-08 09:22"), V: 24892.3})},
		{`stock_day_volume_shares`, matrix(`{}`,
			source.Point{T: ist(t, "2026-10-07 15:00"), V: 9},
			source.Point{T: ist(t, "2026-10-08 09:20"), V: 1200},
			source.Point{T: ist(t, "2026-10-08 09:22"), V: 1500})},
	})
	s := newTestServer(t, prom.URL, prom.URL)

	var env struct {
		Data Chart `json:"data"`
	}
	if code := get(t, s, "/api/v1/chart?symbol=nifty%2050&range=1d", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	c := env.Data
	if c.Symbol != "NIFTY 50" || c.Range != "1D" || len(c.Points) != 2 || len(c.Volume) != 2 {
		t.Fatalf("chart = %+v (1D keeps the latest session only)", c)
	}
	if c.Last == nil || *c.Last != 24892.3 || c.VolumeLabel == "" {
		t.Errorf("last = %v, label %q", c.Last, c.VolumeLabel)
	}
	for _, q := range queries() {
		if !strings.Contains(q, "stock_market_open") {
			t.Errorf("1D query not limited to the open session: %s", q)
		}
	}
	if !strings.Contains(strings.Join(queries(), "\n"), "[2m]") {
		t.Errorf("volume window should match the 1D step: %v", queries())
	}

	for _, bad := range []string{"symbol=NIFTY%2050&range=5Y", "symbol=RELIANCE&range=1D", `symbol=NIFTY%2050%22%7D&range=1D`} {
		if code := get(t, s, "/api/v1/chart?"+bad, nil); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", bad, code)
		}
	}
}

func TestChartEmpty(t *testing.T) {
	prom, _ := fakePromSeq(t, nil, nil)
	var env struct {
		Data Chart `json:"data"`
	}
	if code := get(t, newTestServer(t, prom.URL, prom.URL), "/api/v1/chart?symbol=NIFTY%2050&range=1Y", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if env.Data.Points == nil || len(env.Data.Points) != 0 || env.Data.Last != nil {
		t.Errorf("empty chart = %+v", env.Data)
	}
}

func TestQuotes(t *testing.T) {
	tcs := `{"exchange":"NSE","symbol":"TCS"}`
	prom, queries := fakePromSeq(t, []answer{
		{"stock_previous_close_price", `[{"metric":` + tcs + `,"value":[1,"-12.35"]}]`},
		{"stock_change_percent", `[{"metric":` + tcs + `,"value":[1,"-0.36"]}]`},
		{"stock_last_price", `[{"metric":` + tcs + `,"value":[1,"3412.2"]}]`},
	}, []answer{
		{"stock_last_price", matrix(`{"symbol":"TCS"}`, source.Point{T: ist(t, "2026-10-08 09:20"), V: 3420}, source.Point{T: ist(t, "2026-10-08 09:25"), V: 3412.2})},
	})
	s := newTestServer(t, prom.URL, prom.URL)

	var env struct {
		Data Quotes `json:"data"`
	}
	if code := get(t, s, "/api/v1/quotes?symbols=tcs,M%26M,TCS", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	items := env.Data.Items
	if len(items) != 2 || items[0].Symbol != "TCS" || items[1].Symbol != "M&M" {
		t.Fatalf("items = %+v (request order, deduplicated)", items)
	}
	if items[0].Price == nil || *items[0].Price != 3412.2 || *items[0].Change != -12.35 || *items[0].ChangePercent != -0.36 || len(items[0].Spark) != 2 {
		t.Errorf("TCS = %+v", items[0])
	}
	if items[1].Price != nil || items[1].Spark != nil {
		t.Errorf("unknown symbol must have no values: %+v", items[1])
	}
	if !strings.Contains(strings.Join(queries(), "\n"), `symbol=~"TCS|M&M"`) {
		t.Errorf("queries = %v", queries())
	}

	many := make([]string, maxQuoteSymbols+1)
	for i := range many {
		many[i] = fmt.Sprintf("S%d", i)
	}
	for _, bad := range []string{"", "symbols=", `symbols=TCS%22%7D`, "symbols=" + strings.Join(many, ",")} {
		if code := get(t, s, "/api/v1/quotes?"+bad, nil); code != http.StatusBadRequest {
			t.Errorf("%q: status %d, want 400", bad, code)
		}
	}
}

func TestMoversHighVolumeAndChange(t *testing.T) {
	sbin := `{"exchange":"NSE","symbol":"SBIN","sector":"Banks"}`
	infy := `{"exchange":"NSE","symbol":"INFY","sector":"IT"}`
	prom, _ := fakePromSeq(t, []answer{
		{"stock_previous_close_price", `[{"metric":` + sbin + `,"value":[1,"18.75"]},{"metric":` + infy + `,"value":[1,"-5"]}]`},
		{"topk(5, max by (exchange, symbol, sector) (stock_change", `[{"metric":` + sbin + `,"value":[1,"3.11"]}]`},
		{"bottomk(", `[{"metric":` + infy + `,"value":[1,"-0.5"]}]`},
		{"stock_day_volume_shares", `[{"metric":` + infy + `,"value":[1,"1000"]},{"metric":` + sbin + `,"value":[1,"5000"]}]`},
		{"max by (exchange, symbol) (stock_change_percent", `[{"metric":` + sbin + `,"value":[1,"3.11"]},{"metric":` + infy + `,"value":[1,"-0.5"]}]`},
		{`core"})`, `[{"metric":` + sbin + `,"value":[1,"621.35"]},{"metric":` + infy + `,"value":[1,"1487.3"]}]`},
	}, []answer{
		{`symbol=~"SBIN|INFY", instrument_type="EQ", tier="core"`, matrix(`{"symbol":"SBIN"}`, source.Point{T: ist(t, "2026-10-08 09:20"), V: 600}, source.Point{T: ist(t, "2026-10-08 09:25"), V: 621.35})},
	})
	var env struct {
		Data Movers `json:"data"`
	}
	if code := get(t, newTestServer(t, prom.URL, prom.URL), "/api/v1/movers", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	m := env.Data
	if len(m.Gainers) != 1 || *m.Gainers[0].Change != 18.75 || *m.Gainers[0].Price != 621.35 || len(m.Gainers[0].Spark) != 2 {
		t.Errorf("gainers = %+v", m.Gainers)
	}
	if len(m.Losers) != 1 || m.Losers[0].Spark != nil {
		t.Errorf("losers = %+v", m.Losers)
	}
	if len(m.HighVolume) != 2 || m.HighVolume[0].Symbol != "SBIN" || *m.HighVolume[0].Volume != 5000 || *m.HighVolume[1].ChangePercent != -0.5 || *m.HighVolume[1].Change != -5 {
		t.Errorf("high volume = %+v", m.HighVolume)
	}
}

func TestMarketIndexChangeAndSparks(t *testing.T) {
	nifty := `{"exchange":"NSE","symbol":"NIFTY 50"}`
	prom, _ := fakePromSeq(t, []answer{
		{"stock_previous_close_price", `[{"metric":` + nifty + `,"value":[1,"156.7"]}]`},
		{`stock_change_percent{instrument_type="INDEX"}`, `[{"metric":` + nifty + `,"value":[1,"0.63"]}]`},
		{`(stock_last_price{instrument_type="INDEX"})`, `[{"metric":` + nifty + `,"value":[1,"24892.3"]}]`},
	}, []answer{
		{`symbol=~"NIFTY 50", instrument_type="INDEX"`, matrix(`{"symbol":"NIFTY 50"}`, source.Point{T: ist(t, "2026-10-08 09:20"), V: 24800}, source.Point{T: ist(t, "2026-10-08 09:25"), V: 24892.3})},
	})
	var env struct {
		Data Market `json:"data"`
	}
	if code := get(t, newTestServer(t, prom.URL, prom.URL), "/api/v1/market", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if len(env.Data.Indices) != 1 {
		t.Fatalf("indices = %+v", env.Data.Indices)
	}
	n := env.Data.Indices[0]
	if *n.Price != 24892.3 || *n.Change != 156.7 || *n.ChangePercent != 0.63 || len(n.Spark) != 2 {
		t.Errorf("NIFTY 50 = %+v", n)
	}
}

func TestMarketBreadthAndSectors(t *testing.T) {
	prom, _ := fakePromSeq(t, []answer{
		{"market:advancers:count", vec("1428", nil)},
		{"market:decliners:count", vec("876", nil)},
		{`count(stock_change_percent{instrument_type="EQ"} == 0)`, vec("57", nil)},
		{"count by (sector)", `[{"metric":{"sector":"Banks"},"value":[1,"12"]}]`},
		{"sector:change_percent:avg", `[{"metric":{"sector":"IT"},"value":[1,"-0.64"]},{"metric":{"sector":"Banks"},"value":[1,"1.98"]}]`},
	}, nil)
	var env struct {
		Data Market `json:"data"`
	}
	if code := get(t, newTestServer(t, prom.URL, prom.URL), "/api/v1/market", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	m := env.Data
	if *m.Advancers != 1428 || *m.Decliners != 876 || m.Unchanged == nil || *m.Unchanged != 57 {
		t.Errorf("breadth = %v %v %v", m.Advancers, m.Decliners, m.Unchanged)
	}
	want := []Sector{{Name: "Banks", ChangePercent: 1.98, Constituents: 12}, {Name: "IT", ChangePercent: -0.64}}
	if len(m.Sectors) != 2 || m.Sectors[0] != want[0] || m.Sectors[1] != want[1] {
		t.Errorf("sectors = %+v", m.Sectors)
	}
}

func TestLastDayPoints(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Kolkata")
	pts := []source.Point{{T: ist(t, "2026-10-07 15:29")}, {T: ist(t, "2026-10-08 09:15")}, {T: ist(t, "2026-10-08 09:17")}}
	if got := lastDayPoints(pts, loc); len(got) != 2 || got[0].T != pts[1].T {
		t.Errorf("lastDayPoints = %+v", got)
	}
	if got := lastDayPoints(nil, loc); len(got) != 0 {
		t.Errorf("empty input = %+v", got)
	}
	if promDuration(2*time.Minute) != "2m" || promDuration(3*time.Hour) != "3h" || promDuration(24*time.Hour) != "1d" {
		t.Error("promDuration")
	}
}
