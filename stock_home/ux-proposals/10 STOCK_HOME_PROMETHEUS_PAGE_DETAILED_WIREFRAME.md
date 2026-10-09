# Stock Home — Prometheus Page Wireframe

**Page:** Prometheus  
**Goal:** Bring the important Prometheus UI workflows into Stock Home so users can inspect targets, query metrics, and review rule/alert state without routinely opening the separate Prometheus UI.

**Product constraint:** This is a detailed proposed wireframe, not a claim that the current app already exposes every endpoint. Implement each section through the Prometheus HTTP API or supported app backend. Preserve Prometheus semantics and permissions. A feature should be hidden or clearly marked unavailable if the deployed Prometheus version/API does not support it.

## 1. Page layout

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ STOCK HOME                                                                                 Data updated: —   │
├─────────────────────────┬────────────────────────────────────────────────────────────────────────────────────┤
│                         │ Prometheus                                                                          │
│ Dashboard               │ Query metrics, inspect targets, rules, and alert state                              │
│ Market Overview         │                                                                                     │
│ Stock Explorer          │ [Overview] [Query] [Targets] [Rules] [Alerts] [Status]                              │
│ Watchlist               │                                                                                     │
│ Alerts                  │ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌────────────────┐               │
│ Top Movers              │ │ Targets UP   │ │ Targets DOWN │ │ Rules        │ │ Firing Alerts  │               │
│ Sector View             │ │ — / —        │ │ —            │ │ —            │ │ —              │               │
│ OpenBB Analysis         │ └──────────────┘ └──────────────┘ └──────────────┘ └────────────────┘               │
│ Prometheus              │                                                                                     │
│ Grafana                 │ ┌────────────────────────────────────────────────────────────────────────────────┐ │
│ AlertManager            │ │ Prometheus connection: [Configured endpoint ▼]   Status: —   [Refresh]         │ │
│                         │ └────────────────────────────────────────────────────────────────────────────────┘ │
│                         │                                                                                     │
│                         │ ┌────────────────────────────────────────────────────────────────────────────────┐ │
│                         │ │ MAIN CONTENT AREA — changes with selected tab                                  │ │
│                         │ │                                                                                │ │
│                         │ │ Overview: system health, target/rule/alert summaries and recent query examples │ │
│                         │ │ Query: query editor, execute/clear controls, results and table/graph toggle     │ │
│                         │ │ Targets: scrape target table, labels, scrape status, last scrape/error details │ │
│                         │ │ Rules: rule groups, recording/alerting rules, health and last evaluation       │ │
│                         │ │ Alerts: active/pending/firing alert instances and labels/annotations            │ │
│                         │ │ Status: build/runtime information and supported configuration/status details   │ │
│                         │ └────────────────────────────────────────────────────────────────────────────────┘ │
│                         │                                                                                     │
│                         │ [Open full Prometheus UI] (fallback link; retain for advanced/unsupported tasks)  │
├─────────────────────────┴────────────────────────────────────────────────────────────────────────────────────┤
│ Stock Home • Prometheus endpoint: configured alias • Last successful API check: —                            │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Shared shell and connection status

- Reuse Stock Home header, sidebar, typography, spacing, card styles, and footer.
- Highlight **Prometheus** in the observability navigation.
- Show a configured endpoint alias and connection health; avoid exposing secrets or internal credentials.
- Use a server-side proxy/backend for Prometheus API calls when required by CORS, authentication, or network boundaries. Do not place credentials in browser code.
- The `Open full Prometheus UI` link is a fallback for workflows not yet replicated, advanced debugging, and version-specific features. The objective is to avoid needing it for routine workflows, not to pretend every native UI feature has been reproduced.

## 3. Prometheus overview tab

### Summary cards
- **Targets UP:** count of healthy scrape targets / total targets.
- **Targets DOWN:** count of targets currently reporting scrape errors.
- **Rules:** count of loaded rules, with separate alerting/recording counts if available.
- **Firing Alerts:** current firing alert instances.
- Use `—` when the API is unavailable; do not show zero for a failed request.
- Do not confuse Prometheus alert state with AlertManager notification delivery state.

