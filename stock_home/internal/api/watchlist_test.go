package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/seenimo/stock_home/internal/source"
)

func TestInstrument(t *testing.T) {
	m := func(name, v string) string {
		return `{"metric":{"__name__":"` + name + `","exchange":"NSE","sector":"Energy"},"value":[1,"` + v + `"]}`
	}
	prom, queries := fakePromSeq(t, []answer{
		{"stock:session_volume:shares", vec("14200000", nil)},
		{`__name__=~"stock_(last|open`, `[` + strings.Join([]string{
			m("stock_last_price", "2842.5"), m("stock_open_price", "2810"), m("stock_high_price", "2856"),
			m("stock_low_price", "2798.1"), m("stock_previous_close_price", "2805.7"),
			m("stock_change_percent", "1.31"), m("stock_day_volume_shares", "12400000"),
		}, ",") + `]`},
	}, nil)
	s := newTestServer(t, prom.URL, prom.URL)
	var env struct {
		Data Instrument `json:"data"`
	}
	if code := get(t, s, "/api/v1/instrument?symbol=reliance", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	in := env.Data
	if in.Symbol != "RELIANCE" || in.Exchange != "NSE" || in.Sector != "Energy" || *in.Price != 2842.5 || *in.High != 2856 || *in.Low != 2798.1 {
		t.Fatalf("instrument = %+v", in)
	}
	if in.Change == nil || *in.Change < 36.79 || *in.Change > 36.81 || *in.Volume != 12400000 || *in.AvgVolume20d != 14200000 {
		t.Errorf("change/volume = %v %v %v", in.Change, in.Volume, in.AvgVolume20d)
	}
	if !strings.HasPrefix(in.ChartURL, "/grafana/explore?") || !strings.Contains(strings.Join(queries(), "\n"), `symbol="RELIANCE", instrument_type="EQ"`) {
		t.Errorf("chart url %q, queries %v", in.ChartURL, queries())
	}

	for _, bad := range []string{"", "RELIANCE%22%7D"} {
		if code := get(t, s, "/api/v1/instrument?symbol="+bad, nil); code != http.StatusBadRequest {
			t.Errorf("%q: status %d, want 400", bad, code)
		}
	}
	empty, _ := fakePromSeq(t, nil, nil)
	if code := get(t, newTestServer(t, empty.URL, empty.URL), "/api/v1/instrument?symbol=TCS", nil); code != http.StatusNotFound {
		t.Errorf("unknown symbol: status %d, want 404", code)
	}
}

func TestEquityChart(t *testing.T) {
	prom, queries := fakePromSeq(t, nil, []answer{
		{`stock_last_price{symbol="TCS", instrument_type="EQ"}`, matrix(`{}`,
			source.Point{T: ist(t, "2026-10-08 09:20"), V: 3420}, source.Point{T: ist(t, "2026-10-08 09:30"), V: 3412.2})},
		{`stock_day_volume_shares{symbol="TCS", instrument_type="EQ"}[7d]`, matrix(`{}`, source.Point{T: ist(t, "2026-10-08 09:20"), V: 5})},
	})
	s := newTestServer(t, prom.URL, prom.URL)
	var env struct {
		Data Chart `json:"data"`
	}
	if code := get(t, s, "/api/v1/chart?symbol=TCS&range=5Y", &env); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if len(env.Data.Points) != 2 || len(env.Data.Volume) != 1 || env.Data.VolumeLabel != equityVolumeLabel {
		t.Errorf("chart = %+v, queries %v", env.Data, queries())
	}
}
