import { describe, expect, it } from "vitest";
import { resolveDocHref } from "./links";

const files = new Map([
  ["content/docs/quickstart.mdx", "quickstart"],
  ["../stock-observability-proposal/03-SYSTEM-ARCHITECTURE.md", "architecture"],
]);
const from = "../stock-observability-proposal/01-PRODUCT-MASTER-PLAN.md";

describe("resolveDocHref", () => {
  it("maps published repo files to doc pages", () => {
    expect(resolveDocHref("03-SYSTEM-ARCHITECTURE.md#2-components", from, files)).toEqual({ href: "/docs/architecture#2-components", external: false });
    expect(resolveDocHref("./03-SYSTEM-ARCHITECTURE.md", from, files)?.href).toBe("/docs/architecture");
    expect(resolveDocHref("../website/content/docs/quickstart.mdx", from, files)).toBeNull();
    expect(resolveDocHref("quickstart.mdx", "content/docs/x.mdx", files)?.href).toBe("/docs/quickstart");
  });

  it("keeps anchors, site paths and external links", () => {
    expect(resolveDocHref("#vision", from, files)).toEqual({ href: "#vision", external: false });
    expect(resolveDocHref("/demo", from, files)).toEqual({ href: "/demo", external: false });
    expect(resolveDocHref("https://prometheus.io", from, files)).toEqual({ href: "https://prometheus.io", external: true });
    expect(resolveDocHref("mailto:a@example.com", from, files)?.external).toBe(true);
  });

  it("drops repo-only paths and unsafe schemes", () => {
    expect(resolveDocHref("../deploy/README.md", from, files)).toBeNull();
    expect(resolveDocHref("//evil.example/x", from, files)).toBeNull();
    expect(resolveDocHref("javascript:alert(1)", from, files)).toBeNull();
    expect(resolveDocHref("JavaScript:alert(1)", from, files)).toBeNull();
  });
});
