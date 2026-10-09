# stock_home — Top Movers Page Wireframe

**Scope:** Top Movers page only  
**Reference:** Previously generated stock_home Top Movers UI image  
**Goal:** Reproduce the reference image's page structure and component placement, without adding unrelated features.

---

## 1. Page layout

```text
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ HEADER                                                                                                             │
│ stock_home  ☰  [ Search stocks, indices, sectors... ]         Date / Time   ● Live Market   ⚙  🔔12  [AS]          │
├────────────────────────┬───────────────────────────────────────────────────────────────────────────────────────────┤
│ SIDEBAR                │ TOP MOVERS                                                                               │
│                        │ Find the biggest price movers across the NSE market                                      │
│ 🏠 Home                │ [NSE ▼] [NIFTY 50 ▼] [1D ▼] [All Sectors ▼] [All Prices ▼] [↻ Refresh] [Export]          │
│ 📈 Market Overview     ├───────────────────────────────────────────────────────────────────────────────────────────┤
│ ♡ Watchlist            │ MARKET MOVEMENT SUMMARY                                                                   │
│ 🔔 Alerts          12  │ ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌────────────────────────────┐ │
│ ↗ Top Movers      ◀    │ │ TOP GAINERS    │ │ TOP LOSERS     │ │ MOST ACTIVE    │ │ HIGHEST VOLUME CHANGE      │ │
│ ◈ Sector View          │ │ count / best % │ │ count / worst %│ │ symbol / value │ │ symbol / volume change     │ │
│ 🔎 Stock Explorer      │ └────────────────┘ └────────────────┘ └────────────────┘ └────────────────────────────┘ │
│ ▦ Dashboards           │                                                                                           │
│ ◈ OpenBB Analysis      │ GAINERS / LOSERS                                                                          │
│ 🔵 Prometheus      ↗   │ ┌───────────────────────────────────────┐ ┌────────────────────────────────────────────┐ │
│ 🟠 Grafana         ↗   │ │ TOP GAINERS                           │ │ TOP LOSERS                                 │ │
│ 🔴 AlertManager    ↗   │ │ # Symbol LTP Change %Change Sparkline │ │ # Symbol LTP Change %Change Sparkline      │ │
│ ⚙ Settings             │ │ 1 ADANIENT ...                        │ │ 1 LT ...                                  │ │
│                        │ │ 2 SBIN ...                            │ │ 2 HCLTECH ...                             │ │
│ ┌──────────────────┐   │ │ 3 BEL ...                             │ │ 3 WIPRO ...                               │ │
│ │ Market Status    │   │ │ 4 TATAMOTORS ...                     │ │ 4 TECHM ...                               │ │
│ │ ● OPEN            │   │ │ 5 NTPC ...                            │ │ 5 INFY ...                                │ │
│ │ NSE               │   │ │ ...                                  │ │ ...                                       │ │
│ │ 09:15–03:30 PM    │   │ │ [View All →]                         │ │ [View All →]                              │ │
│ │ Live via Kite API │   │ └───────────────────────────────────────┘ └────────────────────────────────────────────┘ │
│ └──────────────────┘   │                                                                                           │
│                        │ VOLUME / RANGE / MOMENTUM ANALYSIS                                                       │
│                        │ ┌────────────────────────────┐ ┌────────────────────────────┐ ┌────────────────────────┐ │
│                        │ │ MOST ACTIVE BY VOLUME      │ │ 52-WEEK HIGHS               │ │ 52-WEEK LOWS           │ │
│                        │ │ # Symbol Volume LTP %      │ │ Symbol LTP Distance %       │ │ Symbol LTP Distance %  │ │
│                        │ │ RELIANCE ...               │ │ stock ...                   │ │ stock ...              │ │
│                        │ │ TCS ...                    │ │ stock ...                   │ │ stock ...              │ │
│                        │ │ INFY ...                   │ │ stock ...                   │ │ stock ...              │ │
│                        │ │ HDFCBANK ...               │ │ stock ...                   │ │ stock ...              │ │
│                        │ │ ICICIBANK ...              │ │ stock ...                   │ │ stock ...              │ │
│                        │ └────────────────────────────┘ └────────────────────────────┘ └────────────────────────┘ │
│                        │                                                                                           │
│                        │ MOVEMENT DISTRIBUTION / RANKING                                                          │
│                        │ ┌────────────────────────────────────────────┐ ┌────────────────────────────────────────┐ │
│                        │ │ DAILY % CHANGE DISTRIBUTION                │ │ TOP MOVERS BY SECTOR                   │ │
│                        │ │ histogram / compact bar chart              │ │ sector labels + ranked movers          │ │
│                        │ │ negative ← 0% → positive                   │ │ Banking / IT / Auto / Pharma / Metal    │ │
│                        │ └────────────────────────────────────────────┘ └────────────────────────────────────────┘ │
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
- Settings shortcut
- Notifications bell with count badge
- User avatar

## 3. Sidebar

Items in order, with **Top Movers selected**:

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

Persistent Market Status card:
- `OPEN` indicator
- Exchange: NSE
- Session: `09:15 AM – 03:30 PM`
- `Live data via Zerodha Kite API`

## 4. Page title and filters

- Title: `Top Movers`
- Subtitle: `Find the biggest price movers across the NSE market`
- Exchange selector: `NSE`
- Universe selector: `NIFTY 50`
- Time range selector: `1D`
- Sector filter: `All Sectors`
- Price filter: `All Prices`
- `Refresh` button
- `Export` button

## 5. Market movement summary

Four summary cards in one row:

1. **Top Gainers** — number of advancing movers and strongest percentage gain
2. **Top Losers** — number of declining movers and largest percentage loss
3. **Most Active** — leading symbol and its activity/volume value
4. **Highest Volume Change** — symbol with the strongest volume increase

Each card includes a concise key value, comparison/change, and small visual indicator. Use green for gainers, red for losers, and neutral blue for volume/activity.

## 6. Top Gainers and Top Losers

Two equally weighted tables side by side.

### Top Gainers

Columns:
- Rank
- Symbol
- LTP (₹)
- Absolute Change
- % Change
- Day Sparkline

Reference symbols: `ADANIENT`, `SBIN`, `BEL`, `TATAMOTORS`, `NTPC`.

### Top Losers

Columns:
- Rank
- Symbol
- LTP (₹)
- Absolute Change
- % Change
- Day Sparkline

Reference symbols: `LT`, `HCLTECH`, `WIPRO`, `TECHM`, `INFY`.

Both tables:
- Use green for positive movement and red for negative movement.
- Keep rows compact.
- Show a `View All` action.
- Allow selecting a symbol to open its detail view.

## 7. Volume, range and momentum analysis

Three cards in one horizontal row.

### Most Active by Volume

Columns:
- Rank
- Symbol
- Volume
- LTP (₹)
- % Change

Reference symbols: `RELIANCE`, `TCS`, `INFY`, `HDFCBANK`, `ICICIBANK`.

### 52-Week Highs

- Symbol
- LTP (₹)
- 52-week high
- Distance from high (%)
- Compact distance/progress indicator

### 52-Week Lows

- Symbol
- LTP (₹)
- 52-week low
- Distance from low (%)
- Compact distance/progress indicator

## 8. Movement distribution and sector ranking

Two panels side by side.

### Daily % Change Distribution

- Compact histogram or bar chart
- Negative returns to the left of zero
- Positive returns to the right of zero
- Axis and labels showing percentage-change ranges

### Top Movers by Sector

- Sector label
- Leading symbol(s) in the sector
- Percentage change
- Compact ranking or mini bar
- Reference sectors: Banking, IT, Auto, Pharma, Metal

## 9. Footer

- `stock_home`
- Version `v0.1.0`
- `Data via Zerodha Kite API`
- `NSE`
- Green `Live` indicator

## 10. Layout and visual constraints

- Preserve the reference image's dark navy/charcoal palette, thin blue-gray borders, rounded panels, and compact typography.
- Keep filter controls in one horizontal row below the page title.
- Keep four summary cards in one row.
- Keep Gainers and Losers side by side.
- Keep Most Active, 52-Week Highs, and 52-Week Lows in one row.
- Keep movement distribution and sector ranking in the bottom row.
- Use green for gains and red for losses.
- Keep tables dense and scan-friendly; avoid replacing tables with large stock cards.
- Do not add unrelated configuration panels.
- Prices, counts, and rankings are placeholders for wireframe purposes, not live market data.

## Out of scope

This document specifies only the **Top Movers** page. It does not define Market Overview, Watchlist, Alerts, Sector View, Stock Explorer, or Settings.

**End of wireframe**
