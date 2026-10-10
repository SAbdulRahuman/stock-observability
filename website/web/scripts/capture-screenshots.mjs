// Captures the showcase screenshots from a running stock_home (fed by the simulator).
// Usage: HOME_URL=http://localhost:8081 npm run screenshots
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import sharp from "sharp";

const base = (process.env.HOME_URL ?? "http://localhost:8080").replace(/\/$/, "");
const warmup = Number(process.env.WARMUP_SECONDS ?? 300);
const here = (p) => fileURLToPath(new URL(p, import.meta.url));
const screens = JSON.parse(await readFile(here("../lib/screens.json"), "utf8"));

// Charts and movers stay empty until Prometheus has a few minutes of samples.
async function waitForData() {
  const deadline = Date.now() + warmup * 1000;
  for (;;) {
    try {
      const res = await fetch(`${base}/api/v1/movers`);
      const body = await res.json();
      if (res.ok && body.data?.gainers?.length > 0) return;
    } catch {}
    if (Date.now() > deadline) throw new Error(`no market data at ${base} after ${warmup}s; is the simulator stack running?`);
    await new Promise((r) => setTimeout(r, 5000));
  }
}

await waitForData();
const browser = await chromium.launch();
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1.5, colorScheme: "dark" });
await ctx.addInitScript(() => localStorage.setItem("theme", "dark"));
const page = await ctx.newPage();
for (const s of screens) {
  await page.goto(base + s.path, { waitUntil: "networkidle" });
  await page.waitForTimeout(Number(process.env.SETTLE_MS ?? 2500));
  const png = await page.screenshot({ type: "png" });
  const out = here(`../public/screenshots/${s.id}.webp`);
  await sharp(png).resize({ width: 1600, height: 1000, fit: "cover", position: "top" }).webp({ quality: 74 }).toFile(out);
  console.log(`${s.id}: ${base}${s.path} -> public/screenshots/${s.id}.webp`);
}
await browser.close();
