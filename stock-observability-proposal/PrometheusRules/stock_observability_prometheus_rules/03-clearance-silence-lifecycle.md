# SO-MVP --- Clearance, Silence and Alert Lifecycle Requirements

## 1. Key semantics

### Alert creation

Prometheus evaluates a PromQL alert expression at its configured
evaluation interval. When the expression returns a vector element, the
alert enters pending; after its `for` duration remains satisfied, it
fires.

### Silence

An Alertmanager silence suppresses notifications for matching alerts
during its active interval. It does **not**: - change a firing alert to
resolved; - delete the alert; - prove that the underlying condition has
cleared; - suppress unmatched alerts.

When the silence expires, notifications may resume if the alert is still
firing, subject to grouping, routing, repeat interval and notification
configuration.

### Clearance

An alert normally resolves when its expression no longer returns the
relevant series. `keep_firing_for`, if configured, keeps an alert firing
for a defined period after the expression stops matching. This is a
recovery stability policy, not a second alert rule.

## 2. Fifteen clearance predicates

These predicates are for test assertions and lifecycle validation. **Do
not deploy them as 15 additional Prometheus alerting rules.** They may
be represented as helper expressions in tests or in implementation
documentation.

  ---------------------------------------------------------------------------------------------------------------------------------
  ID                      Alert                   Clearance predicate / condition
  ----------------------- ----------------------- ---------------------------------------------------------------------------------
  SO-101                  WebSocket disconnected  `stock_exporter_websocket_connected == 1`

  SO-102                  Stale tick              `time() - nse_stock_last_tick_timestamp <= 120`, with session-aware gating

  SO-103                  Missing/invalid price   Expected symbol has a present, valid positive `nse_stock_ltp` and passes
                                                  inventory coverage validation

  SO-104                  Collection failing      `stock_exporter_scrape_success == 1`; separately verify
                                                  `up{job="stock-exporter"} == 1` for target reachability

  SO-105                  Significant price       `abs(stock_return_5m) < 0.02`
                          movement                

  SO-106                  Near upper circuit      `((nse_stock_upper_circuit - nse_stock_ltp) / nse_stock_upper_circuit) > 0.01`,
                                                  with valid positive values

  SO-107                  Near lower circuit      `((nse_stock_ltp - nse_stock_lower_circuit) / nse_stock_lower_circuit) > 0.01`,
                                                  with valid positive values

  SO-108                  VWAP deviation          `abs(nse_stock_ltp / stock_vwap - 1) < 0.02`, with valid positive LTP/VWAP

  SO-109                  Opening gap             Configured opening-gap monitoring window ends or trading session rolls over

  SO-110                  Relative volume         `stock_relative_volume < 3`

  SO-111                  Wide spread             Spread `< 0.005`, with valid positive, non-crossed quotes

  SO-112                  Bid imbalance           Bid share `< 0.80`, with valid non-zero total depth

  SO-113                  Elevated volatility     `stock_realized_volatility_5m < 0.01`, with sufficient fresh samples

  SO-114                  Intraday drawdown       `stock_intraday_drawdown > -0.05`, or documented session lifecycle ends

  SO-115                  Price + volume surge    `stock_return_5m < 0.02 OR stock_relative_volume < 2`, with valid fresh inputs
  ---------------------------------------------------------------------------------------------------------------------------------

**Missing-data rule:** A missing or invalid series should not be blindly
treated as a clearance predicate being true. Missing-data policy must be
handled explicitly with exporter health, expected-symbol inventory, and
stale-data checks.

## 3. Shared Jira requirement: SO-COMMON-01 --- Silence must not resolve an alert

### Given / When / Then

-   **Given** an alert is firing for `symbol="INFY"`.
-   **When** a user creates an Alertmanager silence matching that symbol
    and alert name.
-   **Then** the alert remains firing while its PromQL expression
    remains true.
-   **And** matching notifications are suppressed while the silence is
    active.
-   **And** alerts for other symbols remain unaffected.
-   **And** the silence records creator, creation time, expiry, matchers
    and comment.
