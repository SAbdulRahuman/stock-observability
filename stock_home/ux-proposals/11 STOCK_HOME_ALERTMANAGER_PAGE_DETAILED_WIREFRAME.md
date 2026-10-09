# Stock Home — Alertmanager Page UI Wireframe

**Page:** Alertmanager  
**Goal:** Bring the common Alertmanager workflows into Stock Home so users can inspect grouped alerts, understand routing/grouping, and manage silences without routinely opening the separate Alertmanager UI.

> **Scope note:** This is a detailed proposed wireframe. Implement controls only when supported by the deployed Alertmanager version, API, authentication, and backend integration. Keep the full Alertmanager UI link as a fallback for advanced or unsupported workflows.

## 1. Overall page layout

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ STOCK HOME                                                                           Updated: —   Profile    │
├─────────────────────────┬────────────────────────────────────────────────────────────────────────────────────┤
│                         │ Alertmanager                                                                         │
│ Dashboard               │ Review alert groups, silence notifications, and inspect routing status               │
│ Market Overview         │                                                                                     │
│ Stock Explorer          │ [Alerts] [Silences] [Receivers & Routes] [Status]                                    │
│ Watchlist               │                                                                                     │
│ Alerts                  │ ┌─────────────────┐ ┌─────────────────┐ ┌──────────────────┐ ┌──────────────────┐ │
│ Top Movers              │ │ Alert Groups     │ │ Active Alerts   │ │ Silenced Alerts  │ │ Pending Silences │ │
│ Sector View             │ │ —               │ │ —               │ │ —                │ │ —                │ │
│ OpenBB Analysis         │ └─────────────────┘ └─────────────────┘ └──────────────────┘ └──────────────────┘ │
│ Prometheus              │                                                                                     │
│ Alertmanager            │ ┌────────────────────────────────────────────────────────────────────────────────┐ │
│ Grafana                 │ │ Search alerts… [Severity ▼] [Status ▼] [Receiver ▼] [Clear] [Refresh]          │ │
│                         │ └────────────────────────────────────────────────────────────────────────────────┘ │
│                         │                                                                                     │
│                         │ ┌────────────────────────────────────────────────────────────────────────────────┐ │
│                         │ │ ALERT GROUPS                                                                     │ │
│                         │ ├──────────┬────────────────┬─────────────┬───────────┬────────────┬──────────────┤ │
│                         │ │ Status   │ Group / Alert  │ Severity    │ Active    │ Receiver   │ Actions      │ │
│                         │ ├──────────┼────────────────┼─────────────┼───────────┼────────────┼──────────────┤ │
│                         │ │ Firing   │ —              │ —           │ —         │ —          │ [Details]    │ │
│                         │ │ Pending  │ —              │ —           │ —         │ —          │ [Details]    │ │
│                         │ └──────────┴────────────────┴─────────────┴───────────┴────────────┴──────────────┘ │
│                         │                                                                                     │
│                         │ Selected group / alert details                                                      │
│                         │ ┌────────────────────────────────────────────────────────────────────────────────┐ │
│                         │ │ Alert name • Status • Active since                                               │ │
│                         │ │ Labels: alertname, severity, job, instance, symbol…                              │ │
│                         │ │ Annotations: summary, description, runbook URL (when present)                    │ │
│                         │ │ [Create silence] [Open in Prometheus]                                            │ │
│                         │ └────────────────────────────────────────────────────────────────────────────────┘ │
│                         │                                                                                     │
│                         │ [Open full Alertmanager UI]                                                         │
├─────────────────────────┴────────────────────────────────────────────────────────────────────────────────────┤
│ Alertmanager endpoint: configured alias • Last successful API check: —                                        │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Navigation and summary

### Shared shell
- Reuse Stock Home's header, sidebar, typography, spacing, card styles, and footer.
- Highlight **Alertmanager** in the observability navigation.
- Display a configured endpoint alias and connection state; never expose credentials or secrets.

