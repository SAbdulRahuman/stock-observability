#!/usr/bin/env python3
"""Quant Lab (proposal §A9, Release 5): models that do not belong in PromQL, published as metrics.

Reads inputs from the Prometheus HTTP API, computes and serves /metrics for the `stock_quantlab`
scrape job (stdlib only):
  options   Black-Scholes implied volatility and Greeks of every CE/PE in stock_instrument_info,
            max pain per underlying and expiry                                  (Release 4 hand-off)
  forecasts AR(1) on 60 one-minute log returns with an EWMA (lambda 0.94) volatility: expected
            15m return, 15m volatility and P(up) per core equity, model="ar1"   (§8.3/§8.4 schema)
Outputs are observability signals, not investment advice.

    python3 quantlab.py --prometheus http://localhost:9090 --listen :9108 --interval 60
"""

import argparse
import http.server
import json
import math
import threading
import time
import urllib.parse
import urllib.request

VERSION = "0.1.0"
HORIZON_MIN = 15


# --- Black-Scholes ------------------------------------------------------------------------------


def norm_cdf(x):
    return 0.5 * (1 + math.erf(x / math.sqrt(2)))


def norm_pdf(x):
    return math.exp(-x * x / 2) / math.sqrt(2 * math.pi)


def _d1_d2(s, k, t, r, sigma):
    d1 = (math.log(s / k) + (r + sigma * sigma / 2) * t) / (sigma * math.sqrt(t))
    return d1, d1 - sigma * math.sqrt(t)


def bs_price(kind, s, k, t, r, sigma):
    """European option price; kind is "CE" (call) or "PE" (put); t in years, r and sigma annual."""
    d1, d2 = _d1_d2(s, k, t, r, sigma)
    if kind == "CE":
        return s * norm_cdf(d1) - k * math.exp(-r * t) * norm_cdf(d2)
    return k * math.exp(-r * t) * norm_cdf(-d2) - s * norm_cdf(-d1)


def implied_vol(kind, price, s, k, t, r, lo=1e-4, hi=5.0, tol=1e-8):
    """Bisection on sigma; None when the price is outside the no-arbitrage bounds."""
    if t <= 0 or price <= 0 or s <= 0 or k <= 0:
        return None
    if not bs_price(kind, s, k, t, r, lo) <= price <= bs_price(kind, s, k, t, r, hi):
        return None
    for _ in range(200):
        mid = (lo + hi) / 2
        if bs_price(kind, s, k, t, r, mid) < price:
            lo = mid
        else:
            hi = mid
        if hi - lo < tol:
            break
    return (lo + hi) / 2


def greeks(kind, s, k, t, r, sigma):
    """delta, gamma, theta (per calendar day), vega and rho (per 1 percentage point)."""
    d1, d2 = _d1_d2(s, k, t, r, sigma)
    disc = math.exp(-r * t)
    gamma = norm_pdf(d1) / (s * sigma * math.sqrt(t))
    vega = s * norm_pdf(d1) * math.sqrt(t) / 100
    decay = -s * norm_pdf(d1) * sigma / (2 * math.sqrt(t))
    if kind == "CE":
        delta, theta, rho = norm_cdf(d1), decay - r * k * disc * norm_cdf(d2), k * t * disc * norm_cdf(d2)
    else:
        delta, theta, rho = norm_cdf(d1) - 1, decay + r * k * disc * norm_cdf(-d2), -k * t * disc * norm_cdf(-d2)
    return {"delta": delta, "gamma": gamma, "theta": theta / 365, "vega": vega, "rho": rho / 100}


def max_pain(chain):
    """chain: [(strike, call_oi, put_oi)]. Strike minimising the total payout to option holders."""
    strikes = sorted({c[0] for c in chain})
    if not strikes:
        return None

    def payout(settle):
        return sum(c * max(settle - k, 0) + p * max(k - settle, 0) for k, c, p in chain)

    return min(strikes, key=lambda s: (payout(s), s))


# --- forecasts ----------------------------------------------------------------------------------


def ar1(returns):
    """AR(1) by OLS on consecutive returns: (mu, phi), phi clamped to (-0.99, 0.99)."""
    if len(returns) < 30:
        return None
    x, y = returns[:-1], returns[1:]
    mx, my = sum(x) / len(x), sum(y) / len(y)
    var = sum((a - mx) ** 2 for a in x)
    phi = 0.0 if var == 0 else sum((a - mx) * (b - my) for a, b in zip(x, y)) / var
    phi = max(-0.99, min(0.99, phi))
    mu = sum(returns) / len(returns)
    return mu, phi


def ewma_variance(returns, lam=0.94):
    h = sum(r * r for r in returns[:30]) / min(len(returns), 30)
    for r in returns[30:]:
        h = lam * h + (1 - lam) * r * r
    return h


def forecast(returns, horizon=HORIZON_MIN):
    """Expected log return over `horizon` minutes (AR(1) closed form), its sigma and P(up)."""
    fit = ar1(returns)
    if fit is None:
        return None
    mu, phi = fit
    dev = returns[-1] - mu
    geo = phi * (1 - phi ** horizon) / (1 - phi) if phi != 1 else horizon
    expected = horizon * mu + dev * geo
    sigma = math.sqrt(horizon * ewma_variance(returns))
    prob_up = norm_cdf(expected / sigma) if sigma > 0 else 0.5
    return {"expected_return_percent": (math.exp(expected) - 1) * 100, "volatility_percent": sigma * 100, "prob_up": prob_up}


# --- Prometheus I/O -----------------------------------------------------------------------------


