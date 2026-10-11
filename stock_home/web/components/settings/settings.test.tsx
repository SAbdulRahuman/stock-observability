import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { ThemeProvider } from "next-themes";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Sparkline } from "@/components/dashboard/sparkline";
import { PrivacyProvider } from "@/components/privacy";
import { SETTINGS_KEY } from "@/lib/settings";
import { WATCHLIST_KEY } from "@/lib/watchlist";
import { SettingsPage } from "./settings-page";

const nav = vi.hoisted(() => ({ params: new URLSearchParams(), replace: vi.fn() }));
vi.mock("next/navigation", () => ({
  useSearchParams: () => nav.params,
  useRouter: () => ({ replace: nav.replace }),
  usePathname: () => "/settings",
}));

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const routes: Record<string, unknown> = {
  "/api/v1/info": env({
    version: "0.1.0",
    exchange: "NSE",
    sessionOpen: "09:15",
    sessionClose: "15:30",
    timezone: "Asia/Kolkata",
    indices: ["NIFTY 50", "NIFTY BANK"],
    holidayCount: 0,
  }),
  "/api/v1/integrations": env([
    { id: "kite", name: "Zerodha Kite API", configured: true, auth: "credentials are held by stock_exporter" },
    { id: "openbb", name: "OpenBB", configured: false, auth: "none" },
    { id: "prometheus", name: "Prometheus", configured: true, endpoint: "http://prometheus:9090/prometheus", auth: "none" },
  ]),
  "/api/v1/prom/status": {
    data: { ready: true, healthy: true, build: {}, runtime: { storageRetention: "15d" } },
    updatedAt: "2026-10-08T04:12:00Z",
  },
  "/api/v1/prom/rules": {
    data: {
      data: {
        groups: [
          {
            name: "stock_alerts",
            rules: [
              { type: "alerting", name: "PriceAboveR1", query: "stock_last_price > 1", health: "ok", duration: 60, labels: { severity: "critical" } },
              { type: "recording", name: "market:advancers:count", query: "count(x)", health: "ok" },
            ],
          },
        ],
      },
      durationMs: 1,
    },
    updatedAt: "",
  },
};

type Call = { url: string; method: string };

function renderPage(params = "", ok = true) {
  nav.params = new URLSearchParams(params);
  const calls: Call[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      const u = new URL(url, "http://localhost");
      calls.push({ url: u.pathname, method: init?.method ?? "GET" });
      if (!ok) return new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
      const test = /^\/api\/v1\/integrations\/(\w+)\/test$/.exec(u.pathname);
      if (test && init?.method === "POST") {
        const connected = test[1] === "prometheus";
        return new Response(
          JSON.stringify(
            env({
              id: test[1],
              status: connected ? "connected" : "not_configured",
              message: connected ? "Prometheus is ready" : "No OpenBB server is configured (--openbb.url)",
              latencyMs: 12.4,
              checkedAt: "2026-10-08T04:12:00Z",
            }),
          ),
        );
      }
      const body = routes[u.pathname];
      return body ? new Response(JSON.stringify(body)) : new Response("{}", { status: 404 });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <ThemeProvider attribute="class" defaultTheme="dark" enableSystem>
      <QueryClientProvider client={client}>
        <PrivacyProvider>
          <SettingsPage />
        </PrivacyProvider>
      </QueryClientProvider>
    </ThemeProvider>,
  );
  return calls;
}

const card = (title: string) => screen.getByRole("region", { name: title });
const stored = () => JSON.parse(localStorage.getItem(SETTINGS_KEY) ?? "{}");

beforeEach(() => {
  vi.stubGlobal(
    "matchMedia",
    vi.fn(() => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn(), addListener: vi.fn(), removeListener: vi.fn() })),
  );
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  localStorage.clear();
  nav.replace.mockReset();
});

