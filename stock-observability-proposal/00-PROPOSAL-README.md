# Stock Observability — Product Proposal

## Purpose

This proposal defines **Stock Observability** as a product-grade platform for observing, analyzing, investigating, and researching stock-market data, initially focused on NSE/Indian equities.

The platform combines:

- `stock-exporter` — Zerodha Kite market-data exporter
- Prometheus — metrics and time-series source of truth
- Alertmanager — alert routing and notifications
- Grafana — deep analytics and observability dashboards
- OpenBB — research and market intelligence
- Stock Home — central product/control-plane UI
- Product API / BFF — unified API boundary between Stock Home and backend services

## Product Principle

> **Stock Home is the product. Prometheus, Grafana, Alertmanager, OpenBB, and stock-exporter are specialized platform components.**

The user should enter through Stock Home and move seamlessly into specialized tools when deeper analysis is required.

## Current MVP

The existing POC/MVP contains:

- stock-exporter
- Prometheus
- Alertmanager
- Grafana
- OpenBB
- stock-home

The next objective is to evolve this into a **usable, installable, testable, extensible product**.

## Proposal Documents

| File | Purpose |
|---|---|
| `01-PRODUCT-MASTER-PLAN.md` | Overall product strategy and roadmap |
| `02-CENTRAL-UI-PRODUCT-DESIGN.md` | Stock Home product UX architecture |
| `03-SYSTEM-ARCHITECTURE.md` | Technical architecture |
| `04-FEATURE-BACKLOG.md` | Feature-level implementation backlog |
| `05-FORMULA-RULES-ENGINE.md` | Formula, KPI, recording-rule and alert architecture |
| `06-PACKAGING-DEPLOYMENT.md` | Product packaging and deployment |
| `07-TEST-RELEASE-OPERATIONS.md` | Testing, release and operations |
| `08-GRAFANA-DASHBOARD-PLAN-WIREFRAMES.md` | Grafana dashboard plan |
| `09-OPENBB-NSE-DEEP-DIVE-ROADMAP.md` | OpenBB NSE provider roadmap |
| `10-STOCK-HOME-UX-REDESIGN-WIREFRAMES.md` | Stock Home UX redesign |
| `11-STOCK-HOME-PAGE-BY-PAGE-WIREFRAMES.md` | Page-by-page UI wireframes |

## Recommended Development Method

Implement one vertical feature at a time:

`Specification → Backend/API → Metrics/Rules → UI → Unit Tests → Integration Tests → E2E → Failure Tests → Documentation → Release Evidence`

Each feature is complete only when the entire vertical slice works.

## First Recommended Implementation Slice

Start with:

**F1-01 — Stock Home application shell**

Then:

1. F0-03 health endpoints
2. F0-04 Docker Compose
3. F1-02 Home dashboard
4. F1-03 Stock Search
5. F1-04 Stock Detail
6. F2-01 Data Freshness
7. F3-01 Formula Catalog
8. F4-01 Signal Framework
9. Alert Center
10. OpenBB NSE provider foundation

## Product North Star

Stock Observability should eventually provide a unified workflow:

`Discover → Observe → Detect → Investigate → Research → Decide → Act`

The product should remain observability-first and should not make guaranteed price predictions or represent signals as guaranteed trading recommendations.