def http_query(base):
    def query(expr):
        url = base.rstrip("/") + "/api/v1/query?" + urllib.parse.urlencode({"query": expr})
        with urllib.request.urlopen(url, timeout=30) as resp:
            data = json.load(resp)["data"]
        if data["resultType"] == "matrix":
            return [(r["metric"], [float(v) for _, v in r["values"]]) for r in data["result"]]
        return [(r["metric"], float(r["value"][1])) for r in data["result"]]
    return query


def _labels(d):
    return "{" + ",".join('%s="%s"' % (k, str(v).replace("\\", "\\\\").replace('"', '\\"')) for k, v in sorted(d.items())) + "}"


class Exposition:
    def __init__(self):
        self.families = {}

    def add(self, name, help_, labels, value):
        if value is None or math.isnan(value) or math.isinf(value):
            return
        self.families.setdefault(name, (help_, []))[1].append((labels, value))

    def text(self):
        out = []
        for name in sorted(self.families):
            help_, samples = self.families[name]
            out += ["# HELP %s %s" % (name, help_), "# TYPE %s gauge" % name]
            out += ["%s%s %r" % (name, _labels(l), v) for l, v in samples]
        return "\n".join(out) + "\n"


def options(query, now, exp):
    rate = next((v for _, v in query("max(stock_risk_free_rate_ratio)")), 0.065)
    spot = {m["underlying"]: v for m, v in query("underlying:spot:price")}
    meta = 'on(exchange, symbol) group_left(underlying, expiry, strike) stock_instrument_info{instrument_type=~"CE|PE"}'
    expiry = {m["symbol"]: v for m, v in query("stock_instrument_expiry_timestamp_seconds * " + meta)}
    oi = {m["symbol"]: v for m, v in query("stock_open_interest * " + meta)}
    chains = {}
    for m, price in query("stock_last_price * " + meta):
        sym, und, kind = m["symbol"], m["underlying"], m["instrument_type"]
        k, s = float(m.get("strike") or 0), spot.get(und)
        chain = chains.setdefault((und, m["expiry"]), {})
        row = chain.setdefault(k, [0.0, 0.0])
        row[0 if kind == "CE" else 1] += oi.get(sym, 0.0)
        t = (expiry.get(sym, 0) - now) / (365 * 86400)
        if not s or k <= 0 or t <= 0:
            continue
        iv = implied_vol(kind, price, s, k, t, rate)
        if iv is None:
            continue
        lbl = {key: m[key] for key in ("exchange", "symbol", "instrument_type", "underlying", "expiry", "strike")}
        exp.add("stock_option_implied_volatility_ratio", "Black-Scholes implied volatility (annual) from the last price and the spot.", lbl, iv)
        for g, v in greeks(kind, s, k, t, rate, iv).items():
            exp.add("stock_option_" + g, "Black-Scholes %s at the implied volatility (theta per day, vega and rho per 1 pp)." % g, lbl, v)
    for (und, expiry_date), chain in chains.items():
        mp = max_pain([(k, c, p) for k, (c, p) in chain.items()])
        exp.add("underlying_max_pain_strike_price", "Strike minimising the total payout to option holders at expiry.",
                {"underlying": und, "expiry": expiry_date}, mp)


def forecasts(query, exp):
    for m, returns in query('stock:log_return_1m:ratio{tier="core", instrument_type="EQ"}[60m]'):
        f = forecast(returns)
        if f is None:
            continue
        lbl = {"exchange": m["exchange"], "symbol": m["symbol"], "model": "ar1", "horizon": "%dm" % HORIZON_MIN}
        exp.add("stock_prediction_prob_up", "P(price up over the horizon).", lbl, f["prob_up"])
        exp.add("stock_prediction_expected_return_percent", "Expected return over the horizon, percent.", lbl, f["expected_return_percent"])
        exp.add("stock_prediction_volatility_percent", "Expected standard deviation of the return over the horizon, percent.", lbl, f["volatility_percent"])


def run_once(query, now):
    exp = Exposition()
    for model, fn in (("options", lambda: options(query, now, exp)), ("ar1", lambda: forecasts(query, exp))):
        start = time.monotonic()
        try:
            fn()
            exp.add("stock_prediction_last_run_timestamp_seconds", "Unix time of the last successful run.", {"model": model}, now)
        except Exception as err:  # keep serving the other model and the staleness signal
            print("quantlab: %s failed: %s" % (model, err), flush=True)
        exp.add("stock_prediction_inference_duration_seconds", "Duration of the last run.", {"model": model}, time.monotonic() - start)
        exp.add("stock_prediction_info", "Model metadata (value 1).", {"model": model, "version": VERSION, "trained_until": "online", "features_hash": "none"}, 1)
    return exp.text()


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--prometheus", default="http://prometheus:9090")
    ap.add_argument("--listen", default=":9108")
    ap.add_argument("--interval", type=float, default=60)
    args = ap.parse_args()
    state = {"text": "# no run yet\n"}
    query = http_query(args.prometheus)

    def loop():
        while True:
            state["text"] = run_once(query, time.time())
            time.sleep(args.interval)

    threading.Thread(target=loop, daemon=True).start()

    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path not in ("/metrics", "/-/healthy"):
                self.send_error(404)
                return
            body = (state["text"] if self.path == "/metrics" else "ok\n").encode()
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; version=0.0.4")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *_):
            pass

    host, port = args.listen.rsplit(":", 1)
    http.server.ThreadingHTTPServer((host or "0.0.0.0", int(port)), Handler).serve_forever()


if __name__ == "__main__":
    main()
