package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestFilterStocks(t *testing.T) {
	rows := []moverRow{
		{exchange: "NSE", symbol: "TCS", sector: "IT", price: f(3412.2), pct: f(-0.36), volume: f(2e6)},
		{exchange: "NSE", symbol: "INFY", sector: "IT", price: f(1487.3), pct: f(1.29)},
		{exchange: "NSE", symbol: "INFIBEAM", sector: "IT", price: f(31.4), pct: f(3.5), volume: f(9e6)},
		{exchange: "NSE", symbol: "SBIN", sector: "Banks", price: f(621.35), pct: f(0), volume: f(5e6)},
	}
	all := changeFilters["all"]
	names := func(st Stocks) string {
		var out []string
		for _, q := range st.Items {
			out = append(out, q.Symbol)
		}
		return strings.Join(out, ",")
	}
	if got := names(filterStocks(rows, "", "", priceBands["all"], false, all, nil, false, 1, 10)); got != "INFIBEAM,INFY,SBIN,TCS" {
		t.Errorf("symbol order = %s", got)
	}
	if got := names(filterStocks(rows, "INF", "", priceBands["all"], false, all, nil, false, 1, 10)); got != "INFIBEAM,INFY" {
		t.Errorf("search = %s", got)
	}
	if got := names(filterStocks(rows, "INFY", "", priceBands["all"], false, all, nil, false, 1, 10)); got != "INFY" {
		t.Errorf("exact search = %s", got)
	}
	if got := names(filterStocks(rows, "", "", priceBands["all"], false, all, stockSorts["volume"], true, 1, 10)); got != "INFIBEAM,SBIN,TCS,INFY" {
		t.Errorf("volume desc (missing last) = %s", got)
	}
	if got := names(filterStocks(rows, "", "", priceBands["all"], false, all, stockSorts["volume"], false, 1, 10)); got != "TCS,SBIN,INFIBEAM,INFY" {
		t.Errorf("volume asc (missing last) = %s", got)
	}
	st := filterStocks(rows, "", "IT", priceBands["100-500"], true, all, nil, false, 1, 10)
	if st.Total != 0 || len(st.SectorNames) != 2 {
		t.Errorf("sector+price = %+v", st)
	}
	if got := names(filterStocks(rows, "", "", priceBands["all"], false, changeFilters["gt2"], nil, false, 1, 10)); got != "INFIBEAM" {
		t.Errorf("change >= 2%% = %s", got)
	}
	p2 := filterStocks(rows, "", "", priceBands["all"], false, all, nil, false, 2, 10)
	if p2.Total != 4 || len(p2.Items) != 0 {
		t.Errorf("page past the end = %+v", p2)
	}
}

func TestStocksHandler(t *testing.T) {
	m := `{"metric":{"exchange":"NSE","symbol":"SBIN","sector":"Banks"},"value":[1,"3.11"]}`
	prom, queries := fakePromSeq(t, []answer{{"stock_change_percent", `[` + m + `]`}, {"stock_last_price", `[` + strings.Replace(m, "3.11", "621.35", 1) + `]`}}, nil)
	s := newTestServer(t, prom.URL, prom.URL)
	var env struct {
		Data Stocks `json:"data"`
	}
	if code := get(t, s, "/api/v1/stocks?q=sb&sort=change&dir=desc&per=10", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if env.Data.Total != 1 || env.Data.Items[0].Symbol != "SBIN" || *env.Data.Items[0].Price != 621.35 || env.Data.Per != 10 {
		t.Errorf("stocks = %+v", env.Data)
	}
	for _, q := range queries() {
		if strings.Contains(q, `tier="core"`) {
			t.Errorf("explorer must cover all equities: %s", q)
		}
	}
	for _, bad := range []string{"q=a%22b", "sort=name", "dir=up", "page=0", "per=7", "change=big", "price=x", "sector=a%7Bb"} {
		if code := get(t, s, "/api/v1/stocks?"+bad, nil); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", bad, code)
		}
	}
}
