# Stock Home — Settings Page UI Wireframe

**Page:** Settings  
**Purpose:** Configure Stock Home application behavior, market defaults, display preferences, notifications, dashboard widgets, integrations, data retention, and export/backup actions.

**Design reference:** This wireframe follows the supplied Settings UI image. Preserve its three-column card layout, dark left sidebar, white content area, blue active states, compact form controls, and footer status bar. Values shown below reflect the reference image and should be validated against actual supported settings before implementation.

## 1. Overall page layout

```text
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ STOCK HOME  [Search symbol, company, or navigate…]                 Market Open 09:45 AM IST  [Notifications] [Help] │
├──────────────────────┬─────────────────────────────────────────────────────────────────────────────────────────────┤
│ Stock Home           │ ⚙ Settings                                                                                  │
│ Stock Observability   │ Configure Stock Home, data sources, alerts, and user preferences                            │
│                      │                                                                                             │
│ Dashboard            │ [General] [Data Sources] [Alert Rules] [Watchlist] [User Preferences] [Integrations] [System]│
│ Watchlist             │                                                                                             │
│ Alerts                │ ┌─────────────────────────┐ ┌─────────────────────────┐ ┌─────────────────────────────────┐ │
│ Top Movers            │ │ Application Settings    │ │ Market Settings         │ │ Data Display Settings            │ │
│ Sector View           │ │                         │ │                         │ │                                 │ │
│ Stock Explorer        │ │ Application Name        │ │ Default Exchange        │ │ Price Format                    │ │
│ OpenBB Analysis       │ │ Theme: Light/Dark/System │ │ Default Index           │ │ Number Format                   │ │
│ Prometheus            │ │ Language                │ │ Market Hours             │ │ Change Display                  │ │
│ Alertmanager          │ │ Time Zone               │ │ Holiday Calendar         │ │ Decimal Places                  │ │
│ Settings (active)     │ │ Refresh Interval        │ │                         │ │ Table Rows per Page             │ │
│                      │ └─────────────────────────┘ │                         │ │ Show Sparkline [toggle]         │ │
│                      │                             │                         │ │ Highlight Price Change [toggle] │ │
│                      │ ┌─────────────────────────┐ └─────────────────────────┘ └─────────────────────────────────┘ │
│                      │ │ Notification Settings   │ ┌─────────────────────────┐ ┌─────────────────────────────────┐ │
│                      │ │ Enable Notifications    │ │ Dashboard Settings      │ │ API & Integration Settings       │ │
│                      │ │ Price Alerts [toggle]   │ │ Default Dashboard View  │ │ Zerodha Kite API  Configured [Test]│
│                      │ │ Rule Alerts [toggle]     │ │ Widget Layout           │ │ OpenBB             Configured [Test]│
│                      │ │ Market Open/Close       │ │ Show Index Cards        │ │ Prometheus         Configured [Test]│
│                      │ │ Daily Summary            │ │ Show Recent Alerts      │ │ Alertmanager       Configured [Test]│
│                      │ │ Notification Sound      │ │ Show Top Movers         │ │ Grafana            Configured [Test]│
│                      │ └─────────────────────────┘ │ Show Watchlist Snapshot │ │                                 │
│                      │                             └─────────────────────────┘ └─────────────────────────────────┘ │
│ Quick Links          │                                                                                             │
│ Grafana ↗            │ ┌─────────────────────────┐ ┌─────────────────────────┐ ┌─────────────────────────────────┐ │
│ Prometheus ↗         │ │ Data Retention          │ │ Backup & Export         │ │ Danger Zone                     │ │
│ Alertmanager ↗       │ │ Prometheus Retention    │ │ Export Watchlists       │ │ [Reset All Settings]            │ │
│ OpenBB ↗             │ │ Alert History Retention │ │ Export Alert Rules      │ │ [Clear Cache & Local Data]      │ │
│                      │ │                         │ │ Export Settings         │ │                                 │ │
│                      │ │                         │ │ Import Watchlists       │ │                                 │ │
│                      │ └─────────────────────────┘ └─────────────────────────┘ └─────────────────────────────────┘ │
├──────────────────────┴─────────────────────────────────────────────────────────────────────────────────────────────┤
│ Stock Home v1.0.0 | Connected to: Prometheus • Grafana • Alertmanager | Last Updated: —     Documentation GitHub Support│
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Shared shell

### Header
- Product identity: **Stock Home — Stock Observability Platform**.
- Global search field with placeholder: `Search symbol, company, or navigate…`.
- Market status chip, e.g. `Market Open`, and market-local time.
- Notification icon with unread count, help icon, and user/avatar control, matching the reference image.

### Sidebar
- Navigation entries in the order shown in the reference: Dashboard, Watchlist, Alerts, Top Movers, Sector View, Stock Explorer, OpenBB Analysis, Prometheus, Alertmanager, Settings.
- Highlight **Settings** in blue as the active page.
- A **Quick Links** section contains Grafana, Prometheus, Alertmanager, and OpenBB, each opening its configured destination.
- Use the existing icon style and sidebar width. Do not add new items without design approval.

### Settings tabs
- General (active in the reference image)
- Data Sources
- Alert Rules
- Watchlist
- User Preferences
- Integrations
- System

Tabs should switch the settings category while retaining the same page shell. The General tab contains the cards described below.

## 3. General tab — three-column settings cards

### A. Application Settings

**Subtitle:** General application configuration

| Setting | UI control | Reference value / behavior |
|---|---|---|
| Application Name | Text input | `Stock Home`; used in header/browser title |
| Theme | Three-option selector | Light / Dark / System; System selected in the reference |
| Language | Dropdown | English |
| Time Zone | Dropdown | `(GMT+05:30) Asia/Kolkata` |
| Refresh Interval | Dropdown | `30 seconds`; used for automatic refresh where supported |

Helper text should be shown below controls when visible in the reference. Theme, language, timezone, and refresh settings should persist only if the application supports persistence.

### B. Market Settings

**Subtitle:** Default market and display preferences

| Setting | UI control | Reference value / behavior |
|---|---|---|
| Default Exchange | Dropdown | `NSE (India)` |
| Default Index | Dropdown | `NIFTY 50` |
| Market Hours | Start/end time controls | `09:15` to `15:30` |
| Holiday Calendar | Dropdown | `NSE Official Calendar` |

Market hours and holiday calendar are used for market status and filtering only if the application implements these behaviors. Display the correct timezone for the selected market.

### C. Data Display Settings

**Subtitle:** Configure how data is shown

| Setting | UI control | Reference value / behavior |
|---|---|---|
| Price Format | Dropdown | `₹ (INR)` |
| Number Format | Dropdown | Indian format, e.g. `1,23,456.78` |
| Change Display | Segmented control | Absolute / Percentage; Absolute selected in the reference |
| Decimal Places | Minus/value/plus control | `2` |
| Table Rows per Page | Dropdown | `50` |
| Show Sparkline | Toggle | On |
| Highlight Price Change | Toggle | On |

Apply formatting consistently across Stock Explorer, Watchlist, Top Movers, Sector View, and other pages. A display preference must not alter the underlying numeric data.

### D. Notification Settings

**Subtitle:** Browser and in-app notifications

| Setting | UI control | Reference value / behavior |
|---|---|---|
| Enable Notifications | Toggle | On |
| Price Alerts | Toggle | On |
| Rule Alerts | Toggle | On |
| Market Open/Close | Toggle | On |
| Daily Summary | Toggle | Off |
| Notification Sound | Dropdown + play/test control | `Default` |

- Explain when browser permission is required.
- Respect browser/OS notification permissions.
- Do not claim a notification was delivered just because the setting is enabled.
- The play/test control should provide a preview only if implemented.

### E. Dashboard Settings

**Subtitle:** Configure dashboard layout and widgets

| Setting | UI control | Reference value / behavior |
|---|---|---|
| Default Dashboard View | Dropdown | `Main Dashboard` |
| Widget Layout | Three visual layout options | First layout selected in the reference |
| Show Index Cards | Toggle | On |
| Show Recent Alerts | Toggle | On |
| Show Top Movers | Toggle | On |
| Show Watchlist Snapshot | Toggle | On |

Use these settings only for dashboard elements that exist. Changes should affect the Dashboard page, not other pages.

### F. API & Integration Settings

**Subtitle:** Configure external services and APIs

| Integration | Status in reference | Controls |
|---|---|---|
| Zerodha Kite API | Configured | `Test` button, details chevron |
| OpenBB | Configured | `Test` button, details chevron |
| Prometheus | Configured | `Test` button, details chevron |
| Alertmanager | Configured | `Test` button, details chevron |
| Grafana | Configured | `Test` button, details chevron |

- Use status badges such as **Configured**, **Connected**, **Error**, or **Not configured**, based on actual test results.
- A saved endpoint is not proof that the service is healthy.
- The `Test` action checks connectivity/authentication safely and returns a timestamped success/failure result.
- Integration details open a sub-panel or form with the configured endpoint and supported settings.
- Mask API tokens and secrets. Never return secrets to the browser after saving.
- Use a protected backend for connectivity tests; do not embed credentials in client-side code.

### G. Data Retention

**Subtitle:** Configure data storage and retention policies

| Setting | UI control | Reference value / behavior |
|---|---|---|
| Prometheus Retention | Dropdown | `30 days` |
| Alert History Retention | Dropdown | `90 days` |

- Clarify which system owns each retention setting.
- Prometheus retention should be changed through the supported deployment/configuration mechanism, not merely stored as a UI preference.
- Alert history retention applies only if Stock Home persists its own alert history.
- Show whether a change takes effect immediately or requires a service restart/redeployment.

### H. Backup & Export

**Subtitle:** Export and back up application data

Buttons shown in the reference:
- `Export Watchlists`
- `Export Alert Rules`
- `Export Settings`
- `Import Watchlists` (JSON format)

Behavior:
- Export only data the current user is authorized to access.
- Use explicit file formats and clear filenames.
- Validate imported JSON before applying it.
- Preview import changes and ask for confirmation before overwriting existing data.
- Report success/failure clearly; never claim an export completed until the file is generated.

### I. Danger Zone

**Subtitle:** These actions are irreversible

Red-outline destructive controls:
- `Reset All Settings`
- `Clear Cache & Local Data`

Behavior:
- Keep this card visually distinct from normal settings.
- Require a confirmation dialog before either action.
- Explain exactly what will be reset or deleted.
- `Reset All Settings` restores supported preferences to defaults; it should not silently delete watchlists, alert rules, or integration credentials unless explicitly stated.
- `Clear Cache & Local Data` should specify the exact browser/app data cleared.
- Do not trigger destructive actions from a single accidental click.

## 4. Footer status bar

- Product name and version, e.g. `Stock Home v1.0.0`.
- Connected service names based on current successful checks.
- Last updated timestamp.
- Documentation, GitHub, and Support links when configured.

The timestamp should reflect a real update event, not a hard-coded date.

## 5. General page interactions

- Keep settings grouped into cards with headings, icons, and concise subtitles.
- Provide a consistent save/apply pattern if settings are persisted. If changes auto-save, show a clear saved/error indicator.
- Validate inputs before saving; show field-level errors.
- Avoid losing unsaved edits when switching tabs.
- Test buttons show progress and the actual result.
- Disabled or unsupported settings should be hidden or clearly marked as unavailable rather than appearing functional.
- Maintain keyboard focus, readable contrast, labels for every control, and text descriptions for toggle state.

## 6. Responsive behavior

- Desktop: preserve the reference's three-column card layout.
- Medium screens: flow to two columns.
- Narrow screens: stack cards in a single column.
- Keep controls usable and prevent horizontal overflow.
- Sidebar collapse behavior should match the rest of Stock Home.

## 7. Acceptance checklist

- [ ] Page layout closely matches the supplied Settings reference image.
- [ ] Dark sidebar, top bar, tabs, three-column card grid, and footer are preserved.
- [ ] All nine cards/sections are represented: Application, Market, Data Display, Notification, Dashboard, API & Integration, Data Retention, Backup & Export, Danger Zone.
- [ ] Reference values are treated as defaults/placeholders to validate against actual implementation.
- [ ] Integration status reflects real connectivity checks.
- [ ] API credentials and secrets are protected.
- [ ] Import/export validates input and reports actual outcomes.
- [ ] Destructive actions require explicit confirmation.
- [ ] Loading, validation, save failure, and unavailable states are defined.
- [ ] No nonfunctional controls are presented as working features.
