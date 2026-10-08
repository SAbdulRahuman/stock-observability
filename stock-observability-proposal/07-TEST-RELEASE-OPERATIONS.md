# Testing, Release and Operations

## 1. Test Pyramid

```text
                    E2E
                 /       \
              Integration
             /             \
          Component       API
         /                   \
      Unit                  Rules
```

## 2. Unit Tests

Test:

- formula calculations
- transformations
- symbol mapping
- configuration validation
- API adapters
- alert metadata
- error handling

## 3. Integration Tests

Validate:

```text
stock-exporter → Prometheus
Prometheus → Alertmanager
Prometheus → Grafana
OpenBB → Product API
Product API → Stock Home
```

## 4. Failure Tests

Explicitly test:

- Kite API unavailable
- authentication failure
- rate limiting
- stale market data
- missing symbol
- Prometheus unavailable
- Alertmanager unavailable
- Grafana unavailable
- OpenBB unavailable
- malformed metrics
- rule evaluation failure
- notification failure

The UI should show degraded service state rather than pretending everything is healthy.

## 5. Performance Tests

Important targets:

- Stock Home initial load
- global search latency
- stock detail latency
- alert list query latency
- Prometheus query latency
- Product API aggregation latency
- exporter scrape latency
- large-alert-table rendering

Alert Center must use:

- server-side filtering
- pagination/cursor
- virtualized rendering

Never render thousands of alert rows into the DOM.

## 6. Observability of the Product

The platform should observe itself.

Track:

- request latency
- request error rate
- exporter scrape duration
- exporter API errors
- Prometheus availability
- rule evaluation health
- Alertmanager delivery health
- OpenBB latency
- stale symbols
- data freshness
- UI/API failures

## 7. Release Gate

A feature release requires:

- tests pass
- deployment succeeds
- health checks pass
- sample data validated
- dashboards validated
- alerts validated
- documentation updated
- rollback path tested

## 8. Definition of Done

```text
CODE
CONFIG
TESTS
FAILURE TESTS
METRICS
LOGGING
DOCS
SCREENSHOT/DEMO
ACCEPTANCE CHECKLIST
RELEASE NOTES
```
