# OpenBB → stock_home parity backlog

Single source of truth for the OpenBB porting loop driven by the `openbb-porter` agent.
OpenBB source root: `OpenBB/OpenBB/` (paths below are relative to `OpenBB/OpenBB/openbb_platform/` unless they start with `cli/`).

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked (reason in Notes, WIP in `git stash list`) · `[-]` not portable (no in-repo or free upstream provider; reason in Notes)

Rules:
- Work top to bottom; a feature may start only when every ID in **Deps** is `[x]` or `[-]`.
- One feature = one local commit. The commit hash goes in **Notes**.
- A worker may split a large feature into `F###a`, `F###b`, … rows in place when the real command count is discovered.
- API path = `/api/v2/` + OpenBB command path (dots → slashes). Provider-specific routers mount at `/api/v2/<provider>/...`.

---

## Phase 0 — Foundation

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F001 | [ ] | Command registry, OBBject envelope (`results`, `provider`, `warnings`, `chart`, `extra`), `/api/v2/*` router, `GET /api/v2/commands` metadata, typed errors (400 invalid params, 401 missing credential, 404, 502 upstream) | `core/openbb_core/app/router.py`, `core/openbb_core/app/model/obbject.py`, `core/openbb_core/app/command_runner.py` | – | |
| F002 | [ ] | Provider framework: `Fetcher` (TransformQuery → Extract → Transform), provider registry + default provider per command, shared HTTP client (User-Agent, retry/backoff, per-host rate limit, BaseURL override), response cache reuse (`internal/cache`), `providertest` fixture-replay helper | `core/openbb_core/provider/abstract/fetcher.py`, `core/openbb_core/provider/registry.py`, `core/openbb_core/provider/utils/helpers.py` | F001 | |
| F003 | [ ] | SQLite store (`modernc.org/sqlite`, embedded migrations, `-store.path` flag) + credentials/settings (`STOCK_HOME_<PROVIDER>_<KEY>` env > flag > DB), `GET/PUT /api/v2/settings` (secrets write-only, never echoed) | `core/openbb_core/app/model/credentials.py`, `core/openbb_core/app/model/user_settings.py` | F001 | |
| F004 | [ ] | Web shell: sidebar nav by extension, `/explore` command browser driven by `/api/v2/commands`, query-param routing (static export: no dynamic segments), keep `/` NSE dashboard unchanged | `extensions/platform_api` (widget layout ideas) | F001 | |
| F005 | [ ] | Generic `CommandView` (param form from metadata, data table with sort/filter, chart: lightweight-charts for OHLC, Recharts for line/bar/area), `useCommand` hook with zod | `obbject_extensions/charting/openbb_charting/` | F004 | |
| F006 | [ ] | Export on every command: `?format=json\|csv\|xlsx` (excelize) + UI export menu | `core/openbb_core/app/model/obbject.py` (`to_df`, `to_csv`…) | F001, F005 | |
| F007 | [ ] | Analytics input pipeline: POST body `{data:[...]}` or `source` reference (`{command, params}`) resolved server-side; shared OHLCV types; add `gonum` | `extensions/technical/openbb_technical/indicators/*` (param shapes) | F002 | |
| F008 | [ ] | Parity tooling: `make parity` (registry vs this backlog, fails on drift), `scripts/golden/` dev-only generator for numeric reference outputs (python3/uv + OpenBB) with tolerance helpers in Go | – | F001 | |

## Phase 1 — Equity core

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F010 | [ ] | `equity/search` | `extensions/equity/openbb_equity/equity_router.py::search` | P0 | |
| F011 | [ ] | `equity/price/quote` | `extensions/equity/openbb_equity/price/price_router.py::quote` | F010 | |
| F012 | [ ] | `equity/price/historical` (+ candlestick chart) | `price_router.py::historical` | F010 | |
| F013 | [ ] | `equity/price/nbbo` | `price_router.py::nbbo` | F011 | |
| F014 | [ ] | `equity/price/performance` | `price_router.py::performance` | F012 | |
| F015 | [ ] | `equity/profile` | `equity_router.py::profile` | F010 | |
| F016 | [ ] | `equity/screener` | `equity_router.py::screener` | F010 | |
| F017 | [ ] | `equity/market_snapshots` | `equity_router.py::market_snapshots` | F010 | |
| F018 | [ ] | `equity/historical_market_cap` | `equity_router.py::historical_market_cap` | F012 | |
| F019 | [ ] | `nse` provider: quote + historical for NSE symbols from existing Prometheus metrics (`stock_last_price` etc.) | stock_home `internal/source/prometheus.go` | F011, F012 | |

