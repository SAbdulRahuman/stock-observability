# stock_home — Home Page Wireframe

**Scope:** Home page only  
**Product:** stock_home — Stock Observability Platform  
**Reference:** Generated stock_home Home dashboard screenshot  
**Layout:** Desktop, dark theme

> **Fidelity rule:** This document describes only the Home page layout in the reference image. It does not specify other application pages.

## 1. Page layout

```text
┌──────────────────────────────┬──────────────────────────────────────────────────────────────────────────────────────────┐
│ SIDEBAR                      │ HEADER                                                                                   │
│ stock_home                   │ [☰] [ Search stocks (e.g. RELIANCE, TCS, INFY)... ] Date/time  ● Live Market  ⚙  🔔  AS │
│ Stock Observability Platform │                                                                                          │
├──────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────┤
│ 🏠 Home              ACTIVE  │ ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌─────────────────────────┐ │
│ 📈 Market Overview           │ │ NIFTY 50   │ │ BANK NIFTY │ │ SENSEX     │ │ INDIA VIX  │ │ Market Status           │ │
│ ♡  Watchlist                 │ │ 24,892.30  │ │ 54,321.10  │ │ 81,432.61  │ │ 12.54      │ │ ● OPEN                  │ │
│ 🔔 Alerts             [12]   │ │ +0.63%     │ │ +0.77%     │ │ +0.64%     │ │ -5.14%     │ │ NSE • Live via Kite API │ │
│ ↗ Top Movers                 │ └────────────┘ └────────────┘ └────────────┘ └────────────┘ └─────────────────────────┘ │
│ ◈ Sector View                │                                                                                          │
│ 🔎 Stock Explorer            │ ┌──────────────────────────────────────────────┐ ┌─────────────────────────────────────┐│
│ ▦ Dashboards                 │ │ NIFTY 50 Intraday Chart                      │ │ Top Movers (NIFTY 50)                ││
│ ◈ OpenBB Analysis            │ │ [1D] [1W] [1M] [3M] [1Y]                   │ │ [Gainers] [Losers] [High Volume]     ││
│ 🔥 Prometheus           ↗    │ │                                            │ │                                     ││
│ 🟠 Grafana              ↗    │ │ Price line/area chart                      │ │ # Symbol    LTP      Change    %     ││
│ 🔴 AlertManager         ↗    │ │ Volume bars along bottom                    │ │ 1 ADANIENT  ...      ...       ...   ││
│ ⚙ Settings                  │ │                                            │ │ 2 SBIN      ...      ...       ...   ││
│                              │ │                                            │ │ 3 BEL       ...      ...       ...   ││
│                              │ └──────────────────────────────────────────────┘ │ 4 TATAMOTORS ...    ...       ...   ││
│                              │                                                │ 5 NTPC      ...      ...       ...   ││
│                              │                                                └─────────────────────────────────────┘│
│                              │                                                                                          │
│                              │ ┌───────────────────────────────────────┐ ┌──────────────────────────────────────────┐ │
│                              │ │ 🔔 Active Alerts [12]       View All →│ │ ★ Watchlist                    View All →│ │
│                              │ │ Time Symbol Alert       Value Status  │ │ Symbol   LTP     Change   %   Mini Chart │ │
│                              │ │ 09:41 RELIANCE Price > R1 2,842 Firing│ │ RELIANCE ...     ...      ...   ╱╲╱╲     │ │
│                              │ │ 09:38 INFY Volume Spike  12.4M Firing│ │ TCS      ...     ...      ...   ╲╱╲╱     │ │
│                              │ │ 09:35 TCS RSI > 70       72.3 Firing │ │ INFY     ...     ...      ...   ╱╲╱╲     │ │
│                              │ │ 09:32 HDFCBANK MACD      ...  Firing │ │ HDFCBANK ...     ...      ...   ╱╲╱╲     │ │
│                              │ │ 09:28 ADANIENT > 20 EMA  ... Resolved│ │ ICICIBANK...     ...      ...   ╱╲╱╲     │ │
│                              │ │ 09:25 ITC High Volume    ... Resolved│ │                                          │ │
│                              │ └───────────────────────────────────────┘ └──────────────────────────────────────────┘ │
│                              │                                                                                          │
│                              │ ┌──────────────────────────────────────────────────────────────────────────────────────┐ │
│                              │ │ 🔗 Quick Links                                                                       │ │
│                              │ │ [Grafana Dashboards ↗] [Prometheus ↗] [AlertManager ↗] [OpenBB ↗]                   │ │
│                              │ │ Visualize metrics       Metrics & Rules   Manage alerts    Fundamental & Technical   │ │
│                              │ └──────────────────────────────────────────────────────────────────────────────────────┘ │
├──────────────────────────────┴──────────────────────────────────────────────────────────────────────────────────────────┤
│ stock_home | Stock Observability Platform | v0.1.0        Data via Zerodha Kite API | NSE | ● Live                     │
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Sidebar

**Order and labels must match the reference:**

1. `stock_home` logo and “Stock Observability Platform”
2. Home — active
3. Market Overview
4. Watchlist
5. Alerts — red count badge (`12` in the reference)
6. Top Movers
7. Sector View
8. Stock Explorer
9. Dashboards
10. OpenBB Analysis
11. Prometheus — external-link indicator
12. Grafana — external-link indicator
13. AlertManager — external-link indicator
14. Settings

Keep the Market Status card anchored at the bottom of the sidebar:

```text
┌──────────────────────────────┐
│ Market Status       ● OPEN   │
│ NSE                          │
│ 09:15 AM – 03:30 PM           │
│ Live data via Zerodha Kite API│
└──────────────────────────────┘
```

## 3. Header

Left-to-right order:

1. Sidebar collapse/expand button
2. Global search: `Search stocks (e.g. RELIANCE, TCS, INFY)...`
3. Date and time
4. Green `Live Market` indicator
5. Settings icon
6. Notification bell with count badge
7. User avatar

The header spans the content area to the right of the sidebar.

## 4. Market summary row

Five cards in a single desktop row: four index cards followed by Market Status.

| Card | Main value | Change shown | Visual |
|---|---:|---:|---|
| NIFTY 50 | 24,892.30 | +156.70 (+0.63%) | Green mini sparkline |
| BANK NIFTY | 54,321.10 | +412.80 (+0.77%) | Green mini sparkline |
| SENSEX | 81,432.61 | +521.34 (+0.64%) | Green mini sparkline |
| INDIA VIX | 12.54 | -0.68 (-5.14%) | Red mini sparkline |
| Market Status | OPEN | NSE session hours | Status + data-source text |

These are reference-image examples, not hardcoded market data.

## 5. Main row — chart and movers

The NIFTY chart occupies the wider left panel; Top Movers occupies the right panel.

### 5.1 NIFTY 50 Intraday Chart

```text
┌──────────────────────────────────────────────────────────────────┐
│ NIFTY 50 Intraday Chart                 [1D] [1W] [1M] [3M] [1Y]│
├──────────────────────────────────────────────────────────────────┤
│ Price axis                                                       │
│ 24,950 ┤                                                         │
│ 24,900 ┤       ╱╲___╱╲___╱╲___ current-price marker              │
│ 24,850 ┤  ___╱                                                    │
│        └───────────────────────────────────────────────────────   │
│          09:15    10:00    11:00    12:00    13:00    15:15       │
│          ▂▃▂▅▃▂▃▅▃▂▃▂▅▃  volume bars                             │
└──────────────────────────────────────────────────────────────────┘
```

Required elements:
- Title: `NIFTY 50 Intraday Chart`
- Timeframe controls: `1D`, `1W`, `1M`, `3M`, `1Y`
- Green price line and subtle area fill
- Price-axis labels and time-axis labels
- Volume bars at the bottom
- Horizontal current-price line and value label

Selecting a timeframe updates the period. Hovering a point can reveal time, price, and volume.

### 5.2 Top Movers (NIFTY 50)

```text
┌─────────────────────────────────────────────────────────────┐
│ Top Movers (NIFTY 50)                                       │
│ [Gainers] [Losers] [High Volume]                            │
├─────────────────────────────────────────────────────────────┤
│ #  Symbol       LTP (₹)      Change        %                │
│ 1  ADANIENT     3,142.60     +126.40     +4.20%             │
│ 2  SBIN           621.35      +18.75     +3.11%             │
│ 3  BEL            312.40       +8.90     +2.93%             │
│ 4  TATAMOTORS     992.10      +26.30     +2.72%             │
│ 5  NTPC           412.85      +10.45     +2.59%             │
└─────────────────────────────────────────────────────────────┘
```

Required elements:
- Title and three tabs: `Gainers`, `Losers`, `High Volume`
- Columns: rank, symbol, LTP, change, percentage change
- Five visible rows
- Green positive and red negative values
- Optional `View All` link if it exists in the final screenshot/design

## 6. Lower row — Active Alerts and Watchlist

Two side-by-side panels.

### 6.1 Active Alerts

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ 🔔 Active Alerts [12]                                           View All → │
├────────────────────────────────────────────────────────────────────────────┤
│ Time   Symbol      Alert                      Value       Status            │
│ 09:41  RELIANCE    Price > Resistance (R1)     2,842.50    Firing            │
│ 09:38  INFY        Volume Spike (5x)           12.4M       Firing            │
│ 09:35  TCS         RSI > 70 (Overbought)       72.3        Firing            │
│ 09:32  HDFCBANK    MACD Bullish Crossover      1,672.10    Firing            │
│ 09:28  ADANIENT    Price > 20 EMA              3,142.60    Resolved          │
│ 09:25  ITC         High Volume Breakout        485.20      Resolved          │
└────────────────────────────────────────────────────────────────────────────┘
```

