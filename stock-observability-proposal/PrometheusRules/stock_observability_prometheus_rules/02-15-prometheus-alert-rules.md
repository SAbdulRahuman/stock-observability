# SO-MVP --- 15 Prometheus Alert Rule Requirements

**Status:** Draft for Jira refinement and implementation.\
**Important:** PromQL expressions depend on the proposed metric contract
in `01-metric-contract.md`. Verify actual metric names and units before
merging.

## Common requirements

Every alert rule must define: - Stable alert name and requirement ID. -
Business purpose and severity. - PromQL expression, evaluation interval,
and `for` duration. - Labels that remain stable (`severity`, `category`,
`exchange`, `symbol` as appropriate). - Annotations with summary,
observed value, meaning, limitations, and runbook link. - Positive,
negative, recovery, missing-data and silence tests as applicable. -
Session-aware behaviour where relevant.

Thresholds below are configurable starting values for MVP testing, not
universal financial thresholds.

------------------------------------------------------------------------

## SO-101 --- Stock exporter WebSocket disconnected

**Severity:** critical\
**Proposed `for`:** 1m\
**Category:** data_ingestion

### User story

As an operator, I want to know when the exporter loses its live
market-data connection so that stale dashboards are not mistaken for
current market data.

### PromQL

``` promql
stock_exporter_websocket_connected == 0
```

### BDD acceptance criteria

-   Given the exporter is connected and the metric is `1`, no disconnect
    alert is active.
-   Given the metric remains `0` for one minute, a critical alert fires.
-   Given the connection returns to `1`, the alert resolves according to
    the configured recovery policy.
-   Given the feed is disconnected, the UI identifies market data as
    potentially stale.

### Clearance

The connection metric returns to `1` and remains healthy through the
configured recovery policy.

### Silence

Allow time-limited maintenance silence scoped to exporter instance. Do
not use market-closed periods as a substitute for modelling expected
feed availability.

### TDD

Test `1 → 0 → 0 → 1`, exact `for` timing, missing connection metric, and
exporter instance labels. Missing series must be handled separately from
value `0`.

------------------------------------------------------------------------

## SO-102 --- Stock tick data stale

**Severity:** warning\
**Proposed `for`:** 2m\
**Category:** data_freshness

### User story

As a market watcher, I want to know when a monitored instrument stops
receiving ticks during expected active trading so that I can identify
stale data.

### PromQL

``` promql
time() - nse_stock_last_tick_timestamp > 120
```

### BDD acceptance criteria

-   Given a recent tick timestamp, no stale-tick alert fires.
-   Given the timestamp age exceeds 120 seconds continuously for two
    minutes, a warning fires.
-   Given fresh ticks resume, the alert resolves after recovery
    validation.
-   Given the market is closed or the instrument is legitimately
    inactive, no incident is generated solely due to no ticks.

### Clearance

`time() - nse_stock_last_tick_timestamp <= 120`, with session-aware
gating and recovery stability.

### Silence

Prefer gating with an authoritative market-session metric/calendar. Use
symbol-scoped silence for known temporary inactivity only.

### TDD

Test tick ages of 30, 121 and 300 seconds; missing timestamp series;
market-closed periods; and recovery. Confirm timestamp units are Unix
seconds.

------------------------------------------------------------------------

## SO-103 --- Required price metric missing or invalid

**Severity:** critical\
**Proposed `for`:** 1m\
**Category:** data_quality

### User story

As an operator, I want to detect a missing or invalid LTP so that a
broken data path does not silently produce misleading market views.

### PromQL

``` promql
nse_stock_ltp <= 0
or
absent(nse_stock_ltp)
```

### Implementation note

`absent()` detects a wholly absent metric, but does not reliably
enumerate each missing symbol when other symbols still exist. For
per-symbol detection, compare the authoritative expected-instrument
inventory with observed metrics. Ensure this alert is scoped to the
intended target/metric contract.

### BDD acceptance criteria

-   Given a valid positive LTP, no alert fires.
-   Given an observed LTP is zero or negative for one minute, an alert
    fires.
-   Given a required symbol disappears while other symbols remain,
    inventory-versus-observed coverage detects that missing symbol.
-   Given valid LTP data returns, the alert resolves.

### Clearance

The expected instrument has a present, valid positive LTP and passes
coverage validation.

### Silence

Permit symbol-scoped maintenance suppression. Do not silence
missing-price data globally across the exchange.

### TDD

Test positive, zero, negative values, absent metric, partial symbol
coverage, and invalid values rejected by the exporter.

------------------------------------------------------------------------

