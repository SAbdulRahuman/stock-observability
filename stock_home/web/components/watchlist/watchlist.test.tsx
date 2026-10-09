import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { formatVolume } from "@/lib/format";
import { WATCHLIST_KEY } from "@/lib/watchlist";
import { WatchlistPage } from "./watchlist-page";

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const spark = [
  { t: 1759895100000, v: 2800 },
  { t: 1759895400000, v: 2842.5 },
];
const quotes: Record<string, unknown> = {
  RELIANCE: { exchange: "NSE", symbol: "RELIANCE", sector: "Energy", price: 2842.5, change: 36.8, changePercent: 1.31, volume: 12_400_000, spark, exploreUrl: "/grafana/explore?r" },
  TCS: { exchange: "NSE", symbol: "TCS", price: 3412.2, change: -12.35, changePercent: -0.36, exploreUrl: "/grafana/explore?t" },
  HDFCBANK: { exchange: "NSE", symbol: "HDFCBANK", price: 1672.1, change: 22.45, changePercent: 1.36, exploreUrl: "" },
  INFY: { exchange: "NSE", symbol: "INFY", price: 1487.3, change: 18.9, changePercent: 1.29, exploreUrl: "" },
};

const routes: Record<string, unknown> = {
  "/api/v1/info": env({ version: "0.1.0", exchange: "NSE", sessionOpen: "09:15", sessionClose: "15:30", timezone: "Asia/Kolkata", quoteSymbolsMax: 20 }),
  "/api/v1/links": env([{ id: "alertmanager", label: "Alertmanager", href: "/alertmanager/" }]),
  "/api/v1/alerts": env({
    bySeverity: { critical: 1 },
    byCategory: {},
    silenced: 0,
    inhibited: 0,
    activeSilences: 0,
    items: [{ name: "PriceAboveR1", severity: "critical", category: "price", summary: "Price > 2,840", symbol: "RELIANCE", startsAt: "2026-10-08T04:11:00Z", fingerprint: "a", value: "2,842.50" }],
    truncated: false,
  }),
  "/api/v1/timeline": env({ from: "2026-10-07T04:12:00Z", items: [], bySeverity: {}, truncated: false }),
  "/api/v1/instrument": env({
    ...(quotes.RELIANCE as object),
    open: 2810,
    high: 2856,
    low: 2798.1,
    prevClose: 2805.7,
    avgVolume20d: 14_200_000,
    chartUrl: "/grafana/explore?full",
  }),
  "/api/v1/chart": env({ symbol: "RELIANCE", range: "1D", points: spark, volume: [], volumeLabel: "Traded volume", last: 2842.5 }),
};

function respond(url: string, ok: boolean) {
  const u = new URL(url, "http://localhost");
  if (!ok) return new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
  if (u.pathname === "/api/v1/quotes") {
    const items = (u.searchParams.get("symbols") ?? "").split(",").map((s) => quotes[s] ?? { exchange: "", symbol: s, price: null, change: null, changePercent: null, exploreUrl: "" });
    return new Response(JSON.stringify(env({ items })), { status: 200 });
  }
  const body = routes[u.pathname];
  if (body) return new Response(JSON.stringify(body), { status: 200 });
  return new Response(JSON.stringify({ error: "unknown command" }), { status: 404 });
}

function renderPage({ ok = true, groups }: { ok?: boolean; groups?: unknown } = {}) {
  if (groups) localStorage.setItem(WATCHLIST_KEY, JSON.stringify({ groups }));
  const fetchMock = vi.fn(async (url: string) => respond(url, ok));
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WatchlistPage />
    </QueryClientProvider>,
  );
  return fetchMock;
}