Columns: time, symbol, alert description, value, status. Show compact rows, a severity/status indicator, and a count badge.

### 6.2 Watchlist

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ ★ Watchlist                                                      View All →│
├────────────────────────────────────────────────────────────────────────────┤
│ Symbol     LTP (₹)      Change      %       Mini Chart                     │
│ RELIANCE   2,842.50     +36.80    +1.31%    ╱╲╱╲                           │
│ TCS        3,412.20     -12.35    -0.36%    ╲╱╲╱                           │
│ INFY       1,487.30     +18.90    +1.29%    ╱╲╱╲                           │
│ HDFCBANK   1,672.10     +22.45    +1.36%    ╱╲╱╲                           │
│ ICICIBANK  1,245.80     +10.20    +0.82%    ╱╲╱╲                           │
└────────────────────────────────────────────────────────────────────────────┘
```

Columns: symbol, LTP, change, percentage change, mini chart. Keep five visible rows and align the sparklines to the right.

## 7. Quick Links

Full-width panel below Active Alerts and Watchlist, above the footer.

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ 🔗 Quick Links                                                                                           │
├──────────────────────────┬──────────────────────────┬──────────────────────────┬─────────────────────────┤
│ Grafana Dashboards ↗     │ Prometheus ↗             │ AlertManager ↗           │ OpenBB ↗                │
│ Visualize stock metrics  │ Metrics & Rules          │ Manage alerts            │ Fundamental & Technical │
│                          │                          │                          │ Analysis                │
└──────────────────────────┴──────────────────────────┴──────────────────────────┴─────────────────────────┘
```