## SO-104 --- Exporter scrape or collection failing

**Severity:** critical\
**Proposed `for`:** 2m\
**Category:** data_ingestion

### User story

As an operator, I want to distinguish a healthy exporter process from an
exporter that fails to collect usable market data.

### PromQL

``` promql
stock_exporter_scrape_success == 0
```

Also configure a separate target-availability alert for the exporter,
for example:

``` promql
up{job="stock-exporter"} == 0
```

Use the actual Prometheus job label.

### BDD acceptance criteria

-   Given collection succeeds and the metric is `1`, no
    collection-failure alert fires.
-   Given collection fails continuously for two minutes, a critical
    alert fires.
-   Given collection succeeds again and stays healthy, the alert
    resolves.
-   Given the exporter endpoint is unreachable, the separate `up == 0`
    alert detects it.

### Clearance

Collection success returns to `1` and remains healthy through recovery
validation.

### Silence

Allow silence during approved maintenance scoped to the affected
exporter. A scrape failure should not be hidden by a symbol-specific
market silence.

### TDD

Test success, repeated failure, recovery, target unreachable, and
missing collection-success metric.

------------------------------------------------------------------------

## SO-105 --- Significant five-minute price movement

**Severity:** warning\
**Proposed `for`:** 2m\
**Category:** market_movement

### User story

As a market watcher, I want to be notified when a stock's five-minute
return reaches a configured threshold.

### PromQL

``` promql
abs(stock_return_5m) >= 0.02
```

`0.02` means a 2% fractional return.

### BDD acceptance criteria

-   Given valid return data, when absolute return stays at or above 2%
    for two minutes, a warning fires.
-   Given absolute return stays below 2%, no alert fires.
-   The alert identifies symbol, direction and observed return.
-   Missing or stale return data is handled by data-quality rules rather
    than interpreted as a normal return.

### Clearance

`abs(stock_return_5m) < 0.02`, sustained through the configured recovery
policy.

### Silence

Support per-symbol and per-rule time-limited silence. Silence must not
imply the move is harmless.

### TDD

Test `+1.5%`, `+2.0%`, `+3.0%`, `-2.0%`, missing return, stale input and
recovery. Validate the return calculation.

------------------------------------------------------------------------

## SO-106 --- Price approaching upper circuit limit

**Severity:** warning\
**Proposed `for`:** 1m\
**Category:** circuit_proximity

### User story

As a market watcher, I want to know when the LTP is within 1% of a valid
upper circuit limit.

### PromQL

``` promql
(
  (nse_stock_upper_circuit - nse_stock_ltp)
  / nse_stock_upper_circuit
) <= 0.01
and
nse_stock_upper_circuit > 0
and
nse_stock_ltp > 0
```

### BDD acceptance criteria

-   Given a valid upper limit, when remaining price distance is at most
    1% for one minute, the alert fires.
-   Given the distance exceeds 1%, no alert fires.
-   Missing or invalid limit data must not trigger a circuit-proximity
    alert.
-   If the circuit limit changes, evaluation uses the current valid
    limit.

### Clearance

Price distance rises above 1%, remains there through recovery, and
limit/price data remain valid.

### Silence

Allow symbol-scoped silence. Do not suppress exchange-wide circuit risk
alerts by default.

### TDD

Test distances of 2%, 1%, 0.5%, price at the limit, missing limit, zero
limit, and recovery.

------------------------------------------------------------------------

## SO-107 --- Price approaching lower circuit limit

**Severity:** warning\
**Proposed `for`:** 1m\
**Category:** circuit_proximity

### User story

As a market watcher, I want to know when the LTP is within 1% of a valid
lower circuit limit.

### PromQL

``` promql
(
  (nse_stock_ltp - nse_stock_lower_circuit)
  / nse_stock_lower_circuit
) <= 0.01
and
nse_stock_lower_circuit > 0
and
nse_stock_ltp > 0
```

### BDD acceptance criteria

-   Given a valid lower limit, when price distance is at most 1% for one
    minute, the alert fires.
-   Given the distance exceeds 1%, no alert fires.
-   Missing or invalid circuit data must not trigger the alert.
-   The alert identifies the affected symbol and current distance.

### Clearance

Distance above the lower limit increases beyond 1% and stays above the
threshold through recovery.

### Silence

Allow time-limited symbol-level suppression. Preserve alert
state/history.

### TDD

Test 2%, 1%, 0.5% above the lower limit, price at the limit,
missing/invalid limit, and recovery. Confirm the provider's
circuit-limit semantics.

------------------------------------------------------------------------

