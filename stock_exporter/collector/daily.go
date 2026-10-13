//go:build !nodaily

package collector

import (
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// sessionIndexOrigin is the first session counted by stock_daily_session_index.
var sessionIndexOrigin = time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)

var dailySessionDelay = kingpin.Flag("collector.daily.session-delay",
	"Time after a session's 15:30 close by which its daily candle is expected (stock_daily_session_lag_sessions).").Default("18h").Duration()

type dailyCollector struct {
	deps Deps

	close, ret1, ret5, ret20, ret126, sma20, rsi, relVol        typedDesc
	vol5, vol10, vol20, vol60, ewma, macd, adx, diPlus, diMinus typedDesc
	slope, r2, zscore, squeeze, obv, streak, signalAge          typedDesc
	sessionTS, sessionLag, sessionIndex, history, valid         typedDesc

	mu      sync.Mutex
	indexes map[string]float64
}

func init() {
	registerCollector("daily", defaultEnabled, NewDailyCollector)
}

// NewDailyCollector exposes completed-session indicators computed from daily candles (stock forecast PRD §8).
func NewDailyCollector(_ *slog.Logger, deps Deps) (Collector, error) {
	l := instrumentLabels
	return &dailyCollector{
		deps:         deps,
		close:        gaugeDesc("daily", "close_price", "Close of the latest completed session.", l...),
		ret1:         gaugeDesc("daily", "return_1d_percent", "Return of the latest completed session, in percent.", l...),
		ret5:         gaugeDesc("daily", "return_5d_percent", "Return over the last 5 completed sessions, in percent.", l...),
		ret20:        gaugeDesc("daily", "return_20d_percent", "Return over the last 20 completed sessions, in percent.", l...),
		ret126:       gaugeDesc("daily", "return_126_21d_percent", "6-1 momentum: return from 126 to 21 sessions ago, in percent.", l...),
		sma20:        gaugeDesc("sma", "20d_price", "Simple moving average of the last 20 daily closes.", l...),
		rsi:          gaugeDesc("rsi", "14d_index", "Wilder RSI(14) on daily closes (after 100 closes).", l...),
		relVol:       gaugeDesc("daily", "relative_volume_20d_ratio", "Latest-session volume divided by the mean volume of the previous 20 sessions.", l...),
		vol5:         gaugeDesc("realized", "vol_5d_percent", "Annualised realized volatility of the last 5 daily log returns, in percent.", l...),
		vol10:        gaugeDesc("realized", "vol_10d_percent", "Annualised realized volatility of the last 10 daily log returns, in percent.", l...),
		vol20:        gaugeDesc("realized", "vol_20d_percent", "Annualised realized volatility of the last 20 daily log returns, in percent.", l...),
		vol60:        gaugeDesc("realized", "vol_60d_percent", "Annualised realized volatility of the last 60 daily log returns, in percent.", l...),
		ewma:         gaugeDesc("daily", "ewma_vol_forecast_percent", "Annualised one-step EWMA (lambda 0.94) volatility forecast, in percent.", l...),
		macd:         gaugeDesc("daily", "macd_histogram_price", "MACD(12,26) minus its 9-session signal line on daily closes.", l...),
		adx:          gaugeDesc("adx", "14d_index", "Wilder ADX(14) on daily candles.", l...),
		diPlus:       gaugeDesc("di", "plus_14d_index", "Wilder +DI(14) on daily candles.", l...),
		diMinus:      gaugeDesc("di", "minus_14d_index", "Wilder -DI(14) on daily candles.", l...),
		slope:        gaugeDesc("daily", "trend_slope_20d_percent", "Least-squares slope of ln(close) over 20 sessions, annualised, in percent.", l...),
		r2:           gaugeDesc("daily", "trend_r2_20d_ratio", "R squared of the 20-session ln(close) regression.", l...),
		zscore:       gaugeDesc("daily", "close_zscore_20d_sigma", "Distance of the close from its 20-session average in standard deviations.", l...),
		squeeze:      gaugeDesc("daily", "squeeze_ratio", "Bollinger(20,2) width divided by Keltner(20,1.5) width; below 1 is a squeeze.", l...),
		obv:          gaugeDesc("daily", "obv_slope_20d_ratio", "20-session on-balance-volume slope in average volumes per session.", l...),
		streak:       gaugeDesc("daily", "trend_streak_sessions", "Consecutive sessions with close, SMA20 and SMA50 aligned in the direction.", append(l, "direction")...),
		signalAge:    gaugeDesc("daily", "signal_age_sessions", "Sessions since the latest event of the signal (0 = latest session).", append(l, "signal", "direction")...),
		sessionTS:    gaugeDesc("daily", "session_timestamp_seconds", "Date (00:00 exchange time) of the latest completed session candle.", l...),
		sessionLag:   gaugeDesc("daily", "session_lag_sessions", "Trading sessions between the expected latest completed session and the loaded one.", l...),
		sessionIndex: gaugeDesc("daily", "session_index", "Trading-session number of the latest candle, counted by the exchange calendar from 2000-01-03.", l...),
		history:      gaugeDesc("daily", "history_sessions", "Valid daily candles loaded.", l...),
		valid:        gaugeDesc("daily", "inputs_valid", "1 when the daily candles passed validation, else 0.", l...),
		indexes:      map[string]float64{},
	}, nil
}