### Summary cards
Use API-backed counts only:
- **Alert Groups:** number of groups returned by Alertmanager.
- **Active Alerts:** count of active alert instances returned by the API.
- **Silenced Alerts:** alert instances currently matched by active silences, if this count can be derived reliably.
- **Pending Silences:** silences scheduled to start in the future, if supported by the deployed version/API.

Avoid counting the same grouped alert multiple times without documenting the counting convention. Show `—` when the API call fails rather than showing zero.

## 3. Alerts tab — grouped alert inspection

### Filters
- Search alert name, label values, or group labels.
- Filter by supported status/severity values.
- Filter by receiver only if receiver/group metadata is available from the integration.
- Clear filters and refresh.
- Make clear whether filtering is performed client-side on loaded data or server-side.

### Alert group table
Suggested columns:
- **Status:** firing, silenced, or other supported status labels.
- **Group / Alert:** group labels and representative alert names.
- **Severity:** label value when the alert includes a severity label.
- **Active:** start/active-since time, if available.
- **Receiver:** resolved receiver only if it can be accurately obtained.
- **Actions:** open details; create silence if authorized.

Alertmanager's grouping model means one group can contain multiple alert instances. The UI must preserve this distinction and not present a group as a single alert.

### Detail panel
When a row is selected, show:
- Alert name and current status.
- Active start time and end time where applicable.
- Full labels and annotations.
- Group labels and fingerprint when available.
- Existing silence match/state when it can be determined.
- A link to the corresponding Prometheus alert/rule when a valid target can be identified.
- A runbook link only when one is present and trusted.

## 4. Silences tab — create, inspect, expire

```text
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ Silences                                                    [Status ▼] [Search ______]      │
│ [ + Create Silence ] [Refresh]                                                              │
├────────────┬────────────────────────┬────────────────────┬──────────────────┬───────────────┤
│ Status     │ Matchers               │ Creator            │ Starts / Ends    │ Actions       │
├────────────┼────────────────────────┼────────────────────┼──────────────────┼───────────────┤
│ Active     │ severity = warning     │ —                  │ —                │ [Details]     │
│ Pending    │ alertname = —          │ —                  │ —                │ [Details]     │
│ Expired    │ —                      │ —                  │ —                │ [Details]     │
└────────────┴────────────────────────┴────────────────────┴──────────────────┴───────────────┘
```

- List silences with status, matchers, creator, comment, start/end time, and ID where available.
- Support search/filter by status, matcher, creator, or comment.
- Allow users with the appropriate permission to create a silence.
- Allow ending an active silence only if supported and authorized; confirm before submitting.
- Expired silences may be visible for audit/context if returned by the API.

### Create Silence form
Fields:
- **Matchers:** one or more label matchers (label name, operator, value).
- **Starts:** now or a scheduled start time, if supported.
- **Ends / duration:** explicit end time or duration with a visible timezone.
- **Creator:** populated from authenticated identity where possible; do not trust a user-entered identity as proof of authorization.
- **Comment:** required or recommended reason for the silence.
- **Preview:** show the matcher set and estimated matching alerts when this can be computed accurately.
- **Create / Cancel:** validate required fields and display the result.

Safety checks:
- Clearly warn when matchers are broad, such as a matcher that could silence many alerts.
- Show the exact matchers and time window in the confirmation step.
- Never create a silence until the user explicitly confirms.
- Do not offer silence actions to users without permission.
- Use Alertmanager's supported API rather than editing internal state directly.

## 5. Receivers & Routes tab

Purpose: provide visibility into notification routing, not necessarily route editing.

- Show configured route tree and receiver names only when the deployed API/configuration integration exposes them safely.
- Display grouping labels, route matchers, and relevant timing parameters only if available.
- Show receiver destinations in a redacted form; never expose credentials, webhook tokens, passwords, or secret URL parameters.
- Clearly label configuration as read-only unless a separate configuration-management workflow exists.
- Link to configuration source or full Alertmanager UI when safe and configured.

