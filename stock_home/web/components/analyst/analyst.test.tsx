import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Analyst } from "./analyst";

const analysis = {
  symbol: "INFY",
  marketRegime: "TRENDING_BULL",
  regimes: { momentum: "UPTREND" },
  summary: ["INFY is up 2.40% today, 1.10 pp stronger than NIFTY 50."],
  evidence: [
    { kpi: "stock_change_percent", label: "Day change", value: 2.4, unit: "%", quality: "exact" },
    { kpi: "stock:volume_delta_1h:ratio", label: "Volume delta (1h, tick rule)", value: -0.45, unit: "", quality: "approximation", method: "tick-rule" },
  ],
  conditions: [],
  disclaimer: "Observability summary from recorded KPIs; not investment advice.",
};

function renderAnalyst(status: number, body: unknown) {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <Analyst />
    </QueryClientProvider>,
  );
  fireEvent.change(screen.getByLabelText("Symbol"), { target: { value: " infy " } });
  fireEvent.click(screen.getByText("Explain"));
  return fetchMock;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("Analyst", () => {
  it("explains a symbol with evidence and quality badges", async () => {
    const fetchMock = renderAnalyst(200, { data: analysis, updatedAt: "2026-10-08T05:00:00Z", stale: false });
    expect(await screen.findByText("Why is INFY moving?")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/analyst?symbol=INFY", expect.anything());
    expect(screen.getByText(analysis.summary[0])).toBeInTheDocument();
    expect(screen.getByText("approximation")).toBeInTheDocument();
    expect(screen.getByText("-0.45")).toBeInTheDocument();
    expect(screen.getByText(analysis.disclaimer)).toBeInTheDocument();
  });

  it("shows the API error", async () => {
    renderAnalyst(404, { error: "no KPIs recorded for INFY" });
    expect(await screen.findByText("no KPIs recorded for INFY")).toBeInTheDocument();
  });
});