## SO-108 --- Price diverges from VWAP

**Severity:** warning\
**Proposed `for`:** 3m\
**Category:** price_deviation

### User story

As a market watcher, I want to identify sustained LTP deviation of at
least 2% from session VWAP.

### PromQL

``` promql
abs(nse_stock_ltp / stock_vwap - 1) >= 0.02
and
stock_vwap > 0
and
nse_stock_ltp > 0
```

### BDD acceptance criteria

-   Given valid LTP and session VWAP, when absolute deviation remains at
    least 2% for three minutes, the alert fires.
-   Given deviation stays below 2%, no alert fires.
-   Zero, missing or invalid VWAP must not trigger a false deviation
    alert.
-   Annotations report direction and deviation percentage.

### Clearance

Absolute VWAP deviation drops below 2% and remains below it through
recovery.

### Silence

Allow per-symbol suppression for research/testing. Do not silence
automatically just because the market is volatile.

### TDD

Test at VWAP, +1%, +2%, -2%, zero/missing VWAP, missing LTP and
recovery. Validate VWAP session boundaries and formula.

------------------------------------------------------------------------

## SO-109 --- Opening gap from previous close

**Severity:** informational / warning\
**Proposed `for`:** 1m\
**Category:** session_event

### User story

As a market watcher, I want to identify an opening price that differs
from the previous session close by at least 3%.

### PromQL

``` promql
abs(nse_stock_open / nse_stock_close - 1) >= 0.03
and
nse_stock_open > 0
and
nse_stock_close > 0
```

### BDD acceptance criteria

-   Given valid open and previous close, when the opening gap is at
    least 3% during the active session, the event alert fires.
-   Given the gap is below 3%, no alert fires.
-   Missing open or close does not produce a valid opening-gap
    conclusion.
-   The event lifecycle is session-aware and does not repeatedly re-fire
    all day.

### Clearance

Resolve when the configured opening-gap monitoring window ends or the
session rolls over. Do not require the opening price to revert to the
previous close.

### Silence

Allow scheduled opening-window suppression if users explicitly choose
it. Avoid permanent broad silences.

### TDD

Test +3%, -3%, +2.9%, missing open/close and session rollover. Verify
alert lifecycle does not repeatedly re-fire during the day.

------------------------------------------------------------------------

## SO-110 --- Unusual relative volume

**Severity:** warning\
**Proposed `for`:** 3m\
**Category:** volume_anomaly

### User story

As a market watcher, I want to identify volume at least three times a
comparable baseline.

### PromQL

``` promql
stock_relative_volume >= 3
```

### Metric requirement

Relative volume must compare equivalent intervals or time-of-day
buckets. Raw cumulative volume is not a valid substitute.

### BDD acceptance criteria

-   Given sufficient baseline history, when relative volume stays at or
    above 3 for three minutes, a warning fires.
-   Given it stays below 3, no alert fires.
-   Missing baseline or insufficient history is reported as unavailable,
    not as a normal value.
-   Alert annotations describe the comparison baseline.

### Clearance

`stock_relative_volume < 3`, sustained through recovery.

### Silence

Allow per-symbol silence for known events or research. Preserve alert
history.

### TDD

Test ratios 1, 2.99, 3 and 4; insufficient baseline, zero baseline,
missing series and recovery.

------------------------------------------------------------------------

## SO-111 --- Abnormally wide bid-ask spread

**Severity:** warning\
**Proposed `for`:** 2m\
**Category:** liquidity

### User story

As a market watcher, I want to detect when the best bid-ask spread
exceeds 0.5% of the midpoint.

### PromQL

``` promql
(
  (nse_stock_ask_price - nse_stock_bid_price)
  /
  ((nse_stock_ask_price + nse_stock_bid_price) / 2)
) >= 0.005
and
nse_stock_bid_price > 0
and
nse_stock_ask_price >= nse_stock_bid_price
```

### BDD acceptance criteria

-   Given valid positive bid/ask, when spread stays at or above 0.5% for
    two minutes, the alert fires.
-   Given spread stays below 0.5%, no alert fires.
-   Crossed or invalid quotes are treated as data-quality issues, not
    normal spread values.
-   Thresholds may be configurable for inherently illiquid instruments.

### Clearance

Spread falls below 0.5% and remains below the threshold through
recovery, with valid non-crossed quotes.

### Silence

Allow symbol-specific silence for illiquid instruments or planned feed
tests. Prefer per-instrument thresholds when justified.

### TDD

Test 0.1%, 0.5%, 1%, zero bid, crossed quotes, missing quotes and
recovery.

