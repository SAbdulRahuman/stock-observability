import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fixtures } from "./fixtures";
import { Workbench } from "./workbench";

const nav = vi.hoisted(() => ({ replace: vi.fn(), params: new URLSearchParams() }));

vi.mock("next/navigation", () => ({
  useSearchParams: () => nav.params,
  useRouter: () => ({ replace: nav.replace }),
  usePathname: () => "/openbb",
}));

type Reply = { status: number; body: unknown };
let requests: URL[] = [];

function mockFetch(reply: (path: string, url: URL) => Reply) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(String(input), "http://localhost");
      requests.push(url);
      const { status, body } = reply(url.pathname.replace("/api/v2/", ""), url);
      return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
    }),
  );
}

function renderWorkbench() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Workbench />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  requests = [];
  nav.replace.mockReset();
  nav.params = new URLSearchParams();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("Workbench", () => {
  it("renders every panel for the default symbol", async () => {
    mockFetch((path) => ({ status: 200, body: fixtures[path] }));
    renderWorkbench();

    expect(await screen.findByTestId("last-price")).toHaveTextContent("1,412.30");
    expect(screen.getByText(/\+11\.80 \(\+0\.84%\)/)).toBeInTheDocument();
    expect(await screen.findByText("Oil & Gas Refining & Marketing")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Website/ })).toHaveAttribute("href", "https://www.ril.com/");
    expect(await screen.findByText("3 sessions")).toBeInTheDocument();

    const news = await screen.findByRole("link", { name: "Reliance shares rise ahead of quarterly results" });
    expect(news).toHaveAttribute("rel", "noopener noreferrer");
    expect(screen.getByText("Untrusted link is shown as text").closest("a")).toBeNull();

    const quote = requests.find((u) => u.pathname === "/api/v2/equity/price/quote");
    expect(quote?.searchParams.get("symbol")).toBe("RELIANCE.NS");
    expect(quote?.searchParams.get("provider")).toBe("yfinance");
    expect(requests.find((u) => u.pathname === "/api/v2/equity/price/historical")?.searchParams.get("start_date")).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("reads symbol, provider and range from the URL", async () => {
    nav.params = new URLSearchParams("symbol=tcs.ns&provider=cboe&range=1M");
    mockFetch((path) => ({ status: 200, body: fixtures[path] }));
    renderWorkbench();

    expect(await screen.findByText("Quote · TCS.NS")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "1M" })).toHaveAttribute("aria-pressed", "true");
    expect(requests.every((u) => u.searchParams.get("provider") === "cboe")).toBe(true);
  });

  it("explains when stock_home has no OpenBB upstream", async () => {
    mockFetch(() => ({ status: 404, body: { error: "unknown command" } }));
    renderWorkbench();

    expect(await screen.findAllByText(/OpenBB is not connected/)).toHaveLength(4);
  });

  it("shows OpenBB error details per panel", async () => {
    mockFetch((path) =>
      path === "news/company"
        ? { status: 422, body: { detail: [{ msg: "Field required" }] } }
        : { status: 200, body: fixtures[path] },
    );
    renderWorkbench();

    expect(await screen.findByText("Unavailable — Field required")).toBeInTheDocument();
    expect(await screen.findByTestId("last-price")).toBeInTheDocument();
  });

  it("derives the change from the previous close when the provider omits it", async () => {
    const quote = { ...(fixtures["equity/price/quote"] as { results: Record<string, unknown>[] }) };
    quote.results = [{ ...quote.results[0], change: null, change_percent: null }];
    mockFetch((path) => ({ status: 200, body: path === "equity/price/quote" ? quote : fixtures[path] }));
    renderWorkbench();

    expect(await screen.findByText(/\+11\.80 \(\+0\.84%\)/)).toBeInTheDocument();
  });

  it("shows an empty state when OpenBB has no data", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) =>
        String(input).includes("news/company")
          ? new Response(null, { status: 204 })
          : new Response(JSON.stringify(fixtures[new URL(String(input), "http://x").pathname.replace("/api/v2/", "")]), {
              status: 200,
              headers: { "Content-Type": "application/json" },
            }),
      ),
    );
    renderWorkbench();

    const news = await screen.findByRole("region", { name: "Company news" });
    expect(await within(news).findByText("No data")).toBeInTheDocument();
  });

  it("searches companies and switches symbol", async () => {
    mockFetch((path) => ({ status: 200, body: fixtures[path] }));
    renderWorkbench();

    fireEvent.change(screen.getByLabelText("Find company"), { target: { value: "tata consultancy" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    const results = await screen.findByRole("list", { name: "Search results" });
    expect(requests.find((u) => u.pathname === "/api/v2/yfinance/search")?.searchParams.get("query")).toBe("tata consultancy");

    fireEvent.click(within(results).getByText("TCS.BO"));
    expect(nav.replace).toHaveBeenCalledWith("/openbb?symbol=TCS.BO", { scroll: false });
  });

  it("changes range and provider through the URL", () => {
    mockFetch((path) => ({ status: 200, body: fixtures[path] }));
    renderWorkbench();

    fireEvent.click(screen.getByRole("button", { name: "5Y" }));
    expect(nav.replace).toHaveBeenLastCalledWith("/openbb?range=5Y", { scroll: false });
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "tmx" } });
    expect(nav.replace).toHaveBeenLastCalledWith("/openbb?provider=tmx", { scroll: false });
  });
});
