import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { rebase } from "./comparison";
import { SectorViewPage } from "./sector-view-page";

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const spark = [
  { t: 1, v: 1 },
  { t: 2, v: 2 },
];
const ix = (symbol: string, price: number, change: number, changePercent: number) => ({ exchange: "NSE", symbol, price, change, changePercent, exploreUrl: "", spark });
const indices = [
  ix("NIFTY BANK", 54321.1, 412.8, 0.77),
  ix("NIFTY IT", 37124.2, -120.35, -0.32),
  ix("NIFTY AUTO", 18432.6, 256.7, 1.41),
  ix("NIFTY FMCG", 56231.9, 298.45, 0.53),
  ix("NIFTY PHARMA", 15862.4, -82.15, -0.52),
  ix("NIFTY METAL", 8921.3, 143.9, 1.64),
];
const stock = (symbol: string, changePercent: number) => ({ exchange: "NSE", symbol, price: 100, changePercent, exploreUrl: "" });
const sectors = [
  { name: "Metals", changePercent: 1.76, count: 10, advancers: 8, decliners: 2, gainer: stock("TATASTEEL", 2.4), loser: stock("NMDC", -0.2) },
  { name: "Banks", changePercent: 1.24, count: 12, advancers: 9, decliners: 3, gainer: stock("SBIN", 1.8), loser: null },
  { name: "IT", changePercent: -0.62, count: 10, advancers: 2, decliners: 8, gainer: null, loser: stock("WIPRO", -1.2) },
];
const scan = (s: unknown[]) => ({
  universe: "core",
  sector: "",
  price: "all",
  total: 32,
  advancers: 19,
  decliners: 13,
  unchanged: 0,
  gainers: [],
  losers: [],
  mostActive: [],
  volumeChange: [],
  high52: [],
  low52: [],
  distribution: [],
  sectors: s,
  sectorNames: ["Banks", "IT", "Metals"],
});

function renderPage({ ok = true, sectorList = sectors, indexList = indices } = {}) {
  const fetchMock = vi.fn(async (url: string) => {
    const u = new URL(url, "http://localhost");
    if (!ok) return new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
    switch (u.pathname) {
      case "/api/v1/info":
        return new Response(JSON.stringify(env({ version: "0.1.0", exchange: "NSE", sessionOpen: "09:15", sessionClose: "15:30", timezone: "Asia/Kolkata" })));
      case "/api/v1/sectors":
        return new Response(JSON.stringify(env({ indices: indexList })));
      case "/api/v1/movers/scan":
        return new Response(JSON.stringify(env(scan(sectorList))));
      case "/api/v1/chart": {
        const symbol = u.searchParams.get("symbol");
        const base = symbol === "NIFTY IT" ? 200 : 100;
        return new Response(
          JSON.stringify(env({ symbol, range: "1D", points: [{ t: 1, v: base }, { t: 2, v: base * 1.02 }], volume: [], volumeLabel: "", last: base * 1.02 })),
        );
      }
      default:
        return new Response("{}", { status: 404 });
    }
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <SectorViewPage />
    </QueryClientProvider>,
  );
  return fetchMock;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("SectorViewPage", () => {
  it("renders index cards, heatmap, performance, comparison and leaders", async () => {
    const fetchMock = renderPage();
    const bank = await screen.findByLabelText("NIFTY BANK");
    expect(within(bank).getByText("54,321.10")).toBeInTheDocument();
    expect(within(screen.getByLabelText("NIFTY IT")).getByText("-120.35 (-0.32%)")).toBeInTheDocument();
    expect(screen.queryByLabelText("NIFTY METAL")).toBeNull();

    expect(await screen.findByTitle("Metals: +1.76% · 10 instruments · 8 up / 2 down")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Tile size"), { target: { value: "equal" } });
    expect(screen.getByTitle("Banks: +1.24% · 12 instruments · 9 up / 3 down")).toHaveStyle({ flexGrow: "1" });

    const comparison = screen.getByLabelText("Sector Comparison");
    expect(within(comparison).getByText("BANK")).toHaveAttribute("aria-pressed", "true");
    expect(within(comparison).getByText("PHARMA")).toHaveAttribute("aria-pressed", "false");
    expect(await within(comparison).findByLabelText(/Rebased to 100: NIFTY BANK 102/)).toBeInTheDocument();
    fireEvent.click(within(comparison).getByText("1W"));
    await waitFor(() => expect(fetchMock.mock.calls.map(([u]) => u)).toContain("/api/v1/chart?symbol=NIFTY%20BANK&range=1W"));

    const gainers = screen.getByLabelText("Top Sector Gainers");
    expect(within(gainers).getByText("TATASTEEL")).toBeInTheDocument();
    expect(within(gainers).getByLabelText("View sector Metals")).toHaveAttribute("href", "/movers?universe=core&sector=Metals");
    const losers = screen.getByLabelText("Top Sector Losers");
    expect(within(losers).getByText("WIPRO")).toBeInTheDocument();
    expect(within(losers).queryByText("Banks")).toBeNull();
  });

  it("sends the sector and universe filters", async () => {
    const fetchMock = renderPage();
    await screen.findByTitle("Metals: +1.76% · 10 instruments · 8 up / 2 down");
    fireEvent.change(screen.getByLabelText("Universe"), { target: { value: "all" } });
    fireEvent.change(screen.getByLabelText("Sector"), { target: { value: "IT" } });
    await waitFor(() => expect(fetchMock.mock.calls.map(([u]) => u)).toContain("/api/v1/movers/scan?universe=all&price=all&sector=IT"));
    expect(screen.getByLabelText("Time range")).toBeDisabled();
  });

  it("shows empty states", async () => {
    renderPage({ sectorList: [], indexList: [] });
    expect(await screen.findByText("No sector index data recorded")).toBeInTheDocument();
    expect(await screen.findAllByText("No sector data for these filters")).toHaveLength(2);
    expect(screen.getByText("No sector is up")).toBeInTheDocument();
    expect(screen.getByText("No sector indices configured")).toBeInTheDocument();
  });

  it("shows unavailable states when the API fails", async () => {
    renderPage({ ok: false });
    expect(await screen.findAllByText(/Unavailable — upstream unavailable/)).toHaveLength(4);
    expect(screen.getAllByText("Unavailable")).toHaveLength(5);
  });
});

describe("rebase", () => {
  it("rebases each series to 100 at its first point", () => {
    const rows = rebase([
      { symbol: "A", points: [{ t: 1, v: 50 }, { t: 2, v: 55 }] },
      { symbol: "B", points: [{ t: 1, v: 200 }, { t: 2, v: 190 }] },
    ]);
    expect(rows).toEqual([
      { t: 1, A: 100, B: 100 },
      { t: 2, A: 110.00000000000001, B: 95 },
    ]);
  });
});
