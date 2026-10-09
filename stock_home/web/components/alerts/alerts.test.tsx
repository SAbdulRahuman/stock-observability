import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AlertsPage } from "./alerts-page";
import { countsOverTime } from "./panels";

const NOW = new Date("2026-10-08T04:20:00Z");
const env = (data: unknown) => ({ data, updatedAt: NOW.toISOString(), stale: false });

const row = (over: Record<string, unknown>) => ({
  key: "k",
  name: "Rule",
  severity: "info",
  startsAt: "2026-10-08T04:00:00Z",
  status: "firing",
  source: "Prometheus",
  ...over,
});

const items = [
  row({ key: "f1", fingerprint: "f1", name: "PriceAboveR1", severity: "critical", symbol: "RELIANCE", summary: "Price > Resistance (R1)", description: "Price crossed R1 level", value: "2,842.50", threshold: "2840", startsAt: "2026-10-08T04:11:00Z", exploreUrl: "/grafana/explore?r", sourceUrl: "http://prometheus/graph" }),
  row({ key: "f2", fingerprint: "f2", name: "VolumeSpike", severity: "warning", symbol: "INFY", status: "acknowledged", startsAt: "2026-10-08T04:08:00Z", source: "Grafana", silencedUntil: "2026-10-08T05:00:00Z" }),
  row({ key: "f3", fingerprint: "f3", name: "RsiHigh", severity: "info", symbol: "TCS", status: "silenced", startsAt: "2026-10-08T02:05:00Z", source: "Alertmanager" }),
  row({ key: "r1", name: "HighVolume", severity: "info", symbol: "ITC", status: "resolved", startsAt: "2026-10-08T02:50:00Z", endsAt: "2026-10-08T03:00:00Z" }),
];
const timeline = [
  { name: "PriceAboveR1", severity: "critical", category: "price", symbol: "RELIANCE", start: "2026-10-08T04:11:00Z", end: null },
  { name: "HighVolume", severity: "info", category: "volume", symbol: "ITC", start: "2026-10-08T02:50:00Z", end: "2026-10-08T03:00:00Z" },
];

type Opts = { list?: unknown; allowed?: boolean; ok?: boolean };

function renderPage({ list = { items, timeline, from: "2026-10-07T04:20:00Z", truncated: false }, allowed = true, ok = true }: Opts = {}) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const u = new URL(url, "http://localhost");
    if (!ok) return new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
    switch (u.pathname) {
      case "/api/v1/alerts/items":
        return new Response(JSON.stringify(env(list)));
      case "/api/v1/info":
        return new Response(JSON.stringify(env({ version: "0.1.0", exchange: "NSE", sessionOpen: "09:15", sessionClose: "15:30", timezone: "Asia/Kolkata", silencesAllowed: allowed })));
      case "/api/v1/links":
        return new Response(JSON.stringify(env([{ id: "alertmanager", label: "Alertmanager", href: "/alertmanager/" }])));
      case "/api/v1/quotes":
        return new Response(JSON.stringify(env({ items: [{ exchange: "NSE", symbol: "RELIANCE", price: 2842.5, change: 36.8, changePercent: 1.31, exploreUrl: "" }] })));
      case "/api/v1/chart":
        return new Response(
          JSON.stringify(env({ symbol: "RELIANCE", range: "1D", points: [{ t: 1, v: 2830 }, { t: 2, v: 2842.5 }], volume: [], volumeLabel: "Traded volume", last: 2842.5 })),
        );
      case "/api/v1/silences": {
        const body = JSON.parse(String(init?.body));
        return new Response(JSON.stringify(env({ created: body.fingerprints.map((f: string) => ({ fingerprint: f, silenceId: "s" })), failed: [] })));
      }
      default:
        return new Response(JSON.stringify({ error: "not found" }), { status: 404 });
    }
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <AlertsPage />
    </QueryClientProvider>,
  );
  return fetchMock;
}

