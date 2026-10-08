package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Analyst (proposal §A11, Release 6): explains an instrument from the KPI evidence layer. Answers
// cite KPI values with their catalogue quality and the effectiveness ledger; they are deterministic
// summaries, not buy/sell advice.

var symbolPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9&.\- ]{0,31}$`)

const analystDisclaimer = "Observability summary from recorded KPIs; not investment advice."

type evidenceSpec struct {
	kpi, label, unit string
}

var evidenceSpecs = []evidenceSpec{
	{"stock_change_percent", "Day change", "%"},
	{"stock:intraday_return:percent", "Return since the open", "%"},
	{"stock:relative_strength_nifty:percent", "Strength vs NIFTY 50", "pp"},
	{"stock:vwap_deviation:percent", "Distance from VWAP", "%"},
	{"stock:rsi_wilder_14m:index", "Wilder RSI(14), 1m bars", ""},
	{"stock:macd_histogram:price", "MACD histogram", "₹"},
	{"stock:adx_14m:index", "ADX(14)", ""},
	{"stock:relative_volume:ratio", "Relative volume", "x"},
	{"stock:volume_delta_1h:ratio", "Volume delta (1h, tick rule)", ""},
	{"stock:orderbook_weighted_imbalance:ratio", "Weighted book imbalance", ""},
	{"stock:realized_vol_annualised:percent", "Realised volatility (annualised)", "%"},
	{"stock:beta_nifty_60m:ratio", "Beta vs NIFTY 50 (60m)", ""},
	{"stock:abnormal_return_15m:percent", "Abnormal 15m return (market model)", "%"},
	{"stock:var_95_1d:percent", "1-day VaR(95)", "%"},
	{"stock:distance_52w_high:percent", "Distance from the 52-week high", "%"},
}

// Evidence is one KPI value with its catalogue quality.
type Evidence struct {
	KPI     string  `json:"kpi"`
	Label   string  `json:"label"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	Quality string  `json:"quality,omitempty"`
	Method  string  `json:"method,omitempty"`
}

// Condition is a firing condition alert with its 15m hit rate from the effectiveness ledger.
type Condition struct {
	Alert     string   `json:"alert"`
	Direction string   `json:"direction,omitempty"`
	HitRate   *float64 `json:"hitRate,omitempty"`
}

// Analysis is the analyst answer for one instrument.
type Analysis struct {
	Symbol       string            `json:"symbol"`
	MarketRegime string            `json:"marketRegime,omitempty"`
	Regimes      map[string]string `json:"regimes"`
	Summary      []string          `json:"summary"`
	Evidence     []Evidence        `json:"evidence"`
	Conditions   []Condition       `json:"conditions"`
	Disclaimer   string            `json:"disclaimer"`
}

func (s *Server) analystHandler(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	if !symbolPattern.MatchString(symbol) {
		writeJSON(w, http.StatusBadRequest, envelope{Error: "symbol must be an NSE trading symbol", UpdatedAt: s.now()})
		return
	}
	res, err := s.cache.Get("analyst:"+symbol, 30*time.Second, 5*time.Minute, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.QueryTimeout)
		defer cancel()
		return s.analyst(ctx, symbol)
	})
	if err != nil {
		s.errors.WithLabelValues("analyst").Inc()
		s.logger.Warn("Analyst query failed", "symbol", symbol, "err", err)
		writeJSON(w, http.StatusBadGateway, envelope{Error: "upstream unavailable", UpdatedAt: s.now()})
		return
	}
	a := res.Value.(*Analysis)
	if len(a.Evidence) == 0 {
		writeJSON(w, http.StatusNotFound, envelope{Error: "no KPIs recorded for " + symbol, UpdatedAt: s.now()})
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: a, UpdatedAt: res.UpdatedAt, Stale: res.Stale})
}