## Phase 2 — Equity fundamentals (`extensions/equity/openbb_equity/fundamental/fundamental_router.py`)

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F020 | [ ] | `equity/fundamental/income`, `balance`, `cash` | `::income`, `::balance`, `::cash` | P1 | |
| F021 | [ ] | `equity/fundamental/income_growth`, `balance_growth`, `cash_growth` | `::*_growth` | F020 | |
| F022 | [ ] | `equity/fundamental/reported_financials` | `::reported_financials` | F020 | |
| F023 | [ ] | `equity/fundamental/dividends`, `trailing_dividend_yield` | `::dividends`, `::trailing_dividend_yield` | P1 | |
| F024 | [ ] | `equity/fundamental/historical_eps` | `::historical_eps` | P1 | |
| F025 | [ ] | `equity/fundamental/employee_count` | `::employee_count` | P1 | |
| F026 | [ ] | `equity/fundamental/search_attributes`, `latest_attributes`, `historical_attributes` | `::*_attributes` | P1 | |
| F027 | [ ] | `equity/fundamental/metrics` | `::metrics` | F020 | |
| F028 | [ ] | `equity/fundamental/ratios` | `::ratios` | F020 | |
| F029 | [ ] | `equity/fundamental/management`, `management_compensation` | `::management*` | P1 | |
| F030 | [ ] | `equity/fundamental/revenue_per_geography`, `revenue_per_segment` | `::revenue_per_*` | F020 | |
| F031 | [ ] | `equity/fundamental/filings` | `::filings` | P1 | |
| F032 | [ ] | `equity/fundamental/historical_splits` | `::historical_splits` | P1 | |
| F033 | [ ] | `equity/fundamental/transcript` | `::transcript` | P1 | |
| F034 | [ ] | `equity/fundamental/management_discussion_analysis` | `::management_discussion_analysis` | F031 | |
| F035 | [ ] | `equity/fundamental/esg_score` | `::esg_score` | P1 | |

## Phase 3 — Equity extended

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F040 | [ ] | `equity/estimates/price_target`, `historical`, `consensus` | `equity/openbb_equity/estimates/estimates_router.py` | P1 | |
| F041 | [ ] | `equity/estimates/analyst_search` | `estimates_router.py::analyst_search` | P1 | |
| F042 | [ ] | `equity/estimates/forward_sales`, `forward_ebitda`, `forward_eps`, `forward_pe` | `estimates_router.py::forward_*` | P1 | |
| F043 | [ ] | `equity/calendar/ipo` | `equity/openbb_equity/calendar/calendar_router.py::ipo` | P1 | |
| F044 | [ ] | `equity/calendar/dividend`, `splits` | `calendar_router.py` | P1 | |
| F045 | [ ] | `equity/calendar/events`, `earnings` | `calendar_router.py` | P1 | |
| F046 | [ ] | `equity/compare/peers`, `groups` | `equity/openbb_equity/compare/compare_router.py` | P1 | |
| F047 | [ ] | `equity/compare/company_facts` | `compare_router.py::company_facts` | P1 | |
| F048 | [ ] | `equity/discovery/gainers`, `losers`, `active` | `equity/openbb_equity/discovery/discovery_router.py` | P1 | |
| F049 | [ ] | `equity/discovery/undervalued_large_caps`, `undervalued_growth`, `aggressive_small_caps`, `growth_tech` | `discovery_router.py` | P1 | |
| F050 | [ ] | `equity/discovery/top_retail` | `discovery_router.py::top_retail` | P1 | |
| F051 | [ ] | `equity/discovery/filings`, `latest_financial_reports` | `discovery_router.py` | P1 | |
| F052 | [ ] | `equity/ownership/major_holders`, `institutional`, `share_statistics` | `equity/openbb_equity/ownership/ownership_router.py` | P1 | |
| F053 | [ ] | `equity/ownership/insider_trading` | `ownership_router.py::insider_trading` | P1 | |
| F054 | [ ] | `equity/ownership/form_13f` | `ownership_router.py::form_13f` | P1 | |
| F055 | [ ] | `equity/ownership/government_trades` | `ownership_router.py::government_trades` | P1 | |
| F056 | [ ] | `equity/shorts/fails_to_deliver`, `short_volume`, `short_interest` | `equity/openbb_equity/shorts/shorts_router.py` | P1 | |
| F057 | [ ] | `equity/darkpool/otc` | `equity/openbb_equity/darkpool/darkpool_router.py` | P1 | |

