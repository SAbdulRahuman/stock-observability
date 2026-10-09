import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WATCHLIST_KEY } from "@/lib/watchlist";
import { HomePage } from "./home";

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const q = (symbol: string, price: number | null, change: number | null, changePercent: number | null, extra = {}) => ({
  exchange: "NSE",
  symbol,
  price,
  change,
  changePercent,
  exploreUrl: "/grafana/explore",
  ...extra,
});
const spark = [
  { t: 1, v: 1 },
  { t: 2, v: 2 },
];

const populated: Record<string, unknown> = {
  "/api/v1/status": env({
    market: { phase: "open", open: true, tradingDay: true, timezone: "Asia/Kolkata", now: "2026-10-08T04:12:00Z", feedSource: "kite" },
    banners: [],
  }),
  "/api/v1/info": env({ version: "0.1.0", exchange: "NSE", sessionOpen: "09:15", sessionClose: "15:30", timezone: "Asia/Kolkata" }),
  "/api/v1/links": env([
    { id: "grafana", label: "Grafana", href: "/grafana/" },
    { id: "prometheus", label: "Prometheus", href: "/prometheus/" },
    { id: "alertmanager", label: "Alertmanager", href: "/alertmanager/" },
    { id: "openbb", label: "OpenBB", href: "/openbb" },
  ]),
  "/api/v1/market": env({
    indices: [q("NIFTY 50", 24892.3, 156.7, 0.63, { spark }), q("INDIA VIX", 12.54, -0.68, -5.14)],
    sectors: [],
  }),
  "/api/v1/chart": env({
    symbol: "NIFTY 50",
    range: "1D",
    points: [
      { t: 1759895100000, v: 24800 },
      { t: 1759895220000, v: 24892.3 },
    ],
    volume: [{ t: 1759895100000, v: 1200 }],
    volumeLabel: "Core equities traded volume",
    last: 24892.3,
  }),
  "/api/v1/movers": env({
    gainers: [q("ADANIENT", 3142.6, 126.4, 4.2)],
    losers: [q("TCS", 3412.2, -12.35, -0.36)],
    highVolume: [q("SBIN", 621.35, 18.75, 3.11, { volume: 5_000_000 })],
  }),
  "/api/v1/alerts": env({
    bySeverity: { critical: 1, warning: 1, info: 0 },
    byCategory: {},
    silenced: 0,
    inhibited: 0,
    activeSilences: 0,
    items: [
      { name: "PriceAboveR1", severity: "warning", category: "price", summary: "Price > Resistance (R1)", symbol: "RELIANCE", startsAt: "2026-10-08T04:11:00Z", fingerprint: "a", value: "2,842.50" },
      { name: "VolumeSpike", severity: "critical", category: "volume", summary: "", symbol: "INFY", startsAt: "2026-10-08T04:08:00Z", fingerprint: "b" },
    ],
    truncated: false,
  }),
  "/api/v1/timeline": env({
    from: "2026-10-07T04:12:00Z",
    items: [{ name: "HighVolumeBreakout", severity: "info", category: "volume", symbol: "ITC", start: "2026-10-08T03:55:00Z", end: "2026-10-08T04:00:00Z" }],
    bySeverity: {},
    truncated: false,
  }),
  "/api/v1/quotes": env({ items: [q("RELIANCE", 2842.5, 36.8, 1.31, { spark }), { ...q("NOPE", null, null, null), exchange: "", exploreUrl: "" }] }),
};

const empty: Record<string, unknown> = {
  ...populated,
  "/api/v1/market": env({ indices: [], sectors: [] }),
  "/api/v1/chart": env({ symbol: "NIFTY 50", range: "1D", points: [], volume: [], volumeLabel: "Core equities traded volume", last: null }),
  "/api/v1/movers": env({ gainers: [], losers: [], highVolume: [] }),
  "/api/v1/alerts": env({ bySeverity: { critical: 0 }, byCategory: {}, silenced: 0, inhibited: 0, activeSilences: 0, items: [], truncated: false }),
  "/api/v1/timeline": env({ from: "2026-10-07T04:12:00Z", items: [], bySeverity: {}, truncated: false }),
};

