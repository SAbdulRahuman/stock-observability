import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { formatVolume } from "@/lib/format";
import { TopMoversPage } from "./top-movers-page";

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const spark = [
  { t: 1, v: 1 },
  { t: 2, v: 2 },
];
const q = (symbol: string, price: number, change: number, changePercent: number, extra = {}) => ({
  exchange: "NSE",
  symbol,
  sector: "Banks",
  price,
  change,
  changePercent,
  exploreUrl: "",
  ...extra,
});
const gainers = Array.from({ length: 7 }, (_, i) => q(`G${i}`, 100 + i, 1, 5 - i * 0.5, { spark }));
const scan = {
  universe: "core",
  sector: "",
  price: "all",
  total: 200,
  advancers: 120,
  decliners: 70,
  unchanged: 10,
  gainers,
  losers: [q("HCLTECH", 1548.2, -32.8, -2.08)],
  mostActive: [q("RELIANCE", 2842.5, 36.8, 1.31, { volume: 124_500_000 })],
  volumeChange: [{ ...q("SBIN", 621.35, 18.75, 3.11), relativeVolume: 2.96 }],
  high52: [{ ...q("HAL", 5312.4, 10, 0.6), high52: 5280, low52: 3500, distancePercent: 0.61, position: 1 }],
  low52: [{ ...q("IDEA", 7.25, -0.05, -0.68), high52: 18, low52: 7.3, distancePercent: -0.68, position: 0 }],
  distribution: [
    { label: "< -5%", from: null, to: -5, count: 3 },
    { label: "0%", from: 0, to: 0, count: 10 },
    { label: "≥ 5%", from: 5, to: null, count: 4 },
  ],
  sectors: [
    { name: "Banks", changePercent: 1.2, count: 12, gainer: q("SBIN", 621.35, 18.75, 3.11), loser: null },
    { name: "IT", changePercent: -0.8, count: 10, gainer: null, loser: q("HCLTECH", 1548.2, -32.8, -2.08) },
  ],
  sectorNames: ["Banks", "IT"],
};
const empty = { ...scan, total: 0, advancers: 0, decliners: 0, unchanged: 0, gainers: [], losers: [], mostActive: [], volumeChange: [], high52: [], low52: [], sectors: [] };

function renderPage(body: unknown, status = 200) {
  const fetchMock = vi.fn(async (url: string) => {
    if (url.startsWith("/api/v1/info")) return new Response(JSON.stringify(env({ version: "0.1.0", exchange: "NSE", sessionOpen: "09:15", sessionClose: "15:30", timezone: "Asia/Kolkata" })));
    return new Response(JSON.stringify(body), { status });
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <TopMoversPage />
    </QueryClientProvider>,
  );
  return fetchMock;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("TopMoversPage", () => {
  it("renders summary, rankings, ranges, distribution and sectors", async () => {
    renderPage(env(scan));
    const gainersCard = await screen.findByLabelText("Top Gainers summary");
    expect(within(gainersCard).getByText("120")).toBeInTheDocument();
    expect(within(gainersCard).getByText("of 200 · best G0 +5.00%")).toBeInTheDocument();
    expect(within(screen.getByLabelText("Highest Volume Change summary")).getByText("2.96x usual volume pace")).toBeInTheDocument();
    expect(within(screen.getByLabelText("Most Active summary")).getByText("RELIANCE")).toBeInTheDocument();

    expect(screen.getByText("G4")).toBeInTheDocument();
    expect(screen.queryByText("G5")).toBeNull();
    fireEvent.click(screen.getByText("View All (7)"));
    expect(screen.getByText("G6")).toBeInTheDocument();
    expect(screen.getByText("G0").closest("a")).toHaveAttribute("href", "/explorer?symbol=G0");

    expect(screen.getByText(formatVolume(124_500_000))).toBeInTheDocument();
    expect(screen.getByLabelText("HAL position in 52-week range")).toHaveAttribute("aria-valuenow", "100");
    expect(screen.getByTitle("< -5%: 3")).toBeInTheDocument();
    expect(screen.getByText("70 down · 10 unchanged · 120 up")).toBeInTheDocument();
    expect(screen.getByTitle("IT · 10 instruments")).toBeInTheDocument();
  });

  it("sends the filters to the backend", async () => {
    const fetchMock = renderPage(env(scan));
    await screen.findByText("G0");
    fireEvent.change(screen.getByLabelText("Universe"), { target: { value: "all" } });
    fireEvent.change(screen.getByLabelText("Sector"), { target: { value: "IT" } });
    fireEvent.change(screen.getByLabelText("Price"), { target: { value: "500-2000" } });
    await waitFor(() => expect(fetchMock.mock.calls.map(([u]) => u)).toContain("/api/v1/movers/scan?universe=all&price=500-2000&sector=IT"));
    expect(screen.getByLabelText("Time range")).toBeDisabled();
  });

  it("shows empty states", async () => {
    renderPage(env(empty));
    expect(await screen.findByText("No gainers for these filters")).toBeInTheDocument();
    expect(screen.getByText("No losers for these filters")).toBeInTheDocument();
    expect(screen.getAllByText("No 52-week range data")).toHaveLength(2);
    expect(screen.getByText("No instruments for these filters")).toBeInTheDocument();
    expect(within(screen.getByLabelText("Most Active summary")).getByText("No traded volume yet")).toBeInTheDocument();
  });

  it("shows unavailable states when the API fails", async () => {
    renderPage({ error: "upstream unavailable" }, 502);
    expect(await screen.findAllByText(/Unavailable — upstream unavailable/)).toHaveLength(7);
    expect(screen.getAllByText("Unavailable")).toHaveLength(4);
    expect(screen.getByText("Export").closest("button")).toBeDisabled();
  });
});