## Phase 4 — ETF & Index

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F060 | [ ] | `etf/search`, `info` | `extensions/etf/openbb_etf/etf_router.py` | P1 | |
| F061 | [ ] | `etf/historical`, `price_performance` | `etf_router.py` | F060 | |
| F062 | [ ] | `etf/holdings`, `sectors`, `countries` | `etf_router.py` | F060 | |
| F063 | [ ] | `etf/nport_disclosure` | `etf_router.py::nport_disclosure` | F060 | |
| F064 | [ ] | `etf/equity_exposure` | `etf_router.py::equity_exposure` | F062 | |
| F065 | [ ] | `index/available`, `search` | `extensions/index/openbb_index/index_router.py` | P1 | |
| F066 | [ ] | `index/constituents` | `index_router.py::constituents` | F065 | |
| F067 | [ ] | `index/price/historical`, `index/snapshots` | `index/openbb_index/price/price_router.py`, `index_router.py::snapshots` | F065 | |
| F068 | [ ] | `index/sp500_multiples` | `index_router.py::sp500_multiples` | F065 | |
| F069 | [ ] | `index/sectors` | `index_router.py::sectors` | F065 | |

## Phase 5 — Derivatives, Crypto, Currency, Commodity

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F070 | [ ] | `derivatives/options/chains` | `extensions/derivatives/openbb_derivatives/options/options_router.py` | P1 | |
| F071 | [ ] | `derivatives/options/surface` (surface/heatmap chart) | `options_router.py::surface` | F070 | |
| F072 | [ ] | `derivatives/options/unusual`, `snapshots` | `options_router.py` | F070 | |
| F073 | [ ] | `derivatives/futures/historical`, `curve` | `derivatives/openbb_derivatives/futures/futures_router.py` | P0 | |
| F074 | [ ] | `derivatives/futures/instruments`, `info` | `futures_router.py` | F073 | |
| F075 | [ ] | `crypto/search`, `crypto/price/historical` | `extensions/crypto/openbb_crypto/` | P0 | |
| F076 | [ ] | `currency/search`, `currency/price/historical` | `extensions/currency/openbb_currency/` | P0 | |
| F077 | [ ] | `currency/reference_rates`, `snapshots` | `currency_router.py` | F076 | |
| F078 | [ ] | `commodity/price/spot` | `extensions/commodity/openbb_commodity/price/price_router.py` | P0 | |
| F079 | [ ] | `commodity/petroleum_status_report`, `short_term_energy_outlook` | `commodity/openbb_commodity/commodity_router.py` | P0 | |
| F080 | [ ] | `commodity/psd_data`, `psd_report` | `commodity_router.py` | P0 | |
| F081 | [ ] | `commodity/weather_bulletins`, `weather_bulletins_download` | `commodity_router.py` | P0 | |