describe("SettingsPage", () => {
  it("shows the nine General cards with server values and unavailable settings marked", async () => {
    renderPage();
    for (const t of [
      "Application Settings",
      "Market Settings",
      "Data Display Settings",
      "Notification Settings",
      "Dashboard Settings",
      "API & Integration Settings",
      "Data Retention",
      "Backup & Export",
      "Danger Zone",
    ]) {
      expect(card(t)).toBeInTheDocument();
    }
    const market = card("Market Settings");
    expect(await within(market).findByText("NSE")).toBeInTheDocument();
    expect(within(market).getByText("NIFTY 50")).toBeInTheDocument();
    expect(within(market).getByText("09:15 – 15:30")).toBeInTheDocument();
    expect(within(market).getByText("Not configured")).toBeInTheDocument();
    expect(await within(card("Data Retention")).findByText("15d")).toBeInTheDocument();
    expect(within(card("Notification Settings")).getAllByText("Unavailable")).toHaveLength(6);
  });

  it("switches tabs through the URL", async () => {
    renderPage("tab=integrations");
    expect(screen.getByRole("tab", { name: "Integrations" })).toHaveAttribute("aria-selected", "true");
    expect(card("API & Integration Settings")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Danger Zone" })).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: "System" }));
    expect(nav.replace).toHaveBeenCalledWith("/settings?tab=system", { scroll: false });
  });

  it("persists theme, sparkline and Home widget preferences", async () => {
    renderPage();
    fireEvent.click(within(card("Application Settings")).getByRole("radio", { name: "light" }));
    await waitFor(() => expect(localStorage.getItem("theme")).toBe("light"));
    expect(within(card("Application Settings")).getByRole("radio", { name: "light" })).toHaveAttribute("aria-checked", "true");

    const movers = within(card("Dashboard Settings")).getByRole("switch", { name: "Show Top Movers" });
    expect(movers).toHaveAttribute("aria-checked", "true");
    fireEvent.click(movers);
    expect(movers).toHaveAttribute("aria-checked", "false");
    expect(stored().home.topMovers).toBe(false);
    expect(stored().home.indexCards).toBe(true);
    expect(screen.getByText("Saved in this browser.")).toBeInTheDocument();

    fireEvent.click(within(card("Data Display Settings")).getByRole("switch", { name: "Show Sparkline" }));
    expect(stored().showSparklines).toBe(false);
    const { container } = render(<Sparkline points={[{ t: 1, v: 1 }, { t: 2, v: 2 }]} />);
    expect(container.querySelector("[aria-hidden]")).toBeNull();
  });

  it("tests integrations through the backend and shows masked details", async () => {
    const calls = renderPage("tab=integrations");
    const c = card("API & Integration Settings");
    expect(await within(c).findByText("Prometheus")).toBeInTheDocument();
    expect(within(c).getAllByText("Configured")).toHaveLength(1);
    expect(within(c).getByRole("link", { name: "Data Sources" })).toHaveAttribute("href", "/settings?tab=data-sources&source=kite");
    fireEvent.click(within(c).getByRole("button", { name: "Test Prometheus" }));
    expect(await within(c).findByText("Connected")).toBeInTheDocument();
    expect(within(c).getByText(/Prometheus is ready · 12 ms · checked/)).toBeInTheDocument();
    expect(calls).toContainEqual({ url: "/api/v1/integrations/prometheus/test", method: "POST" });
    fireEvent.click(within(c).getByRole("button", { name: "Test OpenBB" }));
    expect(await within(c).findByText(/No OpenBB server is configured/)).toBeInTheDocument();
    fireEvent.click(within(c).getByRole("button", { name: "Prometheus details" }));
    expect(within(c).getByText("http://prometheus:9090/prometheus")).toBeInTheDocument();
    expect(calls.every((x) => x.url.startsWith("/api/v1/"))).toBe(true);
  });

  it("exports watchlists and alert rules as JSON files", async () => {
    localStorage.setItem(WATCHLIST_KEY, JSON.stringify({ groups: [{ id: "default", name: "My Watchlist", symbols: ["RELIANCE"] }] }));
    const blobs: Blob[] = [];
    vi.stubGlobal("URL", Object.assign(URL, { createObjectURL: vi.fn((b: Blob) => (blobs.push(b), "blob:x")), revokeObjectURL: vi.fn() }));
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    renderPage("tab=watchlist");
    fireEvent.click(screen.getByRole("button", { name: "Export Watchlists" }));
    expect(await screen.findByText(/Exported stock_home-watchlists-\d{4}-\d{2}-\d{2}\.json/)).toBeInTheDocument();
    expect(JSON.parse(await blobs[0].text()).groups[0].symbols).toEqual(["RELIANCE"]);
    fireEvent.click(screen.getByRole("button", { name: "Export Alert Rules" }));
    expect(await screen.findByText(/Exported stock_home-alert-rules-/)).toBeInTheDocument();
    const rules = JSON.parse(await blobs[1].text());
    expect(rules.groups[0].rules).toEqual([{ alert: "PriceAboveR1", expr: "stock_last_price > 1", for: 60, labels: { severity: "critical" }, annotations: {} }]);
    expect(click).toHaveBeenCalledTimes(2);
  });

  it("imports watchlists only after preview and confirmation", async () => {
    localStorage.setItem(WATCHLIST_KEY, JSON.stringify({ groups: [{ id: "default", name: "My Watchlist", symbols: ["TCS"] }] }));
    renderPage("tab=watchlist");
    const input = screen.getByLabelText("Watchlist JSON file");
    fireEvent.change(input, { target: { files: [new File(["not json"], "bad.json", { type: "application/json" })] } });
    expect(await screen.findByText("Import rejected: The file is not valid JSON.")).toBeInTheDocument();

    const good = { groups: [{ id: "default", name: "My Watchlist", symbols: ["RELIANCE", "bad symbol!"] }, { id: "banks", name: "Banks", symbols: ["HDFCBANK"] }] };
    fireEvent.change(input, { target: { files: [new File([JSON.stringify(good)], "w.json", { type: "application/json" })] } });
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("Banks")).toBeInTheDocument();
    expect(within(dialog).getByText(/1 invalid or duplicate symbol/)).toBeInTheDocument();
    expect(JSON.parse(localStorage.getItem(WATCHLIST_KEY) ?? "{}").groups[0].symbols).toEqual(["TCS"]);
    fireEvent.click(within(dialog).getByRole("button", { name: "Replace watchlists" }));
    expect(await screen.findByText("Imported 2 group(s) with 2 symbol(s).")).toBeInTheDocument();
    expect(JSON.parse(localStorage.getItem(WATCHLIST_KEY) ?? "{}").groups[0].symbols).toEqual(["RELIANCE"]);
  });

  it("confirms destructive actions before changing anything", async () => {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify({ showSparklines: false }));
    localStorage.setItem(WATCHLIST_KEY, JSON.stringify({ groups: [{ id: "default", name: "My Watchlist", symbols: ["TCS"] }] }));
    renderPage("tab=system");
    fireEvent.click(screen.getByRole("button", { name: "Reset All Settings" }));
    let dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText(/Watchlists are kept/)).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(localStorage.getItem(SETTINGS_KEY)).not.toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Reset All Settings" }));
    dialog = await screen.findByRole("dialog");
    fireEvent.click(within(dialog).getByRole("button", { name: "Reset settings" }));
    expect(localStorage.getItem(SETTINGS_KEY)).toBeNull();
    expect(localStorage.getItem(WATCHLIST_KEY)).not.toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Clear Cache & Local Data" }));
    dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText(new RegExp(WATCHLIST_KEY.replace(/\./g, "\\.")))).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Delete local data" }));
    expect(localStorage.getItem(WATCHLIST_KEY)).toBeNull();
    expect(await screen.findByText("Local data and cached responses were cleared.")).toBeInTheDocument();
  });

  it("shows unavailable states when the server fails", async () => {
    renderPage("", false);
    expect(await screen.findByText(/Integration settings are unavailable: upstream unavailable/)).toBeInTheDocument();
    expect(await screen.findByText(/Server configuration is unavailable/)).toBeInTheDocument();
    expect(await within(card("Data Retention")).findByText("unavailable")).toBeInTheDocument();
    expect(within(card("Market Settings")).getAllByText("—").length).toBeGreaterThan(0);
  });
});
