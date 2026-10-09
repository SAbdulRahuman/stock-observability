import { useQuery } from "@tanstack/react-query";
import { z } from "zod";

const num = z.number().nullish();
const point = z.object({ t: z.number(), v: z.number() });
const link = z.object({ label: z.string(), href: z.string(), description: z.string().optional(), id: z.string().optional() });
const level = z.enum(["critical", "warning", "info"]);

const quote = z.object({
  exchange: z.string(),
  symbol: z.string(),
  sector: z.string().optional(),
  price: num,
  changePercent: num,
  exploreUrl: z.string(),
  change: num,
  volume: num,
  spark: z.array(point).nullish(),
});

const schemas = {
  status: z.object({
    market: z.object({
      phase: z.string(),
      open: z.boolean(),
      tradingDay: z.boolean(),
      secondsToOpen: num,
      secondsToClose: num,
      timezone: z.string(),
      now: z.string(),
      nextHoliday: z.object({ date: z.string(), name: z.string() }).nullish(),
      feedSource: z.string(),
    }),
    banners: z.array(z.object({ level, title: z.string(), message: z.string(), action: link.optional() })),
  }),
  feed: z.object({
    source: z.string(),
    mode: z.string(),
    sessionValid: z.boolean(),
    sessionExpiresAt: z.string().nullish(),
    connections: num,
    subscribed: num,
    lastTickAgeSeconds: num,
    ticksPerSecond: num,
    reconnects1h: num,
    errors1h: num,
    staleRatio: num,
    apiOutcomes10m: z.record(z.string(), z.number()),
    tickRateSpark: z.array(point).nullish(),
    exploreUrl: z.string(),
  }),
  alerts: z.object({
    bySeverity: z.record(z.string(), z.number()),
    byCategory: z.record(z.string(), z.number()),
    silenced: z.number(),
    inhibited: z.number(),
    activeSilences: z.number(),
    items: z.array(
      z.object({
        name: z.string(),
        severity: z.string(),
        category: z.string(),
        summary: z.string(),
        symbol: z.string().optional(),
        exchange: z.string().optional(),
        startsAt: z.string(),
        sourceUrl: z.string().optional(),
        fingerprint: z.string(),
        value: z.string().optional(),
      }),
    ),
    truncated: z.boolean(),
  }),
  market: z.object({
    indices: z.array(quote),
    advancers: num,
    decliners: num,
    unchanged: num,
    advanceDecline: num,
    sectors: z.array(z.object({ name: z.string(), changePercent: z.number(), constituents: z.number().optional() })),
    spark: z.object({ symbol: z.string(), points: z.array(point).nullish() }).nullish(),
  }),
  movers: z.object({ gainers: z.array(quote), losers: z.array(quote), highVolume: z.array(quote).default([]) }),
  circuits: z.object({
    hits: z.array(quote.extend({ side: z.string(), distancePercent: z.number(), hit: z.boolean() })),
    near: z.array(quote.extend({ side: z.string(), distancePercent: z.number(), hit: z.boolean() })),
    thresholdPercent: z.number(),
  }),
  portfolio: z.object({
    enabled: z.boolean(),
    holdingValue: num,
    dayChange: num,
    dayChangePercent: num,
    holdingPnl: num,
    positionPnl: num,
    marginUsed: num,
    marginTotal: num,
    updatedAt: z.string().nullish(),
    topHoldings: z.array(
      z.object({
        exchange: z.string(),
        symbol: z.string(),
        quantity: z.number(),
        value: z.number(),
        pnl: z.number(),
        dayChange: z.number(),
      }),
    ),
  }),
  system: z.object({
    components: z.array(z.object({ name: z.string(), healthy: z.boolean(), latencyMs: z.number(), href: z.string() })),
    targets: z.array(
      z.object({
        job: z.string(),
        up: z.boolean(),
        durationSeconds: num,
        timeoutSeconds: num,
        sampleUsage: num,
      }),
    ),
    collectors: z.record(z.string(), z.boolean()),
    headSeries: num,
  }),
  timeline: z.object({
    from: z.string(),
    items: z.array(
      z.object({
        name: z.string(),
        severity: z.string(),
        category: z.string(),
        subject: z.string().optional(),
        symbol: z.string().optional(),
        start: z.string(),
        end: z.string().nullish(),
      }),
    ),
    bySeverity: z.record(z.string(), z.number()),
    truncated: z.boolean(),
  }),
  links: z.array(link),
  info: z.object({
    version: z.string(),
    exchange: z.string(),
    sessionOpen: z.string(),
    sessionClose: z.string(),
    timezone: z.string(),
  }),
};

