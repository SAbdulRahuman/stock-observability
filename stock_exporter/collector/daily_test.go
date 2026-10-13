package collector

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/seenimo/stock_exporter/internal/daily"
)

func TestDailyCollector(t *testing.T) {
	deps := testDeps(t)
	nan := math.NaN()
	ind := daily.Indicators{
		SessionDate: time.Date(2026, 10, 5, 0, 0, 0, 0, ist), History: 248, Valid: true,
		Close: 1500, SMA20: 1480, SMA50: 1450, Return1D: 1.5, Return5D: 3.2, Return20D: nan, Return126x21: nan,
		RSI14: 64, RelVolume20: nan, Vol5: nan, Vol10: nan, Vol20: nan, Vol60: nan, EWMAVol: nan,
		MACDHist: nan, ADX: nan, DIPlus: nan, DIMinus: nan, Slope20: nan, R2x20: nan, ZScore20: nan, Squeeze: nan, OBVSlope20: nan,
		StreakBullish: 3,
		Events:        []daily.Event{{Signal: daily.SignalSMACross, Bullish: true, Age: 2}},
	}
	deps.Store.UpdateDaily(1, ind)
	deps.Store.UpdateDaily(3, ind)
	stale := ind
	stale.SessionDate, stale.Valid = time.Date(2026, 10, 2, 0, 0, 0, 0, ist), false
	deps.Store.UpdateDaily(2, stale)

	want := `
# HELP stock_daily_return_5d_percent Return over the last 5 completed sessions, in percent.
# TYPE stock_daily_return_5d_percent gauge
stock_daily_return_5d_percent{` + labels + `} 3.2
# HELP stock_daily_session_lag_sessions Trading sessions between the expected latest completed session and the loaded one.
# TYPE stock_daily_session_lag_sessions gauge
stock_daily_session_lag_sessions{` + labels + `} 0
stock_daily_session_lag_sessions{exchange="NSE",instrument_type="INDEX",sector="unknown",symbol="NIFTY 50",tier="core"} 1
# HELP stock_daily_signal_age_sessions Sessions since the latest event of the signal (0 = latest session).
# TYPE stock_daily_signal_age_sessions gauge
stock_daily_signal_age_sessions{direction="bullish",exchange="NSE",instrument_type="EQ",sector="IT",signal="sma_20d_50d_cross",symbol="INFY",tier="core"} 2
# HELP stock_daily_trend_streak_sessions Consecutive sessions with close, SMA20 and SMA50 aligned in the direction.
# TYPE stock_daily_trend_streak_sessions gauge
stock_daily_trend_streak_sessions{direction="bullish",` + labels + `} 3
# HELP stock_daily_inputs_valid 1 when the daily candles passed validation, else 0.
# TYPE stock_daily_inputs_valid gauge
stock_daily_inputs_valid{` + labels + `} 1
stock_daily_inputs_valid{exchange="NSE",instrument_type="INDEX",sector="unknown",symbol="NIFTY 50",tier="core"} 0
`
	c := mustCollector(t, NewDailyCollector, deps)
	core := Scope{Tiers: map[string]bool{"core": true}}
	if err := testutil.CollectAndCompare(adapter{c, core}, strings.NewReader(want),
		"stock_daily_return_5d_percent", "stock_daily_session_lag_sessions", "stock_daily_signal_age_sessions",
		"stock_daily_trend_streak_sessions", "stock_daily_inputs_valid"); err != nil {
		t.Fatal(err)
	}
	if n := testutil.CollectAndCount(adapter{c, core}, "stock_daily_return_20d_percent"); n != 0 {
		t.Fatalf("unavailable indicator exported %d times", n)
	}
	dc := c.(*dailyCollector)
	if mon, fri := dc.index(ind.SessionDate), dc.index(stale.SessionDate); mon-fri != 1 {
		t.Fatalf("session index Monday-Friday = %v, want 1", mon-fri)
	}
}
