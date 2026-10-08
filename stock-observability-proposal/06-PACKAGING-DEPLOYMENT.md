# Packaging and Deployment

## 1. Product Packaging

Ship Stock Observability as one product distribution.

```text
stock-observability/
├── stock-home/
├── product-api/
├── stock-exporter/
├── prometheus/
├── alertmanager/
├── grafana/
├── openbb-nse/
├── rules/
├── dashboards/
├── deploy/
├── docs/
└── tests/
```

## 2. Docker Compose

Recommended for:

- local development
- demos
- POC/MVP
- single-server deployment

Example services:

```text
stock-home
product-api
stock-exporter
prometheus
alertmanager
grafana
openbb
```

## 3. Kubernetes

Recommended for:

- production
- HA
- multi-instance deployments
- enterprise customers

Use:

- Helm
- ConfigMaps
- Secrets
- Services
- Ingress
- PersistentVolumeClaims
- ServiceMonitor/PrometheusRule where appropriate

## 4. Product Entry Point

Always expose:

```text
https://<host>/
```

as Stock Home.

Specialized UIs may be exposed under controlled routes or links.

## 5. Configuration

Central configuration categories:

### Data Sources

- Zerodha Kite

### Observability

- stock-exporter
- Prometheus
- Alertmanager
- Grafana

### Research

- OpenBB

### AI

- OpenAI

### Notifications

- email
- Slack
- webhook
- future channels

## 6. Configuration UX

Every configuration page should show:

- current status
- configuration
- connection test
- validation
- save/reset
- last changed
- error details

Credentials must not be returned to the browser.

## 7. Distribution Considerations

Before external commercial distribution validate:

- Zerodha/Kite licensing and API terms
- NSE market-data licensing
- third-party news licensing
- fundamentals data rights
- OpenBB provider licensing
- redistribution restrictions

## 8. Release Artifacts

Each release should provide:

- version
- container images
- Helm chart
- Compose bundle
- configuration examples
- migration notes
- release notes
- compatibility matrix
- test report
