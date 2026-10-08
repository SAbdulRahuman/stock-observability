# Stock Observability — System Architecture

## 1. Target Architecture

```text
                         +----------------------+
                         |      Stock Home      |
                         |    Next.js Web UI    |
                         +----------+-----------+
                                    |
                                    v
                         +----------------------+
                         |    Product API/BFF    |
                         +----------+-----------+
                                    |
              +-----------------+---+---+-----------------+
              |                 |       |                 |
              v                 v       v                 v
        Prometheus        Alertmanager Grafana          OpenBB
              ^                 ^       ^                 ^
              |                 |       |                 |
              +-----------------+-------+-----------------+
                                    ^
                                    |
                             stock-exporter
                                    ^
                                    |
                              Zerodha Kite
```

## 2. Component Responsibilities

### stock-exporter

Responsibilities:

- Kite authentication
- instrument discovery
- quote/tick ingestion
- transformation into Prometheus metrics
- source health
- freshness
- API error metrics
- symbol coverage

### Prometheus

Responsibilities:

- scrape metrics
- store time series
- execute recording rules
- execute alert rules
- query historical data

### Alertmanager

Responsibilities:

- grouping
- deduplication
- routing
- silencing
- notification delivery

### Grafana

Responsibilities:

- deep dashboards
- formula validation
- time-series exploration
- engineering/operational observability

### OpenBB

Responsibilities:

- financial research
- fundamentals
- news
- comparison
- screening
- market intelligence

### Product API/BFF

Responsibilities:

- stable product API
- authentication/authorization
- backend aggregation
- caching
- rate limiting
- normalization
- secrets protection
- audit logging

### Stock Home

Responsibilities:

- product UX
- workflows
- watchlists
- stock investigation
- alert investigation
- configuration
- deep-link navigation

## 3. Browser Access Principle

The browser should not directly orchestrate five independent backend systems.

Prefer:

```text
Browser
   |
Stock Home
   |
Product API
   |
Backend Services
```

## 4. Stable Entity Identity

Use a canonical stock identity:

```text
market=IN
exchange=NSE
segment=EQUITY
symbol=RELIANCE
instrument_key=NSE:RELIANCE
```

Additional metadata:

- company_name
- instrument_token
- ISIN
- instrument_type
- tick_size
- lot_size
- sector
- industry
- currency
- country

## 5. Security

- secrets remain server-side
- encrypted credentials
- least privilege
- role-based access
- audit logs
- no broker secrets in browser responses
- API authentication
- secure configuration storage

## 6. Real-Time Updates

Use WebSocket or SSE for:

- new alert
- resolved alert
- signal changes
- market status
- data-quality degradation

## 7. Deep Links

Examples:

```text
/stocks/RELIANCE
/alerts/<alert-id>
/signals/<signal-id>
/formulas/rsi14
/rules/VolumeAnomaly
/research/RELIANCE
/dashboards/stock-deep-dive?symbol=RELIANCE
/prometheus/query?...
```
