import math
import random
import unittest

import quantlab as q


class BlackScholesTest(unittest.TestCase):
    def test_put_call_parity(self):
        s, k, t, r, v = 25000, 25100, 30 / 365, 0.065, 0.14
        c, p = q.bs_price("CE", s, k, t, r, v), q.bs_price("PE", s, k, t, r, v)
        self.assertAlmostEqual(c - p, s - k * math.exp(-r * t), places=6)

    def test_known_value(self):
        # Hull, Options Futures and Other Derivatives: S=42, K=40, r=10%, sigma=20%, T=0.5 -> c=4.76, p=0.81
        self.assertAlmostEqual(q.bs_price("CE", 42, 40, 0.5, 0.1, 0.2), 4.76, places=2)
        self.assertAlmostEqual(q.bs_price("PE", 42, 40, 0.5, 0.1, 0.2), 0.81, places=2)

    def test_implied_vol_round_trip(self):
        for kind in ("CE", "PE"):
            for k in (24000, 25000, 26000):
                price = q.bs_price(kind, 25000, k, 20 / 365, 0.065, 0.17)
                self.assertAlmostEqual(q.implied_vol(kind, price, 25000, k, 20 / 365, 0.065), 0.17, places=6)

    def test_implied_vol_rejects_arbitrage(self):
        self.assertIsNone(q.implied_vol("CE", 0.01, 25000, 24000, 20 / 365, 0.065))  # below intrinsic
        self.assertIsNone(q.implied_vol("CE", 100, 25000, 25000, 0, 0.065))

    def test_greeks_match_finite_differences(self):
        s, k, t, r, v, h = 25000.0, 25200.0, 25 / 365, 0.065, 0.16, 1e-3
        for kind in ("CE", "PE"):
            g = q.greeks(kind, s, k, t, r, v)
            price = lambda **kw: q.bs_price(kind, kw.get("s", s), k, kw.get("t", t), kw.get("r", r), kw.get("v", v))
            self.assertAlmostEqual(g["delta"], (price(s=s + h) - price(s=s - h)) / (2 * h), places=5)
            self.assertAlmostEqual(g["gamma"], (price(s=s + 1) - 2 * price() + price(s=s - 1)), places=6)
            self.assertAlmostEqual(g["vega"], (price(v=v + h) - price(v=v - h)) / (2 * h) / 100, places=4)
            self.assertAlmostEqual(g["rho"], (price(r=r + h) - price(r=r - h)) / (2 * h) / 100, places=4)
            self.assertAlmostEqual(g["theta"], -(price(t=t + 1e-6) - price(t=t - 1e-6)) / 2e-6 / 365, places=3)


class MaxPainTest(unittest.TestCase):
    def test_toy_chain(self):
        # Heavy call OI above 100 and put OI below 100 pin the settlement at 100.
        chain = [(90, 0, 500), (95, 10, 300), (100, 50, 50), (105, 300, 10), (110, 500, 0)]
        self.assertEqual(q.max_pain(chain), 100)
        self.assertIsNone(q.max_pain([]))


class ForecastTest(unittest.TestCase):
    def test_ar1_recovers_phi(self):
        rnd, r, xs = random.Random(1), 0.0, []
        for _ in range(5000):
            r = 0.0001 + 0.3 * (r - 0.0001) + rnd.gauss(0, 0.001)
            xs.append(r)
        mu, phi = q.ar1(xs)
        self.assertAlmostEqual(phi, 0.3, delta=0.05)
        self.assertAlmostEqual(mu, 0.0001, delta=0.0001)
        self.assertIsNone(q.ar1(xs[:10]))

    def test_forecast_bounds(self):
        rnd = random.Random(2)
        f = q.forecast([rnd.gauss(0, 0.001) for _ in range(60)])
        self.assertTrue(0 < f["prob_up"] < 1)
        self.assertGreater(f["volatility_percent"], 0)
        # EWMA with lambda 0.94 averages ~30 effective samples: within 50% of the true 15m sigma.
        self.assertTrue(0.5 < f["volatility_percent"] / (0.001 * math.sqrt(15) * 100) < 1.5)


class RunOnceTest(unittest.TestCase):
    def test_exposition(self):
        now = 1_790_000_000.0
        k, s, t = 25000, 25000, 30 * 86400
        call = q.bs_price("CE", s, k, t / 365 / 86400, 0.065, 0.15)
        meta = {"exchange": "NFO", "instrument_type": "CE", "underlying": "NIFTY", "expiry": "2026-10-27", "strike": "25000"}
        sym = dict(meta, symbol="NIFTY26OCT25000CE")
        put = dict(meta, symbol="NIFTY26OCT25000PE", instrument_type="PE")
        answers = {
            "max(stock_risk_free_rate_ratio)": [({}, 0.065)],
            "underlying:spot:price": [({"underlying": "NIFTY"}, s)],
            "stock_instrument_expiry_timestamp_seconds": [(sym, now + t), (put, now + t)],
            "stock_open_interest": [(sym, 1000.0), (put, 2000.0)],
            "stock_last_price": [(sym, call), (put, 0.0)],
            "stock:log_return_1m:ratio": [({"exchange": "NSE", "symbol": "INFY"}, [0.001 * (-1) ** i for i in range(60)])],
        }
        query = lambda expr: next(v for key, v in answers.items() if expr.startswith(key))
        text = q.run_once(query, now)
        iv = [l for l in text.splitlines() if l.startswith("stock_option_implied_volatility_ratio{")]
        self.assertEqual(len(iv), 1)  # the put has no price
        self.assertAlmostEqual(float(iv[0].split()[-1]), 0.15, places=6)
        self.assertIn('underlying_max_pain_strike_price{expiry="2026-10-27",underlying="NIFTY"} 25000', text)
        self.assertIn('stock_prediction_prob_up{exchange="NSE",horizon="15m",model="ar1",symbol="INFY"}', text)
        self.assertIn('stock_prediction_last_run_timestamp_seconds{model="options"} 1790000000.0', text)


if __name__ == "__main__":
    unittest.main()
