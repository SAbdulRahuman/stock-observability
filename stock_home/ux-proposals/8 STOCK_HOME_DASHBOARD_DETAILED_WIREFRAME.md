# Stock Home — Dashboard UI Detailed Wireframe

**Page:** Dashboard / Stock Home  
**Purpose:** Provide a concise, read-only overview of market conditions, watchlist movement, active alerts, and links into the underlying observability tools.

> **Design constraint:** Keep this page aligned with the approved Stock Home dashboard visual. This document defines layout and behavior, not new product scope. Values shown as `—` are placeholders; do not hard-code sample market values. If the approved screenshot differs, treat the screenshot as the source of truth.

## 1. Page layout

```text
┌────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ STOCK HOME                                           Market status • Last updated: —       Profile │
├──────────────────────┬─────────────────────────────────────────────────────────────────────────────┤
│                      │ Dashboard                                                                   │
│  [Stock Home logo]   │ Market and stock observability at a glance                                  │
│                      │                                                                             │
│  ▸ Dashboard         │ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐ ┌──────────┐│
│    Market Overview   │ │ NIFTY 50         │ │ SENSEX           │ │ Watchlist         │ │ Alerts   ││
│    Stock Explorer    │ │ Value: —         │ │ Value: —          │ │ Stocks: —         │ │ Active: —││
│    Watchlist         │ │ Change: —        │ │ Change: —         │ │ Gainers/Losers: — │ │ Firing: —││
│    Alerts            │ └──────────────────┘ └──────────────────┘ └──────────────────┘ └──────────┘│
│    Top Movers        │                                                                             │
│    Sector View       │ ┌────────────────────────────────────────────┐ ┌──────────────────────────┐│
│                      │ │ Market Overview                            │ │ Watchlist Snapshot       ││
│  ──────────────────  │ │                                            │ │                          ││
│  OBSERVABILITY       │ │ Index / market trend visualization         │ │ Symbol  Price  Change    ││
│    Prometheus        │ │ (only if present in approved UI)           │ │ —       —      —         ││
│    Grafana           │ │                                            │ │ —       —      —         ││
│    AlertManager      │ │ [Open Market Overview]                     │ │ [View Watchlist]         ││
│                      │ └────────────────────────────────────────────┘ └──────────────────────────┘│
│                      │                                                                             │
│                      │ ┌────────────────────────────────────────────┐ ┌──────────────────────────┐│
│                      │ │ Recent / Active Alerts                     │ │ Top Movers               ││
│                      │ │                                            │ │                          ││
│                      │ │ Severity  Alert        Stock   Started     │ │ Top Gainers / Top Losers ││
│                      │ │ —         —            —       —           │ │ Symbol  Change  Price    ││
│                      │ │ —         —            —       —           │ │ —       —       —       ││
│                      │ │ [View All Alerts]                          │ │ [View Top Movers]        ││
│                      │ └────────────────────────────────────────────┘ └──────────────────────────┘│
│                      │                                                                             │
│                      │ ┌─────────────────────────────────────────────────────────────────────────┐│
│                      │ │ Service / Data Source Status (only if shown in approved UI)             ││
│                      │ │ stock_exporter: —   Prometheus: —   AlertManager: —   Grafana: —        ││
│                      │ └─────────────────────────────────────────────────────────────────────────┘│
├──────────────────────┴─────────────────────────────────────────────────────────────────────────────┤
│ Data source: — • Market-data timestamp: — • Read-only dashboard                                   │
└────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Global shell

### Header
- Product label/logo: `STOCK HOME`.
- Market status and/or last-updated timestamp, using the established header pattern.
- Profile or settings entry only if present in the approved design.
- Avoid duplicate timestamps if the header and footer already show the same information.

### Left navigation
- Dashboard (active).
- Market Overview.
- Stock Explorer.
- Watchlist.
- Alerts.
- Top Movers.
- Sector View.
- Observability links: Prometheus, Grafana, AlertManager, if these are part of the established navigation.
- Preserve the existing ordering, labels, icons, spacing, and grouping from the approved Stock Home UI. Do not add new navigation items solely because they appear in this proposal.

### Main content
- Page title: `Dashboard` or the exact title used in the approved UI.
- Optional one-line description only if it appears in the visual reference.
- Use the existing content width, spacing, typography, border, radius, and card style.

## 3. Dashboard sections

### A. Summary cards
Represent only the KPIs shown in the approved design. Candidate content in the sketch above:
- **NIFTY 50:** latest index value and absolute/percentage movement.
- **SENSEX:** latest index value and absolute/percentage movement.
- **Watchlist:** tracked stock count and/or gainers-versus-losers summary.
- **Alerts:** active/firing alert count.

Rules:
- Do not add more cards than the reference shows.
- Show units and change conventions consistently.
- Positive and negative changes should use consistent semantic colors and also retain readable signs/text.
- If a data point is unavailable, display `—` and a clear stale/unavailable status rather than `0`.

### B. Market overview panel
- Show the exact visualization present in the approved UI (for example, an index trend chart only if one is actually shown).
- Keep chart title, timeframe controls, legend, and axis labels only when visible in the reference.
- If the dashboard uses a compact summary card with a link instead of a chart, retain the compact card; do not expand it into a chart.
- Link to **Market Overview** using the app's established navigation.

### C. Watchlist snapshot
- Compact table of the few rows shown in the design.
- Suggested fields when present: symbol, latest price, percentage change.
- Use the same formatting and color conventions as the Watchlist page.
- Provide a clear link to **Watchlist**.
- Do not make the dashboard table a second full watchlist page.

### D. Recent / active alerts
- Show the same number of rows as the approved screenshot.
- Suggested fields when present: severity, alert name, related symbol, and start time.
- Severity must be text-readable as well as color-coded.
- Link to **Alerts** for the complete list.
- Do not invent alert names or show example alerts as real data.

### E. Top movers
- Show only the compact gainers/losers content visible in the reference.
- Keep symbol, percentage change, and price only if those fields appear in the design.
- Link to **Top Movers** for the full view.
- Do not add prediction scores or recommendations unless explicitly included in the approved design.

### F. Service/data status
- Include this section only if it appears in the approved UI.
- If included, show statuses for existing components such as `stock_exporter`, Prometheus, AlertManager, and Grafana only when reliable health data is available.
- Do not imply that a service is healthy merely because its URL exists.
- Keep stock-market data freshness separate from application/service health.

## 4. Navigation and interaction

- Dashboard cards and section links navigate to their corresponding Stock Home pages.
- Prometheus, Grafana, and AlertManager links open the configured destinations; do not hard-code environment-specific URLs in the UI.
- If the dashboard is intended to be read-only, do not add editing, trading, order placement, or alert-management actions.
- Refresh behavior should match the existing app. If manual refresh is shown, indicate loading while refreshing and update the timestamp only after a successful refresh.
- Do not make cards clickable unless the destination is clear and consistent with the rest of the app.

## 5. Data states

### Loading
- Preserve card and panel dimensions where possible.
- Use the app's established loading treatment.

### Empty
- For a section with no rows, show a short empty-state message and a link to the relevant page when appropriate.

### Error / unavailable
- Distinguish “data unavailable” from a legitimate zero value.
- Keep the rest of the dashboard usable if one data source fails.
- Avoid dumping raw stack traces or API errors into the UI.

### Stale data
- Show the last successful update timestamp or stale label.
- Do not present cached data as live.
- Keep market-data freshness distinct from Prometheus scrape or service health.

## 6. Responsive behavior

- Desktop: preserve the approved multi-column layout.
- Narrow screens: stack panels in a clear reading order without changing their meaning.
- Tables may scroll horizontally or reduce to the columns already defined by the design.
- Sidebar collapse behavior should follow the existing app shell.

## 7. Accessibility and visual consistency

- Use semantic headings in order.
- Do not rely on color alone for gain/loss or alert severity.
- Ensure text contrast and keyboard focus for links and controls.
- Keep icon meaning consistent across pages.
- Align all card padding, table density, and heading styles with the approved Stock Home pages.

## 8. Acceptance checklist

- [ ] Dashboard uses the same shared shell as the other Stock Home pages.
- [ ] Dashboard is visibly selected in the sidebar.
- [ ] Only approved KPI cards and panels are present.
- [ ] No fake market values, alert examples, or health statuses are displayed.
- [ ] Each section links to the corresponding full page where appropriate.
- [ ] External observability URLs come from configuration.
- [ ] Loading, empty, error, and stale-data states are defined.
- [ ] Dashboard remains read-only if that is the established product behavior.
- [ ] The final implementation matches the approved UI image rather than expanding the scope of this wireframe.
