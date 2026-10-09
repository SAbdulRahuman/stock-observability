import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AlertmanagerPage } from "./alertmanager-page";

const nav = vi.hoisted(() => ({ params: new URLSearchParams(), replace: vi.fn() }));
vi.mock("next/navigation", () => ({
  useSearchParams: () => nav.params,
  useRouter: () => ({ replace: nav.replace }),
  usePathname: () => "/observability/alertmanager",
}));

const env = (data: unknown) => ({ data, updatedAt: "2026-10-08T04:12:00Z", stale: false });
const alert = (fingerprint: string, labels: Record<string, string>, extra = {}) => ({
  fingerprint,
  labels,
  annotations: { summary: `${labels.alertname} summary`, runbook_url: "https://runbooks.example/price" },
  startsAt: "2026-10-08T04:00:00Z",
  endsAt: "2026-10-08T05:00:00Z",
  state: "active",
  silencedBy: [],
  inhibitedBy: [],
  generatorUrl: "javascript:alert(1)",
  ...extra,
});
const reliance = alert("aaa1", { alertname: "PriceAboveR1", severity: "critical", symbol: "RELIANCE" });
const routes: Record<string, unknown> = {
  "/api/v1/links": { data: [{ id: "alertmanager", label: "Alertmanager", href: "/alertmanager/" }], updatedAt: "" },
  "/api/v1/am/groups": env({
    groups: [
      { labels: { alertname: "PriceAboveR1" }, receiver: "critical-pager", alerts: [reliance] },
      {
        labels: { alertname: "VolumeSpike" },
        receiver: "default",
        alerts: [alert("bbb2", { alertname: "VolumeSpike", severity: "warning", symbol: "INFY" }, { state: "suppressed", silencedBy: ["s-1"] })],
      },
      // The same instance routed to a second receiver must not be double counted.
      { labels: { alertname: "PriceAboveR1" }, receiver: "default", alerts: [reliance] },
    ],
  }),
  "/api/v1/am/silences": env({
    silences: [
      {
        id: "11111111-2222-3333-4444-555555555555",
        matchers: [{ name: "alertname", value: "VolumeSpike", isRegex: false, isEqual: true }],
        createdBy: "ops",
        comment: "maintenance window",
        startsAt: "2026-10-08T03:00:00Z",
        endsAt: "2026-10-08T06:00:00Z",
        state: "active",
      },
      {
        id: "66666666-2222-3333-4444-555555555555",
        matchers: [{ name: "symbol", value: "TCS|INFY", isRegex: true, isEqual: true }],
        createdBy: "ops",
        comment: "results day",
        startsAt: "2026-10-09T03:00:00Z",
        endsAt: "2026-10-09T06:00:00Z",
        state: "pending",
      },
    ],
  }),
  "/api/v1/am/status": env({
    ready: true,
    version: { version: "0.28.1", revision: "abc" },
    uptime: "2026-10-08T00:00:00Z",
    cluster: "disabled",
    peers: 0,
    configAvailable: true,
    route: {
      receiver: "default",
      matchers: [],
      groupBy: ["alertname"],
      groupWait: "10s",
      continue: false,
      routes: [{ receiver: "critical-pager", matchers: ['severity="critical"'], groupBy: [], continue: true, routes: [] }],
    },
    receivers: [
      { name: "default", integrations: [] },
      { name: "critical-pager", integrations: [{ type: "webhook", count: 2 }] },
    ],
  }),
};

type Call = { url: string; method: string; body?: unknown };

