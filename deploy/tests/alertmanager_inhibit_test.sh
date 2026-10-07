#!/usr/bin/env bash
# Verifies Alertmanager inhibition against a throwaway Alertmanager.
# Each case posts its alerts to a fresh instance and checks whether the target alert is inhibited.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT=$PWD
AM=${AM:-bin/alertmanager}
PORT=${AM_INHIBIT_PORT:-19193}
WORK=$(mktemp -d)
pid=
cleanup() { [[ -n $pid ]] && kill "$pid" 2>/dev/null; wait 2>/dev/null || true; rm -rf "$WORK"; }
trap cleanup EXIT
sed -e "s|/etc/alertmanager/templates|$ROOT/alertmanager/templates|" alertmanager/alertmanager.yml > "$WORK/am.yml"

start() {
  rm -rf "$WORK/data"
  "$AM" --config.file="$WORK/am.yml" --storage.path="$WORK/data" --web.listen-address=127.0.0.1:$PORT \
    --cluster.listen-address= --log.level=error > "$WORK/am.log" 2>&1 &
  pid=$!
  for _ in $(seq 50); do curl -fsS "http://127.0.0.1:$PORT/-/ready" >/dev/null 2>&1 && return; sleep 0.1; done
  echo "alertmanager did not start"; cat "$WORK/am.log"; exit 1
}
stop() { kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null || true; pid=; }

# post '<json label object>'...
post() {
  local body
  body=$(printf '%s\n' "$@" | python3 -c 'import json,sys; print(json.dumps([{"labels": json.loads(l)} for l in sys.stdin if l.strip()]))')
  curl -fsS -XPOST -H 'Content-Type: application/json' "http://127.0.0.1:$PORT/api/v2/alerts" -d "$body" >/dev/null
}

# inhibited <alertname> -> "true" or "false"
inhibited() {
  curl -fsS "http://127.0.0.1:$PORT/api/v2/alerts?inhibited=true" | python3 -c '
import json, sys
name = sys.argv[1]
st = [a["status"] for a in json.load(sys.stdin) if a["labels"]["alertname"] == name]
print("true" if st and st[0]["inhibitedBy"] else "false")' "$1"
}

fail=0
# case <description> <target alertname> <want true|false> <alert json>...
case_() {
  local desc=$1 target=$2 want=$3; shift 3
  start
  post "$@"
  local got; got=$(inhibited "$target")
  stop
  if [[ $got == "$want" ]]; then echo "ok   $desc"; else echo "FAIL $desc (inhibited=$got, want $want)"; fail=1; fi
}

case_ "critical without symbol does not inhibit market-level conditions" SectorRotation false \
  '{"alertname":"PortfolioDailyLoss","category":"portfolio","severity":"critical","layer":"incident"}' \
  '{"alertname":"SectorRotation","category":"market","severity":"info","layer":"condition","condition":"sector_rotation","sector":"IT"}'
case_ "critical on a symbol inhibits its info conditions" StockMomentumExhaustion true \
  '{"alertname":"StockCircuitHit","category":"circuit","severity":"critical","layer":"incident","exchange":"NSE","symbol":"INFY","band":"upper"}' \
  '{"alertname":"StockMomentumExhaustion","category":"price","severity":"info","layer":"condition","condition":"momentum_exhaustion","exchange":"NSE","symbol":"INFY"}'
case_ "critical on another symbol does not inhibit" StockMomentumExhaustion false \
  '{"alertname":"StockCircuitHit","category":"circuit","severity":"critical","layer":"incident","exchange":"NSE","symbol":"TCS","band":"upper"}' \
  '{"alertname":"StockMomentumExhaustion","category":"price","severity":"info","layer":"condition","condition":"momentum_exhaustion","exchange":"NSE","symbol":"INFY"}'
case_ "exporter down inhibits KPI data quality" KPIDataQualityDegraded true \
  '{"alertname":"StockExporterDown","category":"infra","severity":"critical","layer":"incident","job":"stock_core"}' \
  '{"alertname":"KPIDataQualityDegraded","category":"quality","severity":"warning","layer":"incident","incident":"kpi_data_quality","kpi":"stock:rsi_14m:index"}'
case_ "stale market data inhibits breadth stress" MarketBreadthStress true \
  '{"alertname":"MarketDataStale","category":"infra","severity":"critical","layer":"incident"}' \
  '{"alertname":"MarketBreadthStress","category":"market","severity":"warning","layer":"incident","incident":"breadth_stress","direction":"bearish"}'
exit $fail
