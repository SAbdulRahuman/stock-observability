# Stock Home — OpenBB Analysis Page Wireframe

**Page:** OpenBB Analysis  
**Purpose:** Provide an OpenBB-powered research and analysis workspace inside Stock Home, complementing (not replacing) the existing `stock_exporter`, Prometheus, AlertManager, Grafana, and Stock Home views.

> **Scope note:** This is a proposed page wireframe. If an approved OpenBB Analysis UI image exists, use that image as the visual source of truth and adjust this document to match it exactly. Do not infer that every optional panel below is already implemented by OpenBB or by the current app. Render only the analysis features and data sources that are actually integrated.

## 1. Page layout

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ STOCK HOME                                                       Market status / Updated: —   Profile │
├───────────────────────┬──────────────────────────────────────────────────────────────────────────────┤
│                       │ OpenBB Analysis                                                              │
│  Dashboard            │ Research and inspect a selected stock                                        │
│  Market Overview      │                                                                              │
│  Stock Explorer       │ ┌──────────────────────────────────────────────────────────────────────────┐ │
│  Watchlist            │ │ Search symbol / company name…  [NSE ▼]  [RELIANCE ▼]       [Analyze]      │ │
│  Alerts               │ └──────────────────────────────────────────────────────────────────────────┘ │
│  Top Movers           │                                                                              │
│  Sector View          │ [Overview] [Price & Volume] [Fundamentals] [Financials] [News]               │
│  OpenBB Analysis      │                                                                              │
│                       │ ┌──────────────────────────────────────────────────────────────────────────┐ │
│  OBSERVABILITY        │ │ Selected Stock                                                           │ │
│  Prometheus           │ │ RELIANCE — Reliance Industries Ltd.                     Data status: —    │ │
│  Grafana              │ │ Price: —   Change: —   Volume: —   Market cap: — (when available)         │ │
│  AlertManager         │ └──────────────────────────────────────────────────────────────────────────┘ │
│                       │                                                                              │
│                       │ ┌────────────────────────────────────────────┐ ┌─────────────────────────┐ │
│                       │ │ Price & Volume                             │ │ Key Metrics             │ │
│                       │ │                                            │ │                         │ │
│                       │ │ [Time range ▼]                             │ │ Metric        Value     │ │
│                       │ │                                            │ │ —             —         │ │
│                       │ │             [Chart area]                   │ │ —             —         │ │
│                       │ │                                            │ │ —             —         │ │
│                       │ └────────────────────────────────────────────┘ └─────────────────────────┘ │
│                       │                                                                              │
│                       │ ┌────────────────────────────────────────────┐ ┌─────────────────────────┐ │
│                       │ │ Financial / Fundamental Summary            │ │ Related News            │ │
│                       │ │                                            │ │                         │ │
│                       │ │ Only available fields and periods          │ │ Headline • Source • Time │ │
│                       │ │ from connected OpenBB providers             │ │ Headline • Source • Time │ │
│                       │ │                                            │ │ [View news]              │ │
│                       │ └────────────────────────────────────────────┘ └─────────────────────────┘ │
│                       │                                                                              │
│                       │ [Open full OpenBB workspace] (only if a configured destination exists)      │
├───────────────────────┴──────────────────────────────────────────────────────────────────────────────┤
│ Data provider: — • Last successful update: — • Values may be delayed                                 │
└──────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Shared application shell

- Reuse the Stock Home header, sidebar, typography, spacing, card styles, and footer.
- Highlight **OpenBB Analysis** as the active navigation item only if this page is included in the approved navigation.
- Keep existing links to Dashboard, Market Overview, Stock Explorer, Watchlist, Alerts, Top Movers, Sector View, Prometheus, Grafana, and AlertManager consistent with the app.
- Do not create a second navigation system inside the page unless the approved design includes tabs.

## 3. Page header and stock selector

### Page heading
- Title: `OpenBB Analysis`.
- Optional subtitle: `Research and inspect a selected stock`, only if consistent with the product's existing headings.

