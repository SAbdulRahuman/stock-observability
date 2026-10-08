#!/usr/bin/env bash
# End-to-end test without Docker: stock_exporter (simulated feed) -> Prometheus -> Alertmanager.
# Requires bin/prometheus, bin/alertmanager, bin/amtool (see Makefile `tools`) and ../stock_exporter/stock_exporter.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT=$PWD
DURATION=${DURATION:-180}
WORK=$(mktemp -d)
EXP_PORT=19101 PROM_PORT=19090 AM_PORT=19093
pids=()
cleanup() { kill "${pids[@]}" 2>/dev/null || true; wait 2>/dev/null || true; rm -rf "$WORK"; }
trap cleanup EXIT

mkdir -p "$WORK/rules" "$WORK/am"
cp prometheus/rules/*.yml "$WORK/rules/"
sed -e "s|/etc/prometheus/rules|$WORK/rules|" \
    -e "s|stock_exporter:9101|127.0.0.1:$EXP_PORT|" \
    -e "s|alertmanager:9093|127.0.0.1:$AM_PORT|" \
    -e "s|localhost:9090|127.0.0.1:$PROM_PORT|" \
    prometheus/prometheus.yml > "$WORK/prometheus.yml"
sed -e "s|/etc/alertmanager/templates|$ROOT/alertmanager/templates|" alertmanager/alertmanager.yml > "$WORK/alertmanager.yml"

../stock_exporter/stock_exporter --feed.source=sim --market.force-open --sim.derivatives --sim.event-probability=0.002 \
  --stock.watchlist-file=../stock_exporter/examples/watchlist.yml \
  --market.holidays-file=../stock_exporter/examples/holidays-nse-2026.yml \
  --collector.textfile.directory=stock_exporter/textfile \
  --web.listen-address=127.0.0.1:$EXP_PORT --log.level=warn > "$WORK/exporter.log" 2>&1 &
pids+=($!)
bin/alertmanager --config.file="$WORK/alertmanager.yml" --storage.path="$WORK/am" \
  --web.listen-address=127.0.0.1:$AM_PORT --cluster.listen-address= --log.level=warn > "$WORK/am.log" 2>&1 &
pids+=($!)
bin/prometheus --config.file="$WORK/prometheus.yml" --storage.tsdb.path="$WORK/tsdb" \
  --web.listen-address=127.0.0.1:$PROM_PORT --log.level=warn > "$WORK/prom.log" 2>&1 &
pids+=($!)

echo ">> running stack for ${DURATION}s"
sleep "$DURATION"

q() { curl -fsS --get "http://127.0.0.1:$PROM_PORT/api/v1/query" --data-urlencode "query=$1" | python3 -c 'import json,sys; r=json.load(sys.stdin)["data"]["result"]; print(r[0]["value"][1] if r else "none")'; }

fail=0
check() { # name, value, python condition on v
  if python3 -c "import sys; v=float('$2') if '$2'!='none' else None; sys.exit(0 if ($3) else 1)"; then
    echo "ok   $1 = $2"
  else
    echo "FAIL $1 = $2 (want $3)"; fail=1
  fi
}

check "up stock_core"                       "$(q 'up{job="stock_core"}')"  "v==1"
check "up stock_broad"                      "$(q 'up{job="stock_broad"}')" "v==1"
check "core samples/scrape"                 "$(q 'scrape_samples_scraped{job="stock_core"}')" "v is not None and 11000<v<16000"
check "broad samples/scrape"                "$(q 'scrape_samples_scraped{job="stock_broad"}')" "v is not None and v>20000"
check "ingest samples/s (1m)"               "$(q 'rate(prometheus_tsdb_head_samples_appended_total[1m])')" "v is not None and v>5000"
check "core scrape duration p_max (s)"      "$(q 'max_over_time(scrape_duration_seconds{job="stock_core"}[1m])')" "v is not None and v<0.9"
check "broad scrape duration p_max (s)"     "$(q 'max_over_time(scrape_duration_seconds{job="stock_broad"}[1m])')" "v is not None and v<4"
check "rule evaluation failures"            "$(q 'sum(prometheus_rule_evaluation_failures_total)')" "v==0"
check "rule group max duration (s)"         "$(q 'max(prometheus_rule_group_last_duration_seconds)')" "v is not None and v<1"
check "sample_limit exceeded"               "$(q 'sum(prometheus_target_scrapes_exceeded_sample_limit_total)')" "v==0"
check "recorded advance/decline ratio"      "$(q 'market:advance_decline:ratio')" "v is not None and v>0"
check "firing alerts in Prometheus"         "$(q 'count(ALERTS{alertstate="firing"})')" "v is not None and v>=1"

# NSE Release 1 KPIs (EOD-based KPIs stay empty: sim mode never reaches post_close).
kpis=$(grep -c '^stock_kpi_info' stock_exporter/textfile/kpi_info.prom)
check "stock_kpi_info series"               "$(q 'count(stock_kpi_info)')" "v==$kpis"
check "RSI(14) core instruments"            "$(q 'count(stock:rsi_14m:index)')" "v is not None and v>150"
check "turnover equities"                   "$(q 'count(stock:turnover:inr)')" "v is not None and v>1000"
check "TRIN"                                "$(q 'market:trin:ratio')" "v is not None and v>0"
check "sector relative strength sectors"    "$(q 'count(sector:relative_strength:percent)')" "v is not None and v>=5"
check "KPI coverage series"                 "$(q 'count(kpi:coverage:ratio)')" "v is not None and v>=25"
check "min intraday KPI coverage"           "$(q 'min(kpi:coverage:ratio{kpi=~"stock:(return_1m|rsi_14m|turnover|orderbook_depth|microprice_deviation).*"})')" "v is not None and v>=0.9"
curl -fsS --get "http://127.0.0.1:$PROM_PORT/api/v1/query" --data-urlencode 'query=kpi:coverage:ratio < 0.9' \
  | python3 -c 'import json,sys; [print("     low coverage %s = %s" % (r["metric"]["kpi"], r["value"][1])) for r in json.load(sys.stdin)["data"]["result"]]'
nse_records=$(sed -n 's/^ *- record: \(.*\)$/\1/p' prometheus/rules/nse_*.rules.yml | sort -u | paste -sd'|')
check "R1+R2 recorded series (cap 27000)"   "$(q "count({__name__=~\"$nse_records\"})")" "v is not None and v<27000"
check "NSE rule group duration / interval"  "$(q 'max(prometheus_rule_group_last_duration_seconds{rule_group=~".*nse_.*"} / prometheus_rule_group_interval_seconds)')" "v is not None and v<=0.25"

# NSE Release 2 (1m and stateful KPIs need a few minutes; DURATION >= 120).
check "market regime"                       "$(q 'count(market:regime:info)')" "v==1"
check "market regime code"                  "$(q 'market:regime:code')" "v is not None and 1<=v<=10"
check "WOBI core equities"                  "$(q 'count(stock:orderbook_weighted_imbalance:ratio)')" "v is not None and v>150"
check "1m bars core instruments"            "$(q 'count(stock:bar_1m_close:price)')" "v is not None and v>150"
check "stock liquidity regimes"             "$(q 'count(stock:liquidity_regime:info)')" "v is not None and v>150"

# NSE Release 3 (reference levels from the simulated reference collector) and Release 4 (simulated NIFTY F&O).
check "52-week position core instruments"   "$(q 'count(stock:range_52w_position:ratio)')" "v is not None and v>150"
check "NIFTY implied move"                  "$(q 'market:implied_move:points')" "v is not None and v>0"
check "F&O contracts"                       "$(q 'count(stock_instrument_info)')" "v==15"
check "NIFTY PCR(OI)"                       "$(q 'underlying:pcr_oi:ratio{underlying="NIFTY"}')" "v is not None and v>0"
check "NIFTY futures premium (%)"           "$(q 'underlying:futures_premium:percent{underlying="NIFTY"}')" "v is not None and 0<v<2"

# Every dashboard query must execute (empty results are fine).
dash_errors=$(python3 - "$PROM_PORT" <<'EOF'
import glob, json, sys, urllib.parse, urllib.request
port = sys.argv[1]
exprs = []
for path in sorted(glob.glob("grafana/dashboards/*.json")):
    d = json.load(open(path))
    exprs += [t["expr"] for p in d["panels"] for t in p.get("targets", [])] + [a["expr"] for a in d["annotations"]["list"]]
bad = 0
for e in exprs:
    e = e.replace("$symbol", "RELIANCE").replace("$sector", ".*")
    url = "http://127.0.0.1:%s/api/v1/query?%s" % (port, urllib.parse.urlencode({"query": e}))
    try:
        urllib.request.urlopen(url, timeout=10).read()
    except Exception as err:
        bad += 1
        print("query failed: %s: %s" % (e, err), file=sys.stderr)
print(bad)
EOF
)
check "dashboard query errors"              "$dash_errors" "v==0"

echo ">> slowest rule groups (s):"
curl -fsS --get "http://127.0.0.1:$PROM_PORT/api/v1/query" \
  --data-urlencode 'query=topk(6, max_over_time(prometheus_rule_group_last_duration_seconds[2m]))' \
  | python3 -c 'import json,sys; [print("  %.3f %s" % (float(r["value"][1]), r["metric"]["rule_group"].split(";")[-1])) for r in json.load(sys.stdin)["data"]["result"]]'
echo ">> slowest rules, last evaluation (s):"
curl -fsS "http://127.0.0.1:$PROM_PORT/api/v1/rules" | python3 -c '
import json, sys
rules = [(r["evaluationTime"], g["name"], r["name"], r.get("labels", {}).get("kpi", "")) for g in json.load(sys.stdin)["data"]["groups"] for r in g["rules"]]
for t, g, n, k in sorted(rules, reverse=True)[:8]:
    print("  %.3f %s %s %s" % (t, g, n, k))'

echo ">> alerts received by Alertmanager:"
alerts=$(bin/amtool --alertmanager.url="http://127.0.0.1:$AM_PORT" alert query -o simple) || fail=1
head -15 <<<"$alerts"
n=$(curl -fsS "http://127.0.0.1:$AM_PORT/api/v2/alerts" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')
check "alerts in Alertmanager" "$n" "v>=1"

if [[ $fail -ne 0 ]]; then
  echo ">> logs"; tail -n 20 "$WORK"/*.log
fi
exit $fail