/** Poll intervals match the server-side cache TTLs. */
const refreshMs: Record<SectionName, number | false> = {
  status: 5_000,
  feed: 5_000,
  alerts: 5_000,
  market: 10_000,
  movers: 10_000,
  circuits: 10_000,
  portfolio: 30_000,
  system: 15_000,
  timeline: 60_000,
  links: false,
  info: false,
};

export type SectionName = keyof typeof schemas;
export type SectionData<K extends SectionName> = z.infer<(typeof schemas)[K]>;
export type Envelope<T> = { data: T; updatedAt: string; stale: boolean };

export type Status = SectionData<"status">;
export type Banner = Status["banners"][number];

export async function fetchSection<K extends SectionName>(name: K, signal?: AbortSignal): Promise<Envelope<SectionData<K>>> {
  return fetchEnvelope(`/api/v1/${name}`, { parse: (d) => schemas[name].parse(d) as SectionData<K> }, signal);
}

async function fetchEnvelope<T>(url: string, schema: { parse: (data: unknown) => T }, signal?: AbortSignal): Promise<Envelope<T>> {
  const res = await fetch(url, { signal, cache: "no-store", headers: { Accept: "application/json" } });
  const body: unknown = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = (body as { error?: string }).error ?? `HTTP ${res.status}`;
    throw new Error(msg);
  }
  const env = body as { data: unknown; updatedAt: string; stale?: boolean };
  return { data: schema.parse(env.data), updatedAt: env.updatedAt, stale: env.stale ?? false };
}

export function useSection<K extends SectionName>(name: K) {
  return useQuery({
    queryKey: ["section", name],
    queryFn: ({ signal }) => fetchSection(name, signal),
    refetchInterval: refreshMs[name],
    staleTime: refreshMs[name] === false ? Number.POSITIVE_INFINITY : 0,
  });
}

export const chartRanges = ["1D", "1W", "1M", "3M", "1Y"] as const;
export type ChartRange = (typeof chartRanges)[number];

const chartSchema = z.object({
  symbol: z.string(),
  range: z.string(),
  points: z.array(point),
  volume: z.array(point),
  volumeLabel: z.string(),
  last: num,
});
export type Chart = z.infer<typeof chartSchema>;

/** Price history of a configured index; the server keeps only session-time points. */
export function useChart(symbol: string, range: ChartRange) {
  return useQuery({
    queryKey: ["chart", symbol, range],
    queryFn: ({ signal }) =>
      fetchEnvelope(`/api/v1/chart?symbol=${encodeURIComponent(symbol)}&range=${range}`, chartSchema, signal),
    refetchInterval: range === "1D" ? 30_000 : 120_000,
  });
}

const quotesSchema = z.object({ items: z.array(quote) });
export type Quote = z.infer<typeof quote>;

export function useQuotes(symbols: string[]) {
  const list = symbols.join(",");
  return useQuery({
    queryKey: ["quotes", list],
    queryFn: ({ signal }) => fetchEnvelope(`/api/v1/quotes?symbols=${encodeURIComponent(list)}`, quotesSchema, signal),
    enabled: symbols.length > 0,
    refetchInterval: 10_000,
  });
}
