// Copyright 2026 The Stock Exporter Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package questdb

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/seenimo/stock_exporter/internal/daily"
	"github.com/seenimo/stock_exporter/internal/store"
)

const (
	queueSize     = 4096
	batchSize     = 500
	flushInterval = time.Second
)

// Writer sends market-state updates to QuestDB using its HTTP line protocol endpoint.
type Writer struct {
	endpoint string
	logger   *slog.Logger
	client   *http.Client
	lines    chan []string
}

// New creates a writer for a QuestDB HTTP base URL.
func New(baseURL string, logger *slog.Logger) (*Writer, error) {
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid QuestDB URL %q: expected an HTTP(S) base URL", baseURL)
	}
	if logger == nil {
		return nil, fmt.Errorf("QuestDB logger is required")
	}
	u.Path = "/write"
	u.RawQuery = "precision=ns"
	return &Writer{
		endpoint: u.String(),
		logger:   logger,
		client:   &http.Client{Timeout: 5 * time.Second},
		lines:    make(chan []string, queueSize),
	}, nil
}

// Observe queues market-state updates without blocking feed processing.
func (w *Writer) Observe(change store.Change) {
	lines := linesForChange(change, time.Now())
	if len(lines) == 0 {
		return
	}
	select {
	case w.lines <- lines:
	default:
		w.logger.Warn("QuestDB queue is full; dropping market-state update", "kind", change.Kind, "exchange", change.Instrument.Exchange, "symbol", change.Instrument.Symbol)
	}
}

// Run batches queued market-state updates and retries failed writes until the context ends.
func (w *Writer) Run(ctx context.Context) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	batch := make([]string, 0, batchSize)
	for {
		select {
		case <-ctx.Done():
			return
		case lines := <-w.lines:
			batch = append(batch, lines...)
			if len(batch) >= batchSize {
				if !w.flush(ctx, batch) {
					return
				}
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if !w.flush(ctx, batch) {
					return
				}
				batch = batch[:0]
			}
		}
	}
}

func (w *Writer) flush(ctx context.Context, lines []string) bool {
	body := strings.Join(lines, "\n")
	backoff := time.Second
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint, strings.NewReader(body))
		if err != nil {
			w.logger.Error("Building QuestDB write request failed", "err", err)
			return false
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		resp, err := w.client.Do(req)
		if err == nil {
			_, readErr := io.Copy(io.Discard, resp.Body)
			closeErr := resp.Body.Close()
			if readErr == nil && closeErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return true
			}
			if readErr != nil {
				err = readErr
			} else if closeErr != nil {
				err = closeErr
			} else {
				err = fmt.Errorf("QuestDB returned HTTP %s", resp.Status)
			}
		}
		w.logger.Error("QuestDB write failed; retrying", "err", err, "batch_size", len(lines))
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

func quoteLine(instrument store.Instrument, quote store.Quote) string {
	return quoteLineAt(instrument, quote, time.Now())
}

type lineBuilder struct {
	strings.Builder
	hasField bool
}

func newLineBuilder(table string, instrument store.Instrument) *lineBuilder {
	b := &lineBuilder{}
	b.WriteString(table)
	b.tag("exchange", instrument.Exchange)
	b.tag("symbol", instrument.Symbol)
	b.tag("sector", instrument.Sector)
	b.tag("instrument_type", instrument.Type)
	b.tag("tier", instrument.Tier)
	b.tag("underlying", instrument.Underlying)
	if !instrument.Expiry.IsZero() {
		b.tag("expiry", instrument.Expiry.Format("2006-01-02"))
	}
	if instrument.Strike != 0 {
		b.tag("strike", formatFloat(instrument.Strike))
	}
	return b
}

func (b *lineBuilder) tag(key, value string) {
	if value == "" {
		return
	}
	b.WriteByte(',')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(escapeTag(value))
}

func (b *lineBuilder) field(key, value string) {
	if !b.hasField {
		b.WriteByte(' ')
		b.hasField = true
	} else {
		b.WriteByte(',')
	}
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(value)
}

func (b *lineBuilder) floatField(key string, value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	b.field(key, formatFloat(value))
}

func (b *lineBuilder) intField(key string, value int64) {
	b.field(key, strconv.FormatInt(value, 10)+"i")
}

