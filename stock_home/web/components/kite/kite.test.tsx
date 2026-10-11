import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DataSourcesPage } from "@/components/settings/data-sources-page";

const nav = vi.hoisted(() => ({ params: new URLSearchParams(), replace: vi.fn() }));
vi.mock("next/navigation", () => ({
  useSearchParams: () => nav.params,
  useRouter: () => ({ replace: nav.replace }),
  usePathname: () => "/settings",
}));

const base = {
  available: true,
  configured: false,
  source: "none",
  editable: true,
  sessionValid: false,
  redirectUrl: "http://127.0.0.1:8080/exporter/kite/callback",
  loginUrl: "/exporter/login",
  consoleUrl: "https://developers.kite.trade/apps",
};
type Setup = Record<string, unknown>;
type Call = { url: string; method: string; body?: string };
const env = (data: unknown) => ({ data, updatedAt: "2026-10-11T04:00:00Z", stale: false });

function mockServer(initial: Setup = {}) {
  let state: Setup = { ...base, ...initial };
  const calls: Call[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      const path = new URL(url, "http://localhost").pathname;
      const method = init?.method ?? "GET";
      calls.push({ url: path, method, body: init?.body as string | undefined });
      if (path === "/api/v1/integrations") {
        return new Response(JSON.stringify(env([
          { id: "kite", name: "Zerodha Kite API", configured: Boolean(state.configured), auth: "credentials held by stock_exporter" },
          { id: "prometheus", name: "Prometheus", configured: true, endpoint: "http://prometheus:9090", auth: "none" },
          { id: "alertmanager", name: "Alertmanager", configured: true, endpoint: "http://alertmanager:9093", auth: "none" },
          { id: "grafana", name: "Grafana", configured: true, endpoint: "http://grafana:3000", auth: "anonymous" },
          { id: "openbb", name: "OpenBB", configured: false, auth: "none" },
        ])));
      }
      if (path === "/api/v1/kite/setup") {
        if (method === "PUT") state = { ...state, configured: true, source: "file", apiKeyHint: "••••5678", sessionValid: false };
        if (method === "DELETE") state = { ...base };
        return new Response(JSON.stringify(env(state)));
      }
      return new Response("{}", { status: 404 });
    }),
  );
  return calls;
}

function renderPage(params = "") {
  nav.params = new URLSearchParams(params);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const notify = vi.fn();
  render(<QueryClientProvider client={client}><DataSourcesPage notify={notify} /></QueryClientProvider>);
  return { notify };
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  nav.replace.mockReset();
});

describe("Settings Data Sources", () => {
  it("lists configured sources and opens the Kite configuration panel", async () => {
    mockServer();
    renderPage("tab=data-sources");
    expect(await screen.findByRole("heading", { name: "Data sources" })).toBeInTheDocument();
    expect(await screen.findAllByText("Not configured")).not.toHaveLength(0);
    expect(screen.getByRole("region", { name: "Configure Zerodha Kite Connect" })).toBeInTheDocument();
    expect(await screen.findByLabelText("API key")).toBeInTheDocument();
    expect(screen.getByLabelText("API secret")).toHaveAttribute("type", "password");
    expect(screen.getByLabelText("Kite app redirect URL")).toHaveValue("http://127.0.0.1:8080/exporter/kite/callback");

    fireEvent.click(screen.getByRole("button", { name: /Prometheus/ }));
    expect(nav.replace).toHaveBeenCalledWith("/settings?tab=data-sources&source=prometheus", { scroll: false });
  });

  it("tests and saves credentials, then reports that Kite sign-in is required", async () => {
    const calls = mockServer();
    const { notify } = renderPage("tab=data-sources&source=kite");
    const panel = await screen.findByRole("region", { name: "Configure Zerodha Kite Connect" });
    fireEvent.change(await within(panel).findByLabelText("API key"), { target: { value: "abcd12345678" } });
    fireEvent.change(within(panel).getByLabelText("API secret"), { target: { value: "s3cretvalue42" } });
    fireEvent.click(within(panel).getByRole("button", { name: "Save credentials" }));

    expect(await within(panel).findByText(/Configuration saved. Sign in to Kite to test/)).toBeInTheDocument();
    expect(calls.find((c) => c.url === "/api/v1/kite/setup" && c.method === "PUT")?.body).toBe(JSON.stringify({ apiKey: "abcd12345678", apiSecret: "s3cretvalue42" }));
    expect(await within(panel).findByRole("link", { name: /Test with Kite/ })).toHaveAttribute("href", "/exporter/login");
    expect(calls.every((c) => !c.body?.includes("s3cretvalue42") || c.url === "/api/v1/kite/setup")).toBe(true);
    expect(notify).toHaveBeenCalledWith(true, "Kite configuration saved. Sign in to test it.");

    fireEvent.click(within(panel).getByRole("button", { name: "Refresh status" }));
    expect(await within(panel).findByText(/Credentials are saved. Sign in to Kite to test/)).toBeInTheDocument();
  });

  it("shows a read-only state for deployment-managed credentials", async () => {
    mockServer({ configured: true, source: "flags", editable: false, readOnlyReason: "managed", apiKeyHint: "••••5678", sessionValid: true, userId: "AB1234" });
    renderPage("tab=data-sources&source=kite");
    const panel = await screen.findByRole("region", { name: "Configure Zerodha Kite Connect" });
    expect(await within(panel).findByText("Connected")).toBeInTheDocument();
    expect(within(panel).getByText(/set by the deployment/)).toBeInTheDocument();
    expect(within(panel).queryByLabelText("API key")).toBeNull();
    expect(within(panel).queryByRole("button", { name: "Disconnect" })).toBeNull();
  });

  it("confirms before disconnecting the saved data source", async () => {
    const calls = mockServer({ configured: true, source: "file", editable: true, apiKeyHint: "••••5678" });
    renderPage("tab=data-sources&source=kite");
    const panel = await screen.findByRole("region", { name: "Configure Zerodha Kite Connect" });
    fireEvent.click(await within(panel).findByRole("button", { name: "Disconnect" }));
    const dialog = await screen.findByRole("dialog", { name: "Disconnect Zerodha?" });
    expect(calls.some((c) => c.method === "DELETE")).toBe(false);
    fireEvent.click(within(dialog).getByRole("button", { name: "Disconnect" }));
    await waitFor(() => expect(calls.some((c) => c.url === "/api/v1/kite/setup" && c.method === "DELETE")).toBe(true));
    expect(await within(panel).findByLabelText("API key")).toBeInTheDocument();
  });
});