const posted = (fetchMock: ReturnType<typeof vi.fn>) =>
  fetchMock.mock.calls.filter(([u]) => u === "/api/v1/silences").map(([, init]) => JSON.parse(String((init as RequestInit).body)));

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(NOW);
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("AlertsPage", () => {
  it("renders summary, table, detail and supporting panels", async () => {
    renderPage();
    const total = await screen.findByLabelText("Total Alerts");
    expect(within(total).getByText("3")).toBeInTheDocument();
    expect(within(total).getByText("2 fired in the last 24h")).toBeInTheDocument();
    expect(within(screen.getByLabelText("Critical")).getByText("1 new in the last hour")).toBeInTheDocument();
    expect(within(screen.getByLabelText("Info")).getByText("0 new in the last hour")).toBeInTheDocument();

    const table = screen.getByLabelText("Alerts list");
    for (const s of ["Firing", "Acknowledged", "Silenced", "Resolved"]) expect(within(table).getByText(s)).toBeInTheDocument();
    expect(within(table).getByLabelText("Select ITC")).toBeDisabled();
    expect(within(table).getByText("Showing 1–4 of 4")).toBeInTheDocument();

    const detail = screen.getByLabelText("Selected alert");
    expect(within(detail).getByText("Price crossed R1 level")).toBeInTheDocument();
    expect(within(detail).getByText("₹2,840.00")).toBeInTheDocument();
    expect(await within(detail).findByText("₹2,842.50")).toBeInTheDocument();
    expect(await within(detail).findByLabelText(/trigger 2,840.00/)).toBeInTheDocument();
    expect(within(detail).getByText("9m 00s")).toBeInTheDocument();

    fireEvent.click(within(table).getByText("VolumeSpike"));
    expect(within(screen.getByLabelText("Selected alert")).getByText("INFY")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Status"), { target: { value: "resolved" } });
    expect(within(table).getByText("Showing 1–1 of 1")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Search alerts"), { target: { value: "nothing" } });
    expect(screen.getByText("No alerts match the filters")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Reset"));
    expect(screen.getByText("Showing 1–4 of 4")).toBeInTheDocument();

    expect(within(screen.getByLabelText("Alert Activity")).getByText("HighVolume resolved")).toBeInTheDocument();
  });

  it("acknowledges and silences only after confirmation", async () => {
    const fetchMock = renderPage();
    const detail = await screen.findByLabelText("Selected alert");
    await waitFor(() => expect(within(detail).getByText("Acknowledge").closest("button")).toBeEnabled());
    fireEvent.click(within(detail).getByText("Acknowledge"));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("Acknowledge 1 alert?")).toBeInTheDocument();
    expect(posted(fetchMock)).toHaveLength(0);
    fireEvent.click(within(dialog).getByText("Confirm acknowledge"));
    expect(await screen.findByText("Acknowledged 1 alert for 1h.")).toBeInTheDocument();
    expect(posted(fetchMock)[0]).toEqual({ fingerprints: ["f1"], kind: "acknowledge", duration: "1h", comment: "" });

    fireEvent.click(screen.getByLabelText("Select RELIANCE"));
    fireEvent.click(within(screen.getByText("1 selected").parentElement as HTMLElement).getByText("Silence"));
    const bulk = await screen.findByRole("dialog");
    fireEvent.change(within(bulk).getByLabelText("Duration"), { target: { value: "30m" } });
    fireEvent.change(within(bulk).getByLabelText("Comment"), { target: { value: "deploy" } });
    fireEvent.click(within(bulk).getByText("Confirm silence"));
    await waitFor(() => expect(posted(fetchMock)).toHaveLength(2));
    expect(posted(fetchMock)[1]).toEqual({ fingerprints: ["f1"], kind: "silence", duration: "30m", comment: "deploy" });
  });

  it("disables silences when the backend does not allow them", async () => {
    renderPage({ allowed: false });
    const detail = await screen.findByLabelText("Selected alert");
    expect(within(detail).getByText("Acknowledge").closest("button")).toBeDisabled();
    expect(within(detail).getByText("Silence").closest("button")).toBeDisabled();
  });

  it("paginates", async () => {
    const many = Array.from({ length: 12 }, (_, i) => row({ key: `k${i}`, fingerprint: `k${i}`, symbol: `S${i}` }));
    renderPage({ list: { items: many, timeline: [], from: "2026-10-07T04:20:00Z", truncated: false } });
    expect(await screen.findByText("Showing 1–10 of 12")).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Next page"));
    expect(screen.getByText("Showing 11–12 of 12")).toBeInTheDocument();
  });

  it("shows empty and error states", async () => {
    renderPage({ list: { items: [], timeline: [], from: "2026-10-07T04:20:00Z", truncated: false } });
    expect(await screen.findByText("No alerts firing or resolved in the last 24 hours")).toBeInTheDocument();
    expect(screen.getByText("No alert activity")).toBeInTheDocument();
    expect(screen.getByText("Select an alert to see its details")).toBeInTheDocument();
    cleanup();
    renderPage({ ok: false });
    expect(await screen.findByText("Unavailable — upstream unavailable")).toBeInTheDocument();
  });
});

describe("countsOverTime", () => {
  it("counts firing runs per severity at each instant", () => {
    const now = NOW.getTime();
    const data = countsOverTime(timeline, "1h", now, 4);
    expect(data).toHaveLength(5);
    expect(data.at(-1)).toMatchObject({ critical: 1, info: 0 });
    expect(data[0]).toMatchObject({ critical: 0 });
  });
});
