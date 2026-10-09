# stock_home UI backlog (UX wireframes)

Single source of truth for the `stock-home-ui` agent (`/implement-ui`).
Designs: `stock_home/ux-proposals/<N> STOCK_HOME_*.md` (wins on conflicts) and `stock_home/ux-proposals/Design_UI_images/*.png` (visual reference).

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked (reason in Notes) · `[-]` dropped (reason in Notes)

Rules:
- Work top to bottom; an item may start only when every ID in **Deps** is `[x]` or `[-]`.
- One item = one local commit (`git log --grep U##`).
- Wireframe sample values are placeholders; missing data → `—` or an explicit "unavailable" panel, never fake values.
- New data comes from stock_home Go endpoints backed by Prometheus, Alertmanager or OpenBB; `/api/v1` shapes are only extended, never changed.

Decisions:
- Dark theme by default; light/system selectable in Settings.
- The sidebar shell replaces `NavTabs`; `/analyst` and `/openbb` stay reachable (Stock Explorer detail links to `/analyst`).
- Persistence (watchlist groups, settings) uses browser storage until the SQLite store (openbb-parity F003) exists.
- Alertmanager has no acknowledge/resolve: "Acknowledge"/"Silence" create a silence after confirmation; "Resolve" is not offered.

| ID | Status | Item | Wireframe | Image | Deps | Notes |
|----|--------|------|-----------|-------|------|-------|
| U00 | [ ] | Design reconcile: compare each MD with its PNG; record decisions here (shell differences between MDs 1–7, 8–11 and 12; bell / sidebar Market Status card; Stock Explorer and Settings MDs vs their images; Dashboards vs Home overlap; panels without a data source). No code. | all | all | – | |
| U01 | [x] | App shell: sidebar (13 items, Alerts badge, external-link icons, Market Status card), header (search, IST clock, Live Market, settings, bell, avatar), footer, dark default theme; replaces `NavTabs`; links from `/api/v1/links` | [1](../../ux-proposals/1%20STOCK_HOME_HOME_PAGE_WIREFRAME.md) §2–3, §8 | Home_UI.png | U00 | Started ahead of U00 on explicit user request (Home wireframe). Added `GET /api/v1/info` (version, exchange, session hours from new flags `--market.exchange`, `--market.session-hours`), `id` field on `/api/v1/links` items plus `prometheus` and `openbb` entries. Nav defined once in `web/components/shell/nav.ts`; items without a page yet are listed but disabled ("coming soon") and flip via `ready`. MD vs PNG: bell (MD only) and sidebar Market Status card (MD only) kept per MD; avatar shows a generic user icon (no user identity source); header search opens `/analyst?symbol=` until Stock Explorer (U08); bell links to AlertManager until Alerts (U05); theme/privacy toggles removed from the header (theme moves to Settings, U13). |
| U02 | [x] | Home: index cards + Market Status, NIFTY intraday chart (1D–1Y), Top Movers tabs, Active Alerts, Watchlist snapshot, Quick Links | [1](../../ux-proposals/1%20STOCK_HOME_HOME_PAGE_WIREFRAME.md) | Home_UI.png | U01 | Endpoints: new `GET /api/v1/chart?symbol=<configured index>&range=1D\|1W\|1M\|3M\|1Y` (session-only points; 1D = latest session) and `GET /api/v1/quotes?symbols=` (≤20, validated, uncached); added fields `change`, `volume`, `spark` on quotes in `/market` and `/movers`, `highVolume` on `/movers`, `value` (from a rule's `value` annotation) on `/alerts` items, `symbol` on `/timeline` items. Watchlist reads browser storage key `stock_home.watchlist.v1` (shared with U04); empty → explicit empty state. Gaps: movers universe is the exporter core tier (no NIFTY 50 membership label) so the title reads "Top Movers (core equities)"; indices have no volume, so chart bars are core-equities traded volume and labelled as such; alert Value shows `—` unless rules add a `value` annotation; Active Alerts lists firing alerts then alerts resolved in the last 24h (from `/timeline`); Watchlist "View All" and per-row kebab (PNG only) omitted until U04; operational banners (Kite login etc.) kept above the summary row. |
| U03 | [ ] | Market Overview: index cards, intraday chart, breadth donut, gainers/losers, sector performance, sector heatmap, most active, market news | [2](../../ux-proposals/2%20STOCK_HOME_MARKET_OVERVIEW_WIREFRAME.md) | Market_Overview_UI.png | U02 | News via OpenBB if available |
| U04 | [ ] | Watchlist: group tabs, table + selected-stock panel, performance, recent alerts, quick actions | [3](../../ux-proposals/3%20STOCK_HOME_WATCHLIST_WIREFRAME.md) | Watchlist_ui.png | U02 | |
| U05 | [ ] | Alerts: severity cards, filters, table + detail panel, bulk actions, activity and counts-over-time panels | [4](../../ux-proposals/4%20STOCK_HOME_ALERTS_WIREFRAME.md) | Alerts_UI.png | U02 | |
| U06 | [ ] | Top Movers: filters, summary cards, gainers/losers, most active, 52-week highs/lows, % change distribution, movers by sector | [5](../../ux-proposals/5%20STOCK_HOME_TOP_MOVERS_WIREFRAME.md) | Top_Movers_UI.png | U02 | |
| U07 | [ ] | Sector View: sector index cards, heatmap, sector performance and comparison, sector gainers/losers | [6](../../ux-proposals/6%20STOCK_HOME_SECTOR_VIEW_WIREFRAME.md) | Sector_View_UI.png | U03 | |
| U08 | [ ] | Stock Explorer: search, filters, results table, pagination, detail via `?symbol=` linking to `/analyst` and `/openbb` | [7](../../ux-proposals/7%20STOCK_HOME_STOCK_EXPLORER_WIREFRAME.md) | Stock_Explorer_UI.png | U02 | |
| U09 | [ ] | Dashboards: scope per U00 decision (default: configured Grafana dashboard links) | [8](../../ux-proposals/8%20STOCK_HOME_DASHBOARD_DETAILED_WIREFRAME.md) | – | U01 | |
| U10 | [ ] | OpenBB Analysis: existing workbench inside the new shell, stock selector and tabs per wireframe | [9](../../ux-proposals/9%20STOCK_HOME_OPENBB_ANALYSIS_WIREFRAME.md) | – | U01 | |
| U11 | [ ] | Prometheus: overview, query (instant/range), targets, rules, alerts, status via read-only allowlisted proxy | [10](../../ux-proposals/10%20STOCK_HOME_PROMETHEUS_PAGE_DETAILED_WIREFRAME.md) | – | U01 | |
| U12 | [ ] | AlertManager: alert groups, silences (create/expire with confirmation), receivers/routes (read-only, redacted), status | [11](../../ux-proposals/11%20STOCK_HOME_ALERTMANAGER_PAGE_DETAILED_WIREFRAME.md) | – | U05 | |
| U13 | [ ] | Settings: tabs and nine cards; preferences persisted, integration Test via backend, secrets never echoed, destructive actions confirmed | [12](../../ux-proposals/12%20STOCK_HOME_SETTINGS_PAGE_DETAILED_WIREFRAME.md) | Settings_UI.png, Settings_UI_dark.png | U01 | |