## Phase 6 — Economy (`extensions/economy/openbb_economy/`)

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F090 | [ ] | `economy/calendar` | `economy_router.py::calendar` | P0 | |
| F091 | [ ] | `economy/cpi`, `retail_prices`, `pce` | `economy_router.py` | P0 | |
| F092 | [ ] | `economy/fred_search`, `fred_series`, `fred_release_table`, `fred_regional` | `economy_router.py` | P0 | |
| F093 | [ ] | `economy/risk_premium` | `economy_router.py` | P0 | |
| F094 | [ ] | `economy/balance_of_payments`, `direction_of_trade`, `export_destinations` | `economy_router.py` | P0 | |
| F095 | [ ] | `economy/money_measures` | `economy_router.py` | P0 | |
| F096 | [ ] | `economy/unemployment`, `economy/survey/nonfarm_payrolls` | `economy_router.py`, `survey/survey_router.py` | P0 | |
| F097 | [ ] | `economy/composite_leading_indicator` | `economy_router.py` | P0 | |
| F098 | [ ] | `economy/country_profile` | `economy_router.py` | P0 | |
| F099 | [ ] | `economy/available_indicators`, `indicators` | `economy_router.py` | P0 | |
| F100 | [ ] | `economy/central_bank_holdings` | `economy_router.py` | P0 | |
| F101 | [ ] | `economy/share_price_index`, `house_price_index` | `economy_router.py` | P0 | |
| F102 | [ ] | `economy/interest_rates` | `economy_router.py` | P0 | |
| F103 | [ ] | `economy/primary_dealer_positioning`, `primary_dealer_fails` | `economy_router.py` | P0 | |
| F104 | [ ] | `economy/fomc_documents` | `economy_router.py` | P0 | |
| F105 | [ ] | `economy/total_factor_productivity` | `economy_router.py` | P0 | |
| F106 | [ ] | `economy/gdp/forecast`, `nominal`, `real` | `gdp/gdp_router.py` | P0 | |
| F107 | [ ] | `economy/shipping/port_info`, `port_volume`, `chokepoint_info`, `chokepoint_volume` | `shipping/shipping_router.py` | P0 | |
| F108 | [ ] | `economy/survey/bls_series`, `bls_search` | `survey/survey_router.py` | P0 | |
| F109 | [ ] | `economy/survey/sloos`, `university_of_michigan`, `inflation_expectations` | `survey/survey_router.py` | P0 | |
| F110 | [ ] | `economy/survey/economic_conditions_chicago`, `manufacturing_outlook_texas`, `manufacturing_outlook_ny` | `survey/survey_router.py` | P0 | |

## Phase 7 — Fixed income (`extensions/fixedincome/openbb_fixedincome/`)

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F120 | [ ] | `fixedincome/government/yield_curve`, `svensson_yield_curve` | `government/government_router.py` | P0 | |
| F121 | [ ] | `fixedincome/government/treasury_rates`, `tips_yields` | `government_router.py` | P0 | |
| F122 | [ ] | `fixedincome/government/treasury_auctions`, `treasury_prices` | `government_router.py` | P0 | |
| F123 | [ ] | `fixedincome/rate/sofr`, `effr`, `effr_forecast`, `iorb`, `overnight_bank_funding`, `dpcredit` | `rate/rate_router.py` | P0 | |
| F124 | [ ] | `fixedincome/rate/ameribor`, `sonia`, `estr`, `ecb` | `rate/rate_router.py` | P0 | |
| F125 | [ ] | `fixedincome/spreads/tcm`, `tcm_effr`, `treasury_effr` | `spreads/spreads_router.py` | P0 | |
| F126 | [ ] | `fixedincome/corporate/hqm`, `spot_rates`, `commercial_paper` | `corporate/corporate_router.py` | P0 | |
| F127 | [ ] | `fixedincome/corporate/bond_prices` | `corporate_router.py::bond_prices` | P0 | |
| F128 | [ ] | `fixedincome/bond_indices`, `mortgage_indices` | `fixedincome_router.py` | P0 | |

## Phase 8 — News (`extensions/news/openbb_news/`)

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F130 | [ ] | `news/world`, `news/company` | `news_router.py` | P0 | |
| F131 | [ ] | `news/rss`, `rss_providers`, `rss_feeds` (port feed registry) | `rss.py`, `registry.py` | P0 | |
| F132 | [ ] | RSS article body fetch + HTML→markdown/truncate (SSRF-safe: registry URLs only) | `parser.py` | F131 | |