function renderPage(params = "", opts: { ok?: boolean; allowed?: boolean; empty?: boolean } = {}) {
  const { ok = true, allowed = true, empty = false } = opts;
  nav.params = new URLSearchParams(params);
  const calls: Call[] = [];
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const u = new URL(url, "http://localhost");
    const method = init?.method ?? "GET";
    calls.push({ url: u.pathname, method, body: init?.body ? JSON.parse(String(init.body)) : undefined });
    if (u.pathname === "/api/v1/info") {
      return new Response(JSON.stringify(env({ version: "t", exchange: "NSE", sessionOpen: "09:15", sessionClose: "15:30", timezone: "Asia/Kolkata", silencesAllowed: allowed })));
    }
    if (!ok && u.pathname.startsWith("/api/v1/am/")) return new Response(JSON.stringify({ error: "alertmanager unreachable" }), { status: 502 });
    if (u.pathname === "/api/v1/am/silences" && method === "POST") {
      const b = JSON.parse(String(init?.body));
      return new Response(
        JSON.stringify({
          data: { silenceId: b.dryRun ? undefined : "new-id", matches: 1, sample: ["PriceAboveR1 RELIANCE"], broad: false, startsAt: "2026-10-08T04:12:00Z", endsAt: "2026-10-08T06:12:00Z" },
        }),
      );
    }
    if (u.pathname.startsWith("/api/v1/am/silences/") && method === "DELETE") return new Response(JSON.stringify({ data: { expired: true } }));
    if (empty && u.pathname === "/api/v1/am/groups") return new Response(JSON.stringify(env({ groups: [] })));
    if (empty && u.pathname === "/api/v1/am/silences") return new Response(JSON.stringify(env({ silences: [] })));
    const body = routes[u.pathname];
    return body ? new Response(JSON.stringify(body)) : new Response("{}", { status: 404 });
  });
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <AlertmanagerPage />
    </QueryClientProvider>,
  );
  return calls;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  nav.replace.mockReset();
});

const card = (label: string) => screen.getByLabelText(label, { selector: "section" });

