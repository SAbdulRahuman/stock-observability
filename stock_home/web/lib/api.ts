import { keepPreviousData, useQuery } from "@tanstack/react-query";
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

const timelineItem = z.object({
  name: z.string(),
  severity: z.string(),
  category: z.string(),
  subject: z.string().optional(),
  symbol: z.string().optional(),
  start: z.string(),
  end: z.string().nullish(),
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
    items: z.array(timelineItem),
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
    quoteSymbolsMax: z.number().optional(),
    silencesAllowed: z.boolean().optional(),
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

export const chartRanges = ["1D", "1W", "1M", "3M", "1Y", "5Y"] as const;
export type ChartRange = (typeof chartRanges)[number];
/** Index charts on Home and Market Overview stop at 1Y. */
export const indexRanges: readonly ChartRange[] = ["1D", "1W", "1M", "3M", "1Y"];

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
export function useChart(symbol: string, range: ChartRange, enabled = true) {
  return useQuery({
    queryKey: ["chart", symbol, range],
    queryFn: ({ signal }) =>
      fetchEnvelope(`/api/v1/chart?symbol=${encodeURIComponent(symbol)}&range=${range}`, chartSchema, signal),
    enabled,
    refetchInterval: range === "1D" ? 30_000 : 120_000,
  });
}

const quotesSchema = z.object({ items: z.array(quote) });
export type Quote = z.infer<typeof quote>;

const instrumentSchema = quote.extend({
  open: num,
  high: num,
  low: num,
  prevClose: num,
  avgVolume20d: num,
  chartUrl: z.string(),
});
export type Instrument = z.infer<typeof instrumentSchema>;

export function useInstrument(symbol: string | null) {
  return useQuery({
    queryKey: ["instrument", symbol],
    queryFn: ({ signal }) => fetchEnvelope(`/api/v1/instrument?symbol=${encodeURIComponent(symbol ?? "")}`, instrumentSchema, signal),
    enabled: !!symbol,
    refetchInterval: 10_000,
  });
}

export function useQuotes(symbols: string[]) {
  const list = symbols.join(",");
  return useQuery({
    queryKey: ["quotes", list],
    queryFn: ({ signal }) => fetchEnvelope(`/api/v1/quotes?symbols=${encodeURIComponent(list)}`, quotesSchema, signal),
    enabled: symbols.length > 0,
    refetchInterval: 10_000,
  });
}

export async function fetchQuotes(symbols: string[]): Promise<Quote[]> {
  const env = await fetchEnvelope(`/api/v1/quotes?symbols=${encodeURIComponent(symbols.join(","))}`, quotesSchema);
  return env.data.items;
}

const alertRow = z.object({
  key: z.string(),
  fingerprint: z.string().optional(),
  name: z.string(),
  severity: z.string(),
  category: z.string().optional(),
  summary: z.string().optional(),
  description: z.string().optional(),
  symbol: z.string().optional(),
  exchange: z.string().optional(),
  value: z.string().optional(),
  threshold: z.string().optional(),
  startsAt: z.string(),
  endsAt: z.string().nullish(),
  status: z.enum(["firing", "acknowledged", "silenced", "inhibited", "resolved"]),
  silencedUntil: z.string().nullish(),
  source: z.string(),
  sourceUrl: z.string().optional(),
  exploreUrl: z.string().optional(),
});
const alertListSchema = z.object({
  items: z.array(alertRow),
  timeline: z.array(timelineItem),
  from: z.string(),
  truncated: z.boolean(),
});
export type AlertRow = z.infer<typeof alertRow>;
export type AlertList = z.infer<typeof alertListSchema>;
export type TimelineItem = z.infer<typeof timelineItem>;

export const ALERT_LIST_KEY = ["alert-list"];

/** Active alerts (incl. silenced) plus alerts resolved in the last 24h. */
export function useAlertList() {
  return useQuery({
    queryKey: ALERT_LIST_KEY,
    queryFn: ({ signal }) => fetchEnvelope("/api/v1/alerts/items", alertListSchema, signal),
    refetchInterval: 5_000,
  });
}

export type SilenceKind = "acknowledge" | "silence";
export const silenceDurations = ["30m", "1h", "2h", "4h", "24h"] as const;
export type SilenceResult = { created: { fingerprint: string; silenceId: string }[]; failed: { fingerprint: string; error: string }[] };

/** Creates Alertmanager silences through stock_home; the server derives matchers from each alert. */
export async function createSilences(req: { fingerprints: string[]; kind: SilenceKind; duration: string; comment: string }): Promise<SilenceResult> {
  const res = await fetch("/api/v1/silences", {
    method: "POST",
    cache: "no-store",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(req),
  });
  const body = (await res.json().catch(() => ({}))) as { data?: SilenceResult; error?: string };
  if (body.data) return body.data;
  throw new Error(body.error ?? `HTTP ${res.status}`);
}

const rangeItem = quote.extend({ high52: z.number(), low52: z.number(), distancePercent: z.number(), position: z.number() });
const moversScanSchema = z.object({
  universe: z.string(),
  sector: z.string(),
  price: z.string(),
  total: z.number(),
  advancers: z.number(),
  decliners: z.number(),
  unchanged: z.number(),
  gainers: z.array(quote),
  losers: z.array(quote),
  mostActive: z.array(quote),
  volumeChange: z.array(quote.extend({ relativeVolume: z.number() })),
  high52: z.array(rangeItem),
  low52: z.array(rangeItem),
  distribution: z.array(z.object({ label: z.string(), from: num, to: num, count: z.number() })),
  sectors: z.array(z.object({ name: z.string(), changePercent: z.number(), count: z.number(), gainer: quote.nullish(), loser: quote.nullish() })),
  sectorNames: z.array(z.string()),
});
export type MoversScan = z.infer<typeof moversScanSchema>;
export type RangeItem = z.infer<typeof rangeItem>;
export type MoversFilters = { universe: string; sector: string; price: string };

export function useMoversScan(f: MoversFilters) {
  const qs = new URLSearchParams({ universe: f.universe, price: f.price, ...(f.sector ? { sector: f.sector } : {}) });
  return useQuery({
    queryKey: ["movers-scan", f.universe, f.sector, f.price],
    queryFn: ({ signal }) => fetchEnvelope(`/api/v1/movers/scan?${qs}`, moversScanSchema, signal),
    refetchInterval: 10_000,
    placeholderData: keepPreviousData,
  });
}
