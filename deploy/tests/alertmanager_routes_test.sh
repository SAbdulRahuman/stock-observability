#!/usr/bin/env bash
# Verifies Alertmanager routing: each case is "<expected receiver>|<label set>".
set -euo pipefail
cd "$(dirname "$0")/.."
AMTOOL=${AMTOOL:-bin/amtool}
CONFIG=alertmanager/alertmanager.yml

cases=(
  'stock-infra|alertname=StockExporterDown category=infra severity=critical job=stock_core'
  'stock-infra|alertname=KiteSessionInvalid category=infra severity=critical'
  'stock-portfolio|alertname=PortfolioDailyLoss category=portfolio severity=critical'
  'stock-portfolio|alertname=MarginUtilisationHigh category=portfolio severity=warning segment=equity'
  'stock-critical|alertname=StockPriceSharpDrop category=price severity=critical exchange=NSE symbol=INFY'
  'stock-critical|alertname=StockCircuitHit category=circuit severity=critical exchange=NSE symbol=INFY'
  'stock-critical|alertname=IndexSharpFall category=market severity=critical exchange=NSE symbol="NIFTY 50"'
  'stock-market|alertname=StockPriceSpikeUp category=price severity=warning exchange=NSE symbol=INFY'
  'stock-market|alertname=StockVolumeSurge category=flow severity=warning exchange=NSE symbol=INFY'
  'stock-market|alertname=SectorSelloff category=market severity=warning sector=IT'
  'stock-info|alertname=StockGapOpen category=price severity=info exchange=NSE symbol=INFY'
  'stock-info|alertname=StockMACrossover category=price severity=info direction=bullish'
  'stock-default|alertname=Unclassified severity=warning'
)

fail=0
for c in "${cases[@]}"; do
  want=${c%%|*}
  labels=${c#*|}
  # shellcheck disable=SC2086
  if eval "$AMTOOL" config routes test --config.file="$CONFIG" --verify.receivers="$want" $labels >/dev/null; then
    echo "ok   $want <- $labels"
  else
    echo "FAIL $want <- $labels (got: $(eval "$AMTOOL" config routes test --config.file="$CONFIG" $labels || true))"
    fail=1
  fi
done
exit $fail