### Overview panels
- **Target health summary:** compact count by health state.
- **Alert state summary:** pending and firing counts if returned by the API.
- **Recent query examples:** optional list of saved/documented PromQL examples relevant to stock observability, only if implemented.
- **API freshness:** last successful fetch timestamp.

## 4. Query tab — native-style PromQL workflow

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ Query                                                                                       │
│                                                                                             │
│ [Metric browser / autocomplete]                                                              │
│ ┌─────────────────────────────────────────────────────────────────────────────────────────┐ │
│ │ stock_price{symbol="RELIANCE"}                                                          │ │
│ └─────────────────────────────────────────────────────────────────────────────────────────┘ │
│ [Execute] [Clear] [Use example ▼]          Evaluation time: [Now ▼]                          │
│                                                                                             │
│ Result view: (•) Table  ( ) Graph             Range: [Last 1 hour ▼]  Step: [auto ▼]         │
│ ┌─────────────────────────────────────────────────────────────────────────────────────────┐ │
│ │ Query results                                                                            │ │
│ │ metric labels / timestamp / value, or graph visualization for range queries               │ │
│ │                                                                                         │ │
│ └─────────────────────────────────────────────────────────────────────────────────────────┘ │
│ [Copy query] [Copy results]                                                                 │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Query functionality
- Support instant queries and range queries through the Prometheus query APIs.
- Provide an editor for PromQL, execute, clear, and loading/error states.
- Offer metric-name autocomplete/browser only if the required API and UI support are implemented.
- Support table results for instant queries and graph/time-series results for range queries.
- Provide configurable evaluation time, time range, and step where appropriate.
- Show query duration, result count, and API errors when available.
- Allow copying the query and optionally exporting results in a supported format.
- Include a few clearly labeled example queries only when their metric names are confirmed in the deployed stock exporter. Examples must not be presented as guaranteed to exist.
- Never run arbitrary user-supplied PromQL with elevated credentials or bypass access controls. Apply authentication, authorization, rate limits, and query timeouts at the backend.

## 5. Targets tab — native target inspection

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ Targets                                                       [All states ▼] [Search ____]   │
│ [Refresh]                                                                                   │
├──────────────┬────────────┬────────────────────┬───────────────────┬────────────────────────┤
│ State        │ Job        │ Instance           │ Last scrape       │ Duration / Error       │
├──────────────┼────────────┼────────────────────┼───────────────────┼────────────────────────┤
│ UP / DOWN    │ —          │ —                  │ —                 │ —                      │
│ UP / DOWN    │ —          │ —                  │ —                 │ —                      │
└──────────────┴────────────┴────────────────────┴───────────────────┴────────────────────────┘

Select a target to inspect:
  • Discovered labels
  • Target labels / scrape pool
  • Last scrape time and duration
  • Last scrape error, if any
```

- Source target information from the Prometheus targets API.
- Support filtering by state, job, or instance.
- Clearly distinguish active and dropped targets if the API data is included.
- Show full label sets in a detail drawer/panel rather than overloading the table.
- Avoid showing sensitive labels without considering access control.

## 6. Rules tab — alerting and recording rules

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ Rules                                         [All groups ▼] [Alerting ▼] [Search ______]   │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ Rule group: —   Evaluation interval: —   Last evaluation: —   Health: —                     │
├────────────┬───────────────────────┬───────────────┬───────────────────┬────────────────────┤
│ Type       │ Rule name             │ State         │ Last evaluation   │ Health / Error     │
├────────────┼───────────────────────┼───────────────┼───────────────────┼────────────────────┤
│ Alerting   │ —                     │ inactive/...  │ —                 │ —                  │
│ Recording  │ —                     │ —             │ —                 │ —                  │
└────────────┴───────────────────────┴───────────────┴───────────────────┴────────────────────┘

Select a rule to inspect:
  • Expression (PromQL)
  • Labels and annotations
  • Current alert instances, where applicable
  • Last evaluation / health / evaluation error
```