function renderHome(routes: Record<string, unknown> | null) {
  const fetchMock = vi.fn(async (url: string) => {
    const body = routes?.[url.split("?")[0]];
    return body
      ? new Response(JSON.stringify(body), { status: 200 })
      : new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <HomePage />
    </QueryClientProvider>,
  );
  return fetchMock;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe("HomePage", () => {
  it("renders index cards and the NIFTY chart", async () => {
    const fetchMock = renderHome(populated);

    const nifty = await screen.findByLabelText("NIFTY 50");
    expect(await within(nifty).findByText("24,892.30")).toBeInTheDocument();
    expect(within(nifty).getByText("+156.70 (+0.63%)")).toBeInTheDocument();
    expect(within(screen.getByLabelText("INDIA VIX")).getByText("-0.68 (-5.14%)")).toBeInTheDocument();
    expect(within(screen.getByLabelText("SENSEX")).getByText("— (—)")).toBeInTheDocument();
    expect(await screen.findByText("OPEN")).toBeInTheDocument();

    expect(await screen.findByLabelText(/NIFTY 50 1D: last 24,892.30, up/)).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/chart?symbol=NIFTY%2050&range=1D", expect.anything());
    fireEvent.click(screen.getByText("1W"));
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/chart?symbol=NIFTY%2050&range=1W", expect.anything());
  });

  it("switches Top Movers lists", async () => {
    renderHome(populated);
    expect(await screen.findByText("ADANIENT")).toBeInTheDocument();
    expect(screen.getByText("+126.40")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Losers"));
    expect(screen.getByText("-0.36%")).toBeInTheDocument();
    fireEvent.click(screen.getByText("High Volume"));
    expect(screen.getByText("SBIN").closest("tr")).toHaveAttribute("title", "Volume 50,00,000");
  });

  it("renders alerts, watchlist and quick links", async () => {
    localStorage.setItem(WATCHLIST_KEY, JSON.stringify(["RELIANCE", "NOPE", "bad symbol!"]));
    const fetchMock = renderHome(populated);
    expect(await screen.findByLabelText("2 active alerts")).toHaveTextContent("2");
    const alerts = screen.getByLabelText("Active Alerts");
    expect(within(alerts).getByText("Price > Resistance (R1)")).toBeInTheDocument();
    expect(within(alerts).getByText("2,842.50")).toBeInTheDocument();
    expect(within(alerts).getByText("VolumeSpike")).toBeInTheDocument();
    expect(within(alerts).getAllByText("Firing")).toHaveLength(2);
    expect(await within(alerts).findByText("Resolved")).toBeInTheDocument();
    expect(within(alerts).getByLabelText("View all alerts")).toHaveAttribute("href", "/alerts");

    const watchlist = screen.getByLabelText("Watchlist");
    expect(await within(watchlist).findByText("+36.80")).toBeInTheDocument();
    expect(within(watchlist).getByText("NOPE").closest("tr")).toHaveTextContent("NOPE———");
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/quotes?symbols=RELIANCE%2CNOPE", expect.anything());

    const grafana = await screen.findByTitle("Grafana Dashboards (opens in a new tab)");
    expect(grafana).toHaveAttribute("href", "/grafana/");
    expect(screen.getByText("Fundamental & Technical Analysis")).toBeInTheDocument();
  });

  it("shows empty states", async () => {
    renderHome(empty);
    expect(await screen.findByText("No NIFTY 50 price data for this period")).toBeInTheDocument();
    expect(await screen.findByText("No gainers right now")).toBeInTheDocument();
    expect(await screen.findByText("No active alerts")).toBeInTheDocument();
    expect(await screen.findByText("Your watchlist is empty.", { exact: false })).toBeInTheDocument();
    expect(within(await screen.findByLabelText("NIFTY 50")).getByText("—")).toBeInTheDocument();
  });

  it("shows unavailable states when the API fails", async () => {
    localStorage.setItem(WATCHLIST_KEY, JSON.stringify(["RELIANCE"]));
    renderHome(null);
    expect(await within(screen.getByLabelText("NIFTY 50")).findByText("Unavailable")).toBeInTheDocument();
    expect(await screen.findAllByText(/Unavailable — upstream unavailable/)).toHaveLength(4);
    expect(screen.getAllByText("Not configured")).toHaveLength(4);
    expect(screen.getByText("Status unavailable")).toBeInTheDocument();
  });
});