func (s *Server) analyst(ctx context.Context, symbol string) (*Analysis, error) {
	q := s.querier(ctx)
	names := make([]string, len(evidenceSpecs))
	for i, e := range evidenceSpecs {
		names[i] = regexp.QuoteMeta(e.kpi)
	}
	sel := fmt.Sprintf("symbol=%q, exchange=\"NSE\"", symbol)
	values := q.vector(fmt.Sprintf(`{__name__=~%q, %s}`, strings.Join(names, "|"), sel))
	info := q.vector("stock_kpi_info")
	regimes := q.vector(fmt.Sprintf(`{__name__=~"stock:(volatility|liquidity|momentum|orderflow)_regime:info", %s}`, sel))
	market := q.vector("market:regime:info")
	alerts := q.vector(fmt.Sprintf(`ALERTS{alertstate="firing", layer="condition", %s}`, sel))
	ledger := q.vector(`alert:hit_rate_5d:ratio{horizon="15m"}`)
	if q.err != nil {
		return nil, q.err
	}

	quality := map[string][2]string{}
	for _, m := range info {
		quality[m.Labels["kpi"]] = [2]string{m.Labels["quality"], m.Labels["method"]}
	}
	byName := map[string]float64{}
	for _, v := range values {
		if !math.IsNaN(v.Value) && !math.IsInf(v.Value, 0) {
			byName[v.Labels["__name__"]] = v.Value
		}
	}
	a := &Analysis{Symbol: symbol, Regimes: map[string]string{}, Disclaimer: analystDisclaimer, Conditions: []Condition{}}
	for _, e := range evidenceSpecs {
		v, ok := byName[e.kpi]
		if !ok {
			continue
		}
		qm := quality[e.kpi]
		a.Evidence = append(a.Evidence, Evidence{KPI: e.kpi, Label: e.label, Value: v, Unit: e.unit, Quality: qm[0], Method: qm[1]})
	}
	for _, r := range regimes {
		family := strings.TrimSuffix(strings.TrimPrefix(r.Labels["__name__"], "stock:"), "_regime:info")
		a.Regimes[family] = r.Labels["regime"]
	}
	if len(market) > 0 {
		a.MarketRegime = market[0].Labels["regime"]
	}
	hit := map[string]float64{}
	for _, l := range ledger {
		hit[l.Labels["alertname"]+"/"+l.Labels["direction"]] = l.Value
	}
	seen := map[string]bool{}
	for _, al := range alerts {
		c := Condition{Alert: al.Labels["alertname"], Direction: al.Labels["direction"]}
		if seen[c.Alert+"/"+c.Direction] {
			continue
		}
		seen[c.Alert+"/"+c.Direction] = true
		if h, ok := hit[c.Alert+"/"+c.Direction]; ok {
			c.HitRate = &h
		}
		a.Conditions = append(a.Conditions, c)
	}
	sort.Slice(a.Conditions, func(i, j int) bool { return a.Conditions[i].Alert < a.Conditions[j].Alert })
	a.Summary = summarize(a, byName)
	return a, nil
}

// summarize turns the evidence into short factual sentences; approximations are flagged.
func summarize(a *Analysis, v map[string]float64) []string {
	var out []string
	flag := func(kpi string) string {
		for _, e := range a.Evidence {
			if e.KPI == kpi && e.Quality != "" && e.Quality != "exact" {
				return " (" + e.Quality + ")"
			}
		}
		return ""
	}
	if c, ok := v["stock_change_percent"]; ok {
		s := fmt.Sprintf("%s is %s %.2f%% today", a.Symbol, map[bool]string{true: "up", false: "down"}[c >= 0], math.Abs(c))
		if rs, ok := v["stock:relative_strength_nifty:percent"]; ok {
			s += fmt.Sprintf(", %.2f pp %s than NIFTY 50", math.Abs(rs), map[bool]string{true: "stronger", false: "weaker"}[rs >= 0])
		}
		out = append(out, s+".")
	}
	if ab, ok := v["stock:abnormal_return_15m:percent"]; ok && math.Abs(ab) >= 0.5 {
		out = append(out, fmt.Sprintf("The last 15 minutes deviate %.2f%% from what NIFTY 50 explains (market model%s): a stock-specific move.", ab, flag("stock:abnormal_return_15m:percent")))
	}
	if rsi, ok := v["stock:rsi_wilder_14m:index"]; ok && (rsi >= 70 || rsi <= 30) {
		out = append(out, fmt.Sprintf("Momentum is stretched: Wilder RSI %.0f%s.", rsi, flag("stock:rsi_wilder_14m:index")))
	}
	if rv, ok := v["stock:relative_volume:ratio"]; ok && rv >= 1.5 {
		out = append(out, fmt.Sprintf("Volume runs at %.1fx its usual pace for this time of day%s.", rv, flag("stock:relative_volume:ratio")))
	}
	if vd, ok := v["stock:volume_delta_1h:ratio"]; ok && math.Abs(vd) >= 0.3 {
		out = append(out, fmt.Sprintf("Tick-rule volume delta over the last hour is %+.2f%s: %s.", vd, flag("stock:volume_delta_1h:ratio"),
			map[bool]string{true: "buyer-initiated flow dominates", false: "seller-initiated flow dominates"}[vd > 0]))
	}
	if r, ok := a.Regimes["momentum"]; ok {
		out = append(out, fmt.Sprintf("Stock regimes: momentum %s, volatility %s, order flow %s.", strings.ToLower(r),
			strings.ToLower(a.Regimes["volatility"]), strings.ToLower(a.Regimes["orderflow"])))
	}
	if a.MarketRegime != "" {
		out = append(out, "Market regime: "+strings.ReplaceAll(strings.ToLower(a.MarketRegime), "_", " ")+".")
	}
	for _, c := range a.Conditions {
		s := "Active condition: " + c.Alert
		if c.Direction != "" {
			s += " (" + c.Direction + ")"
		}
		if c.HitRate != nil {
			s += fmt.Sprintf(", %.0f%% 15m hit rate over the last 5 days", *c.HitRate*100)
		} else {
			s += ", no effectiveness history yet"
		}
		out = append(out, s+".")
	}
	return out
}
