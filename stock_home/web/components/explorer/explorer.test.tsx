import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WATCHLIST_KEY } from "@/lib/watchlist";
import { ExplorerPage } from "./explorer-page";

const nav = vi.hoisted(() => {
  const state = {
    params: new URLSearchParams(),
    replace: vi.fn((url: string) => {
      state.params = new URLSearchParams(url.split("?")[1] ?? "");
    }),
  };
  return state;
});
vi.mock("next/navigation", () => ({
  useSearchParams: () => nav.params,
  useRouter: () => ({ replace: nav.replace, push: vi.fn() }),
  usePathname: () => "/explorer",
}));

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const q = (symbol: string, sector: string, price: number, change: number, changePercent: number, volume?: number) => ({
  exchange: "NSE",
  symbol,
  sector,
  price,
  change,
  changePercent,
  volume,
  exploreUrl: "/grafana/explore?x",
});
const page = (items: unknown[], total = items.length, p = 1) => env({ items, total, page: p, per: 25, sectorNames: ["Banks", "IT"] });

function renderPage({ stocks = page([q("INFY", "IT", 1487.3, 18.9, 1.29, 9e6), q("SBIN", "Banks", 621.35, 18.75, 3.11), q("TCS", "IT", 3412.2, -12.35, -0.36)], 30) as unknown, ok = true } = {}) {
  const fetchMock = vi.fn(async (url: string) => {
    const u = new URL(url, "http://localhost");
    if (!ok) return new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
    switch (u.pathname) {
      case "/api/v1/info":
        return new Response(JSON.stringify(env({ version: "0.1.0", exchange: "NSE", sessionOpen: "09:15", sessionClose: "15:30", timezone: "Asia/Kolkata" })));
      case "/api/v1/stocks":
        return new Response(JSON.stringify(stocks));
      case "/api/v1/instrument":
        return new Response(JSON.stringify(env({ ...q("RELIANCE", "Energy", 2842.5, 36.8, 1.31, 12e6), open: 2810, high: 2856, low: 2798.1, prevClose: 2805.7, avgVolume20d: null, chartUrl: "/grafana/explore?full" })));
      case "/api/v1/chart":
        return new Response(JSON.stringify(env({ symbol: "RELIANCE", range: "1D", points: [{ t: 1, v: 2800 }, { t: 2, v: 2842.5 }], volume: [], volumeLabel: "Traded volume", last: 2842.5 })));
      default:
        return new Response(JSON.stringify({ error: "unknown command" }), { status: 404 });
    }
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ExplorerPage />
    </QueryClientProvider>,
  );
  return fetchMock;
}

const urls = (m: ReturnType<typeof vi.fn>) => m.mock.calls.map(([u]) => String(u));

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  localStorage.clear();
  nav.params = new URLSearchParams();
  nav.replace.mockClear();
});

describe("ExplorerPage", () => {
  it("lists, searches, filters, sorts and pages stocks", async () => {
    const fetchMock = renderPage();
    const results = screen.getByLabelText("Stock results");
    expect(await within(results).findByText("Results: 30 stocks")).toBeInTheDocument();
    expect(within(results).getByText("-12.35 (-0.36%)")).toBeInTheDocument();
    expect(within(results).getAllByText("—", { selector: "td" }).length).toBeGreaterThanOrEqual(3);
    expect(within(results).getByText("Showing 1–25 of 30 stocks")).toBeInTheDocument();
    expect(urls(fetchMock)).toContain("/api/v1/stocks?q=&sector=&price=all&change=all&sort=symbol&dir=asc&page=1&per=25");

    fireEvent.change(screen.getByLabelText("Search stocks by symbol"), { target: { value: " inf " } });
    fireEvent.click(screen.getByText("Search"));
    expect(nav.replace).toHaveBeenCalledWith("/explorer?q=INF");
    await waitFor(() => expect(urls(fetchMock).some((u) => u.includes("q=INF&"))).toBe(true));

    fireEvent.change(screen.getByLabelText("Change"), { target: { value: "up" } });
    fireEvent.change(screen.getByLabelText("Sort"), { target: { value: "volume:desc" } });
    await waitFor(() => expect(urls(fetchMock).some((u) => u.includes("change=up&sort=volume&dir=desc&page=1"))).toBe(true));
    fireEvent.click(screen.getByText("Next ›"));
    await waitFor(() => expect(urls(fetchMock).some((u) => u.includes("page=2"))).toBe(true));

    fireEvent.click(screen.getByLabelText("View TCS"));
    expect(nav.replace).toHaveBeenLastCalledWith("/explorer?q=INF&symbol=TCS");
  });

  it("rejects malformed searches", async () => {
    renderPage();
    const input = screen.getByLabelText("Search stocks by symbol");
    fireEvent.change(input, { target: { value: "<script>" } });
    fireEvent.submit(input);
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(nav.replace).not.toHaveBeenCalled();
  });

  it("shows details for ?symbol= and adds to the watchlist after confirmation", async () => {
    nav.params = new URLSearchParams("symbol=reliance");
    renderPage();
    const detail = await screen.findByLabelText("RELIANCE details");
    expect(await within(detail).findAllByText("2,856.00", { selector: "dd" })).toHaveLength(2);
    expect(screen.getByText("Explain RELIANCE in Analyst").closest("a")).toHaveAttribute("href", "/analyst?symbol=RELIANCE");
    expect(within(detail).getByText("OpenBB Analysis").closest("a")).toHaveAttribute("href", "/openbb?symbol=RELIANCE.NS");

    await waitFor(() => expect(within(detail).getByLabelText("Add to My Watchlist")).toBeEnabled());
    fireEvent.click(within(detail).getByLabelText("Add to My Watchlist"));
    const dialog = await screen.findByRole("dialog");
    expect(localStorage.getItem(WATCHLIST_KEY)).toBeNull();
    fireEvent.click(within(dialog).getByText("Add"));
    await waitFor(() => expect(JSON.parse(localStorage.getItem(WATCHLIST_KEY) ?? "{}").groups[0].symbols).toEqual(["RELIANCE"]));

    fireEvent.click(screen.getByText("Close details"));
    expect(nav.replace).toHaveBeenCalledWith("/explorer");
  });

  it("explains empty results and clears filters", async () => {
    const fetchMock = renderPage({ stocks: page([]) });
    expect(await screen.findByText(/No stock data recorded yet/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Sector"), { target: { value: "IT" } });
    expect(await screen.findByText("No stocks match your search or filters.", { exact: false })).toBeInTheDocument();
    fireEvent.click(screen.getByText("Clear search and filters"));
    await waitFor(() => expect(urls(fetchMock).at(-1)).toContain("sector=&"));
  });

  it("shows the error state", async () => {
    renderPage({ ok: false });
    expect(await screen.findByText("Unavailable — upstream unavailable")).toBeInTheDocument();
  });
});
