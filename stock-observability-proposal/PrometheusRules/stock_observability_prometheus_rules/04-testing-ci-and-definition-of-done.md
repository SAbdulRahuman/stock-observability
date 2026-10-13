# SO-MVP --- Testing, CI and Definition of Done

## 1. Testing strategy

The release should test more than whether YAML parses. Each rule needs
deterministic tests for firing, not firing, recovery, missing data and
any relevant session/silence behaviour.

### Test layers

1.  **Metric-contract tests:** Required metrics, labels, units,
    freshness, and derived metric semantics.
2.  **PromQL expression tests:** Positive and negative expression
    outcomes using synthetic samples.
3.  **Alert lifecycle tests:** Pending → firing → resolved, including
    `for` and `keep_firing_for`.
4.  **Missing-data tests:** Missing series, stale values, exporter
    target down, and partial symbol coverage.
5.  **Alertmanager integration tests:** Notification delivery, silence
    matching, silence expiry, grouping and resolved notifications.
6.  **End-to-end tests:** Simulated or controlled feed condition reaches
    Prometheus, fires an alert, appears in the UI, and follows the
    configured notification route.

## 2. Promtool commands

Validate rules:

``` bash
promtool check rules prometheus/rules/*.yml
```

Run rule tests:

``` bash
promtool test rules prometheus/tests/stock_alerts_test.yml
```

Use `promtool`'s supported test YAML schema for `rule_files`,
`evaluation_interval`, `tests`, `input_series`, `alert_rule_test`, and
`promql_expr_test`. Pin the Prometheus/promtool version used in CI to
the same version family as the deployment.

Promtool unit tests do not test Alertmanager silence delivery. Use an
integration test with a test receiver or equivalent controlled
notification endpoint.

## 3. Example test-file skeleton

This is a structural example; add actual `input_series`, expected alert
times and expected labels for each rule before treating it as complete.

``` yaml
rule_files:
  - ../rules/stock_data_quality.yml
  - ../rules/stock_price_alerts.yml
  - ../rules/stock_volume_liquidity.yml
  - ../rules/stock_risk_alerts.yml

evaluation_interval: 30s

tests:
  # Add one or more test cases per alert:
  # - alert_rule_test:
  #     - eval_time: 3m
  #       alertname: StockSignificantPriceMovement
  #       exp_alerts:
  #         - exp_labels:
  #             severity: warning
  #             category: market_movement
  #             symbol: INFY
  #           exp_annotations:
  #             summary: "Significant price movement: INFY"
  #
  # Also add promql_expr_test cases for recording rules and clearance predicates.
```

## 4. Required scenarios per alert

For each of SO-101 through SO-115, create tests for:

-   **Positive:** expression remains true long enough to satisfy `for`.
-   **Negative:** expression remains below threshold and does not fire.
-   **Boundary:** just below, exactly at, and just above the threshold.
-   **Recovery:** expression becomes false; resolve timing matches
    documented policy.
-   **Flap:** condition briefly becomes false then true again.
-   **Missing data:** series disappears while healthy and while firing.
-   **Labels:** `symbol`/`exchange` identities are stable and
    multi-metric joins match correctly.
-   **Silence:** integration test confirms notifications are suppressed
    without resolving the alert.
-   **Session behaviour:** relevant rules do not alert merely because
    the exchange is closed.

Not every case is meaningful for every rule; document justified
exceptions.

## 5. CI pipeline gates

A proposed CI sequence:

1.  Lint YAML.
2.  Run `promtool check rules`.
3.  Run PromQL unit tests.
4.  Validate metric-contract fixtures and derived-metric tests.
5.  Run Alertmanager routing/silence integration tests.
6.  Check for missing required labels/annotations/runbook links.
7.  Package rules with the release artifact.
8.  Run a staging smoke test before production release.

Reject a release if: - rule syntax is invalid; - firing occurs too
early/late; - recovery behaviour is wrong; - a missing metric is
incorrectly interpreted as a healthy value; - notification routing or
silence scope is wrong; - a rule has no documented runbook or metric
methodology.

## 6. Suggested repository layout

``` text
stock-observability/
├── prometheus/
│   ├── prometheus.yml
│   ├── rules/
│   │   ├── stock_recording_rules.yml
│   │   ├── stock_data_quality.yml
│   │   ├── stock_price_alerts.yml
│   │   ├── stock_volume_liquidity.yml
│   │   └── stock_risk_alerts.yml
│   └── tests/
│       ├── stock_alerts_test.yml
│       └── fixtures/
├── alertmanager/
│   └── alertmanager.yml
└── docs/
    └── alerts/
        ├── SO-101.md
        ├── SO-102.md
        └── ...
```

## 7. Definition of Done

-   [ ] Actual exporter metric names and label schema are verified.
-   [ ] Units and timestamp conventions are documented.
-   [ ] Derived metrics have reviewed formulas and deterministic tests.
-   [ ] All 15 alert rules are version-controlled.
-   [ ] All 15 clearance predicates are covered by tests.
-   [ ] Thresholds and durations are configurable and identified as
    defaults.
-   [ ] Silence and resolution are tested independently.
-   [ ] Missing data, stale data and exporter failures are covered.
-   [ ] Market-session behaviour is tested where applicable.
-   [ ] Grafana shows affected symbol, observed value, state, and rule
    documentation.
-   [ ] Each rule has a runbook with meaning, limitations and
    troubleshooting.
-   [ ] CI blocks release on rule-test failures.
-   [ ] No alert is presented as a guaranteed prediction or trading
    recommendation.