## Phase 9 — Technical (`extensions/technical/openbb_technical/`)

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F140 | [ ] | `technical/sma`, `ema`, `wma`, `hma` | `indicators/overlays.py` | F007, F012 | |
| F141 | [ ] | `technical/zlma`, `tema`, `dema` | `indicators/overlays.py` | F140 | |
| F142 | [ ] | `technical/kama`, `frama`, `vwma` | `indicators/overlays.py` | F140 | |
| F143 | [ ] | `technical/bbands`, `donchian`, `kc` | `indicators/overlays.py` | F140 | |
| F144 | [ ] | `technical/ichimoku`, `supertrend` | `indicators/overlays.py` | F140 | |
| F145 | [ ] | `technical/rsi`, `stoch`, `williams_r` | `indicators/oscillators.py` | F140 | |
| F146 | [ ] | `technical/cci`, `fisher`, `cg` | `indicators/oscillators.py` | F140 | |
| F147 | [ ] | `technical/mfi`, `trix`, `ultimate_oscillator`, `awesome_oscillator` | `indicators/oscillators.py` | F140 | |
| F148 | [ ] | `technical/macd`, `adx`, `di` | `indicators/trend.py` | F140 | |
| F149 | [ ] | `technical/aroon`, `choppiness` | `indicators/trend.py` | F140 | |
| F150 | [ ] | `technical/atr`, `realized_volatility` (std, parkinson, garman_klass, hodges_tompkins, rogers_satchell, yang_zhang) | `indicators/volatility.py`, `helpers.py` | F140 | |
| F151 | [ ] | `technical/realized_volatility_compare`, `cones` | `indicators/volatility.py` | F150 | |
| F152 | [ ] | `technical/obv`, `ad`, `adosc`, `vwap` | `indicators/volume.py` | F140 | |
| F153 | [ ] | `technical/clenow`, `drawdown`, `returns_stats` | `indicators/statistics.py` | F140 | |
| F154 | [ ] | `technical/stationarity`, `hurst`, `autocorrelation` | `indicators/statistics.py` | F140 | |
| F155 | [ ] | `technical/fib`, `demark`, `pivot_points` | `indicators/structure.py` | F140 | |
| F156 | [ ] | `technical/relative_rotation` (RRG chart) | `indicators/relative_rotation.py`, `relative_rotation.py` | F140 | |
| F157 | [ ] | `technical/crossovers`, `breakouts` | `signals/crossovers.py`, `signals/breakouts.py` | F148 | |
| F158 | [ ] | `technical/divergences`, `oscillator_signals` | `signals/divergences.py`, `signals/thresholds.py` | F145 | |
| F159 | [ ] | `technical/candlestick_patterns` | `signals/patterns.py` | F140 | |
| F160 | [ ] | `technical/regime` | `signals/regime.py` | F150 | |
| F161 | [ ] | `technical/indicators` (catalog), `technical/multi` | `multi/catalog.py`, `multi/compose.py` | F140–F156 | |
| F162 | [ ] | `technical/correlation`, `correlation_matrix` | `multi/correlation.py` | F140 | |
| F163 | [ ] | `technical/screen` | `multi/screen.py` | F161 | |
| F164 | [ ] | Price-chart indicator picker (overlay/sub-pane any indicator on `equity/price/historical` chart) | `obbject_extensions/charting/indicators.md` | F161 | |

## Phase 10 — Quantitative (`extensions/quantitative/openbb_quantitative/`)

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F170 | [ ] | `quantitative/stats/skew`, `variance`, `stdev`, `kurtosis`, `mean`, `quantile` | `stats.py` | F007 | |
| F171 | [ ] | `quantitative/rolling/skew`, `variance`, `stdev`, `kurtosis`, `mean`, `quantile` | `rolling.py` | F170 | |
| F172 | [ ] | `quantitative/normality`, `unitroot_test`, `summary` | `metrics.py` | F170 | |
| F173 | [ ] | `quantitative/capm` | `metrics.py::capm` | F170, F175 | |
| F174 | [ ] | `quantitative/performance/omega_ratio`, `sharpe_ratio`, `sortino_ratio` | `performance.py` | F170 | |
| F175 | [ ] | `famafrench` provider factors + `quantitative/factors`, `quantitative/rolling/factors` | `factors.py`, `rolling.py::factors`, `_factor_helpers.py`, `providers/famafrench` | F170 | |
| F176 | [ ] | `quantitative/attribution` | `attribution.py` | F175 | |
| F177 | [ ] | `quantitative/risk_decomposition` | `risk_decomposition.py` | F175 | |
| F178 | [ ] | Quant charts: factors heatmap, attribution bar, risk-decomposition bar, rolling-factors line | `views/*.py` | F175–F177 | |

