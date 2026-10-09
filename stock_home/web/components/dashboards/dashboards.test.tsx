import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DashboardsPage } from "./dashboards-page";

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const items = [
  { uid: "nse-r1", title: "NSE Market Observability (R1)", folder: "NSE", tags: ["nse", "release-1"], href: "/grafana/d/nse-r1" },
  { uid: "nse-r3", title: "NSE Risk Intelligence (R3)", tags: ["risk"], href: "/grafana/d/nse-r3" },
];

function renderPage(body: unknown, status = 200) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string) => {
      if (url === "/api/v1/links") return new Response(JSON.stringify(env([{ id: "grafana", label: "Grafana", href: "/grafana/" }])));
      return new Response(JSON.stringify(body), { status });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <DashboardsPage />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("DashboardsPage", () => {
  it("lists Grafana dashboards and filters them", async () => {
    renderPage(env({ items }));
    const r1 = await screen.findByTitle("NSE Market Observability (R1) (opens in a new tab)");
    expect(r1).toHaveAttribute("href", "/grafana/d/nse-r1");
    expect(r1).toHaveAttribute("rel", "noopener noreferrer");
    expect(screen.getByText("Folder: NSE")).toBeInTheDocument();
    expect(await screen.findByLabelText("Open Grafana")).toHaveAttribute("href", "/grafana/");

    fireEvent.change(screen.getByLabelText("Filter dashboards"), { target: { value: "risk" } });
    expect(screen.queryByText("NSE Market Observability (R1)")).toBeNull();
    expect(screen.getByText("NSE Risk Intelligence (R3)")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Filter dashboards"), { target: { value: "zzz" } });
    expect(screen.getByText("No dashboards match “zzz”")).toBeInTheDocument();
  });

  it("shows the empty state", async () => {
    renderPage(env({ items: [] }));
    expect(await screen.findByText("Grafana has no dashboards visible to stock_home")).toBeInTheDocument();
  });

  it("shows the unavailable state", async () => {
    renderPage({ error: "upstream unavailable" }, 502);
    expect(await screen.findByText(/Unavailable — upstream unavailable/)).toBeInTheDocument();
  });
});