const seeded = [
  { id: "default", name: "My Watchlist", symbols: ["RELIANCE", "TCS"] },
  { id: "banking", name: "Banking", symbols: ["HDFCBANK"] },
];
const stored = () => JSON.parse(localStorage.getItem(WATCHLIST_KEY) ?? "{}").groups;

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe("WatchlistPage", () => {
  it("renders groups, table, details and supporting panels", async () => {
    const fetchMock = renderPage({ groups: seeded });
    expect(await screen.findByText("My Watchlist (2)")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("Banking (1)")).toBeInTheDocument();

    const table = screen.getByLabelText("My Watchlist symbols");
    expect(await within(table).findByText("+36.80")).toBeInTheDocument();
    expect(within(table).getByText("-0.36%")).toBeInTheDocument();
    expect(within(table).getByLabelText("RELIANCE: alert firing")).toBeInTheDocument();
    expect(within(table).getByLabelText("TCS: no active alert")).toBeInTheDocument();

    const detail = screen.getByLabelText("RELIANCE details");
    expect(await within(detail).findAllByText("2,856.00", { selector: "dd" })).toHaveLength(2);
    expect(within(detail).getByText(formatVolume(14_200_000), { selector: "dd" })).toBeInTheDocument();
    expect(within(detail).getByText("Energy")).toBeInTheDocument();
    expect(within(detail).getByLabelText("Remove from My Watchlist")).toHaveAttribute("aria-pressed", "true");
    expect(within(detail).getByText("Full Chart").closest("a")).toHaveAttribute("href", "/grafana/explore?full");
    expect(within(detail).getByText("OpenBB Analysis").closest("a")).toHaveAttribute("href", "/openbb?symbol=RELIANCE.NS");
    expect(within(detail).getByText("Set Alert").closest("button")).toBeDisabled();
    expect(await within(detail).findByLabelText(/RELIANCE 1D: last 2,842.50/)).toBeInTheDocument();
    fireEvent.click(within(detail).getByText("5Y"));
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/chart?symbol=RELIANCE&range=5Y", expect.anything());

    expect(screen.getAllByText("1 (50.0%)", { selector: "div" })).toHaveLength(2);
    expect(screen.getByText("0 (0.0%)", { selector: "div" })).toBeInTheDocument();
    const recent = screen.getByLabelText("Recent Alerts (Watchlist)");
    expect(await within(recent).findByText("Price > 2,840")).toBeInTheDocument();

    fireEvent.click(within(table).getByLabelText("Show TCS details"));
    expect(await screen.findByLabelText("TCS details")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Search in watchlist"), { target: { value: "tc" } });
    expect(within(table).queryByText("RELIANCE")).toBeNull();
    fireEvent.change(screen.getByLabelText("Search in watchlist"), { target: { value: "" } });
    fireEvent.change(screen.getByLabelText("Filter"), { target: { value: "gainers" } });
    expect(within(table).queryByText("TCS")).toBeNull();

    fireEvent.click(screen.getByText("Banking (1)"));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/v1/quotes?symbols=HDFCBANK", expect.anything()));
  });

  it("validates symbols with the backend before adding them", async () => {
    const fetchMock = renderPage({ groups: seeded });
    await screen.findByText("+36.80");
    fireEvent.click(screen.getByText("Add Symbol"));
    const dialog = await screen.findByRole("dialog");
    fireEvent.change(within(dialog).getByLabelText("Symbols"), { target: { value: "infy, bad!, tcs, nope" } });
    fireEvent.click(within(dialog).getByText("Check"));
    expect(await within(dialog).findByText("✓ INFY")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/quotes?symbols=INFY%2CNOPE", expect.anything());
    expect(within(dialog).getByText("Not a trading symbol: BAD!")).toBeInTheDocument();
    expect(within(dialog).getByText("Already listed: TCS")).toBeInTheDocument();
    expect(within(dialog).getByText("No market data: NOPE")).toBeInTheDocument();
    fireEvent.click(within(dialog).getByText("Add 1 symbol"));
    await waitFor(() => expect(stored()[0].symbols).toEqual(["RELIANCE", "TCS", "INFY"]));
    expect(await screen.findByText("My Watchlist (3)")).toBeInTheDocument();
  });

  it("removes selected symbols only after confirmation", async () => {
    renderPage({ groups: seeded });
    await screen.findByText("+36.80");
    fireEvent.click(screen.getByLabelText("Select TCS"));
    fireEvent.click(screen.getByText("Remove selected (1)"));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("Remove TCS?")).toBeInTheDocument();
    fireEvent.click(within(dialog).getByText("Cancel"));
    expect(stored()[0].symbols).toEqual(["RELIANCE", "TCS"]);

    fireEvent.click(screen.getByText("Remove selected (1)"));
    fireEvent.click(within(await screen.findByRole("dialog")).getByText("Remove"));
    await waitFor(() => expect(stored()[0].symbols).toEqual(["RELIANCE"]));
  });

  it("imports CSV and creates watchlists", async () => {
    renderPage({ groups: seeded });
    await screen.findByText("+36.80");
    const file = new File(["symbol\nINFY\nRELIANCE\n"], "list.csv", { type: "text/csv" });
    fireEvent.change(screen.getByTestId("import-file"), { target: { files: [file] } });
    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText("✓ INFY")).toBeInTheDocument();
    fireEvent.click(within(dialog).getByText("Add 1 symbol"));
    await waitFor(() => expect(stored()[0].symbols).toContain("INFY"));

    fireEvent.click(screen.getByText("Create New Watchlist"));
    const create = await screen.findByRole("dialog");
    fireEvent.change(within(create).getByLabelText("Watchlist name"), { target: { value: "Swing Trade" } });
    fireEvent.click(within(create).getByText("Create"));
    expect(await screen.findByText("Swing Trade (0)")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("Swing Trade is empty.", { exact: false })).toBeInTheDocument();
  });

  it("shows empty states without a stored watchlist", async () => {
    renderPage();
    expect(await screen.findByText("My Watchlist is empty.", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("Select a symbol to see its details")).toBeInTheDocument();
    expect(await screen.findByText("No alerts for these symbols")).toBeInTheDocument();
    expect(screen.getByText("Export").closest("button")).toBeDisabled();
  });

  it("disables editing and shows unavailable states when the API fails", async () => {
    renderPage({ ok: false, groups: seeded });
    expect(await screen.findByText("Unavailable — upstream unavailable")).toBeInTheDocument();
    expect(screen.getByText("Add Symbol").closest("button")).toBeDisabled();
    expect(screen.getByText("Import").closest("button")).toBeDisabled();
    expect(await screen.findByText(/Alerts unavailable/)).toBeInTheDocument();
  });
});