## Phase 11 — Econometrics (`extensions/econometrics/openbb_econometrics/`)

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F180 | [ ] | `econometrics/correlation_matrix`, `summary_statistics` | `tools.py` | F007 | |
| F181 | [ ] | `econometrics/ols_regression`, `ols_regression_summary` | `regression.py` | F180 | |
| F182 | [ ] | `econometrics/autocorrelation`, `residual_autocorrelation`, `heteroskedasticity`, `normality`, `variance_inflation_factor` | `diagnostics.py` | F181 | |
| F183 | [ ] | `econometrics/unit_root` (ADF), `kpss` | `time_series.py` | F180 | |
| F184 | [ ] | `econometrics/cointegration` (Engle-Granger), `cointegration_johansen` | `time_series.py`, `utils.py` | F181, F183 | |
| F185 | [ ] | `econometrics/causality` (Granger) | `time_series.py::causality` | F181 | |
| F186 | [ ] | `econometrics/garch` | `time_series.py::garch` | F180 | |
| F187 | [ ] | `econometrics/panel_random_effects`, `panel_between`, `panel_pooled`, `panel_fixed`, `panel_first_difference`, `panel_fmac` | `panel.py` | F181 | |

## Phase 12 — Provider-specific routers (`providers/*/openbb_*/*_router.py`)

Commands registered directly by provider packages. Many register dynamically; the worker enumerates exact commands from source on start and splits rows (`F2xxa`, `F2xxb`, …) when a row exceeds ~8 commands.

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F200 | [ ] | `sec/*` filings & search: filing_headers, full_text_search, htm_file, cik_map, symbol_map, sic_search, schema_files, rss_litigation, institutions_search | `providers/sec/openbb_sec/sec_router.py` | P0 | |
| F201 | [ ] | `sec/*` advisers: adviser_firms, adviser_individuals, adviser_documents, adviser_universe | `sec_router.py` | F200 | |
| F202 | [ ] | `sec/*` structured disclosures: financial_statements, company_overview, executive_compensation, beneficial_ownership, management_profiles, pay_versus_performance, disclosures, risk_factors, segment_revenue, legal_proceedings, exhibit, … (rest of router) | `sec_router.py` | F200 | |
| F203 | [ ] | `cboe/*` provider commands | `providers/cboe/openbb_cboe/cboe_router.py` | P0 | |
| F204 | [ ] | `nasdaq/*` provider commands | `providers/nasdaq/openbb_nasdaq/nasdaq_router.py` | P0 | |
| F205 | [ ] | `tmx/*` provider commands (covered-call screener, equity trades, symbol reference, short interest, bond trades, …) | `providers/tmx/openbb_tmx/tmx_router.py` | P0 | |
| F206 | [ ] | `finra/*` provider commands | `providers/finra/openbb_finra/finra_router.py` | P0 | |
| F207 | [ ] | `deribit/*` provider commands (announcements, book summary, combos, currencies, delivery prices, expirations, funding, …) | `providers/deribit/openbb_deribit/deribit_router.py` | P0 | |
| F208 | [ ] | `cftc/*` COT: cot_search, cot, cot_index, cot_movers, cot_positioning | `providers/cftc/openbb_cftc/cftc_router.py` | P0 | |
| F209 | [ ] | `cftc/*` swaps: swap_trades, swap_valuation, historical_fixings, swap_summary, cds_index_trades, ois_curve, fx forward curve/points, … | `cftc_router.py` | F208 | |
| F210 | [ ] | `bls/*` provider commands (CPI/Empsit documents, CES tables, CPI news-release tables, …) | `providers/bls/openbb_bls/bls_router.py` | P0 | |
| F211 | [ ] | `fred/*` provider commands | `providers/fred/openbb_fred/fred_router.py` | P0 | |
| F212 | [ ] | `ecb/*` dataflows, topics, concepts, dimensions, tables | `providers/ecb/openbb_ecb/ecb_router.py` | P0 | |
| F213 | [ ] | `eia/*` datasets, facets, coal/petroleum/natural_gas/electricity/electricity_grid/… groups | `providers/eia/openbb_us_eia/eia_router.py` | P0 | |
| F214 | [ ] | `famafrench/*` factors, factor_choices, us/regional/country portfolio returns, breakpoints | `providers/famafrench/openbb_famafrench/famafrench_router.py` | F175 | |
| F215 | [ ] | `federal_reserve/*` FOMC docs download/choices, market probabilities, regional publications/reports | `providers/federal_reserve/openbb_federal_reserve/federal_reserve_router.py` | P0 | |
| F216 | [ ] | `federal_reserve/*` bank data: institutions, call reports (+sectioned), UBPR, BHCPR, peer groups, state averages, structure, financial reports | `federal_reserve_router.py`, `models/call_report_router.py` | F215 | |
| F217 | [ ] | `federal_reserve/*` regional Fed datasets (Chicago, Cleveland, Dallas, New York, Philadelphia, …) | `federal_reserve_router.py` | F215 | |
| F218 | [ ] | `federal_reserve/*` remaining (money market funds, country exposure, international portfolio investment, data download, …) | `federal_reserve_router.py` | F215 | |
| F219 | [ ] | `imf/*` dataflows, dimensions, tables, choices | `providers/imf/openbb_imf/imf_router.py` | P0 | |
| F220 | [ ] | `imf/*` PortWatch: ports, countries, regions, disruptions, country_activity, … | `providers/imf/openbb_imf/portwatch_router.py` | F219 | |
| F221 | [ ] | `oecd/*` topics, subtopics, dataflows, parameters, tables | `providers/oecd/openbb_oecd/oecd_router.py` | P0 | |
| F222 | [ ] | `jodi/oil/*`, `jodi/gas/*` balance, production, demand, imports, exports, … | `providers/jodi/openbb_jodi/{jodi,oil,gas}_router.py` | P0 | |
| F223 | [ ] | `government_us/treasury/*` debt_to_penny, daily_statement, upcoming_auctions, auction_results, weekly_bill_offerings, bulletin | `providers/government_us/openbb_government_us/treasury/treasury_router.py` | P0 | |
| F224 | [ ] | `government_us/treasury/*` awards & spending: award, award_info, spending_explorer, … | `treasury_router.py` | F223 | |
| F225 | [ ] | `government_us/congress/*` bills, bill_info, bill_text(+urls), amendments(+info/text) | `providers/government_us/openbb_government_us/congress/congress_router.py` | P0 | |
| F226 | [ ] | `government_us/congress/*` laws, calendars, mandated_reports, search (+document urls) | `congress_router.py` | F225 | |
| F227 | [ ] | `government_us/usda/*` ERS charts, price spreads, bell report, publications, report calendar | `providers/government_us/openbb_government_us/usda/usda_router.py` | P0 | |
| F228 | [ ] | `government_us/usda/*` PSD & trade: psd report/data choices, agricultural_trade (+measures/commodities/periods), us_food_imports | `usda_router.py` | F227 | |
| F229 | [ ] | `government_us/usda/*` yearbooks & commodity data: dairy, milk cost, wheat, rice, sugar, oil crops, feed grains, fruit, vegetables, cotton, fertilizer, livestock, costs and returns, major land uses | `usda_router.py` | F227 | |
| F230 | [ ] | `government_us/usda/*` farm income & food economics: food price outlook, food dollar, food expenditure, farm income/household/wealth, season average price forecasts, … (rest of router) | `usda_router.py` | F227 | |