------------------------------------------------------------------------

## SO-112 --- Order-book imbalance

**Severity:** informational\
**Proposed `for`:** 3m\
**Category:** order_book

### User story

As a market watcher, I want to identify sustained bid-side dominance in
displayed top-of-book quantities.

### PromQL

``` promql
(
  nse_stock_bid_quantity
  /
  (nse_stock_bid_quantity + nse_stock_ask_quantity)
) >= 0.80
and
(
  nse_stock_bid_quantity + nse_stock_ask_quantity
) > 0
```

A separate ask-side variant should alert when ask share is at least 80%.

### BDD acceptance criteria

-   Given valid non-zero quantities, when bid share stays at or above
    80% for three minutes, an informational alert fires.
-   Given share is below 80%, no alert fires.
-   Zero total depth or missing quantity data must not trigger the rule.
-   UI text must not represent imbalance as a prediction or guaranteed
    directional signal.

### Clearance

Bid share drops below 80% and remains below the threshold through
recovery.

### Silence

Allow user-configured per-symbol silence for noisy or illiquid
instruments.

### TDD

Test balanced quantities, 79.9%, 80%, 90%, zero total depth, missing
metrics and recovery.

------------------------------------------------------------------------

## SO-113 --- Sustained realized volatility elevated

**Severity:** warning\
**Proposed `for`:** 5m\
**Category:** volatility

### User story

As a market watcher, I want to identify instruments whose defined
five-minute realized-volatility metric exceeds an initial threshold of
1%.

### PromQL

``` promql
stock_realized_volatility_5m >= 0.01
```

### Metric requirement

Document whether the metric is standard deviation of returns, annualized
volatility, or another measure. Thresholds only make sense with stable
units and methodology.

### BDD acceptance criteria

-   Given sufficient valid price samples, when the metric stays at or
    above 1% for five minutes, a warning fires.
-   Given it stays below 1%, no alert fires.
-   Insufficient samples or stale data are not treated as normal
    volatility.
-   The runbook explains the calculation and limitations.

### Clearance

Volatility drops below 1% and remains below it through recovery.

### Silence

Allow symbol-level silence during planned events while preserving
visibility into the current state.

### TDD

Test 0.5%, 0.99%, 1%, 2%, insufficient samples and recovery. Validate
the calculation using deterministic price series.

------------------------------------------------------------------------

## SO-114 --- Significant intraday drawdown

**Severity:** warning\
**Proposed `for`:** 2m\
**Category:** drawdown

### User story

As a market watcher, I want to detect a decline of at least 5% from the
running intraday high.

### PromQL

``` promql
stock_intraday_drawdown <= -0.05
```

### Metric requirement

`stock_intraday_drawdown = (current_price / running_intraday_high) - 1`.
The running high resets at the start of a valid session.

### BDD acceptance criteria

-   Given a valid high and current price, when drawdown stays at or
    below -5% for two minutes, a warning fires.
-   Given drawdown is less severe than 5%, no alert fires.
-   Missing baseline or invalid session state does not produce a valid
    drawdown conclusion.
-   The alert explains that it is a drawdown measurement, not a trading
    instruction.

### Clearance

Drawdown improves above -5% and stays above the threshold through
recovery, or the documented session lifecycle ends.

### Silence

Allow per-symbol silence, but do not automatically silence drawdown
alerts because a stock is volatile.

### TDD

Test -2%, -4.99%, -5%, -10%, running-high updates, session reset,
missing baseline and recovery.

------------------------------------------------------------------------

## SO-115 --- Price and volume surge together

**Severity:** warning\
**Proposed `for`:** 2m\
**Category:** combined_market_condition

### User story

As a market watcher, I want a combined alert when a stock has both a
positive five-minute return of at least 2% and relative volume of at
least 2x baseline.

### PromQL

``` promql
(
  stock_return_5m >= 0.02
)
and on (exchange, symbol)
(
  stock_relative_volume >= 2
)
```

### BDD acceptance criteria

-   Given valid return and relative-volume metrics for the same exchange
    and symbol, when both thresholds hold for two minutes, the alert
    fires.
-   Given only one condition is true, no alert fires.
-   Given labels do not match, the expression must not accidentally
    combine different instruments.
-   Alert copy must not present this condition as a buy signal.

### Clearance

The combined expression becomes false because return drops below 2% or
relative volume drops below 2x, and remains false through recovery.

### Silence

Allow per-symbol and per-rule time-limited silence. Preserve the alert
condition and history.

### TDD

Test all four Boolean combinations, mismatched labels, missing return,
missing relative volume and recovery.