- Load loaded rule groups and rule state from the Prometheus rules API.
- Support filters for group, rule type, state, and text.
- Distinguish rule health from alert state.
- Display expressions read-only unless a separate, safe rule-management workflow is explicitly implemented.
- Do not imply that editing a displayed rule will update Prometheus configuration.

## 7. Alerts tab — Prometheus alert state

- Display alert instances from Prometheus's alerts API.
- Provide filters for state (`inactive`, `pending`, `firing`) and label/search terms.
- Table columns: state, alert name, active since, key labels, and annotations/summary where available.
- A detail view should show the full label and annotation sets.
- Link to AlertManager for grouping, silencing, routing, and notification workflows; these are not equivalent to Prometheus's alert evaluation.
- Do not add silence or notification controls here unless explicitly integrated with AlertManager and permissioned.

## 8. Status tab — supported server information

Show only information exposed by supported APIs in the deployed Prometheus version, such as:
- Version/build information.
- Runtime or readiness/health status.
- Selected configuration/status details if explicitly supported and access-controlled.

Do not expose raw configuration, credentials, environment details, or sensitive operational metadata to unauthorized users. Treat API capabilities as version-dependent and label unsupported information clearly.

## 9. Common interactions and states

- **Refresh:** re-fetch data; update “last successful API check” only after success.
- **Search/filter:** filter the current table without losing the selected tab.
- **Details:** open a side panel or expandable row for labels, expressions, annotations, or errors.
- **Loading:** keep page structure stable and show a local loading state.
- **Empty:** explain that no targets/rules/alerts/results match the current filter.
- **Connection failure:** show endpoint health and a concise error; keep other Stock Home pages usable.
- **Stale data:** show the last successful fetch timestamp and avoid labeling the data as current.
- **Permissions:** disable or hide unsupported actions; routine inspection can be read-only.
- **Responsive:** tables may scroll horizontally on narrow screens; query editor remains usable without overflowing the page.

## 10. API mapping (implementation guide)

Use the Prometheus HTTP API supported by the deployed version. Common endpoints include:

| Page feature | Typical API |
|---|---|
| Readiness/health | `/-/ready`, `/-/healthy` (deployment/version dependent) |
| Build information | `/api/v1/status/buildinfo` |
| Runtime information | `/api/v1/status/runtimeinfo` |
| Configuration status | `/api/v1/status/config` (sensitive; protect access) |
| Instant query | `/api/v1/query` |
| Range query | `/api/v1/query_range` |
| Metric names | `/api/v1/label/__name__/values` |
| Target inspection | `/api/v1/targets` |
| Rules | `/api/v1/rules` |
| Alerts | `/api/v1/alerts` |

Verify availability, authentication, response shapes, and version behavior against the deployed Prometheus release. Use a backend proxy where needed. Do not call unsupported endpoints or expose protected endpoints directly to the browser.

## 11. Out of scope unless separately approved

- Editing Prometheus scrape configuration or rule files from the UI.
- Reloading/restarting Prometheus.
- Silencing alerts or managing notification routes without AlertManager integration.
- Arbitrary administrative operations.
- Replacing Grafana's dashboarding and visualization workflows in full.
- Reproducing every version-specific Prometheus native UI feature.

## 12. Acceptance checklist

- [ ] Stock Home shell and observability navigation are consistent.
- [ ] Overview, Query, Targets, Rules, Alerts, and Status tabs are provided where APIs support them.
- [ ] PromQL instant and range queries work with safe limits and clear errors.
- [ ] Targets show state, job, instance, last scrape, duration, and error details where available.
- [ ] Rules show type, state, health, expression, and evaluation details where available.
- [ ] Prometheus alert states are clearly distinguished from AlertManager notification/silence functions.
- [ ] API failure, empty, loading, and stale states are implemented.
- [ ] Sensitive endpoints and credentials are protected.
- [ ] A fallback link to the full Prometheus UI remains available for advanced or unsupported tasks.
- [ ] No fake values or unimplemented controls are displayed as functional.