| Card | Subtitle | Action |
|---|---|---|
| Grafana Dashboards | Visualize stock metrics | Open configured Grafana URL |
| Prometheus | Metrics & Rules | Open configured Prometheus URL |
| AlertManager | Manage alerts | Open configured AlertManager URL |
| OpenBB | Fundamental & Technical Analysis | Open configured OpenBB destination |

Use configured URLs rather than assuming ports or hosts.

## 8. Footer

```text
stock_home | Stock Observability Platform | v0.1.0
Data via Zerodha Kite API | NSE | ● Live
```

## 9. Desktop proportions and ordering

| Region | Guidance |
|---|---|
| Sidebar | Fixed width, approximately 240–260 px |
| Main content | Remaining viewport width with consistent padding |
| Summary row | Five cards in one row |
| Chart + Top Movers | Chart wider than movers panel |
| Active Alerts + Watchlist | Two approximately equal-width panels |
| Quick Links | Four cards across one row |
| Footer | Full content width at bottom |

**Preserve this vertical order:**
1. Header
2. Four index cards + Market Status
3. NIFTY 50 Intraday Chart + Top Movers
4. Active Alerts + Watchlist
5. Quick Links
6. Footer

## 10. Visual style

- Dark navy/charcoal background.
- Slightly lighter cards with thin, low-contrast borders.
- Blue active navigation and primary highlights.
- Green for positive market movement; red for negative movement.
- Compact, readable sans-serif typography.
- Rounded cards and consistent spacing.
- Dense dashboard layout with no oversized whitespace.
- Charts use subtle grid lines, clear axes, a green price line/area, and volume bars.
- Status indicators include text as well as color.

## 11. Out of scope

Do not add the following to this Home page wireframe:
- Settings configuration panels
- Full Watchlist management controls
- Full Alerts filtering and rule editor
- Full Top Movers analytics
- Sector heatmap and sector performance tables
- Stock Exporter operational controls
- New KPI cards or panels not shown in the Home reference

## 12. Acceptance checklist

- [ ] Sidebar labels and order match the reference.
- [ ] Home is the active navigation item.
- [ ] Header contains search, date/time, Live Market, settings, notifications, and avatar.
- [ ] Four market-index cards plus Market Status appear in one desktop row.
- [ ] Intraday chart is left; Top Movers is right.
- [ ] Active Alerts is lower left; Watchlist is lower right.
- [ ] Quick Links contains four cards in the reference order.
- [ ] Footer includes product, version, data source, exchange, and live status.
- [ ] No unrelated panels are introduced.
- [ ] Sample values are placeholders, not hardcoded live data.

**End of Home page wireframe.**
