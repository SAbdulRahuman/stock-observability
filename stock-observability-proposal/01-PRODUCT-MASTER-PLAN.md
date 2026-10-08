# Stock Observability — Product Master Plan

## 1. Vision

Build an **AI-native stock observability platform** that turns raw market data into measurable KPIs, signals, alerts, investigations, research workflows, and operational visibility.

Initial market:

- India
- NSE
- Equity instruments
- Zerodha Kite as the primary live-data source

Future markets:

- BSE
- US
- Saudi
- Japan
- Europe

## 2. Product Layers

### Layer 1 — Data

- Zerodha Kite
- stock-exporter
- market instruments
- quotes
- ticks
- historical data

### Layer 2 — Observability

- Prometheus
- recording rules
- alert rules
- Alertmanager
- data-quality metrics

### Layer 3 — Analytics

- Grafana
- technical indicators
- formula dashboards
- market dashboards

### Layer 4 — Research

- OpenBB
- fundamentals
- news
- peers
- comparisons
- screening

### Layer 5 — Product Experience

- Stock Home
- Product API/BFF
- search
- watchlists
- stock investigation
- alerts
- signals
- research workspace
- administration

### Layer 6 — Intelligence

- AI investigation
- alert explanation
- anomaly explanation
- natural-language queries
- contextual research

## 3. Product Architecture Principle

Stock Home is the central product experience.

Specialized systems remain specialized:

| Component | Primary responsibility |
|---|---|
| stock-exporter | Market-data acquisition and Prometheus exposition |
| Prometheus | Metrics, time-series queries, recording rules, alert rules |
| Alertmanager | Alert grouping, deduplication, routing, silencing |
| Grafana | Deep analytical dashboards |
| OpenBB | Research and standardized financial workflows |
| Product API/BFF | Product-facing API boundary |
| Stock Home | Unified product UX |

## 4. Major Product Domains

### Market

- NSE overview
- indices
- sectors
- breadth
- movers

### Stocks

- search
- stock detail
- chart workspace
- technical analysis
- signals
- alerts
- watchlists

### Alerts

- active alerts
- all alerts
- critical alerts
- alerts by stock
- alerts by rule
- alert history
- grouped alerts
- alert investigation

### Signals

Signals are derived market conditions, separate from operational alerts.

Examples:

- momentum acceleration
- unusual volume
- breakout
- volatility expansion
- trend reversal
- breadth deterioration

### Research

- OpenBB
- fundamentals
- technical analysis
- news
- peer comparison
- screening

### Observability

- Prometheus
- Grafana
- Alertmanager
- stock-exporter
- data quality
- platform health

### Administration

- data sources
- rules
- formulas
- notifications
- users
- roles
- audit
- system settings

## 5. Roadmap

### Phase 0 — Product Foundation

- repository standardization
- configuration management
- health endpoints
- Docker Compose
- Kubernetes deployment

### Phase 1 — Central Product Portal

- application shell
- home dashboard
- stock search
- stock detail
- watchlists
- platform health
- deep links

### Phase 2 — Market Data Observability

- data freshness
- exporter health
- symbol coverage
- tick ingestion rate
- data quality score

### Phase 3 — Formula & KPI Engine

- trend
- momentum
- volatility
- volume
- liquidity
- breadth
- risk

### Phase 4 — Signal & Alert Intelligence

- signal framework
- momentum alerts
- volume anomaly alerts
- breakout alerts
- volatility alerts
- data quality alerts
- notification routing

### Phase 5 — Stock Intelligence UI

- stock investigation workspace
- signal timeline
- alert timeline
- news
- peer comparison
- sector view

### Phase 6 — OpenBB Integration

- OpenBB gateway
- research panel
- research deep links
- NSE provider

### Phase 7 — Advanced Observability

- anomaly detection
- market regime analysis
- correlation
- cross-market analytics
- advanced risk analytics

### Phase 8 — Multi-Market

- BSE
- US
- Saudi
- Japan
- Europe

### Phase 9 — AI-Native Observability

- AI alert investigation
- natural-language PromQL assistance
- contextual research
- anomaly explanations
- incident-style investigation timelines

## 6. Product North Star

The platform should enable:

**Discover → Observe → Detect → Investigate → Research → Decide**

It should provide evidence and context rather than guarantee future price movement.
