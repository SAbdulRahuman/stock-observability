import { useQuery } from "@tanstack/react-query";
import { z } from "zod";

const labels = z.record(z.string(), z.string());

const target = z.object({
  labels,
  scrapePool: z.string(),
  scrapeUrl: z.string(),
  health: z.string(),
  lastError: z.string(),
  lastScrape: z.string(),
  lastScrapeDuration: z.number(),
  scrapeInterval: z.string().optional(),
  scrapeTimeout: z.string().optional(),
});

const promAlert = z.object({
  labels,
  annotations: labels.optional().default({}),
  state: z.string(),
  activeAt: z.string().nullish(),
  value: z.string().optional(),
});

const rule = z.object({
  type: z.enum(["alerting", "recording"]),
  name: z.string(),
  query: z.string(),
  health: z.string(),
  lastError: z.string().optional(),
  state: z.string().optional(),
  labels: labels.nullish(),
  annotations: labels.nullish(),
  alerts: z.array(promAlert).nullish(),
  lastEvaluation: z.string().optional(),
  evaluationTime: z.number().optional(),
  duration: z.number().optional(),
});

const sample = z.tuple([z.number(), z.string()]);

export const promSchemas = {
  targets: z.object({ activeTargets: z.array(target) }),
  rules: z.object({
    groups: z.array(
      z.object({
        name: z.string(),
        interval: z.number().optional(),
        lastEvaluation: z.string().optional(),
        evaluationTime: z.number().optional(),
        rules: z.array(rule),
      }),
    ),
  }),
  alerts: z.object({ alerts: z.array(promAlert) }),
  query: z.discriminatedUnion("resultType", [
    z.object({ resultType: z.literal("vector"), result: z.array(z.object({ metric: labels, value: sample })) }),
    z.object({ resultType: z.literal("matrix"), result: z.array(z.object({ metric: labels, values: z.array(sample) })) }),
    z.object({ resultType: z.literal("scalar"), result: sample }),
    z.object({ resultType: z.literal("string"), result: sample }),
  ]),
  names: z.array(z.string()),
};

export type PromTarget = z.infer<typeof target>;
export type PromRule = z.infer<typeof rule>;
export type PromAlert = z.infer<typeof promAlert>;
export type PromQueryData = z.infer<typeof promSchemas.query>;
export type PromStatus = { ready: boolean; healthy: boolean; build: Record<string, string>; runtime: Record<string, unknown> };
export type PromEnvelope<T> = { data: T; warnings: string[]; durationMs: number; updatedAt: string };

/** Fetches one stock_home Prometheus endpoint; the browser never talks to Prometheus directly. */
export async function fetchProm<T>(path: string, schema: { parse: (d: unknown) => T }, params?: Record<string, string>, signal?: AbortSignal): Promise<PromEnvelope<T>> {
  const qs = params ? `?${new URLSearchParams(params)}` : "";
  const res = await fetch(`/api/v1/prom/${path}${qs}`, { signal, cache: "no-store", headers: { Accept: "application/json" } });
  const body = (await res.json().catch(() => ({}))) as { data?: { data: unknown; warnings?: string[]; durationMs?: number }; error?: string; updatedAt?: string };
  if (!res.ok || !body.data) throw new Error(body.error ?? `HTTP ${res.status}`);
  return { data: schema.parse(body.data.data), warnings: body.data.warnings ?? [], durationMs: body.data.durationMs ?? 0, updatedAt: body.updatedAt ?? "" };
}

export function usePromSection<K extends "targets" | "rules" | "alerts">(name: K, enabled = true) {
  return useQuery({
    queryKey: ["prom", name],
    queryFn: ({ signal }) => fetchProm(name, { parse: (d) => promSchemas[name].parse(d) as z.infer<(typeof promSchemas)[K]> }, undefined, signal),
    refetchInterval: 15_000,
    enabled,
  });
}

export function usePromStatus() {
  return useQuery({
    queryKey: ["prom", "status"],
    queryFn: async ({ signal }) => {
      const res = await fetch("/api/v1/prom/status", { signal, cache: "no-store", headers: { Accept: "application/json" } });
      const body = (await res.json().catch(() => ({}))) as { data?: PromStatus; error?: string; updatedAt?: string };
      if (!res.ok || !body.data) throw new Error(body.error ?? `HTTP ${res.status}`);
      return { ...body.data, updatedAt: body.updatedAt ?? "" };
    },
    refetchInterval: 30_000,
  });
}

export function useMetricNames(enabled: boolean) {
  return useQuery({
    queryKey: ["prom", "metric-names"],
    queryFn: ({ signal }) => fetchProm("metric-names", promSchemas.names, undefined, signal),
    staleTime: 5 * 60_000,
    enabled,
  });
}

/** Example PromQL for metrics this stack's exporter and rules are known to publish. */
export const exampleQueries = [
  { label: "Scrape health by job", query: "up" },
  { label: "Last price of NIFTY 50", query: 'stock_last_price{symbol="NIFTY 50"}' },
  { label: "Top 5 core gainers", query: 'topk(5, stock_change_percent{instrument_type="EQ", tier="core"})' },
  { label: "Advancing equities", query: "market:advancers:count" },
  { label: "Feed ticks per second", query: "sum(rate(stock_feed_ticks_total[1m]))" },
];
