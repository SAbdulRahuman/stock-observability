#!/usr/bin/env bash
# End-to-end test without Docker: stock_exporter (simulated feed) -> Prometheus -> Alertmanager.
# Requires bin/prometheus, bin/alertmanager, bin/amtool (see Makefile `tools`) and ../stock_exporter/stock_exporter.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT=$PWD
DURATION=${DURATION:-120}
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

../stock_exporter/stock_exporter --feed.source=sim --market.force-open --sim.event-probability=0.002 \
  --stock.watchlist-file=../stock_exporter/examples/watchlist.yml \
  --market.holidays-file=../stock_exporter/examples/holidays-nse-2026.yml \
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
check "core samples/scrape"                 "$(q 'scrape_samples_scraped{job="stock_core"}')" "v is not None and 2500<v<4000"
check "broad samples/scrape"                "$(q 'scrape_samples_scraped{job="stock_broad"}')" "v is not None and v>20000"
check "ingest samples/s (1m)"               "$(q 'rate(prometheus_tsdb_head_samples_appended_total[1m])')" "v is not None and v>5000"
check "core scrape duration p_max (s)"      "$(q 'max_over_time(scrape_duration_seconds{job="stock_core"}[1m])')" "v is not None and v<0.9"
check "broad scrape duration p_max (s)"     "$(q 'max_over_time(scrape_duration_seconds{job="stock_broad"}[1m])')" "v is not None and v<4"
check "rule evaluation failures"            "$(q 'sum(prometheus_rule_evaluation_failures_total)')" "v==0"
check "rule group max duration (s)"         "$(q 'max(prometheus_rule_group_last_duration_seconds)')" "v is not None and v<1"
check "sample_limit exceeded"               "$(q 'sum(prometheus_target_scrapes_exceeded_sample_limit_total)')" "v==0"
check "recorded advance/decline ratio"      "$(q 'market:advance_decline:ratio')" "v is not None and v>0"
check "firing alerts in Prometheus"         "$(q 'count(ALERTS{alertstate="firing"})')" "v is not None and v>=1"

echo ">> alerts received by Alertmanager:"
bin/amtool --alertmanager.url="http://127.0.0.1:$AM_PORT" alert query -o simple | head -15 || fail=1
n=$(curl -fsS "http://127.0.0.1:$AM_PORT/api/v2/alerts" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')
check "alerts in Alertmanager" "$n" "v>=1"

if [[ $fail -ne 0 ]]; then
  echo ">> logs"; tail -n 20 "$WORK"/*.log
fi
exit $fail