describe("AlertmanagerPage", () => {
  it("shows summary cards with deduplicated alert counts and the group table", async () => {
    renderPage();
    expect(await screen.findByText("Ready")).toBeTruthy();
    await waitFor(() => expect(within(card("Alert Groups")).getByText("3")).toBeTruthy());
    expect(within(card("Active Alerts")).getByText("1")).toBeTruthy();
    expect(within(card("Silenced Alerts")).getByText("1")).toBeTruthy();
    expect(within(card("Pending Silences")).getByText("1")).toBeTruthy();
    expect(screen.getByRole("link", { name: /Open full Alertmanager UI/ }).getAttribute("href")).toBe("/alertmanager/");
    const table = screen.getByLabelText("Alert groups", { selector: "section" });
    expect(within(table).getByText("critical-pager")).toBeTruthy();
    expect(within(table).getAllByText("1 firing / 1 alert")).toHaveLength(2);
  });

  it("filters groups client-side and shows alert detail with safe links only", async () => {
    renderPage();
    await screen.findByLabelText("Alert groups", { selector: "section" });
    fireEvent.change(screen.getByLabelText("Status"), { target: { value: "silenced" } });
    const table = screen.getByLabelText("Alert groups", { selector: "section" });
    expect(within(table).queryByText("critical-pager")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Clear" }));
    fireEvent.change(screen.getByLabelText("Receiver"), { target: { value: "critical-pager" } });
    fireEvent.click(within(table).getByRole("button", { name: /Expand group/ }));
    fireEvent.click(within(table).getByRole("button", { name: "Details for PriceAboveR1" }));
    const detail = await screen.findByLabelText("PriceAboveR1 details");
    expect(within(detail).getByText("aaa1")).toBeTruthy();
    expect(within(detail).getByRole("link", { name: /Runbook/ }).getAttribute("href")).toBe("https://runbooks.example/price");
    expect(within(detail).queryByRole("link", { name: /Prometheus/ })).toBeNull();
  });

  it("creates a silence only after preview and confirm", async () => {
    const calls = renderPage();
    const table = await screen.findByLabelText("Alert groups", { selector: "section" });
    fireEvent.click(within(table).getAllByRole("button", { name: /Expand group/ })[0]);
    fireEvent.click(within(table).getByRole("button", { name: "Details for PriceAboveR1" }));
    const detail = await screen.findByLabelText("PriceAboveR1 details");
    await waitFor(() => expect((within(detail).getByRole("button", { name: "Create silence" }) as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(within(detail).getByRole("button", { name: "Create silence" }));
    const dialog = await screen.findByRole("dialog");
    expect((within(dialog).getByLabelText("Matcher 1 label") as HTMLInputElement).value).toBe("alertname");
    const preview = within(dialog).getByRole("button", { name: "Preview" }) as HTMLButtonElement;
    expect(preview.disabled).toBe(true);
    fireEvent.change(within(dialog).getByLabelText("Comment"), { target: { value: "known event" } });
    fireEvent.click(preview);
    expect(await within(dialog).findByText(/Silences 1 active alert/)).toBeTruthy();
    expect(calls.filter((c) => c.method === "POST").map((c) => (c.body as { dryRun: boolean }).dryRun)).toEqual([true]);
    fireEvent.click(within(dialog).getByRole("button", { name: "Confirm create silence" }));
    expect(await screen.findByText(/Silence new-id created/)).toBeTruthy();
    const posts = calls.filter((c) => c.method === "POST").map((c) => c.body as { dryRun: boolean; matchers: unknown[]; comment: string });
    expect(posts[1].dryRun).toBe(false);
    expect(posts[1].comment).toBe("known event");
    expect(posts[1].matchers).toContainEqual({ name: "symbol", op: "=", value: "RELIANCE" });
  });

  it("lists silences and expires one after confirmation", async () => {
    const calls = renderPage("tab=silences");
    expect(await screen.findByText('alertname="VolumeSpike"')).toBeTruthy();
    expect(screen.getByText('symbol=~"TCS|INFY"')).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Silence status"), { target: { value: "pending" } });
    expect(screen.queryByText('alertname="VolumeSpike"')).toBeNull();
    fireEvent.change(screen.getByLabelText("Silence status"), { target: { value: "all" } });
    const expire = screen.getByRole("button", { name: "Expire silence 11111111-2222-3333-4444-555555555555" }) as HTMLButtonElement;
    await waitFor(() => expect(expire.disabled).toBe(false));
    fireEvent.click(expire);
    const dialog = await screen.findByRole("dialog");
    expect(calls.some((c) => c.method === "DELETE")).toBe(false);
    fireEvent.click(within(dialog).getByRole("button", { name: "Expire silence" }));
    expect(await screen.findByText(/Silence 11111111-2222-3333-4444-555555555555 expired/)).toBeTruthy();
    expect(calls.find((c) => c.method === "DELETE")?.url).toBe("/api/v1/am/silences/11111111-2222-3333-4444-555555555555");
  });

  it("disables silence actions when the backend does not allow them", async () => {
    renderPage("tab=silences", { allowed: false });
    await screen.findByText('alertname="VolumeSpike"');
    expect((screen.getByRole("button", { name: "+ Create Silence" }) as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByRole("button", { name: /Expire silence 1111/ }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByText(/Silence actions are disabled/)).toBeTruthy();
  });

  it("shows the redacted route tree and status", async () => {
    renderPage("tab=routes");
    expect(await screen.findByText("webhook × 2")).toBeTruthy();
    expect(screen.getByText('severity="critical"')).toBeTruthy();
    expect(screen.getByText(/credentials are redacted/)).toBeTruthy();
    cleanup();
    renderPage("tab=status");
    expect(await screen.findByText("0.28.1")).toBeTruthy();
  });

  it("shows empty states", async () => {
    renderPage("", { empty: true });
    expect(await screen.findByText("Alertmanager has no alerts")).toBeTruthy();
    expect(within(card("Alert Groups")).getByText("0")).toBeTruthy();
  });

  it("shows unavailable states when Alertmanager fails", async () => {
    renderPage();
    cleanup();
    renderPage("", { ok: false });
    expect(await screen.findByText(/Unavailable — alertmanager unreachable/)).toBeTruthy();
    expect(within(card("Alert Groups")).getByText("unavailable")).toBeTruthy();
    expect(screen.getByText("Unreachable")).toBeTruthy();
  });
});