-   **And** after silence expiry, notifications may resume if the alert
    remains firing, subject to Alertmanager notification configuration.

### TDD scenarios

-   Symbol-specific silence.
-   Rule-specific silence.
-   Silence that matches no alert.
-   Expired silence while the alert remains firing.
-   Silence while the alert resolves.
-   Silence matching more alerts than intended; verify matcher scope.

## 4. Shared Jira requirement: SO-COMMON-02 --- Recovery resolves the alert

### Given / When / Then

-   **Given** a stock alert is firing.
-   **When** the underlying PromQL condition becomes false and remains
    false through the configured recovery policy.
-   **Then** Prometheus transitions the alert to resolved.
-   **And** the alert is no longer counted as active.
-   **And** a resolved notification is sent if configured and supported
    by the receiver.
-   **And** history records firing and resolution timestamps.
-   **And** a later recurrence creates a new firing lifecycle/recurrence
    in alert history.

### TDD scenarios

-   Immediate condition recovery.
-   Sustained recovery.
-   Condition briefly becomes false then true again.
-   Recovery while a silence is active.
-   Recurrence after resolution.
-   Validate `keep_firing_for` behaviour where configured.

## 5. Shared Jira requirement: SO-COMMON-03 --- Missing data must not masquerade as recovery

### Given / When / Then

-   **Given** an alert is firing because a stock metric crosses its
    threshold.
-   **When** the exporter stops exposing that metric.
-   **Then** missing data is not automatically interpreted as proof that
    the market condition cleared.
-   **And** feed-health or missing-metric monitoring identifies the
    collection problem.
-   **And** any policy that resolves an alert after data disappears is
    explicit and documented.
-   **And** the UI distinguishes firing, pending, suppressed, resolved
    and unknown/stale-data states.

### TDD scenarios

-   Remove the series while alert is firing.
-   Restore the series with the condition still true.
-   Restore the series with the condition cleared.
-   Stop the whole exporter.
-   Remove one symbol while other symbols continue.
-   Test stale data during market-open and market-closed periods.

## 6. Alertmanager routing and silence guidance

-   Use stable labels such as `severity`, `category`, `exchange`,
    `symbol`, and `team`.
-   Keep volatile numeric values in annotations, not labels.
-   Scope silences as narrowly as practical: symbol + alert name or
    exporter instance.
-   Require a comment and expiry for manual silences.
-   Do not create permanent broad silences for data-quality or
    feed-health alerts.
-   Test grouping, repeat intervals, resolved notifications, and
    receiver failures.
-   Silence should suppress notification delivery only; dashboards
    should still display active alert state and silence status.

## 7. Rule-specific caveats

-   **SO-102:** stale-tick logic requires session-aware gating or it
    will alert for inactive symbols/closed markets.
-   **SO-103:** `absent(nse_stock_ltp)` detects a wholly absent metric
    but does not enumerate every missing symbol. Use expected-instrument
    inventory comparison for per-symbol completeness.
-   **SO-106/SO-107:** invalid/missing circuit limits must not trigger
    proximity alerts.
-   **SO-109:** opening gap is an event-style alert; resolve at the end
    of a defined window/session rather than requiring the gap to revert.
-   **SO-110:** relative volume needs a comparable baseline; raw
    cumulative volume is insufficient.
-   **SO-111:** crossed/invalid quotes are data-quality cases, not
    normal spreads.
-   **SO-113:** volatility units and calculation methodology must be
    fixed.
-   **SO-114:** intraday high must reset at session start.
-   **SO-115:** return and volume series must match on `exchange` and
    `symbol`.

## 8. Acceptance checklist

-   [ ] Silencing never changes the underlying Prometheus alert state.
-   [ ] Recovery semantics are documented for all 15 alerts.
-   [ ] `keep_firing_for` is used intentionally, not as a substitute for
    missing-data handling.
-   [ ] Missing series, stale data, and exporter failure have dedicated
    tests.
-   [ ] Silence matchers are scoped and have expiry.
-   [ ] Resolved notification behaviour is tested.
-   [ ] Session-aware behaviour is tested for relevant rules.
