# Stock Home — Page-by-Page Wireframes

## 1. Application Shell

```text
+---------------------------------------------------------------------+
| LOGO | Search Ctrl+K | NSE OPEN | Alerts 12 | AI | User             |
+------------------+--------------------------------------------------+
| Home             |                                                  |
| Market           |                  PAGE CONTENT                   |
| Stocks           |                                                  |
| Alerts           |                                                  |
| Signals          |                                                  |
| Research         |                                                  |
| Dashboards       |                                                  |
| Observability    |                                                  |
| Administration   |                                                  |
+------------------+--------------------------------------------------+
```

## 2. Global Search

```text
+------------------------------------------------------------+
| Search stocks, alerts, signals, formulas...                |
+------------------------------------------------------------+
| STOCKS                                                     |
| RELIANCE                                                   |
| TCS                                                        |
|                                                          |
| ALERTS                                                     |
| RELIANCE Volume Anomaly                                    |
|                                                          |
| FORMULAS                                                   |
| RSI(14)                                                     |
+------------------------------------------------------------+
```

## 3. Home

```text
MARKET STATUS
NSE OPEN | NIFTY +0.72% | BANKNIFTY -0.21%

ATTENTION
3 Critical | 21 Warning | 87 Info

MARKET PULSE
Advancers | Decliners | Volume | Volatility

TOP MOVERS
...

TOP SIGNALS
...

DATA HEALTH
Exporter | Prometheus | Alertmanager | OpenBB

WATCHLIST
...
```

## 4. Market Overview

```text
NSE MARKET OVERVIEW
------------------------------------------------
Indices
Breadth
Sectors
Movers
Volume
Volatility
------------------------------------------------
Market chart
------------------------------------------------
```

## 5. Indices

```text
NIFTY 50
Price | Change | Volume | Volatility

[chart]

Constituent breadth
Top gainers
Top losers
```

## 6. Sector Overview

```text
SECTORS

Sector             Return     Breadth    Volume
IT                 +1.8%      70%        1.3x
Banking            -0.4%      35%        0.9x
Energy             +0.8%      62%        1.2x
```

## 7. Market Breadth

```text
ADVANCERS  1240
DECLINERS   980
UNCHANGED   130

A/D Ratio
New Highs
New Lows
Sector Breadth
```

## 8. Market Movers

```text
GAINERS
SYMBOL | PRICE | CHANGE | VOLUME

LOSERS
SYMBOL | PRICE | CHANGE | VOLUME

UNUSUAL VOLUME
SYMBOL | RELATIVE VOLUME | CHANGE
```

## 9. Stocks Landing

```text
STOCKS

[ Search ]

Recent
Watchlists
Top Signals
Recently Investigated
```

## 10. Stock Search

```text
Search NSE stocks

RELIANCE
TCS
INFY
HDFCBANK
...

Filters:
Sector | Market Cap | Signal | Alert
```

## 11. Stock Detail

```text
RELIANCE
₹xxxx | +1.42% | NSE | HEALTHY

[Overview] [Chart] [Technical] [Signals] [Alerts] [Research] [Observability]

Price chart
Key KPIs
Signals
Alerts
Research
```

## 12. Chart Workspace

```text
RELIANCE

Timeframe: 5m 15m 1h 1D 1W
Indicators: SMA EMA RSI MACD ATR

+------------------------------------------------+
|                 PRICE CHART                    |
+------------------------------------------------+
|                 VOLUME                         |
+------------------------------------------------+
```

## 13. Technical Analysis

```text
TREND
SMA20 | SMA50 | SMA200 | EMA20

MOMENTUM
RSI | MACD | ROC | ADX

VOLATILITY
ATR | Bollinger | Rolling Volatility

VOLUME
Relative Volume | OBV
```

## 14. Stock Signals

```text
SIGNALS

Breakout             ACTIVE
Volume Anomaly       ACTIVE
Momentum Acceleration ACTIVE
Volatility Expansion WATCH
```

## 15. Stock Alerts

```text
ALERTS FOR RELIANCE

Critical | Warning | Info

Rule | Value | Threshold | Duration
```

## 16. Watchlists

```text
MY WATCHLIST

Core Portfolio
  RELIANCE
  TCS
  INFY

High Momentum
  ...
```

## 17. Alert Center

```text
ALERT CENTER

3 Critical | 21 Warning | 87 Info | 111 Active

Search [                              ]

Filters:
Severity | Symbol | Rule | Category | Status

+-------------------------------------------------------+
| Severity | Symbol | Rule | Value | Duration           |
+-------------------------------------------------------+
```

## 18. Alert Detail

