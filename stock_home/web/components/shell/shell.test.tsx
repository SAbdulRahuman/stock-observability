import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AppShell } from "./app-shell";

const nav = vi.hoisted(() => ({ push: vi.fn(), pathname: "/" }));
vi.mock("next/navigation", () => ({
  usePathname: () => nav.pathname,
  useRouter: () => ({ push: nav.push }),
}));

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });

const ok: Record<string, unknown> = {
  "/api/v1/links": env([
    { id: "grafana", label: "Grafana", href: "/grafana/" },
    { id: "prometheus", label: "Prometheus", href: "/prometheus/" },
    { id: "alertmanager", label: "Alertmanager", href: "/alertmanager/" },
    { id: "openbb", label: "OpenBB", href: "/openbb" },
  ]),
  "/api/v1/info": env({ version: "0.1.0", exchange: "NSE", sessionOpen: "09:15", sessionClose: "15:30", timezone: "Asia/Kolkata" }),
  "/api/v1/status": env({
    market: { phase: "open", open: true, tradingDay: true, timezone: "Asia/Kolkata", now: "2026-10-08T04:12:00Z", feedSource: "kite" },
    banners: [],
  }),
  "/api/v1/alerts": env({
    bySeverity: { critical: 2, warning: 1, info: 0 },
    byCategory: {},
    silenced: 0,
    inhibited: 0,
    activeSilences: 0,
    items: [],
    truncated: false,
  }),
};

function renderShell(routes: Record<string, unknown> | null) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string) => {
      const body = routes?.[url];
      return body
        ? new Response(JSON.stringify(body), { status: 200 })
        : new Response(JSON.stringify({ error: "upstream unavailable" }), { status: 502 });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <AppShell>
        <p>page body</p>
      </AppShell>
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  nav.push.mockReset();
});

describe("AppShell", () => {
  it("renders the sidebar, header and footer from the API", async () => {
    renderShell(ok);
    const sidebar = screen.getByLabelText("Main");
    const labels = Array.from(sidebar.querySelectorAll("li")).map((li) => li.textContent?.replace(/\d+$/, "").trim());
    expect(labels).toEqual([
      "Home",
      "Market Overview",
      "Watchlist",
      "Alerts",
      "Top Movers",
      "Sector View",
      "Stock Explorer",
      "Dashboards",
      "OpenBB Analysis",
      "Prometheus",
      "Grafana",
      "AlertManager",
      "Settings",
    ]);
    expect(within(sidebar).getByText("Home").closest("a")).toHaveAttribute("aria-current", "page");
    expect(await within(sidebar).findByLabelText("3 active alerts")).toHaveTextContent("3");
    const prom = await within(sidebar).findByTitle("Prometheus (opens in a new tab)");
    expect(prom).toHaveAttribute("href", "/prometheus/");
    expect(prom).toHaveAttribute("rel", "noopener noreferrer");
    expect(within(sidebar).getByText("Market Overview").closest("a")).toHaveAttribute("href", "/market");
    expect(within(sidebar).getByText("Top Movers").closest("[aria-disabled]")).not.toBeNull();

    expect(await screen.findByText("Live Market")).toBeInTheDocument();
    expect(screen.getByLabelText("Alerts: 3 active")).toHaveAttribute("href", "/alertmanager/");
    expect(await screen.findByText("09:15 AM – 03:30 PM")).toBeInTheDocument();
    expect(screen.getByText("OPEN")).toBeInTheDocument();
    expect(screen.getByText("Live data via Zerodha Kite API")).toBeInTheDocument();
    expect(await screen.findByText("v0.1.0")).toBeInTheDocument();
    expect(screen.getByText("Data via Zerodha Kite API")).toBeInTheDocument();
    expect(screen.getByText("page body")).toBeInTheDocument();
  });

  it("shows unavailable states when the API fails", async () => {
    renderShell(null);
    expect(await screen.findByText("Status unavailable", { selector: "span[role=status]" })).toBeInTheDocument();
    expect(await screen.findByText("unavailable")).toBeInTheDocument();
    expect(screen.queryByLabelText(/active alerts/)).toBeNull();
    expect(screen.getByTitle("Prometheus link not configured")).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByText("Data via —")).toBeInTheDocument();
  });

  it("searches a valid symbol and toggles the sidebar", () => {
    renderShell(ok);
    const input = screen.getByLabelText("Search stocks");
    fireEvent.change(input, { target: { value: "<script>" } });
    fireEvent.submit(input);
    expect(nav.push).not.toHaveBeenCalled();
    expect(input).toHaveAttribute("aria-invalid", "true");

    fireEvent.change(input, { target: { value: " infy " } });
    fireEvent.submit(input);
    expect(nav.push).toHaveBeenCalledWith("/analyst?symbol=INFY");

    const toggle = screen.getByLabelText("Collapse sidebar");
    fireEvent.click(toggle);
    expect(screen.getByLabelText("Expand sidebar")).toHaveAttribute("aria-expanded", "false");
  });
});
