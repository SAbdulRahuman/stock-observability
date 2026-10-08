# Grafana Dashboard Plan and Wireframes

## Role

Grafana is the deep analytics and observability engine.

Stock Home is the product UX.

## Dashboard Set

### 00 — Executive / Market Overview

```text
+--------------------------------------------------------------+
| NSE MARKET OVERVIEW                                          |
+--------------------------------------------------------------+
| NIFTY | BANKNIFTY | INDIA VIX | ADV/DEC | VOLUME             |
+--------------------------------------------------------------+
| Market Trend                                                 |
|                         chart                                |
+--------------------------------------------------------------+
| Top Movers | Sector Pulse | Signals | Data Health            |
+--------------------------------------------------------------+
```

### 01 — NSE Market Pulse

Focus:

- index movement
- breadth
- sector performance
- volume
- volatility
- market status

### 02 — Stock Deep Dive

```text
RELIANCE
Price | Return | Volume | RSI | MACD | ATR

+------------------- PRICE -------------------+
|                    chart                    |
+---------------------------------------------+

+------------------- VOLUME ------------------+
|                    chart                    |
+---------------------------------------------+

Signals | Alerts | Formula KPIs | Data Quality
```

### 03 — Price & Trend

- price
- SMA20/50/200
- EMA
- trend slope
- crossovers
- returns

### 04 — Momentum

- RSI
- MACD
- ROC
- stochastic
- ADX

### 05 — Volume & Liquidity

- volume
- relative volume
- volume ratio
- turnover
- liquidity metrics

### 06 — Volatility & Risk

- ATR
- rolling volatility
- Bollinger width
- drawdown
- risk metrics

### 07 — Market Breadth

- advance/decline
- breadth ratio
- new highs/lows
- sector breadth

### 08 — Signals & Anomalies

- breakout
- unusual volume
- volatility expansion
- momentum shift
- anomalies

### 09 — Formula Observatory

Every formula as a measurable KPI.

### 10 — Data Quality

- freshness
- stale symbols
- missing data
- API failures
- coverage

### 11 — Stock Exporter

- scrape health
- Kite API health
- latency
- symbol count
- ingestion rate

### 12 — Alertmanager

- firing alerts
- resolved alerts
- groups
- notification status
- silences

### 13 — Platform / Infrastructure

- CPU
- memory
- container health
- API latency
- Prometheus health

## Implementation Phases

### G0

Provisioning and dashboard-as-code.

### G1

Core market dashboards.

### G2

Formula dashboards.

### G3

Operational dashboards.

### G4

Intelligence dashboards.

### G5

Stock Home integration.
