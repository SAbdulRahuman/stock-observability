# OpenBB NSE Deep-Dive Roadmap

## 1. Objective

Provide an OpenBB experience for NSE/Indian equities comparable in workflow quality to mature US-market research workflows.

## 2. Architectural Decision

Do not replace `stock-exporter`.

Do not force US-specific providers to understand NSE.

Build an NSE-specific provider/extension:

```text
openbb-nse/
```

Architecture:

```text
Stock Home
    |
Product API
    |
+---+----------------+
|                    |
OpenBB            Prometheus
|                    |
openbb-nse       stock-exporter
                     |
                   Kite
```

## 3. Provider Structure

```text
openbb-nse/
├── pyproject.toml
├── openbb_nse/
│   ├── nse_router.py
│   ├── constants.py
│   ├── symbol_map.py
│   ├── market_calendar.py
│   ├── auth/
│   │   └── kite_auth.py
│   ├── clients/
│   │   ├── stock_exporter.py
│   │   └── kite.py
│   ├── models/
│   └── transforms/
└── tests/
```

## 4. Canonical NSE Identity

```text
market=IN
exchange=NSE
segment=EQUITY
symbol=RELIANCE
instrument_key=NSE:RELIANCE
```

## 5. Provider Phases

### N0 — Discovery

- OpenBB compatibility
- provider architecture
- source/license matrix

### N1 — Provider Skeleton

- package
- router
- models
- tests

### N2 — Symbol Layer

- search
- instrument mapping
- identity normalization

### N3 — Market Data

- quote
- historical
- market status
- OHLCV

### N4 — Technical Analytics

- SMA
- EMA
- RSI
- MACD
- ATR
- Bollinger Bands
- ROC
- ADX
- stochastic
- OBV

### N5 — NSE Intelligence

- indices
- breadth
- sectors
- movers
- volume anomalies
- volatility

### N6 — Fundamentals

- company profile
- valuation
- financials
- ratios
- ownership
- dividends
- corporate actions

### N7 — Research

- peers
- comparison
- screener
- news
- research workspace

### N8 — Stock Home Integration

- research panel
- deep links
- unified stock workspace

### N9 — Production Hardening

- cache
- retry
- rate limiting
- stale data detection
- failover
- metrics
- security
- performance

### N10 — Release

- documentation
- packaging
- compatibility
- licensing review

## 6. MVP Cut

```text
Provider
  ↓
Symbol Search
  ↓
Quote
  ↓
Historical
  ↓
Market Status
  ↓
Technical Indicators
  ↓
NSE Indices
  ↓
Breadth
  ↓
Stock Home
  ↓
Grafana
  ↓
Tests
  ↓
Release
```

Fundamentals, corporate actions, screening, peers, and news follow after MVP.