### Symbol/company selector
- Search by NSE symbol or company name.
- Exchange/market selector should list only markets supported by the connected provider and current implementation.
- `Analyze` loads the selected instrument's available analysis.
- Validate the symbol and show a clear no-match message when it is not recognized.
- Do not assume OpenBB supports every NSE endpoint or that every provider supplies all fields.

## 4. Selected-stock summary

Show only data fields actually returned by the configured source. Potential fields:
- Symbol and company name.
- Latest price and change.
- Volume.
- Market capitalization, if available.

Rules:
- Use the same currency, decimal precision, and gain/loss styling as the rest of Stock Home.
- Show `—` for unavailable fields, not fabricated values or misleading zeros.
- Display provider and freshness information where available.
- Do not imply real-time data unless the provider and update path confirm it.

## 5. Analysis tabs or sections

The tabs in the diagram are a proposed organization, not a requirement to implement all tabs immediately.

### Overview
- Compact summary of the selected instrument.
- Include only the metrics and summary blocks supported by the integration.

### Price & Volume
- Price chart and volume visualization only when historical/time-series data is available.
- Time-range controls should reflect the periods supported by the provider.
- Keep the chart clearly labeled with currency, time interval, and data freshness.
- Do not duplicate a full Grafana dashboard inside this page unless that is the intended design.

### Fundamentals
- Show only available company metrics (for example, valuation or profitability ratios) returned by a configured provider.
- Label metric definitions and periods where needed.
- Do not present missing metrics as zero or compute them without a defined formula and valid inputs.

### Financials
- Financial statement summaries only if the current OpenBB integration supports them.
- Make reporting period and units explicit.
- Avoid suggesting that annual, quarterly, and trailing-twelve-month values are interchangeable.

### News
- Show news only when a provider is configured and the data is available.
- Display headline, source, and timestamp; open the source using a valid destination.
- Clearly distinguish news timestamps from the time the page fetched the data.

## 6. Relationship with Stock Observability

- `stock_exporter` remains the source for the market fields it currently exposes.
- Prometheus and Grafana remain responsible for the existing metrics and observability dashboards.
- AlertManager remains responsible for the alerting workflow.
- OpenBB Analysis adds research-oriented data only where the OpenBB integration actually supplies it.
- Do not replace or silently override the existing exporter data with a different provider. If two sources provide the same field, define which source is authoritative and label provider differences where necessary.

## 7. Interactions

- Selecting a stock updates the page to that instrument.
- Changing a tab changes the visible analysis section without losing the selected symbol.
- Changing a time range reloads the chart only if supported.
- News links open the relevant article or source page.
- A full-workspace link is displayed only when its destination is configured.
- Keep this page analytical and read-only; do not add trade/order actions.

## 8. Loading, empty, error, and freshness states

- **Loading:** Preserve the panel layout and show a loading indicator in the relevant section.
- **No stock selected:** Show a prompt to search for a symbol.
- **No results:** Explain that no matching instrument was found.
- **Feature unavailable:** Explain that the provider or integration does not currently supply that section; do not show a blank chart.
- **Provider/API error:** Show a concise error and allow retry if supported.
- **Stale data:** Show the last successful update and do not label stale values as live.
- **Partial results:** Allow one panel to fail without hiding other successful panels.

## 9. Acceptance checklist

- [ ] Uses the established Stock Home application shell.
- [ ] Selected symbol remains visible as the user changes analysis sections.
- [ ] Only implemented OpenBB features and configured providers are shown.
- [ ] Data source and freshness are clear.
- [ ] No invented stock values, financial metrics, news, or provider capabilities.
- [ ] OpenBB does not replace the existing `stock_exporter`/Prometheus/Grafana/AlertManager stack.
- [ ] Unavailable panels have explicit states.
- [ ] Page remains read-only.
- [ ] Final implementation matches the approved UI image if one is provided.