func (b *lineBuilder) boolField(key string, value bool) {
	b.field(key, strconv.FormatBool(value))
}

func (b *lineBuilder) line(at time.Time) string {
	if !b.hasField {
		return ""
	}
	if at.IsZero() {
		at = time.Now()
	}
	b.WriteByte(' ')
	b.WriteString(strconv.FormatInt(at.UnixNano(), 10))
	return b.String()
}

func quoteLineAt(instrument store.Instrument, quote store.Quote, fallback time.Time) string {
	at := quote.ReceivedAt
	if at.IsZero() {
		at = fallback
	}
	b := newLineBuilder("stock_quotes", instrument)
	b.intField("token", int64(instrument.Token))
	b.floatField("last_price", quote.LastPrice)
	b.floatField("open", quote.Open)
	b.floatField("high", quote.High)
	b.floatField("low", quote.Low)
	b.floatField("previous_close", quote.PrevClose)
	b.floatField("average_price", quote.AveragePrice)
	b.floatField("volume", quote.Volume)
	b.floatField("last_quantity", quote.LastQuantity)
	b.floatField("buy_quantity", quote.BuyQuantity)
	b.floatField("sell_quantity", quote.SellQuantity)
	b.floatField("open_interest", quote.OpenInterest)
	b.floatField("oi_day_high", quote.OIDayHigh)
	b.floatField("oi_day_low", quote.OIDayLow)
	b.boolField("has_depth", quote.HasDepth)
	if !quote.LastTradeTime.IsZero() {
		b.intField("last_trade_time", quote.LastTradeTime.UnixNano())
	}
	if !quote.ExchangeTime.IsZero() {
		b.intField("exchange_time", quote.ExchangeTime.UnixNano())
	}
	if quote.HasDepth {
		for i := range 5 {
			writeDepthFields(b, "bid", i+1, quote.Depth.Bids[i])
			writeDepthFields(b, "ask", i+1, quote.Depth.Asks[i])
		}
	}
	return b.line(at)
}

func writeDepthFields(b *lineBuilder, side string, level int, depth store.DepthLevel) {
	prefix := side + strconv.Itoa(level) + "_"
	b.floatField(prefix+"price", depth.Price)
	b.floatField(prefix+"quantity", depth.Quantity)
	b.floatField(prefix+"orders", depth.Orders)
}

func linesForChange(change store.Change, now time.Time) []string {
	switch change.Kind {
	case store.ChangeQuote:
		return nonEmptyLines(quoteLineAt(change.Instrument, change.Quote, now))
	case store.ChangeCircuit:
		at := change.Circuit.UpdatedAt
		if at.IsZero() {
			at = now
		}
		b := newLineBuilder("stock_circuits", change.Instrument)
		b.intField("token", int64(change.Instrument.Token))
		b.floatField("upper", change.Circuit.Upper)
		b.floatField("lower", change.Circuit.Lower)
		return nonEmptyLines(b.line(at))
	case store.ChangeReference:
		at := change.Reference.UpdatedAt
		if at.IsZero() {
			at = now
		}
		b := newLineBuilder("stock_references", change.Instrument)
		b.intField("token", int64(change.Instrument.Token))
		b.floatField("high_52w", change.Reference.High52W)
		b.floatField("low_52w", change.Reference.Low52W)
		b.floatField("sma_50d", change.Reference.SMA50D)
		b.floatField("sma_200d", change.Reference.SMA200D)
		b.intField("cross", int64(change.Reference.Cross))
		return nonEmptyLines(b.line(at))
	case store.ChangeDaily:
		return dailyLines(change.Instrument, change.Daily, now)
	case store.ChangePortfolio:
		return portfolioLines(change.Portfolio, now)
	}
	return nil
}