```text
ALERT
RELIANCE Volume Anomaly

FIRING | WARNING | 18m

Current 4.2x
Threshold 3.0x

Why?
Volume is significantly above rolling baseline.

Chart
Related KPIs
Timeline
AI Investigation
Grafana | Prometheus | Rule
```

## 19. Alert AI Drawer

```text
AI INVESTIGATION

What happened?
...

Evidence
...

Related signals
...

Related alerts
...

Potential explanations
...

Open research
```

## 20. Signal Center

```text
SIGNAL CENTER

Momentum
Breakout
Volume anomaly
Volatility
Trend reversal

Signal | Symbol | Strength | Age
```

## 21. Signal Detail

```text
SIGNAL: RELIANCE MOMENTUM ACCELERATION

Detected: 09:42
Strength: HIGH

Evidence
RSI
ROC
Volume
Price trend

Timeline
Related alerts
Related research
```

## 22. Research Landing

```text
RESEARCH

Company Research
Fundamentals
Technical
News
Peers
Compare
Screeners
```

## 23. Company Research

```text
RELIANCE RESEARCH

Overview
Fundamentals
Financials
Valuation
Ownership
Dividends
News
Peers
```

## 24. Research Compare

```text
COMPARE

RELIANCE | TCS | INFY

Valuation
Growth
Profitability
Technical
Risk
Signals
```

## 25. Dashboards Landing

```text
DASHBOARDS

Market
Stock
Formula
Data Quality
Platform
```

## 26. Grafana Launch Page

```text
GRAFANA

Stock Deep Dive
Market Pulse
Formula Observatory
Data Quality
Platform

[Open Grafana]
```

## 27. Prometheus Explorer

```text
PROMETHEUS EXPLORER

Query:
stock:volume_ratio:20{symbol="RELIANCE"}

[Run]

Instant Result
Range Result
```

## 28. Prometheus Rules

```text
PROMETHEUS RULES

Recording Rules
Alert Rules

Rule | Category | Status | Last Evaluation
```

## 29. Rule Detail

```text
VolumeAnomaly

Expression
Threshold
Labels
Annotations

Current Evaluation
Firing History
Related Formula
Grafana Dashboard
```

## 30. Alertmanager Operations

```text
ALERTMANAGER

Firing
Groups
Silences
Notifications

[Open Native Alertmanager]
```

## 31. Data Quality

```text
DATA QUALITY

Freshness
Stale Symbols
Missing Data
API Errors
Coverage

Overall Score: 97%
```

## 32. Stock Exporter

```text
STOCK EXPORTER

Kite API: HEALTHY
Last scrape: 2s
Symbols: 1900
Ingestion rate: ...
Errors: ...

[Open Metrics]
```

## 33. Platform Health

```text
PLATFORM HEALTH

Stock Home       HEALTHY
Product API      HEALTHY
Exporter         HEALTHY
Prometheus       HEALTHY
Alertmanager     HEALTHY
Grafana          HEALTHY
OpenBB           HEALTHY
```

## 34. Configuration Center

```text
CONFIGURATION

Data Sources
Observability
Research
AI
Notifications
```

## 35. Zerodha Configuration

```text
ZERODHA KITE

Status: CONNECTED

API Key
Secret: ********
Session: ACTIVE

[Test Connection]
[Save]
```

## 36. Rules Management

```text
RULE MANAGEMENT

Trend
Momentum
Volume
Volatility
Liquidity
Breadth
Risk
Signals
Data Quality
```

## 37. Formula Catalog

```text
FORMULA CATALOG

Search formula

RSI
MACD
SMA
EMA
ATR
Bollinger Width
Volume Ratio
A/D Ratio
```

## 38. Formula Detail

```text
RSI(14)

Definition
Mathematical Formula
Inputs
PromQL
Interpretation
Limitations
Test Cases
Dashboards
Alerts
```

## 39. Users / Roles

```text
USERS
Name | Role | Status | Last Login

ROLES
Admin
Analyst
Viewer
Operator
```

## 40. Audit Log

```text
AUDIT LOG

Timestamp | User | Action | Resource | Result
```

## 41. Global AI Assistant

```text
AI ASSISTANT

Ask:
"Why is RELIANCE showing unusual volume?"

Context:
Current metrics
Alerts
Signals
Historical data
Research

Answer
Evidence
Links
```

## 42. Standard States

Every page must support:

- loading
- empty
- error
- stale
- degraded
- permission denied
- disconnected

## 43. Implementation Order

```text
Shell
 ↓
Home
 ↓
Search
 ↓
Stock Detail
 ↓
Alert Center
 ↓
Signals
 ↓
Research
 ↓
Observability
 ↓
Administration
 ↓
AI
```