**Important:** Do not assume every Alertmanager deployment exposes full routing configuration through its public API. If the configuration is not available safely, show an informative unavailable state rather than scraping internal pages.

## 6. Status tab

Show only supported and authorized operational information:
- Alertmanager readiness/health.
- Version/build information if available.
- Config load status or cluster status only when exposed by the deployed version and allowed by access policy.
- API connection status and last successful fetch time.

Do not expose secrets or sensitive configuration values. Keep Prometheus evaluation status distinct from Alertmanager notification/routing status.

## 7. Integration with the existing Stock Home stack

- **Prometheus:** evaluates alerting rules and exposes alert state; provide a link to the relevant rule/alert where possible.
- **Alertmanager:** groups, routes, deduplicates, inhibits, and manages silences/notification workflow.
- **Grafana:** remains the destination for dashboards and broader visualization.
- **Stock Home Alerts page:** may show the product-level alert summary, while this Alertmanager page exposes native Alertmanager group and silence workflows. Keep their roles clear.
- Use configured service URLs and a server-side backend/proxy where needed. Never hard-code environment-specific endpoints or put service credentials in browser code.

## 8. Loading, empty, error, and permissions states

- **Loading:** keep layout stable and show local loading indicators.
- **No alerts:** show a clear empty state; distinguish “no alerts” from “could not load alerts.”
- **No filter matches:** offer a clear-filters action.
- **API unavailable:** show endpoint status and last successful fetch time.
- **Stale data:** label the last successful update; do not present stale status as current.
- **Silence creation failure:** show a concise reason when safe, preserve the form values, and let the user retry.
- **Permission denied:** explain that the action is unavailable to the current user.
- **Partial failure:** allow alerts to remain visible if the silence or status request fails independently.

## 9. Typical API integration map

Use the API supported by the deployed Alertmanager version. Common v2 API resources include:

| Feature | Typical endpoint |
|---|---|
| Alert groups / alert instances | `GET /api/v2/alerts` |
| List silences | `GET /api/v2/silences` |
| Create silence | `POST /api/v2/silences` |
| Get silence by ID | `GET /api/v2/silence/{silenceID}` |
| Expire/delete silence | `DELETE /api/v2/silence/{silenceID}` |
| Readiness/health | Deployment/version-dependent health endpoints |

Verify exact routes, request/response schemas, permissions, and behavior against the deployed Alertmanager version. The browser should call a protected Stock Home backend endpoint where authentication, authorization, CSRF protections (as applicable), input validation, and audit logging can be enforced.

## 10. Out of scope unless explicitly approved

- Editing routes or receiver configuration directly from this page.
- Viewing or editing receiver secrets.
- Restarting or reloading Alertmanager.
- Arbitrary administrative API access.
- Claiming a notification was delivered merely because an alert exists.
- Removing confirmation or permission checks for silence creation/expiration.

## 11. Acceptance checklist

- [ ] Uses the shared Stock Home shell and active navigation state.
- [ ] Shows Alertmanager alert groups and preserves group-versus-alert semantics.
- [ ] Supports filtering and inspection of alert labels and annotations.
- [ ] Lists silences and supports create/expire actions only when authorized.
- [ ] Silence creation shows exact matchers and time window before confirmation.
- [ ] Receivers/routes are read-only and secrets are redacted.
- [ ] Prometheus alert evaluation is distinguished from Alertmanager routing/notification behavior.
- [ ] Loading, empty, error, stale-data, and permission-denied states are defined.
- [ ] API access is protected and credentials are not exposed in the browser.
- [ ] Full Alertmanager UI remains available as a fallback for unsupported workflows.
- [ ] No fake alert rows, fake receiver names, or nonfunctional controls appear in production.