func (c *dailyCollector) Update(ch chan<- prometheus.Metric, scope Scope) error {
	expected := c.expectedSession(c.deps.Now())
	for _, e := range c.deps.Store.Snapshot() {
		if !e.HasDaily || !scope.Includes(e.Instrument.Tier) {
			continue
		}
		lv := instrumentLabelValues(e.Instrument)
		d := e.Daily
		ch <- c.history.mustNewConstMetric(float64(d.History), lv...)
		if d.SessionDate.IsZero() {
			continue
		}
		ch <- c.sessionTS.mustNewConstMetric(unixSeconds(d.SessionDate), lv...)
		ch <- c.sessionLag.mustNewConstMetric(c.lag(d.SessionDate, expected), lv...)
		ch <- c.sessionIndex.mustNewConstMetric(c.index(d.SessionDate), lv...)
		if !d.Valid {
			ch <- c.valid.mustNewConstMetric(0, lv...)
			continue
		}
		ch <- c.valid.mustNewConstMetric(1, lv...)
		for _, g := range []struct {
			d *typedDesc
			v float64
		}{
			{&c.close, d.Close}, {&c.ret1, d.Return1D}, {&c.ret5, d.Return5D}, {&c.ret20, d.Return20D}, {&c.ret126, d.Return126x21},
			{&c.sma20, d.SMA20}, {&c.rsi, d.RSI14}, {&c.relVol, d.RelVolume20},
			{&c.vol5, d.Vol5}, {&c.vol10, d.Vol10}, {&c.vol20, d.Vol20}, {&c.vol60, d.Vol60}, {&c.ewma, d.EWMAVol},
			{&c.macd, d.MACDHist}, {&c.adx, d.ADX}, {&c.diPlus, d.DIPlus}, {&c.diMinus, d.DIMinus},
			{&c.slope, d.Slope20}, {&c.r2, d.R2x20}, {&c.zscore, d.ZScore20}, {&c.squeeze, d.Squeeze}, {&c.obv, d.OBVSlope20},
		} {
			if !math.IsNaN(g.v) {
				ch <- g.d.mustNewConstMetric(g.v, lv...)
			}
		}
		if d.StreakBullish > 0 {
			ch <- c.streak.mustNewConstMetric(float64(d.StreakBullish), append(lv, "bullish")...)
		}
		if d.StreakBearish > 0 {
			ch <- c.streak.mustNewConstMetric(float64(d.StreakBearish), append(lv, "bearish")...)
		}
		for _, ev := range d.Events {
			dir := "bearish"
			if ev.Bullish {
				dir = "bullish"
			}
			ch <- c.signalAge.mustNewConstMetric(float64(ev.Age), append(lv, ev.Signal, dir)...)
		}
	}
	return nil
}

func (c *dailyCollector) sessionDay(t time.Time) time.Time {
	y, m, d := t.In(c.deps.Calendar.Location()).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, c.deps.Calendar.Location())
}

// expectedSession is the latest trading day whose close plus the session delay has passed.
func (c *dailyCollector) expectedSession(now time.Time) time.Time {
	delay := *dailySessionDelay
	if delay <= 0 {
		delay = 18 * time.Hour
	}
	day := c.sessionDay(now)
	for range 60 {
		closeAt := day.Add(15*time.Hour + 30*time.Minute)
		if c.deps.Calendar.IsTradingDay(closeAt) && !closeAt.Add(delay).After(now) {
			return day
		}
		day = day.AddDate(0, 0, -1)
	}
	return time.Time{}
}

func (c *dailyCollector) lag(session, expected time.Time) float64 {
	session = c.sessionDay(session)
	n := 0
	for d := expected; d.After(session) && n < 60; d = d.AddDate(0, 0, -1) {
		if c.deps.Calendar.IsTradingDay(d.Add(12 * time.Hour)) {
			n++
		}
	}
	return float64(n)
}

// index counts trading days from sessionIndexOrigin to the session, memoised per date.
func (c *dailyCollector) index(session time.Time) float64 {
	day := c.sessionDay(session)
	key := day.Format(time.DateOnly)
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.indexes[key]; ok {
		return v
	}
	loc := c.deps.Calendar.Location()
	n := 0.0
	for d := time.Date(sessionIndexOrigin.Year(), sessionIndexOrigin.Month(), sessionIndexOrigin.Day(), 12, 0, 0, 0, loc); !d.After(day.Add(12 * time.Hour)); d = d.AddDate(0, 0, 1) {
		if c.deps.Calendar.IsTradingDay(d) {
			n++
		}
	}
	c.indexes[key] = n
	return n
}
