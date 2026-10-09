import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { autoStep } from "./query-tab";
import { PrometheusPage } from "./prometheus-page";

const nav = vi.hoisted(() => ({ params: new URLSearchParams(), replace: vi.fn() }));
vi.mock("next/navigation", () => ({
  useSearchParams: () => nav.params,
  useRouter: () => ({ replace: nav.replace }),
  usePathname: () => "/observability/prometheus",
}));

const wrap = (data: unknown, extra = {}) => ({ data: { data, durationMs: 12.3, ...extra }, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const target = (job: string, instance: string, health: string, lastError = "") => ({
  labels: { job, instance },
  scrapePool: job,
  scrapeUrl: `http://${instance}/metrics`,
  health,
  lastError,
  lastScrape: "2026-10-08T04:11:50Z",
  lastScrapeDuration: 0.0123,
  scrapeInterval: "1s",
  scrapeTimeout: "1s",
});
const routes: Record<string, unknown> = {
  "/api/v1/links": { data: [{ id: "prometheus", label: "Prometheus", href: "/prometheus/" }, { id: "alertmanager", label: "Alertmanager", href: "/alertmanager/" }], updatedAt: "" },
  "/api/v1/prom/status": {
    data: { ready: true, healthy: true, build: { version: "3.5.0", revision: "abc" }, runtime: { storageRetention: "15d", goroutineCount: 120 } },
    updatedAt: "2026-10-08T04:12:00Z",
  },
  "/api/v1/prom/targets": wrap({
    activeTargets: [target("stock_core", "exporter:9101", "up"), target("stock_broad", "exporter:9101", "up"), target("quantlab", "quantlab:9000", "down", "connection refused")],
  }),
  "/api/v1/prom/rules": wrap({
    groups: [
      {
        name: "stock_recording",
        interval: 15,
        lastEvaluation: "2026-10-08T04:11:55Z",
        rules: [
          { type: "recording", name: "market:advancers:count", query: 'count(stock_change_percent{instrument_type="EQ"} > 0)', health: "ok", lastEvaluation: "2026-10-08T04:11:55Z" },
          {
            type: "alerting",
            name: "StockExporterDown",
            query: 'up{job="stock_core"} == 0',
            health: "ok",
            state: "inactive",
            duration: 60,
            labels: { severity: "critical" },
            annotations: { summary: "exporter down" },
            alerts: [],
          },
          { type: "alerting", name: "Broken", query: "foo(", health: "err", lastError: "parse error", state: "inactive" },
        ],
      },
    ],
  }),
  "/api/v1/prom/alerts": wrap({
    alerts: [
      { labels: { alertname: "PriceAboveR1", severity: "critical", symbol: "RELIANCE" }, annotations: { summary: "Price > R1" }, state: "firing", activeAt: "2026-10-08T04:11:00Z", value: "2842.5" },
      { labels: { alertname: "VolumeSpike", severity: "warning", symbol: "INFY" }, annotations: {}, state: "pending", activeAt: "2026-10-08T04:10:00Z" },
    ],
  }),
  "/api/v1/prom/metric-names": wrap(["stock_last_price", "up"]),
};

function renderPage(params = "", ok = true) {
  nav.params = new URLSearchParams(params);
  const fetchMock = vi.fn(async (url: string) => {
    const u = new URL(url, "http://localhost");
    if (!ok && u.pathname !== "/api/v1/links") return new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
    if (u.pathname === "/api/v1/prom/query") {
      const q = u.searchParams.get("query") ?? "";
      if (q.includes("((")) return new Response(JSON.stringify({ error: "bad_data: parse error: unexpected (" }), { status: 400 });
      return new Response(JSON.stringify(wrap({ resultType: "vector", result: [{ metric: { __name__: "up", job: "stock_core" }, value: [1759895100, "1"] }] }, { warnings: ["w1"] })));
    }
    if (u.pathname === "/api/v1/prom/query_range") {
      return new Response(JSON.stringify(wrap({ resultType: "matrix", result: [{ metric: { job: "stock_core" }, values: [[1759895100, "1"], [1759895160, "1"]] }] })));
    }
    const body = routes[u.pathname];
    return body ? new Response(JSON.stringify(body)) : new Response(JSON.stringify({ error: "not found" }), { status: 404 });
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <PrometheusPage />
    </QueryClientProvider>,
  );
  return fetchMock;
}

const urls = (m: ReturnType<typeof vi.fn>) => m.mock.calls.map(([u]) => String(u));

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  nav.replace.mockReset();
});

describe("PrometheusPage", () => {
  it("summarises targets, rules and alerts on the overview", async () => {
    renderPage();
    expect(await within(screen.getByLabelText("Targets UP")).findByText("2 / 3")).toBeInTheDocument();
    expect(within(screen.getByLabelText("Targets DOWN")).getByText("1")).toBeInTheDocument();
    expect(await within(screen.getByLabelText("Rules")).findByText("2 alerting · 1 recording")).toBeInTheDocument();
    expect(await within(screen.getByLabelText("Firing Alerts")).findByText("1 pending")).toBeInTheDocument();
    expect(await screen.findByText("Ready")).toBeInTheDocument();
    expect(screen.getByText("v3.5.0")).toBeInTheDocument();
    expect((await screen.findByText("Open full Prometheus UI")).closest("a")).toHaveAttribute("href", "/prometheus/");
    expect(within(screen.getByLabelText("Target health by job")).getByText("0/1 up")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Scrape health by job"));
    expect(nav.replace).toHaveBeenCalledWith("/observability/prometheus?tab=query&query=up", { scroll: false });
  });

  it("filters targets and shows details", async () => {
    renderPage("tab=targets");
    expect(await screen.findByText("connection refused")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Target state"), { target: { value: "down" } });
    expect(screen.queryByText("stock_core")).toBeNull();
    fireEvent.click(screen.getByText("quantlab"));
    const detail = screen.getByLabelText("quantlab · quantlab:9000 details");
    expect(within(detail).getByText("http://quantlab:9000/metrics")).toBeInTheDocument();
    expect(within(detail).getByText('job="quantlab"')).toBeInTheDocument();
  });

  it("lists rules by group and shows expressions read-only", async () => {
    renderPage("tab=rules");
    expect(await screen.findByText("stock_recording", { selector: "th" })).toBeInTheDocument();
    expect(screen.getByText("parse error")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Rule type"), { target: { value: "recording" } });
    expect(screen.queryByText("StockExporterDown")).toBeNull();
    fireEvent.change(screen.getByLabelText("Rule type"), { target: { value: "all" } });
    fireEvent.click(screen.getByText("StockExporterDown"));
    const detail = screen.getByLabelText("StockExporterDown details");
    expect(within(detail).getByText('up{job="stock_core"} == 0')).toBeInTheDocument();
    expect(within(detail).getByText('summary="exporter down"')).toBeInTheDocument();
  });

  it("shows Prometheus alert state and links to AlertManager", async () => {
    renderPage("tab=alerts");
    expect(await screen.findByText("PriceAboveR1")).toBeInTheDocument();
    expect(screen.getByText("open AlertManager")).toHaveAttribute("href", "/alertmanager/");
    fireEvent.change(screen.getByLabelText("Alert state"), { target: { value: "pending" } });
    expect(screen.queryByText("PriceAboveR1")).toBeNull();
    fireEvent.click(screen.getByText("VolumeSpike"));
    expect(within(screen.getByLabelText("VolumeSpike details")).getByText('symbol="INFY"')).toBeInTheDocument();
  });

  it("runs instant and range queries and shows errors", async () => {
    const fetchMock = renderPage("tab=query&query=up");
    expect(screen.getByLabelText("PromQL expression")).toHaveValue("up");
    fireEvent.click(screen.getByText("Execute"));
    const results = screen.getByLabelText("Query results");
    expect(await within(results).findByText("vector · 1 series · 12 ms")).toBeInTheDocument();
    expect(within(results).getByText("w1")).toBeInTheDocument();
    expect(urls(fetchMock)).toContain("/api/v1/prom/query?query=up");

    fireEvent.click(screen.getByText("graph"));
    fireEvent.click(screen.getByText("Execute"));
    expect(await within(results).findByLabelText("Graph of 1 series")).toBeInTheDocument();
    const range = new URL(urls(fetchMock).find((u) => u.includes("query_range")) ?? "", "http://x");
    expect(Number(range.searchParams.get("end")) - Number(range.searchParams.get("start"))).toBe(3600);
    expect(range.searchParams.get("step")).toBe(String(autoStep(3600)));

    fireEvent.change(screen.getByLabelText("PromQL expression"), { target: { value: "sum((up)" } });
    fireEvent.click(screen.getByText("table"));
    fireEvent.click(screen.getByText("Execute"));
    expect(await within(results).findByText(/bad_data: parse error/)).toBeInTheDocument();
  });

  it("shows build and runtime status", async () => {
    renderPage("tab=status");
    expect(await screen.findByText("3.5.0")).toBeInTheDocument();
    expect(screen.getByText("15d")).toBeInTheDocument();
    expect(screen.getByText(/Configuration and command-line flags are intentionally not shown/)).toBeInTheDocument();
  });

  it("marks data unavailable when Prometheus is unreachable", async () => {
    renderPage("", false);
    await waitFor(() => expect(within(screen.getByLabelText("Targets UP")).getByText("unavailable")).toBeInTheDocument());
    expect(await screen.findByText("Unreachable")).toBeInTheDocument();
    expect(within(screen.getByLabelText("Firing Alerts")).getByText("unavailable")).toBeInTheDocument();
  });
});
