# stock_home — Watchlist Page Wireframe

**Scope:** Watchlist page only  
**Reference:** Previously generated stock_home Watchlist UI image  
**Purpose:** Match the reference image's layout and content hierarchy, not redesign it.

---

## 1. Page Layout

```text
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ HEADER                                                                                                             │
│ stock_home  ☰  [ Search stocks, indices, sectors (e.g. RELIANCE, TCS, NIFTY, Bank...) ] Date / Time ● Live Market ⚙ 🔔12 [AS] │
├────────────────────────┬───────────────────────────────────────────────────────────────────────────────────────────┤
│ SIDEBAR                │ WATCHLIST                                                                             │
│                        │ Track your favourite stocks, set alerts, and monitor real-time performance            │
│ 🏠 Home                │                                  [+ Add Symbol] [↓ Import] [↑ Export] [⋮] [▦]         │
│ 📈 Market Overview     ├───────────────────────────────────────────────────────────────────────────────────────────┤
│ ♡ Watchlist       ◀    │ WATCHLIST GROUP TABS                                                                       │
│ 🔔 Alerts          12  │ [My Watchlist (15)] [Banking (8)] [IT (6)] [Auto (6)] [Pharma (6)] [High Dividend (5)] [Swing Trade (8)] │
│ ↗ Top Movers           ├───────────────────────────────────────────────────────────────────────────────────────────┤
│ ◈ Sector View          │ [🔍 Search in watchlist...]                   [All ▼] [▦ Table] [Cards] [⚙ Customize]   │
│ 🔎 Stock Explorer      ├───────────────────────────────────────────────┬───────────────────────────────────────────┤
│ ▦ Dashboards           │ WATCHLIST TABLE                              │ SELECTED STOCK DETAILS                    │
│ ◈ OpenBB Analysis      │ ☐ # Symbol     LTP (₹) Change % Day Chart    │ RELIANCE  Reliance Industries Ltd.        │
│ 🔵 Prometheus      ↗   │ ☐ 1 RELIANCE   2,842.50 +36.80 +1.31% 📈     │ [NSE] [Large Cap] [Energy]                │
│ 🟠 Grafana         ↗   │ ☐ 2 TCS        3,412.20 -12.35 -0.36% 📉     │ ★  ⋮                                      │
│ 🔴 AlertManager    ↗   │ ☐ 3 INFY       1,487.30 +18.90 +1.29% 📈     │                                           │
│ ⚙ Settings             │ ☐ 4 HDFCBANK   1,672.10 +22.45 +1.36% 📈     │ Price: 2,842.50  +36.80 (+1.31%)         │
│                        │ ☐ 5 ICICIBANK  1,245.80 +10.20 +0.82% 📈     │ Day High 2,856.00  Day Low 2,798.10       │
│ ┌──────────────────┐   │ ☐ 6 SBIN         621.35 +18.75 +3.11% 📈     │ Volume 12.4M                              │
│ │ Market Status    │   │ ☐ 7 ADANIENT   3,142.60 +126.40 +4.20% 📈    │                                           │
│ │ ● OPEN            │   │ ☐ 8 TATAMOTORS  992.10 +26.30 +2.72% 📈     │ [1D][1W][1M][3M][1Y][5Y]                  │
│ │ NSE               │   │ ☐ 9 BEL          312.40 +8.90 +2.93% 📈     │                                           │
│ │ 09:15–03:30 PM    │   │ ☐10 LT         3,562.10 -41.20 -1.14% 📉    │ Intraday price chart + volume bars         │
│ │ Live via Kite API │   │ ☐11 WIPRO        412.20 -5.15 -1.23% 📉    │                                           │
│ └──────────────────┘   │ ☐12 ITC          485.20 +6.45 +1.35% 📈     │ Open / High / Low / Prev Close             │
│                        │ ☐13 HCLTECH    1,548.20 -32.80 -2.08% 📉    │ Volume / Avg Volume (20D)                  │
│                        │ ☐14 AXISBANK   1,104.30 +14.70 +1.35% 📈    │                                           │
│                        │ ☐15 BAJFINANCE 7,231.50 -85.30 -1.17% 📉    │ [🔔 Set Alert] [View in Grafana]           │
│                        │                                             │ [OpenBB Analysis] [Full Chart]             │
│                        ├─────────────────────────────────────────────┴───────────────────────────────────────────┤
│                        │ BOTTOM SUPPORTING ROW                                                                       │
│                        │ ┌─────────────────────────────┐ ┌─────────────────────────────┐ ┌──────────────────────┐ │
│                        │ │ WATCHLIST PERFORMANCE      │ │ RECENT ALERTS (WATCHLIST)   │ │ QUICK ACTIONS        │ │
│                        │ │ Total Symbols: 15          │ │ 09:41 RELIANCE Price > 2840 │ │ + Create Watchlist   │ │
│                        │ │ Gainers 11 (73.3%)         │ │ 09:35 INFY Volume Spike     │ │ ↓ Import from CSV    │ │
│                        │ │ Losers 4 (26.7%)          │ │ 09:28 SBIN Price > 620      │ │ ↑ Export Watchlist   │ │
│                        │ │ Unchanged 0 (0%)           │ │ 09:22 HCLTECH Price < 1550  │ │ 🔔 Set Global Alerts │ │
│                        │ │ Mini performance chart    │ │ 09:15 TATAMOTORS RSI > 70   │ │ ⚙ Watchlist Settings │ │
│                        │ └─────────────────────────────┘ └─────────────────────────────┘ └──────────────────────┘ │
├────────────────────────┴───────────────────────────────────────────────────────────────────────────────────────────┤
│ FOOTER: stock_home | v0.1.0                                      Data via Zerodha Kite API | NSE | ● Live           │
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Header

- Product logo and name: `stock_home`
- Sidebar toggle
- Global search for stocks, indices, and sectors
- Date/time
- `Live Market` indicator
- Settings icon
- Notifications bell with count badge
- User avatar

## 3. Sidebar

Items in order, with **Watchlist selected**:

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

Persistent Market Status card at the bottom:
- `OPEN` indicator
- NSE session: `09:15 AM – 03:30 PM`
- `Live data via Zerodha Kite API`

## 4. Page title and primary actions

- Title: `Watchlist`
- Subtitle: `Track your favourite stocks, set alerts, and monitor real-time performance`
- `Add Symbol`
- `Import`
- `Export`
- Overflow menu
- View/layout toggle

## 5. Watchlist group tabs

Tabs appear directly below the page title:

- `My Watchlist (15)` — selected
- `Banking (8)`
- `IT (6)`
- `Auto (6)`
- `Pharma (6)`
- `High Dividend (5)`
- `Swing Trade (8)`

The active tab is highlighted in blue.

## 6. Search, filters, and view controls

- Search input: `Search in watchlist...`
- Filter dropdown: `All`
- View selector: `Table` (selected) and `Cards`
- `Customize` button

## 7. Main content: table and selected-stock panel

The main area is a two-column layout. The watchlist table occupies the larger left column; the selected stock detail panel occupies the right column.

### 7.1 Watchlist table

Columns:

| Column | Purpose |
|---|---|
| Checkbox | Select one or multiple symbols |
| # | Row number |
| Symbol | Stock ticker |
| LTP (₹) | Last traded price |
| Change | Absolute price change |
| % | Percentage change |
| Day Chart | Compact sparkline |
| Volume | Traded volume |
| Alerts | Alert indicator |
| Actions | Row action menu |

Reference symbols, in order:
1. RELIANCE
2. TCS
3. INFY
4. HDFCBANK
5. ICICIBANK
6. SBIN
7. ADANIENT
8. TATAMOTORS
9. BEL
10. LT
11. WIPRO
12. ITC
13. HCLTECH
14. AXISBANK
15. BAJFINANCE

The reference image shows the key columns and 15 rows; keep the table compact with a highlighted selected row. Use green for positive movement and red for negative movement. Sparklines use the same direction color.

### 7.2 Selected stock detail panel

Reference selected stock: `RELIANCE — Reliance Industries Ltd.`

Header:
- Symbol and company name
- Tags: `NSE`, `Large Cap`, `Energy`
- Favourite star
- More-actions menu

Summary:
- LTP: `₹2,842.50`
- Change: `+36.80 (+1.31%)`
- Day High: `2,856.00`
- Day Low: `2,798.10`
- Volume: `12.4M`

Chart:
- Timeframes: `1D`, `1W`, `1M`, `3M`, `1Y`, `5Y`
- Intraday price chart
- Current-price marker
- Volume bars

Additional statistics:
- Open
- High
- Low
- Previous Close
- Volume
- Average Volume (20D)

Actions:
- `Set Alert`
- `View in Grafana`
- `OpenBB Analysis`
- `Full Chart`

## 8. Bottom supporting row

### 8.1 Watchlist Performance

- Total Symbols: `15`
- Gainers: `11 (73.3%)`
- Losers: `4 (26.7%)`
- Unchanged: `0 (0%)`
- Small performance chart with time axis

### 8.2 Recent Alerts (Watchlist)

A compact list with time, symbol, rule, and value. Reference examples:
- 09:41 — RELIANCE — Price > 2,840
- 09:35 — INFY — Volume Spike (5x)
- 09:28 — SBIN — Price > 620
- 09:22 — HCLTECH — Price < 1,550
- 09:15 — TATAMOTORS — RSI > 70

Include `View All`.

### 8.3 Quick Actions

- Create New Watchlist
- Import from CSV
- Export Watchlist
- Set Global Alerts
- Watchlist Settings

## 9. Footer

- `stock_home`
- Version `v0.1.0`
- `Data via Zerodha Kite API`
- `NSE`
- Green `Live` indicator

## 10. Layout and styling constraints

- Preserve the reference image's dark navy/black background, blue selected states, thin blue-gray borders, and compact rounded panels.
- Keep the title/actions at the top, followed by watchlist group tabs and search/filter controls.
- Keep the stock table on the left and the selected-stock detail panel on the right at desktop width.
- Keep Watchlist Performance, Recent Alerts, and Quick Actions in one bottom row.
- Use green for positive price changes and red for negative changes.
- Avoid turning table rows into cards when Table view is selected.
- Do not add unrelated settings panels or new page sections.
- All prices and counts are illustrative values from the generated reference image, not live financial data.

## Out of scope

This document specifies only the **Watchlist** page. It does not define Market Overview, Alerts, Top Movers, Sector View, Stock Explorer, or Settings.

**End of wireframe**
