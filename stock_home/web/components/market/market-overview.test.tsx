import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { formatVolume } from "@/lib/format";
import { MarketOverviewPage } from "./market-overview";

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
  "/api/v1/links": env([]),
  "/api/v1/market": env({
    indices: [q("NIFTY 50", 24892.3, 156.7, 0.63, { spark }), q("NIFTY NEXT 50", 67124.2, 320.15, 0.48)],
    advancers: 1428,
    decliners: 876,
    unchanged: 57,
    sectors: [
      { name: "PSU Bank", changePercent: 2.86, constituents: 12 },
      { name: "IT", changePercent: -0.64, constituents: 10 },
    ],
  }),
  "/api/v1/chart": env({
    symbol: "NIFTY 50",
    range: "1D",
    points: [
      { t: 1759895100000, v: 24800 },
      { t: 1759895220000, v: 24892.3 },
    ],
    volume: [],
    volumeLabel: "Core equities traded volume",
    last: 24892.3,
  }),
  "/api/v1/movers": env({
    gainers: [q("ADANIENT", 3142.6, 126.4, 4.2, { spark })],
    losers: [q("HCLTECH", 1548.2, -32.8, -2.08)],
    highVolume: [q("RELIANCE", 2842.5, 36.8, 1.31, { volume: 12_400_000 })],
  }),
  "/api/v2/news/company": {
    results: [
      { date: "2026-10-08T03:45:00Z", title: "Older headline", url: "javascript:alert(1)" },
      { date: "2026-10-08T04:08:00Z", title: "Nifty trades higher", url: "https://example.com/a" },
    ],
    provider: "yfinance",
  },
};

const empty: Record<string, unknown> = {
  ...populated,
  "/api/v1/market": env({ indices: [], advancers: null, decliners: null, unchanged: null, sectors: [] }),
  "/api/v1/movers": env({ gainers: [], losers: [], highVolume: [] }),
  "/api/v2/news/company": { results: [], provider: "yfinance" },
};

function renderPage(routes: Record<string, unknown> | null) {
  const fetchMock = vi.fn(async (url: string) => {
    const path = url.split("?")[0];
    const body = routes?.[path];
    if (body) return new Response(JSON.stringify(body), { status: 200 });
    if (path.startsWith("/api/v2/")) return new Response(JSON.stringify({ error: "unknown command" }), { status: 404 });
    return new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MarketOverviewPage />
    </QueryClientProvider>,
  );
  return fetchMock;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("MarketOverviewPage", () => {
  it("renders indices, breadth, sectors and movers", async () => {
    renderPage(populated);
    expect(await within(screen.getByLabelText("NIFTY NEXT 50")).findByText("67,124.20")).toBeInTheDocument();
    expect(within(screen.getByLabelText("BANK NIFTY")).getByText("— (—)")).toBeInTheDocument();

    expect(await screen.findByText("2,361")).toBeInTheDocument();
    expect(screen.getByText("1,428 (60.5%)")).toBeInTheDocument();
    expect(screen.getByText("57 (2.4%)")).toBeInTheDocument();
    expect(await screen.findByText("Market Breadth (NSE)")).toBeInTheDocument();

    expect(await screen.findByText("ADANIENT")).toBeInTheDocument();
    expect(screen.getByText("+4.20%")).toBeInTheDocument();
    expect(screen.getByText("-32.80")).toBeInTheDocument();
    expect(screen.getByText(formatVolume(12_400_000))).toBeInTheDocument();

    const perf = screen.getByLabelText("Sector Performance (NSE)");
    expect(within(perf).getByText("+2.86%")).toBeInTheDocument();
    expect(within(perf).getByText("-0.64%")).toBeInTheDocument();
    expect(screen.getByTitle("PSU Bank: +2.86% · 12 constituents")).toBeInTheDocument();
  });

  it("syncs the page timeframe with the chart, refreshes, and lists news safely", async () => {
    const fetchMock = renderPage(populated);
    expect(await screen.findByLabelText(/NIFTY 50 1D: last 24,892.30/)).toBeInTheDocument();
    const [pageWeek, chartWeek] = screen.getAllByText("1W");
    fireEvent.click(pageWeek);
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/chart?symbol=NIFTY%2050&range=1W", expect.anything());
    expect(chartWeek).toHaveAttribute("aria-pressed", "true");

    const before = fetchMock.mock.calls.length;
    fireEvent.click(screen.getByText("Refresh"));
    await waitFor(() => expect(fetchMock.mock.calls.length).toBeGreaterThan(before));

    const news = screen.getByLabelText("Market News");
    const link = await within(news).findByText("Nifty trades higher");
    expect(link).toHaveAttribute("href", "https://example.com/a");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
    expect(within(news).getByText("Older headline").tagName).toBe("SPAN");
    expect(within(news).getAllByRole("listitem")[0]).toHaveTextContent("Nifty trades higher");
    expect(fetchMock).toHaveBeenCalledWith("/api/v2/news/company?symbol=%5ENSEI&provider=yfinance&limit=6", expect.anything());
  });

  it("shows empty states", async () => {
    renderPage(empty);
    expect(await screen.findByText("Breadth data unavailable")).toBeInTheDocument();
    expect(await screen.findAllByText("No sector data yet")).toHaveLength(2);
    expect(screen.getByText("No gainers right now")).toBeInTheDocument();
    expect(screen.getByText("No traded volume yet")).toBeInTheDocument();
    expect(await screen.findByText("No market news")).toBeInTheDocument();
  });

  it("shows unavailable states when upstreams fail", async () => {
    renderPage(null);
    expect(await within(screen.getByLabelText("NIFTY 50")).findByText("Unavailable")).toBeInTheDocument();
    expect(await screen.findAllByText(/Unavailable — upstream unavailable/)).toHaveLength(7);
    expect(await screen.findByText("Market news unavailable — no OpenBB upstream configured")).toBeInTheDocument();
  });
});
