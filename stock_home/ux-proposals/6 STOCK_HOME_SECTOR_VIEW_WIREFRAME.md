# stock_home — Sector View Page UI Wireframe

**Scope:** Sector View page only  
**Reference:** Previously generated stock_home Sector View UI image  
**Goal:** Match the reference page's layout and component hierarchy. This is a structural wireframe, not a redesign.

---

## 1. Page layout

```text
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ HEADER                                                                                                             │
│ stock_home  ☰  [ Search stocks, indices, sectors... ]         Date / Time   ● Live Market   ⚙  🔔12  [AS]          │
├────────────────────────┬───────────────────────────────────────────────────────────────────────────────────────────┤
│ SIDEBAR                │ SECTOR VIEW                                                                              │
│                        │ Compare NSE sector performance and explore market rotation                               │
│ 🏠 Home                │ [NSE ▼] [1D ▼] [Large Cap ▼] [All Sectors ▼] [↻ Refresh] [Export]                        │
│ 📈 Market Overview     ├───────────────────────────────────────────────────────────────────────────────────────────┤
│ ♡ Watchlist            │ SECTOR INDEX SUMMARY CARDS                                                               │
│ 🔔 Alerts          12  │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌──────────────────────┐ │
│ ↗ Top Movers           │ │ NIFTY BANK  │ │ NIFTY IT    │ │ NIFTY AUTO  │ │ NIFTY FMCG  │ │ NIFTY PHARMA          │ │
│ ◈ Sector View     ◀    │ │ index value │ │ index value │ │ index value │ │ index value │ │ index value           │ │
│ 🔎 Stock Explorer      │ │ change %    │ │ change %    │ │ change %    │ │ change %    │ │ change %              │ │
│ ▦ Dashboards           │ │ sparkline   │ │ sparkline   │ │ sparkline   │ │ sparkline   │ │ sparkline             │ │
│ ◈ OpenBB Analysis      │ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └──────────────────────┘ │
│ 🔵 Prometheus      ↗   │                                                                                           │
│ 🟠 Grafana         ↗   │ SECTOR HEATMAP                                                                            │
│ 🔴 AlertManager    ↗   │ ┌───────────────────────────────────────────────────────────────────────────────────────┐ │
│ ⚙ Settings             │ │ NSE SECTOR HEATMAP                                      [1D ▼] [Market Cap ▼]        │ │
│                        │ │                                                                               │ │
│ ┌──────────────────┐   │ │ ┌───────────────┐ ┌───────────────┐ ┌────────────┐ ┌───────────────┐          │ │
│ │ Market Status    │   │ │ │ BANKING       │ │ IT            │ │ AUTO       │ │ FMCG          │          │ │
│ │ ● OPEN            │   │ │ │ +1.24%        │ │ -0.62%        │ │ +0.88%     │ │ +0.31%        │          │ │
│ │ NSE               │   │ │ └───────────────┘ └───────────────┘ └────────────┘ └───────────────┘          │ │
│ │ 09:15–03:30 PM    │   │ │ ┌───────────────┐ ┌───────────────┐ ┌────────────┐ ┌───────────────┐          │ │
│ │ Live via Kite API │   │ │ │ METAL         │ │ PHARMA        │ │ REALTY     │ │ OIL & GAS     │          │ │
│ └──────────────────┘   │ │ │ +1.76%        │ │ +0.42%        │ │ -0.35%     │ │ +0.67%        │          │ │
│                        │ │ └───────────────┘ └───────────────┘ └────────────┘ └───────────────┘          │ │
│                        │ │ POWER | MEDIA | PSU BANK | INFRA | CONSUMER | CHEMICAL | TELECOM                 │ │
│                        │ └───────────────────────────────────────────────────────────────────────────────────────┘ │
│                        │                                                                                           │
│                        │ SECTOR PERFORMANCE / COMPARISON                                                          │
│                        │ ┌──────────────────────────────────────┐ ┌────────────────────────────────────────────┐ │
│                        │ │ SECTOR PERFORMANCE                  │ │ SECTOR COMPARISON                          │ │
│                        │ │ Sector          Change   Bar         │ │ selected sectors line chart                │ │
│                        │ │ Metal           +1.76%   ███████     │ │ [Banking] [IT] [Auto] [Pharma]             │ │
│                        │ │ Banking         +1.24%   █████       │ │ index rebased to 100                        │ │
│                        │ │ Auto            +0.88%   ████        │ │ timeframe: 1D / 1W / 1M / 3M / 1Y           │ │
│                        │ │ Oil & Gas       +0.67%   ███         │ │                                            │ │
│                        │ │ Pharma          +0.42%   ██          │ │                                            │ │
│                        │ │ IT              -0.62%   ◀██         │ │                                            │ │
│                        │ └──────────────────────────────────────┘ └────────────────────────────────────────────┘ │
│                        │                                                                                           │
│                        │ SECTOR LEADERS / SECTOR LAGGARDS                                                        │
│                        │ ┌──────────────────────────────────────┐ ┌────────────────────────────────────────────┐ │
│                        │ │ TOP SECTOR GAINERS                  │ │ TOP SECTOR LOSERS                          │ │
│                        │ │ Sector / Leading stock / % change   │ │ Sector / Leading stock / % change          │ │
│                        │ │ Metal    TATASTEEL       +2.4%      │ │ IT       WIPRO             -1.2%           │ │
│                        │ │ Banking  SBIN            +1.8%      │ │ Realty   DLF               -0.9%           │ │
│                        │ │ Auto     TATAMOTORS      +1.6%      │ │ Media    ZEEL              -0.7%           │ │
│                        │ │ [View Sector →]                     │ │ [View Sector →]                            │ │
│                        │ └──────────────────────────────────────┘ └────────────────────────────────────────────┘ │
├────────────────────────┴───────────────────────────────────────────────────────────────────────────────────────────┤
│ FOOTER: stock_home | v0.1.0                                      Data via Zerodha Kite API | NSE | ● Live           │
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Header

- `stock_home` logo and name
- Sidebar toggle
- Global search for stocks, indices, and sectors
- Date/time display
- `Live Market` indicator
- Settings shortcut
- Notifications bell with count badge
- User avatar

## 3. Sidebar

Items in order, with **Sector View selected**:

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

## 4. Page title and filters

- Title: `Sector View`
- Subtitle: `Compare NSE sector performance and explore market rotation`
- Exchange selector: `NSE`
- Time range: `1D`
- Universe / capitalization selector: `Large Cap`
- Sector selector: `All Sectors`
- `Refresh` and `Export` actions

## 5. Sector index summary cards

Five cards across one row. Each card contains:
- Sector index name
- Latest index value
- Absolute and percentage change
- Mini trend line

Reference index cards:
- NIFTY BANK
- NIFTY IT
- NIFTY AUTO
- NIFTY FMCG
- NIFTY PHARMA

Positive values use green; negative values use red.

## 6. NSE Sector Heatmap

A full-width heatmap sits below the summary cards.

### Header controls

- Title: `NSE Sector Heatmap`
- Time range selector: `1D`
- Sizing selector: `Market Cap`

### Heatmap tiles

Each tile displays:
- Sector name
- Percentage change
- Optional index/market-cap hint on hover

Reference sectors:
- Banking
- IT
- Auto
- FMCG
- Metal
- Pharma
- Realty
- Oil & Gas
- Power
- Media
- PSU Bank
- Infra
- Consumer
- Chemical
- Telecom

Color semantics:
- Green: positive change
- Red: negative change
- Neutral gray: unchanged or unavailable

Tile size should follow the selected sizing metric; labels must remain readable.

## 7. Sector performance and comparison

Two side-by-side panels.

### 7.1 Sector Performance

A ranked horizontal-bar chart with:
- Sector name
- Percentage change
- Positive/negative bar
- Sort order by performance
- `View All` action if the complete sector list is not visible

Reference ordering examples:
- Metal: +1.76%
- Banking: +1.24%
- Auto: +0.88%
- Oil & Gas: +0.67%
- Pharma: +0.42%
- IT: -0.62%

### 7.2 Sector Comparison

- Multi-line comparison chart
- Sector selection chips: Banking, IT, Auto, Pharma
- Timeframes: `1D`, `1W`, `1M`, `3M`, `1Y`
- Shared time axis
- Values normalized to a common starting point for relative performance comparison

## 8. Sector leaders and laggards

Two tables side by side.

### Top Sector Gainers

Columns:
- Sector
- Leading stock
- Stock % change
- Sector % change
- `View Sector` action

Reference examples:
- Metal — TATASTEEL — +2.4%
- Banking — SBIN — +1.8%
- Auto — TATAMOTORS — +1.6%

### Top Sector Losers

Columns:
- Sector
- Leading stock
- Stock % change
- Sector % change
- `View Sector` action

Reference examples:
- IT — WIPRO — -1.2%
- Realty — DLF — -0.9%
- Media — ZEEL — -0.7%

## 9. Footer

- `stock_home`
- Version `v0.1.0`
- `Data via Zerodha Kite API`
- `NSE`
- Green `Live` indicator

## 10. Layout and visual constraints

- Preserve the reference image's dark navy/charcoal background, thin blue-gray borders, rounded cards, and compact typography.
- Keep filters below the title.
- Keep five sector-index cards in one horizontal row at desktop width.
- Keep the heatmap full width.
- Keep Sector Performance and Sector Comparison side by side.
- Keep Sector Gainers and Sector Losers side by side in the bottom row.
- Use green for positive movement and red for negative movement.
- Maintain consistent sector colors across charts and heatmap where possible.
- Do not add unrelated configuration panels.
- All example values are illustrative wireframe placeholders, not live market data.

## Out of scope

This document specifies only the **Sector View** page. It does not define Market Overview, Watchlist, Alerts, Top Movers, Stock Explorer, or Settings.

**End of wireframe**
