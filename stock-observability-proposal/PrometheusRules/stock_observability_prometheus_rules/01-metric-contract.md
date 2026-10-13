# SO-MVP --- Metric Contract and Data Requirements

**Status:** Draft proposal; verify against actual `stock_exporter`
output.

## 1. Required metric conventions

-   Use `symbol` as the stable stock identity label.
-   Include `exchange` where the metric may cover multiple exchanges.
-   Keep labels bounded. Do not add unbounded labels such as raw news
    text, request IDs, or timestamps.
-   Use Unix timestamps in **seconds** for `*_timestamp` metrics.
-   Document units in metric names or metadata. In this proposal, prices
    use the provider's currency units, quantities use shares/units, and
    fractional returns use ratios (`0.02 = 2%`).
-   Prefer exporter-side validation for NaN, infinity, negative values,
    invalid quotes, and impossible timestamps.
-   Do not represent missing data as zero unless zero has a valid
    business meaning.

## 2. Proposed raw metrics

  --------------------------------------------------------------------------------------
  Metric                                 Type / meaning          Units / expected
                                                                 semantics
  -------------------------------------- ----------------------- -----------------------
  `nse_stock_ltp`                        Gauge                   Latest traded price;
                                                                 must be positive for a
                                                                 valid quote

  `nse_stock_volume_total`               Counter or              Cumulative traded
                                         provider-defined        volume; confirm
                                         cumulative value        reset/session semantics
                                                                 before using as a
                                                                 counter

  `nse_stock_close`                      Gauge                   Previous session close;
                                                                 positive

  `nse_stock_open`                       Gauge                   Current session opening
                                                                 price; positive once
                                                                 established

  `nse_stock_last_tick_timestamp`        Gauge                   Unix timestamp in
                                                                 seconds of last valid
                                                                 tick

  `stock_exporter_websocket_connected`   Gauge                   `1` connected, `0`
                                                                 disconnected

  `stock_exporter_scrape_success`        Gauge                   `1` collection
                                                                 succeeded, `0`
                                                                 collection failed

  `nse_stock_upper_circuit`              Gauge                   Current valid upper
                                                                 circuit limit, if
                                                                 provided

  `nse_stock_lower_circuit`              Gauge                   Current valid lower
                                                                 circuit limit, if
                                                                 provided

  `nse_stock_bid_price`                  Gauge                   Best bid price

  `nse_stock_ask_price`                  Gauge                   Best ask price

  `nse_stock_bid_quantity`               Gauge                   Best bid displayed
                                                                 quantity

  `nse_stock_ask_quantity`               Gauge                   Best ask displayed
                                                                 quantity

  `stock_expected_symbol_info`           Gauge/info-style        One series per expected
                                         inventory metric        instrument, used to
                                                                 detect missing
                                                                 per-symbol data
  --------------------------------------------------------------------------------------

## 3. Proposed derived metrics

  ------------------------------------------------------------------------------------
  Metric                              Required methodology
  ----------------------------------- ------------------------------------------------
  `stock_return_5m`                   Fractional return over a defined five-minute
                                      window; document sampling, missing-sample and
                                      session-boundary handling

  `stock_relative_volume`             Current volume over a defined interval divided
                                      by a comparable baseline, preferably matched by
                                      time-of-day/session; not raw cumulative volume

  `stock_vwap`                        Session VWAP; document reset time, input data
                                      and treatment of missing trades

  `stock_realized_volatility_5m`      Defined measure of realized volatility;
                                      explicitly state whether it is standard
                                      deviation of returns, annualized, or another
                                      measure

  `stock_intraday_drawdown`           `(current_price / running_intraday_high) - 1`;
                                      running high resets at valid session start

  `stock_market_session_active`       `1` only when the configured instrument/exchange
                                      session is expected to be active; use an
                                      authoritative source or maintained calendar

  `stock_expected_symbol_info`        Inventory of symbols expected to be monitored;
                                      used to compare expected symbols with observed
                                      metrics
  ------------------------------------------------------------------------------------

Derived metrics should normally be implemented as recording rules or in
the exporter, with unit tests and documented formulas.

## 4. Example recording rule patterns

These are patterns only; confirm metric names, sampling and business
semantics before use.

``` yaml
groups:
  - name: stock-derived-metrics
    interval: 30s
    rules:
      # Example only. This assumes the exporter already exposes a valid
      # fractional five-minute return metric, which would make this rule
      # unnecessary. Implement the actual return calculation at the chosen
      # data-processing layer and test it independently.
      - record: stock_return_5m:validated
        expr: stock_return_5m
```

Do not use a simple `rate()` on a gauge such as last traded price. Do
not treat cumulative volume as a Prometheus counter unless its reset and
monotonicity semantics actually satisfy counter behaviour.

## 5. Data-quality and availability contract

1.  Every monitored symbol must be represented in the
    expected-instrument inventory.
2.  Exporter liveness and data-collection success must be distinct
    signals.
3.  A connected WebSocket does not prove that every instrument is
    receiving ticks.
4.  A missing series is not equivalent to a zero value.
5.  Market-closed periods and legitimately inactive instruments must be
    accounted for in stale-tick alerts.
6.  When the source data is stale or invalid, market-condition alerts
    should be gated or annotated to avoid misleading users.
7.  For alerts involving multiple metrics, labels must match on the
    intended identity keys (normally `exchange` and `symbol`).
8.  Every metric must have documented units, reset semantics, freshness
    expectations, and provider limitations.

## 6. Acceptance criteria

-   [ ] Actual exporter output is captured and compared to this
    proposal.
-   [ ] All metric names and label keys are confirmed.
-   [ ] Units and timestamp conventions are documented.
-   [ ] Derived metric algorithms are reviewed and unit-tested.
-   [ ] Missing, stale, invalid, and partial symbol coverage are tested.
-   [ ] Session/calendar assumptions are documented.