## Phase 13 — Platform surfaces

| ID | Status | Feature | OpenBB reference | Deps | Notes |
|----|--------|---------|------------------|------|-------|
| F240 | [ ] | Charting parity: every command with an OpenBB default chart gets a matching default view; chart params (`chart=true`) in OBBject | `obbject_extensions/charting/openbb_charting/` | P1–P11 | |
| F241 | [ ] | Settings & credentials UI (provider keys, default providers, theme, export prefs) | `core/openbb_core/app/model/user_settings.py` | F003 | |
| F242 | [ ] | Workspace backend: `GET /widgets.json`, `/apps.json`, `/templates.json` generated from registry | `extensions/platform_api/openbb_platform_api/` | P1–P12 | |
| F243 | [ ] | MCP server (`github.com/modelcontextprotocol/go-sdk`, streamable HTTP at `/mcp`), tools generated from registry, tool categories/discovery like OpenBB | `extensions/mcp_server/openbb_mcp_server/` | P1–P12 | |
| F244 | [ ] | Go CLI: `stock_home cli` interactive menus mirroring command tree, `--provider`, output table/json/csv | `cli/openbb_cli/controllers/`, `cli/openbb_cli/argparse_translator/` | P1–P12 | |
| F245 | [ ] | CLI routines (`.openbb` script record/exec), session settings, results registry (`OBB0`-style stack) | `cli/openbb_cli/controllers/`, `cli/openbb_cli/session.py`, `cli/openbb_cli/models/` | F244 | |
| F246 | [ ] | Final parity audit: `make parity` clean, every `[-]` justified, docs in `stock_home/README` section | – | all | |
