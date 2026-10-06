package api

import (
	"context"
	"fmt"
	"time"

	"github.com/seenimo/stock_home/internal/holidays"
)

// Banner is an attention item shown at the top of the page.
type Banner struct {
	Level   string `json:"level"` // critical, warning or info.
	Title   string `json:"title"`
	Message string `json:"message"`
	Action  *Link  `json:"action,omitempty"`
}

// Status combines the market clock (section 6) and the action banner (section 11).
type Status struct {
	Market  MarketClock `json:"market"`
	Banners []Banner    `json:"banners"`
}

// MarketClock describes the exchange session.
type MarketClock struct {
	Phase          string    `json:"phase"` // closed, pre_open, open, post_close or unknown.
	Open           bool      `json:"open"`
	TradingDay     bool      `json:"tradingDay"`
	SecondsToOpen  *float64  `json:"secondsToOpen"`
	SecondsToClose *float64  `json:"secondsToClose"`
	Timezone       string    `json:"timezone"`
	Now            time.Time `json:"now"`
	NextHoliday    *Holiday  `json:"nextHoliday"`
	FeedSource     string    `json:"feedSource"`
}

// Holiday is the next exchange holiday.
type Holiday struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

// status never fails: an unreachable upstream becomes a banner instead.
func (s *Server) status(ctx context.Context) (any, error) {
	now := s.now().In(s.cfg.Location)
	st := Status{Market: MarketClock{Phase: "unknown", Timezone: s.cfg.Location.String(), Now: now}, Banners: []Banner{}}
	if h := holidays.Next(s.cfg.Holidays, now); h != nil {
		st.Market.NextHoliday = &Holiday{Date: h.Date.Format(time.DateOnly), Name: h.Name}
	}

	q := s.querier(ctx)
	phase := q.vector(`max by (phase) (stock_market_phase) == 1`)
	open := q.scalar(`max(stock_market_open)`)
	tradingDay := q.scalar(`max(stock_market_trading_day)`)
	st.Market.SecondsToOpen = q.scalar(`max(stock_market_seconds_to_open)`)
	st.Market.SecondsToClose = q.scalar(`max(stock_market_seconds_to_close)`)
	exporterUp := q.scalar(`min(up{job=~"stock_core|stock_broad"})`)
	session := q.scalar(`max(stock_kite_session_valid)`)
	feed := q.vector(`max by (source) (stock_feed_info)`)
	stale := q.scalar(`max(market:stale_core_instruments:ratio)`)

	if q.err != nil {
		s.logger.Warn("Status: Prometheus query failed", "err", q.err)
		st.Banners = append(st.Banners, Banner{
			Level: "critical", Title: "Prometheus unreachable",
			Message: "Market and feed data cannot be loaded.",
			Action:  &Link{Label: "Open Prometheus", Href: "/prometheus/"},
		})
	} else {
		if len(phase) > 0 {
			st.Market.Phase = phase[0].Labels["phase"]
		}
		st.Market.Open = isOne(open)
		st.Market.TradingDay = isOne(tradingDay)
		if len(feed) > 0 {
			st.Market.FeedSource = feed[0].Labels["source"]
		}
		st.Banners = append(st.Banners, feedBanners(exporterUp, session, stale, st.Market)...)
	}

	if b := s.alertBanner(ctx); b != nil {
		st.Banners = append(st.Banners, *b)
	}
	return st, nil
}

func feedBanners(exporterUp, session, stale *float64, m MarketClock) []Banner {
	var out []Banner
	if exporterUp == nil || *exporterUp == 0 {
		return append(out, Banner{
			Level: "critical", Title: "Stock exporter down",
			Message: "Prometheus cannot scrape the stock exporter; no market data is being collected.",
			Action:  &Link{Label: "Scrape targets", Href: "/prometheus/targets"},
		})
	}
	if m.FeedSource == "kite" && !isOne(session) {
		level := "warning"
		if m.TradingDay {
			level = "critical"
		}
		out = append(out, Banner{
			Level: level, Title: "Kite session expired",
			Message: "Log in to Kite to resume the live market data feed.",
			Action:  &Link{Label: "Log in to Kite", Href: "/exporter/login"},
		})
	}
	if m.Open && stale != nil && *stale > 0.1 {
		out = append(out, Banner{
			Level: "warning", Title: "Market data stale",
			Message: fmt.Sprintf("%.0f%% of core instruments have not ticked for over a minute.", *stale*100),
		})
	}
	return out
}

func (s *Server) alertBanner(ctx context.Context) *Banner {
	alerts, err := s.am.Alerts(ctx)
	if err != nil {
		s.logger.Warn("Status: Alertmanager request failed", "err", err)
		return &Banner{
			Level: "warning", Title: "Alertmanager unreachable",
			Message: "Active alerts cannot be loaded.",
			Action:  &Link{Label: "Open Alertmanager", Href: "/alertmanager/"},
		}
	}
	n := 0
	for _, a := range alerts {
		if a.Status.State == "active" && a.Labels["severity"] == "critical" {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	return &Banner{
		Level: "critical", Title: fmt.Sprintf("%d critical alert%s firing", n, plural(n)),
		Message: "See Active alerts below.",
		Action:  &Link{Label: "Open Alertmanager", Href: "/alertmanager/#/alerts?silenced=false&inhibited=false&filter=%7Bseverity%3D%22critical%22%7D"},
	}
}

func isOne(f *float64) bool { return f != nil && *f == 1 }

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