func dailyLines(instrument store.Instrument, indicators daily.Indicators, now time.Time) []string {
	b := newLineBuilder("stock_daily_indicators", instrument)
	b.intField("token", int64(instrument.Token))
	if !indicators.SessionDate.IsZero() {
		b.intField("session_date", indicators.SessionDate.UnixNano())
	}
	b.intField("history", int64(indicators.History))
	b.boolField("valid", indicators.Valid)
	b.floatField("close", indicators.Close)
	b.floatField("sma_20", indicators.SMA20)
	b.floatField("sma_50", indicators.SMA50)
	b.floatField("return_1d", indicators.Return1D)
	b.floatField("return_5d", indicators.Return5D)
	b.floatField("return_20d", indicators.Return20D)
	b.floatField("return_126x21", indicators.Return126x21)
	b.floatField("rsi_14", indicators.RSI14)
	b.floatField("relative_volume_20", indicators.RelVolume20)
	b.floatField("volatility_5", indicators.Vol5)
	b.floatField("volatility_10", indicators.Vol10)
	b.floatField("volatility_20", indicators.Vol20)
	b.floatField("volatility_60", indicators.Vol60)
	b.floatField("ewma_volatility", indicators.EWMAVol)
	b.floatField("macd_histogram", indicators.MACDHist)
	b.floatField("adx", indicators.ADX)
	b.floatField("di_plus", indicators.DIPlus)
	b.floatField("di_minus", indicators.DIMinus)
	b.floatField("slope_20", indicators.Slope20)
	b.floatField("r_squared_20", indicators.R2x20)
	b.floatField("z_score_20", indicators.ZScore20)
	b.floatField("squeeze", indicators.Squeeze)
	b.floatField("obv_slope_20", indicators.OBVSlope20)
	b.intField("streak_bullish", int64(indicators.StreakBullish))
	b.intField("streak_bearish", int64(indicators.StreakBearish))
	lines := nonEmptyLines(b.line(now))
	for _, event := range indicators.Events {
		eventLine := newLineBuilder("stock_daily_signals", instrument)
		eventLine.tag("signal", event.Signal)
		eventLine.intField("token", int64(instrument.Token))
		eventLine.boolField("bullish", event.Bullish)
		eventLine.intField("age_sessions", int64(event.Age))
		lines = append(lines, nonEmptyLines(eventLine.line(now))...)
	}
	return lines
}

func portfolioLines(portfolio store.Portfolio, now time.Time) []string {
	at := portfolio.UpdatedAt
	if at.IsZero() {
		at = now
	}
	lines := make([]string, 0, 1+len(portfolio.Holdings)+len(portfolio.Positions)+len(portfolio.Margins))
	snapshot := newLineBuilder("stock_portfolio", store.Instrument{})
	snapshot.tag("kind", "snapshot")
	snapshot.intField("holding_count", int64(len(portfolio.Holdings)))
	snapshot.intField("position_count", int64(len(portfolio.Positions)))
	snapshot.intField("margin_count", int64(len(portfolio.Margins)))
	lines = append(lines, snapshot.line(at))
	for _, holding := range portfolio.Holdings {
		b := newLineBuilder("stock_portfolio", store.Instrument{Exchange: holding.Exchange, Symbol: holding.Symbol})
		b.tag("kind", "holding")
		b.floatField("quantity", holding.Quantity)
		b.floatField("average_price", holding.AveragePrice)
		b.floatField("last_price", holding.LastPrice)
		b.floatField("pnl", holding.PnL)
		b.floatField("day_change", holding.DayChange)
		lines = append(lines, b.line(at))
	}
	for _, position := range portfolio.Positions {
		b := newLineBuilder("stock_portfolio", store.Instrument{Exchange: position.Exchange, Symbol: position.Symbol})
		b.tag("kind", "position")
		b.tag("product", position.Product)
		b.floatField("quantity", position.Quantity)
		b.floatField("average_price", position.AveragePrice)
		b.floatField("last_price", position.LastPrice)
		b.floatField("pnl", position.PnL)
		b.floatField("m2m", position.M2M)
		lines = append(lines, b.line(at))
	}
	for _, margin := range portfolio.Margins {
		b := newLineBuilder("stock_portfolio", store.Instrument{})
		b.tag("kind", "margin")
		b.tag("segment", margin.Segment)
		b.floatField("net", margin.Net)
		b.floatField("total", margin.Total)
		b.floatField("utilised", margin.Utilised)
		lines = append(lines, b.line(at))
	}
	return lines
}

func nonEmptyLines(lines ...string) []string {
	out := lines[:0]
	for _, line := range lines {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func escapeTag(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `,`, `\,`)
	value = strings.ReplaceAll(value, ` `, `\ `)
	return strings.ReplaceAll(value, `=`, `\=`)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
