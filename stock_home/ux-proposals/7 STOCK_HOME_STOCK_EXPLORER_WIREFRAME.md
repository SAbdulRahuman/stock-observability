# Stock Home — Stock Explorer Page Wireframe

**Purpose:** Let users find and inspect NSE stocks from one searchable, filterable list.

> **Scope note:** This is a text wireframe for the Stock Explorer page only. It follows the existing Stock Home dashboard pattern (shared header and left navigation). Since no Stock Explorer reference image was available in this turn, the page-specific content below is a proposed baseline—not a claim that it exactly matches an unseen mockup. Keep the implementation aligned to the approved UI image if one exists.

## 1. Page structure

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ STOCK HOME                                      Market status / last updated      Profile    │
├───────────────────┬──────────────────────────────────────────────────────────────────────────┤
│                   │ Stock Explorer                                                           │
│  Home             │ Find and inspect NSE stocks                                               │
│  Market Overview  │                                                                          │
│  Stock Explorer   │ ┌──────────────────────────────────────────────────────────────────────┐ │
│  Watchlist        │ │ Search stocks by symbol or company name…                     [Search]│ │
│  Alerts           │ └──────────────────────────────────────────────────────────────────────┘ │
│  Top Movers       │                                                                          │
│  Sector View      │ [Exchange: NSE ▼] [Sector: All ▼] [Price range ▼] [Change ▼] [Clear]    │
│                   │                                                                          │
│                   │ ┌──────────────────────────────────────────────────────────────────────┐ │
│                   │ │ Results: N stocks                         [Sort: Symbol ▼] [Refresh] │ │
│                   │ ├────────┬────────────────┬────────┬─────────┬──────────┬───────────────┤ │
│                   │ │ Symbol │ Company        │ Price  │ Change  │ Volume   │ Actions       │ │
│                   │ ├────────┼────────────────┼────────┼─────────┼──────────┼───────────────┤ │
│                   │ │ RELIANCE│ Reliance Ind. │ ₹—     │ —       │ —        │ [View] [☆]    │ │
│                   │ │ TCS     │ Tata Cons.    │ ₹—     │ —       │ —        │ [View] [☆]    │ │
│                   │ │ INFY    │ Infosys       │ ₹—     │ —       │ —        │ [View] [☆]    │ │
│                   │ │ …       │ …             │ …      │ …       │ …        │ …             │ │
│                   │ └────────┴────────────────┴────────┴─────────┴──────────┴───────────────┘ │
│                   │                                                                          │
│                   │ Showing 1–N of N stocks                           [‹ Previous] [Next ›]  │
├───────────────────┴──────────────────────────────────────────────────────────────────────────┤
│ Data source / market-data timestamp                                                          │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 2. Component details

### A. Shared application shell
- Reuse the same Stock Home header, left sidebar, active-navigation treatment, and footer/data timestamp used by the other pages.
- Mark **Stock Explorer** as the active navigation item.
- Keep the content area consistent with the existing page grid and spacing.

### B. Page heading
- **Title:** `Stock Explorer`
- **Subtitle:** `Find and inspect NSE stocks`
- Do not add summary KPI cards unless they are present in the approved visual reference.

### C. Search bar
- One prominent search field for stock symbol or company name.
- Search action can be triggered by Enter or the Search control.
- Provide a clear/reset affordance when a query is entered.
- Empty state should explain that users can search by symbol or company name.

### D. Filters
- Exchange selector, defaulting to NSE if NSE is the supported market.
- Sector selector, default `All`.
- Price range filter.
- Percentage-change filter.
- `Clear` resets all filters.
- Only display filters that are supported by the available stock data; do not imply unavailable data is filterable.

### E. Results table
| Column | Content |
|---|---|
| Symbol | NSE ticker symbol |
| Company | Company/display name |
| Price | Latest available price |
| Change | Price change or percentage change, using the same convention as the rest of Stock Home |
| Volume | Latest available volume |
| Actions | Open stock details and add/remove from watchlist, if those actions are supported |

- Clicking a row or **View** opens the corresponding stock detail view.
- Use consistent positive/negative/neutral styling for price changes.
- Show loading, no-results, stale-data, and error states without replacing the overall page layout.
- Use pagination or the project's existing table pagination pattern for large result sets.

## 3. Interaction states

1. **Initial:** Show the default stock list and current data timestamp.
2. **Searching:** Filter results by symbol/company name.
3. **Filtered:** Show active filters and a clear-filters action.
4. **No results:** Display “No stocks match your search or filters” with a way to clear them.
5. **Loading/error:** Keep the table frame and show a concise status message.
6. **Stale data:** Show the last-updated timestamp; do not present old values as live.

## 4. Data and implementation notes

- Use the existing Stock Home data/API contract; do not invent a new backend endpoint in the wireframe.
- Use only fields actually provided by `stock_exporter` or the app's stock catalogue.
- Treat missing values as `—`; never fabricate prices, changes, or volumes.
- If the app's data is delayed or unavailable, label the status accurately.
- Keep this page focused on stock discovery. Detailed charts and advanced analytics belong in the stock detail view unless the approved reference image explicitly includes them.

## 5. Acceptance checklist

- [ ] Shared Stock Home shell matches the other pages.
- [ ] Stock Explorer is active in the sidebar.
- [ ] Search accepts ticker symbols and company names.
- [ ] Filters reflect fields that actually exist in the data.
- [ ] Table columns and number formatting are consistent with Stock Home.
- [ ] Loading, empty, error, and stale-data states are defined.
- [ ] No invented KPIs, charts, or extra panels are added without design approval.
