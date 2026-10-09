/** OBBject responses as returned by OpenBB with the yfinance provider, trimmed for tests. */
const envelope = (results: unknown[], provider = "yfinance") => ({ results, provider, warnings: null, chart: null, extra: {} });

export const fixtures: Record<string, unknown> = {
  "equity/price/quote": envelope([
    {
      symbol: "RELIANCE.NS",
      name: "Reliance Industries Limited",
      exchange: "NSI",
      currency: "INR",
      last_price: 1412.3,
      open: 1398,
      high: 1418.9,
      low: 1395.1,
      prev_close: 1400.5,
      change: 11.8,
      change_percent: 0.008426,
      volume: 8123456,
      year_high: 1551,
      year_low: 1114.85,
    },
  ]),
  "equity/price/historical": envelope([
    { date: "2026-10-01", open: 1380, high: 1395, low: 1377, close: 1390, volume: 7000000 },
    { date: "2026-10-05", open: 1390, high: 1402, low: 1385, close: 1400.5, volume: 6500000 },
    { date: "2026-10-06", open: 1398, high: 1418.9, low: 1395.1, close: 1412.3, volume: 8123456 },
  ]),
  "equity/profile": envelope([
    {
      symbol: "RELIANCE.NS",
      name: "Reliance Industries Limited",
      stock_exchange: "NSI",
      sector: "Energy",
      industry_category: "Oil & Gas Refining & Marketing",
      employees: 347362,
      hq_country: "India",
      company_url: "https://www.ril.com",
      long_description: "Reliance Industries Limited engages in hydrocarbon exploration and production, oil to chemicals, retail, and digital services.",
    },
  ]),
  "news/company": envelope([
    {
      date: "2026-10-06T08:30:00Z",
      title: "Reliance shares rise ahead of quarterly results",
      url: "https://example.com/reliance-results",
      source: "Example Wire",
    },
    {
      date: "2026-10-05T11:00:00Z",
      title: "Untrusted link is shown as text",
      url: "javascript:alert(1)",
    },
  ]),
  "yfinance/search": envelope([
    { symbol: "TCS.NS", name: "Tata Consultancy Services Limited", exchange: "NSI" },
    { symbol: "TCS.BO", name: "Tata Consultancy Services Limited", exchange: "BSE" },
  ]),
  "equity/fundamental/metrics": envelope([
    { symbol: "RELIANCE.NS", market_cap: 19110000000000, pe_ratio: 23.4, return_on_equity: 0.0912, beta: 0.41, fiscal_year: 2026 },
  ]),
  "equity/fundamental/income": envelope([
    { period_ending: "2026-03-31", fiscal_period: "FY", total_revenue: 9640000000000, net_income: 696000000000 },
    { period_ending: "2025-03-31", fiscal_period: "FY", total_revenue: 9010000000000 },
  ]),
};
