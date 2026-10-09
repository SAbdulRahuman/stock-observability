# stock_home — Alerts Page Wireframe

**Scope:** Alerts page only  
**Reference:** Previously generated stock_home Alerts UI image  
**Goal:** Preserve the reference screenshot's layout and information hierarchy. This is a structural wireframe, not a redesign.

---

## 1. Page layout

```text
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ HEADER                                                                                                             │
│ stock_home  ☰  [ Search stocks, indices, sectors... ]       Date / Time   ● Live Market   ⚙  🔔12  [AS]           │
├────────────────────────┬───────────────────────────────────────────────────────────────────────────────────────────┤
│ SIDEBAR                │ ALERTS                                                                                   │
│                        │ Monitor market alerts, investigate triggers, and manage alert rules                      │
│ 🏠 Home                │                                               [Create Alert +] [Export] [⋮]             │
│ 📈 Market Overview     ├───────────────────────────────────────────────────────────────────────────────────────────┤
│ ♡ Watchlist            │ SUMMARY CARDS                                                                            │
│ 🔔 Alerts          ◀   │ ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌────────────────────────────┐ │
│ ↗ Top Movers           │ │ TOTAL ALERTS   │ │ CRITICAL       │ │ WARNING        │ │ INFO                       │ │
│ ◈ Sector View          │ │ 128            │ │ 12             │ │ 36             │ │ 80                         │ │
│ 🔎 Stock Explorer      │ │ +8 today       │ │ 3 new          │ │ 5 new          │ │ 0 new                      │ │
│ ▦ Dashboards           │ └────────────────┘ └────────────────┘ └────────────────┘ └────────────────────────────┘ │
│ ◈ OpenBB Analysis      │                                                                                           │
│ 🔵 Prometheus      ↗   │ FILTER / SEARCH BAR                                                                       │
│ 🟠 Grafana         ↗   │ [Search alerts or symbols...] [Severity ▼] [Status ▼] [Source ▼] [Time range ▼] [Reset] │
│ 🔴 AlertManager    ↗   ├──────────────────────────────────────────────────────────────┬────────────────────────────┤
│ ⚙ Settings             │ ALERTS TABLE                                                │ SELECTED ALERT             │
│                        │ ☐ Severity Symbol Rule          Triggered       Status      │ 🔴 CRITICAL • FIRING       │
│ ┌──────────────────┐   │ ☐ 🔴 RELIANCE R1 Breakout      09:41:12        FIRING      │ RELIANCE                   │
│ │ Market Status    │   │ ☐ 🔴 NIFTY50   Rapid Move       09:40:08        FIRING      │ R1 Breakout                │
│ │ ● OPEN            │   │ ☐ 🟠 INFY      Volume Spike    09:35:21        FIRING      │                            │
│ │ NSE               │   │ ☐ 🟠 SBIN      Price Threshold 09:28:44        FIRING      │ Price crossed R1 level     │
│ │ 09:15–03:30 PM    │   │ ☐ 🟠 HCLTECH   Price Drop      09:22:03        FIRING      │                            │
│ │ Live via Kite API │   │ ☐ 🔵 TCS       RSI Threshold   09:18:51        RESOLVED    │ Current Price   ₹2,842.50  │
│ └──────────────────┘   │ ☐ 🔵 BANKNIFTY Breakout        09:15:10        ACKNOWLEDGED│ Trigger Level   ₹2,840.00  │
│                        │ ...                                                         │ Started        09:41:12    │
│                        │                                                             │ Duration       00:04:32    │
│                        │ [☐ Select all] [Acknowledge] [Resolve] [Silence]             │ Source         Prometheus  │
│                        │                                                             │ Rule          NSE_R1_Breakout│
│                        │ Showing 1–10 of 128      [‹] 1 2 3 ... 13 [›]               │                            │
│                        │                                                             │ PRICE CHART                │
│                        │                                                             │ price line + R1 threshold  │
│                        │                                                             │                            │
│                        │                                                             │ [Acknowledge] [Silence]    │
│                        │                                                             │ [Open in Grafana]          │
│                        ├──────────────────────────────────────────────────────────────┴────────────────────────────┤
│                        │ LOWER PANEL (optional if present in reference viewport)                                  │
│                        │ ┌─────────────────────────────────────┐ ┌─────────────────────────────────────────────┐ │
│                        │ │ ALERT ACTIVITY                      │ │ ALERT COUNTS OVER TIME                     │ │
│                        │ │ 09:41 RELIANCE fired                │ │ compact time-series chart                 │ │
│                        │ │ 09:35 INFY volume spike              │ │ critical / warning / info series          │ │
│                        │ │ 09:28 SBIN threshold crossed        │ │                                             │ │
│                        │ └─────────────────────────────────────┘ └─────────────────────────────────────────────┘ │
├────────────────────────┴───────────────────────────────────────────────────────────────────────────────────────────┤
│ FOOTER: stock_home | v0.1.0                                      Data via Zerodha Kite API | NSE | ● Live           │
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Header

- `stock_home` logo/name
- Sidebar toggle
- Global search for stocks, indices, and sectors
- Date/time display
- `Live Market` status
- Settings shortcut
- Notification bell with count
- User avatar

## 3. Sidebar

Items in order, with **Alerts selected**:

1. Home
2. Market Overview
3. Watchlist
4. Alerts (count badge)
5. Top Movers
6. Sector View
7. Stock Explorer
8. Dashboards
9. OpenBB Analysis
10. Prometheus (external-link indicator)
11. Grafana (external-link indicator)
12. AlertManager (external-link indicator)
13. Settings

Persistent Market Status card:
- `OPEN` indicator
- Exchange: NSE
- Session: `09:15 AM – 03:30 PM`
- `Live data via Zerodha Kite API`

## 4. Page title and actions

- Title: `Alerts`
- Subtitle: `Monitor market alerts, investigate triggers, and manage alert rules`
- Primary action: `Create Alert`
- Secondary action: `Export`
- Overflow menu

## 5. Summary cards

Four cards in one horizontal row:

| Card | Reference value | Supporting text |
|---|---:|---|
| Total Alerts | 128 | +8 today |
| Critical | 12 | 3 new |
| Warning | 36 | 5 new |
| Info | 80 | 0 new |

Use red for Critical, amber/orange for Warning, and blue for Info.

## 6. Filter and search bar

Controls in one row:

- Search: `Search alerts or symbols...`
- Severity dropdown
- Status dropdown
- Source dropdown
- Time range dropdown
- `Reset` action

Suggested severity values: `All`, `Critical`, `Warning`, `Info`  
Suggested status values: `All`, `Firing`, `Acknowledged`, `Resolved`, `Silenced`  
Suggested sources: `All`, `Prometheus`, `AlertManager`, `stock_exporter`, `Custom`

## 7. Main two-column area

The alert table occupies the wider left side. The selected-alert detail panel occupies the right side.

### 7.1 Alerts table

Columns:

| Column | Purpose |
|---|---|
| Checkbox | Bulk selection |
| Severity | Critical / Warning / Info indicator |
| Symbol | Stock or index |
| Rule | Alert rule name |
| Triggered | Trigger timestamp |
| Value | Current observed value |
| Status | Firing / Acknowledged / Resolved / Silenced |
| Source | Originating service |
| Actions | Row menu |

Reference rows:
- RELIANCE — R1 Breakout — Firing
- NIFTY50 — Rapid Move — Firing
- INFY — Volume Spike — Firing
- SBIN — Price Threshold — Firing
- HCLTECH — Price Drop — Firing
- TCS — RSI Threshold — Resolved
- BANKNIFTY — Breakout — Acknowledged

Table behavior:
- Selected row is highlighted.
- Clicking a row updates the detail panel.
- Severity and status remain visually distinct.
- Pagination appears below the table.
- Bulk actions appear when one or more rows are selected.

Bulk actions:
- Acknowledge
- Resolve
- Silence

## 8. Selected alert detail panel

Reference selection: **RELIANCE — R1 Breakout**

### Alert identity

- Critical severity
- Firing status
- Symbol: RELIANCE
- Rule: R1 Breakout
- Description: `Price crossed R1 level`

### Trigger details

- Current Price: `₹2,842.50`
- Trigger Level: `₹2,840.00`
- Started: `09:41:12`
- Duration: `00:04:32`
- Source: `Prometheus`
- Rule ID/name: `NSE_R1_Breakout`

### Chart

- Compact price chart
- R1 trigger threshold line
- Current-price marker
- Time axis and price axis

### Actions

- `Acknowledge`
- `Silence`
- `Open in Grafana`

Actions should apply to the selected alert and display a success or failure message.

## 9. Lower supporting panels

If visible in the reference viewport, retain the two-panel row below the main table/detail area.

### Alert Activity

Compact chronological feed with time, symbol, event, and status:
- 09:41 — RELIANCE — Alert fired
- 09:35 — INFY — Volume spike
- 09:28 — SBIN — Threshold crossed

### Alert Counts Over Time

- Compact time-series chart
- Time range aligned with the page filter
- Separate Critical, Warning, and Info series

Do not let these lower panels push the main alert table and selected-alert detail panel out of visual priority.

## 10. Footer

- `stock_home`
- Version `v0.1.0`
- `Data via Zerodha Kite API`
- `NSE`
- Green `Live` indicator

## 11. Layout and visual constraints

- Preserve the reference image's dark navy/charcoal background, blue-gray borders, compact rounded panels, and crisp typography.
- Keep four summary cards in one row at desktop width.
- Place filters immediately above the alert table.
- Keep the alert table on the left and selected-alert detail panel on the right.
- Use red for Critical, orange/amber for Warning, and blue for Info.
- Make firing/resolved/acknowledged states easy to distinguish.
- Keep table rows compact; do not convert the table into cards.
- Do not add unrelated configuration panels.
- All alert counts, times, prices, and trigger values are illustrative reference placeholders, not live data.

## Out of scope

This document specifies only the **Alerts** page. It does not define Market Overview, Watchlist, Top Movers, Sector View, Stock Explorer, or Settings.

**End of wireframe**
