# Stock Home — Central UI Product Design

## 1. Objective

Stock Home becomes the canonical entry point for Stock Observability.

It should not attempt to replace every capability of Grafana, Prometheus, or Alertmanager.

## 2. Global Navigation

```text
HOME
MARKET
  NSE Overview
  Indices
  Sectors
  Breadth
  Movers

STOCKS
  Search
  Watchlists
  Stock Detail

ALERTS
  Active
  All
  Critical
  By Stock
  By Rule
  History

SIGNALS

RESEARCH
  OpenBB
  Company Research
  Fundamentals
  Technical
  News
  Compare

DASHBOARDS
  Market
  Stock
  Formula
  Platform

OBSERVABILITY
  Prometheus
  Grafana
  Alertmanager
  Data Quality
  Exporter

ADMINISTRATION
  Configuration
  Data Sources
  Rules
  Formula Catalog
  Notifications
  Users
  Roles
  Audit
  System Health
```

## 3. Global Shell

```text
+--------------------------------------------------------------------------+
| Stock Observability | Search Ctrl+K | NSE: OPEN | Alerts 12 | AI | User |
+--------------------+-----------------------------------------------------+
| Home               |                                                     |
| Market             |                  PAGE CONTENT                       |
| Stocks             |                                                     |
| Alerts             |                                                     |
| Signals            |                                                     |
| Research           |                                                     |
| Dashboards         |                                                     |
| Observability      |                                                     |
| Administration     |                                                     |
+--------------------+-----------------------------------------------------+
```

## 4. Home

Home should answer:

- What is happening?
- Which stocks require attention?
- Are data sources healthy?
- What are the major signals?
- What should I investigate next?

```text
MARKET STATUS
NSE OPEN     NIFTY +0.72%     BANKNIFTY -0.21%

ATTENTION
3 Critical | 21 Warning | 87 Info

MARKET PULSE
Advancers  | Decliners | Unchanged | Volume

TOP MOVERS
...

TOP SIGNALS
...

DATA HEALTH
Exporter | Prometheus | Alertmanager | OpenBB

WATCHLIST
...

RECENT INVESTIGATIONS
...
```

## 5. Alert Scaling

Do not place hundreds of alerts on the home page.

Alert Center must support:

- server-side filtering
- sorting
- pagination/cursor
- virtualization
- grouping
- saved views
- incremental updates
- search

Example:

```text
ALERT CENTER
3 Critical | 21 Warning | 87 Info | 111 Active | 300 Today

Search: [RELIANCE                         ]

Filters: Severity | Symbol | Rule | Category | Status | Time

+--------------------------------------------------------------+
| Severity | Symbol   | Rule             | Value | Duration    |
+--------------------------------------------------------------+
| CRITICAL | RELIANCE | Volume Anomaly   | 4.2x  | 12m         |
| WARNING  | TCS      | RSI Overbought   | 73    | 8m          |
+--------------------------------------------------------------+
```

## 6. Stock Detail

Stock Detail is a multi-tab investigation workspace:

- Overview
- Chart
- Technical
- Signals
- Alerts
- Research
- Observability

```text
RELIANCE
₹2,xxx.xx  +1.42%   NSE   DATA HEALTH: GOOD

[Overview] [Chart] [Technical] [Signals] [Alerts] [Research] [Observability]

PRICE / TREND
------------------------------------------------
        chart

KEY KPIs
RSI | MACD | ATR | Volume Ratio | Volatility

SIGNALS
...

ACTIVE ALERTS
...

RESEARCH
...

OBSERVABILITY
...
```

## 7. Alert Detail

```text
ALERT: RELIANCE Volume Anomaly

Status: FIRING
Severity: WARNING
Duration: 18m

Current Value: 4.2x
Threshold: 3.0x

WHY DID IT FIRE?
Volume is 4.2x the rolling baseline.

LIVE CHART
...

RELATED KPIs
Price | Return | Volume | Volatility | RSI

TIMELINE
...

AI INVESTIGATION
[Explain this alert]

OPEN IN
[Grafana] [Prometheus] [Rule]
```

## 8. AI UX

AI should be contextual.

Input context may include:

- alert
- rule expression
- labels
- annotations
- current metrics
- historical metrics
- related alerts
- related signals
- OpenBB research

AI should explain evidence and uncertainty.

It should not claim guaranteed future price movements.
