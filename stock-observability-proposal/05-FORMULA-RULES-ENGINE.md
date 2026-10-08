# Formula, KPI and Prometheus Rules Engine

## 1. Objective

Convert financial/market formulas into measurable Prometheus KPIs and alertable signals.

## 2. Formula Object

Every formula should have:

```yaml
id:
name:
category:
description:
mathematical_definition:
inputs:
promql:
unit:
window:
labels:
interpretation:
limitations:
test_cases:
dashboard:
alerts:
```

## 3. Rule Organization

```text
prometheus/rules/stock/
  price.yml
  trend.yml
  momentum.yml
  volatility.yml
  volume.yml
  liquidity.yml
  breadth.yml
  risk.yml
  signals.yml
  data_quality.yml
```

## 4. Stable Recording Rule Names

Examples:

```text
stock:price_change_pct:5m
stock:price_change_pct:15m
stock:price_change_pct:1h
stock:sma:20
stock:sma:50
stock:sma:200
stock:ema:20
stock:rsi:14
stock:macd
stock:atr:14
stock:bollinger_width:20
stock:volume_ratio:20
stock:volatility:20
stock:data_freshness_seconds
```

## 5. Example Formula Categories

### Price

- absolute change
- percentage change
- gap
- intraday return
- rolling return

### Trend

- SMA
- EMA
- price/SMA ratio
- EMA crossover
- trend slope

### Momentum

- RSI
- MACD
- ROC
- stochastic oscillator
- ADX

### Volatility

- ATR
- rolling standard deviation
- Bollinger Bands
- Bollinger width
- volatility expansion

### Volume

- volume ratio
- volume spike
- relative volume
- OBV
- volume trend

### Liquidity

- traded value
- turnover
- spread
- volume concentration
- liquidity score

### Breadth

- advance/decline ratio
- advancing percentage
- declining percentage
- new highs/lows
- sector breadth

### Risk

- drawdown
- beta
- volatility
- downside deviation
- Sharpe-like measures where valid data permits

## 6. Alert Metadata Contract

Recommended labels:

```text
market
exchange
symbol
category
severity
signal_type
```

Recommended annotations:

```text
summary
description
formula
metric
threshold
dashboard
runbook
```

## 7. Important Principle

Prometheus rules should provide measurable evidence.

A formula-derived alert is not automatically a trading recommendation.

## 8. Testing

Each rule should have:

- mathematical test cases
- known input/output cases
- edge cases
- missing-data behavior
- stale-data behavior
- threshold boundary tests
- false-positive review
- recording-rule validation
- alert firing/resolution tests
