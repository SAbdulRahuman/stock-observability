# Stock Home UX Redesign

## Design Goal

Transform Stock Home from a link/landing page into a true product control plane.

## UX Model

```text
                    STOCK HOME
                        |
        +---------------+---------------+
        |               |               |
      OPERATE        INVESTIGATE       ADMIN
        |               |               |
     Alerts           Stocks          Config
     Signals          Research        Rules
     Market           OpenBB          Users
     Health           AI              Audit
```

## Home Wireframe

```text
+--------------------------------------------------------------------+
| Stock Observability | Search | NSE OPEN | Alerts | AI | User      |
+----------------------+---------------------------------------------+
| Navigation           | MARKET PULSE                                |
|                      | NIFTY | BANKNIFTY | VIX | ADV/DEC           |
| Home                 +---------------------------------------------+
| Market               | ATTENTION                                   |
| Stocks               | 3 Critical | 21 Warning | 87 Info         |
| Alerts               +---------------------------------------------+
| Signals              | TOP MOVERS | TOP SIGNALS | WATCHLIST       |
| Research             +---------------------------------------------+
| Dashboards           | DATA HEALTH                                 |
| Observability        | Exporter | Prometheus | AM | OpenBB        |
| Administration       +---------------------------------------------+
```

## Search

`Ctrl/Cmd + K`

Search entities:

- stocks
- alerts
- signals
- formulas
- rules
- dashboards
- research
- configuration

## Alert UX

Support:

- filtering
- grouping
- sorting
- saved views
- bulk operations
- virtualization
- incremental updates

Grouping example:

```text
RELIANCE
  └── 8 active alerts
      ├── Volume anomaly
      ├── RSI
      ├── Volatility
      └── Breakout
```

## Signal UX

Signal is not the same as alert.

```text
Alert
= condition requiring attention

Signal
= derived market condition
```

## Responsive Design

Desktop is primary.

Mobile should prioritize:

1. Home
2. Alerts
3. Stocks
4. Signals
5. Watchlists

## Keyboard Shortcuts

```text
Ctrl/Cmd + K  Search
G H           Home
G A           Alerts
G S           Stocks
G R           Research
G D           Dashboards
G O           Observability
G C           Configuration
?             Help
Esc           Close
```
