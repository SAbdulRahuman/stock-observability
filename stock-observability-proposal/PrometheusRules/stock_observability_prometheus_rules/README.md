# Stock Observability MVP --- Prometheus Alert Rules

**Status:** Draft for technical review\
**Scope:** 15 Prometheus alert rules, 15 clearance specifications,
Alertmanager silence requirements, and BDD/TDD acceptance criteria.\
**Target:** Stock Observability MVP using `stock_exporter`, Prometheus,
Alertmanager, Grafana, and `stock_home`.

## Files in this package

-   `01-metric-contract.md` --- required/proposed metrics, labels,
    units, and data-quality assumptions.
-   `02-15-prometheus-alert-rules.md` --- 15 detailed Jira-style alert
    requirements and PromQL expressions.
-   `03-clearance-silence-lifecycle.md` --- 15 clearance predicates and
    shared alert lifecycle/silence requirements.
-   `04-testing-ci-and-definition-of-done.md` --- BDD/TDD plan,
    `promtool` guidance, CI gates, and release checklist.

## Important implementation status

The metric names in this proposal are a **proposed contract**, not a
claim that all metrics currently exist in the exporter. Before
implementation, compare them against the real exporter output and adjust
names, labels, types, units, and semantics.

Some rules require derived metrics, such as five-minute returns,
relative volume, VWAP, realized volatility, intraday drawdown, and
expected-symbol inventory. Those must be implemented and tested before
the related alert rules can be accepted.

## Product semantics

-   **Firing:** Prometheus evaluates an alert expression as true for the
    configured `for` duration.
-   **Silence:** Alertmanager suppresses matching notifications for a
    time window. A silence does not resolve or delete the underlying
    alert.
-   **Clearance/resolution:** The alert expression stops matching,
    subject to any configured `keep_firing_for` policy.
-   **Missing data:** A missing series must not automatically be
    interpreted as a healthy condition.
-   **Market interpretation:** These rules detect observable market
    conditions. They do not promise a prediction, profitable trade, or
    investment recommendation.

## Important caveats

Thresholds are initial, configurable MVP defaults---not universal
financial thresholds. Validate them against historical data, instrument
liquidity, exchange rules, data-provider limitations, and the actual
metric methodology.

Use a valid market-session signal or documented session schedule for
rules that depend on active trading hours. Do not treat silence as a
substitute for session-aware rule design.

## Suggested implementation order

1.  Confirm the metric contract.
2.  Implement and test derived recording rules.
3.  Implement data-quality alerts SO-101 through SO-104.
4.  Implement price movement and circuit/VWAP rules SO-105 through
    SO-109.
5.  Implement volume/liquidity and risk rules SO-110 through SO-115.
6.  Add clearance, silence, missing-data, and notification integration
    tests.
7.  Ship only after the definition-of-done checklist passes.
