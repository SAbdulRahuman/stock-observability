import { useQuery } from "@tanstack/react-query";
import { z } from "zod";

const num = z.number().nullish();
const str = z.string().nullish();

const warning = z.object({ category: str, message: z.string() });

/** OpenBB's OBBject envelope with a list of `item` results. */
export function obbject<T extends z.ZodType>(item: T) {
  return z.object({
    results: z.array(item).nullish(),
    provider: str,
    warnings: z.array(warning).nullish(),
  });
}

export const schemas = {
  search: z.object({ symbol: str, name: str, exchange: str }),
  quote: z.object({
    symbol: z.string(),
    name: str,
    exchange: str,
    currency: str,
    last_price: num,
    open: num,
    high: num,
    low: num,
    prev_close: num,
    change: num,
    // Normalized: 0.0123 means 1.23%.
    change_percent: num,
    volume: num,
    year_high: num,
    year_low: num,
  }),
  historical: z.object({
    date: z.string(),
    open: z.number(),
    high: z.number(),
    low: z.number(),
    close: z.number(),
    volume: num,
  }),
  profile: z.object({
    symbol: z.string(),
    name: str,
    stock_exchange: str,
    sector: str,
    industry_category: str,
    employees: num,
    hq_country: str,
    company_url: str,
    short_description: str,
    long_description: str,
  }),
  news: z.object({
    date: z.string(),
    title: z.string(),
    url: z.string(),
    source: str,
    author: str,
    excerpt: str,
  }),
};

export type Quote = z.infer<typeof schemas.quote>;
export type Bar = z.infer<typeof schemas.historical>;
export type Profile = z.infer<typeof schemas.profile>;
export type NewsItem = z.infer<typeof schemas.news>;
export type SearchHit = z.infer<typeof schemas.search>;

export const PROVIDERS = ["yfinance", "cboe", "nasdaq", "tmx", "sec"] as const;
export const RANGES = { "1M": 1, "6M": 6, "1Y": 12, "5Y": 60 } as const;
export type Range = keyof typeof RANGES;

export const DEFAULTS = { symbol: "RELIANCE.NS", provider: "yfinance", range: "1Y" as Range };

export class CommandError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
  }
}

/** stock_home answers 404 "unknown command" when no --openbb.url is configured. */
export function isNotConfigured(err: unknown): boolean {
  return err instanceof CommandError && err.status === 404 && err.message === "unknown command";
}

function errorMessage(body: unknown, status: number): string {
  const b = body as { error?: unknown; detail?: unknown };
  if (typeof b.error === "string") return b.error;
  if (typeof b.detail === "string") return b.detail;
  if (Array.isArray(b.detail)) {
    const msgs = b.detail.map((d: { msg?: unknown }) => d?.msg).filter((m): m is string => typeof m === "string");
    if (msgs.length) return msgs.join("; ");
  }
  return `HTTP ${status}`;
}

export type Params = Record<string, string | number | undefined>;

export async function fetchCommand(path: string, params: Params, signal?: AbortSignal): Promise<unknown> {
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") qs.set(k, String(v));
  }
  const res = await fetch(`/api/v2/${path}?${qs}`, { signal, cache: "no-store", headers: { Accept: "application/json" } });
  const body: unknown = await res.json().catch(() => ({}));
  if (!res.ok) throw new CommandError(errorMessage(body, res.status), res.status);
  return body;
}

export function useCommand<T extends z.ZodType>(
  path: string,
  params: Params,
  item: T,
  opts: { enabled?: boolean; refetchInterval?: number } = {},
) {
  return useQuery({
    queryKey: ["openbb", path, params],
    queryFn: async ({ signal }) => obbject(item).parse(await fetchCommand(path, params, signal)),
    enabled: opts.enabled ?? true,
    refetchInterval: opts.refetchInterval ?? false,
    staleTime: 60_000,
    retry: (count, err) => !(err instanceof CommandError && err.status < 500) && count < 1,
  });
}

/** First day of the range as YYYY-MM-DD. */
export function rangeStart(range: Range, now = new Date()): string {
  const d = new Date(now);
  d.setMonth(d.getMonth() - RANGES[range]);
  return d.toISOString().slice(0, 10);
}

/** Only http(s) links from upstream data are rendered. */
export function safeUrl(u: string | null | undefined): string | undefined {
  if (!u) return undefined;
  try {
    const url = new URL(u);
    return url.protocol === "https:" || url.protocol === "http:" ? url.href : undefined;
  } catch {
    return undefined;
  }
}
