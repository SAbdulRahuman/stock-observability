# stock_home — Market Overview Page Wireframe

**Scope:** Market Overview page only  
**Product:** stock_home — Stock Observability Platform  
**Reference:** Previously generated Market Overview UI image  
**Purpose:** Mirror the reference layout and component hierarchy; do not introduce new features.

## 1. Page layout

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ HEADER: stock_home  ☰  [Search stocks, indices, sectors...]   Date / Time   ● Live Market   ⚙  🔔12  [AS]  │
├──────────────────────┬───────────────────────────────────────────────────────────────────────────────────────┤
│ SIDEBAR              │ Market Overview                                      [1D][1W][1M][3M][1Y] [NSE ▼] │
│                      │ Live market data, key indices, sectors, and market breadth              [↻ Refresh] │
│ 🏠 Home              ├───────────────────────────────────────────────────────────────────────────────────────┤
│ 📈 Market Overview ◀ │ INDEX SUMMARY CARDS                                                                  │
│ ♡ Watchlist          │ ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌────────────┐ ┌───────────┐             │
│ 🔔 Alerts        12  │ │ NIFTY 50  │ │ BANK NIFTY│ │ SENSEX    │ │ NIFTY NEXT │ │ INDIA VIX │             │
│ ↗ Top Movers         │ │ value     │ │ value     │ │ value     │ │ 50 value   │ │ value     │             │
│ ◈ Sector View        │ │ change %  │ │ change %  │ │ change %  │ │ change %   │ │ change %  │             │
│ 🔎 Stock Explorer    │ │ sparkline │ │ sparkline │ │ sparkline │ │ sparkline  │ │ sparkline │             │
│ ▦ Dashboards         │ └───────────┘ └───────────┘ └───────────┘ └────────────┘ └───────────┘             │
│ ◈ OpenBB Analysis    │                                                                                       │
│ 🔵 Prometheus    ↗   │ MAIN ANALYTICS ROW                                                                    │
│ 🟠 Grafana       ↗   │ ┌─────────────────────────────────────┐ ┌─────────────────────────────────────────┐ │
│ 🔴 AlertManager  ↗   │ │ NIFTY 50 INTRADAY CHART             │ │ MARKET BREADTH (NSE)                   │ │
│ ⚙ Settings          │ │ [1D][1W][1M][3M][1Y]                │ │              Doughnut chart             │ │
│                      │ │                                     │ │              Total: 2,361                │ │
│ ┌──────────────────┐ │ │ line/area price chart + volume bars │ │ Advances 1,428 (60.5%)                 │ │
│ │ Market Status    │ │ │ price axis and time axis            │ │ Declines 876 (37.1%)                   │ │
│ │ ● OPEN            │ │ └─────────────────────────────────────┘ │ Unchanged 57 (2.4%)                    │ │
│ │ NSE               │ │                                         └─────────────────────────────────────────┘ │
│ │ 09:15–03:30 PM    │ │                                                                                       │
│ │ Live via Kite API │ │ GAINERS / LOSERS / SECTOR PERFORMANCE                                                │
│ └──────────────────┘ │ ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────────────────┐ │
│                      │ │ TOP GAINERS (NIFTY) │ │ TOP LOSERS (NIFTY)  │ │ SECTOR PERFORMANCE (NSE)        │ │
│                      │ │ # Symbol LTP Change │ │ # Symbol LTP Change │ │ Sector            % change      │ │
│                      │ │ ADANIENT ...        │ │ LT ...              │ │ PSU Bank           green bar     │ │
│                      │ │ SBIN ...            │ │ HCLTECH ...         │ │ Metal              green bar     │ │
│                      │ │ BEL ...             │ │ WIPRO ...           │ │ Auto               green bar     │ │
│                      │ │ TATAMOTORS ...      │ │ TECHM ...           │ │ Realty / FMCG...   green bars    │ │
│                      │ │ NTPC ...            │ │ INFY ...            │ │ IT / Pharma...     red bars      │ │
│                      │ └─────────────────────┘ └─────────────────────┘ └─────────────────────────────────┘ │
│                      │                                                                                       │
│                      │ SECTOR HEATMAP / MOST ACTIVE / MARKET NEWS                                            │
│                      │ ┌────────────────────────────┐ ┌────────────────────────┐ ┌────────────────────────┐ │
│                      │ │ SECTOR HEATMAP (NSE)      │ │ MOST ACTIVE (BY VOLUME) │ │ MARKET NEWS            │ │
│                      │ │ [1D ▼]                    │ │ # Symbol LTP Volume     │ │ time • headline        │ │
│                      │ │ BANK  IT  AUTO  FMCG       │ │ RELIANCE ...            │ │ 09:38 Nifty trades...   │ │
│                      │ │ METAL PHARMA REALTY OIL/GAS│ │ TCS ...                 │ │ 09:22 RBI MPC...        │ │
│                      │ │ PSU BANK / MEDIA / OTHER  │ │ INFY ...                │ │ 09:15 IT stocks...      │ │
│                      │ │ green = up, red = down    │ │ HDFCBANK ...            │ │ 08:50 Adani group...    │ │
│                      │ └────────────────────────────┘ └────────────────────────┘ └────────────────────────┘ │
├──────────────────────┴───────────────────────────────────────────────────────────────────────────────────────┤
│ FOOTER: stock_home | v0.1.0                   Data via Zerodha Kite API | NSE | ● Live                      │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Header

