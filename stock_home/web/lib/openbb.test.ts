import { afterEach, describe, expect, it, vi } from "vitest";
import { CommandError, fetchCommand, isNotConfigured, rangeStart, safeUrl } from "./openbb";

afterEach(() => vi.unstubAllGlobals());

function stubFetch(status: number, body: unknown) {
  const fn = vi.fn(async () => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fn);
  return fn;
}

describe("fetchCommand", () => {
  it("drops empty params", async () => {
    const fn = stubFetch(200, { results: [] });
    await fetchCommand("equity/price/quote", { symbol: "A B", provider: "", limit: 5, x: undefined });
    expect(fn).toHaveBeenCalledWith("/api/v2/equity/price/quote?symbol=A+B&limit=5", expect.anything());
  });

  it.each([
    [{ error: "unknown command" }, 404, "unknown command"],
    [{ detail: "Provider error" }, 502, "Provider error"],
    [{ detail: [{ msg: "a" }, { msg: "b" }] }, 422, "a; b"],
    [{}, 500, "HTTP 500"],
  ])("maps %j to an error", async (body, status, msg) => {
    stubFetch(status, body);
    await expect(fetchCommand("x", {})).rejects.toMatchObject({ message: msg, status });
  });
});

describe("helpers", () => {
  it("detects a missing OpenBB upstream", () => {
    expect(isNotConfigured(new CommandError("unknown command", 404))).toBe(true);
    expect(isNotConfigured(new CommandError("Not Found", 404))).toBe(false);
  });

  it("computes range starts", () => {
    expect(rangeStart("1M", new Date("2026-10-07T12:00:00Z"))).toBe("2026-09-07");
    expect(rangeStart("5Y", new Date("2026-10-07T12:00:00Z"))).toBe("2021-10-07");
  });

  it("only allows http(s) links", () => {
    expect(safeUrl("https://example.com/a")).toBe("https://example.com/a");
    expect(safeUrl("javascript:alert(1)")).toBeUndefined();
    expect(safeUrl("data:text/html,x")).toBeUndefined();
    expect(safeUrl("not a url")).toBeUndefined();
    expect(safeUrl(null)).toBeUndefined();
  });
});