- Product logo and name: `stock_home`
- Sidebar toggle
- Global search: `Search stocks, indices, sectors (e.g. RELIANCE, TCS, NIFTY, Bank...)`
- Date/time display
- Green `Live Market` status
- Settings icon, notification bell with count, user avatar

## 3. Sidebar

Items in order, with **Market Overview selected**:

1. Home
2. Market Overview
3. Watchlist
4. Alerts (count badge)
5. Top Movers
6. Sector View
7. Stock Explorer
8. Dashboards
9. OpenBB Analysis
10. Prometheus (external-link icon)
11. Grafana (external-link icon)
12. AlertManager (external-link icon)
13. Settings

Persistent bottom card:
- Market Status and `OPEN` indicator
- Exchange: NSE
- Session: `09:15 AM – 03:30 PM`
- `Live data via Zerodha Kite API`

## 4. Page title and controls

- Title: `Market Overview`
- Subtitle: `Live market data, key indices, sectors, and market breadth`
- Time range: `1D`, `1W`, `1M`, `3M`, `1Y`
- Exchange selector: `NSE`
- Refresh button

## 5. Index summary cards

Five cards in one horizontal row. Each includes index name, latest value, absolute/percentage change, and a mini trend line.

| Card | Reference value | Change |
|---|---:|---:|
| NIFTY 50 | 24,892.30 | +156.70 (+0.63%) |
| BANK NIFTY | 54,321.10 | +412.80 (+0.77%) |
| SENSEX | 81,432.61 | +521.34 (+0.64%) |
| NIFTY NEXT 50 | 67,124.20 | +320.15 (+0.48%) |
| INDIA VIX | 12.54 | -0.68 (-5.14%) |

*These values are illustrative placeholders copied from the generated reference image, not a live feed.*

## 6. Main analytics row

### Left card — NIFTY 50 Intraday Chart

- Heading: `NIFTY 50 Intraday Chart`
- Timeframe controls: `1D`, `1W`, `1M`, `3M`, `1Y`
- Intraday line/area price chart
- Current-price horizontal marker and value label
- Price axis and time axis
- Volume bars at chart bottom

### Right card — Market Breadth (NSE)

- Doughnut chart with total in the center
- Advances, declines, and unchanged counts and percentages

Reference values:
- Total: 2,361
- Advances: 1,428 (60.5%)
- Declines: 876 (37.1%)
- Unchanged: 57 (2.4%)

## 7. Gainers, losers and sector performance row

### Top Gainers (NIFTY 50)

Columns: `#`, `Symbol`, `LTP (₹)`, `Change`, `% Change`, `Day Chart`  
Reference symbols: ADANIENT, SBIN, BEL, TATAMOTORS, NTPC.  
Include `View All` action.

### Top Losers (NIFTY 50)

Columns: `#`, `Symbol`, `LTP (₹)`, `Change`, `% Change`, `Day Chart`  
Reference symbols: LT, HCLTECH, WIPRO, TECHM, INFY.  
Include `View All` action.

### Sector Performance (NSE)

- Sector labels
- Horizontal performance bars and percentage labels
- Green for positive, red for negative
- `1D` selector and `View All` action

## 8. Sector heatmap, most active and market news row

### Sector Heatmap (NSE)

- Heading and `1D` selector
- Treemap-style tiles with sector name and percentage change
- Green indicates positive movement; red indicates negative movement
- Reference sectors: Banking, IT, Auto, FMCG, Metal, Pharma, Realty, Oil & Gas, Power, Media, PSU Bank, Infra, Consumer, Chemical, Telecom, Other

### Most Active (By Volume)

Columns: `#`, `Symbol`, `LTP (₹)`, `Volume`, `Change`, `%`  
Reference symbols: RELIANCE, TCS, INFY, HDFCBANK, ICICIBANK.  
Include `View All` action.

### Market News

- Timestamped headlines, newest first
- Small colored status/impact indicators
- `View All` action

## 9. Footer

- `stock_home`
- Version `v0.1.0`
- `Data via Zerodha Kite API`
- `NSE`
- Green `Live` indicator

## 10. Layout constraints

- Preserve the reference's dark navy/black background, thin blue-gray borders, and compact rounded cards.
- Keep five index cards on one horizontal row at desktop width.
- Keep the NIFTY chart and Market Breadth side by side.
- Keep Gainers, Losers, and Sector Performance on one row.
- Keep Sector Heatmap, Most Active, and Market News on one row.
- Use green for positive market movement and red for negative movement.
- Keep table rows compact and typography readable.
- Do not add Settings forms, configuration panels, or features not shown in the reference.
- Treat all example prices and counts as static wireframe content, not live financial data.

## Out of scope

This document defines only the **Market Overview** page. It does not specify Home, Watchlist, Alerts, Top Movers, Sector View, Stock Explorer, or Settings.

**End of wireframe**
