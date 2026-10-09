import { useQuery } from "@tanstack/react-query";
import { z } from "zod";

const labels = z.record(z.string(), z.string());

const amAlert = z.object({
  fingerprint: z.string(),
  labels,
  annotations: labels,
  startsAt: z.string(),
  endsAt: z.string(),
  state: z.string(),
  silencedBy: z.array(z.string()),
  inhibitedBy: z.array(z.string()),
  generatorUrl: z.string().optional(),
});
const matcher = z.object({ name: z.string(), value: z.string(), isRegex: z.boolean(), isEqual: z.boolean().optional().default(true) });
export type RouteNode = {
  receiver: string;
  matchers: string[];
  groupBy: string[];
  groupWait?: string;
  groupInterval?: string;
  repeatInterval?: string;
  continue: boolean;
  routes: RouteNode[];
};
const routeNode: z.ZodType<RouteNode> = z.lazy(() =>
  z.object({
    receiver: z.string(),
    matchers: z.array(z.string()),
    groupBy: z.array(z.string()),
    groupWait: z.string().optional(),
    groupInterval: z.string().optional(),
    repeatInterval: z.string().optional(),
    continue: z.boolean(),
    routes: z.array(routeNode),
  }),
);

export const amSchemas = {
  groups: z.object({ groups: z.array(z.object({ labels, receiver: z.string(), alerts: z.array(amAlert) })) }),
  silences: z.object({
    silences: z.array(
      z.object({ id: z.string(), matchers: z.array(matcher), createdBy: z.string(), comment: z.string(), startsAt: z.string(), endsAt: z.string(), state: z.string() }),
    ),
  }),
  status: z.object({
    ready: z.boolean(),
    version: z.record(z.string(), z.string()),
    uptime: z.string().nullish(),
    cluster: z.string(),
    peers: z.number(),
    configAvailable: z.boolean(),
    route: routeNode.nullish(),
    receivers: z.array(z.object({ name: z.string(), integrations: z.array(z.object({ type: z.string(), count: z.number() })) })),
  }),
};

export type AMAlert = z.infer<typeof amAlert>;
export type AMGroup = z.infer<typeof amSchemas.groups>["groups"][number];
export type AMSilence = z.infer<typeof amSchemas.silences>["silences"][number];
export type AMMatcher = z.infer<typeof matcher>;
export type AMStatus = z.infer<typeof amSchemas.status>;
type Env<T> = { data: T; updatedAt: string; stale: boolean };

async function getAM<T>(path: string, schema: { parse: (d: unknown) => T }, signal?: AbortSignal): Promise<Env<T>> {
  const res = await fetch(`/api/v1/am/${path}`, { signal, cache: "no-store", headers: { Accept: "application/json" } });
  const body = (await res.json().catch(() => ({}))) as { data?: unknown; error?: string; updatedAt?: string; stale?: boolean };
  if (!res.ok) throw new Error(body.error ?? `HTTP ${res.status}`);
  return { data: schema.parse(body.data), updatedAt: body.updatedAt ?? "", stale: body.stale ?? false };
}

export function useAM<K extends keyof typeof amSchemas>(name: K, refetchMs = 10_000) {
  return useQuery({
    queryKey: ["am", name],
    queryFn: ({ signal }) => getAM(name, { parse: (d) => amSchemas[name].parse(d) as z.infer<(typeof amSchemas)[K]> }, signal),
    refetchInterval: refetchMs,
  });
}

export type MatcherInput = { name: string; op: "=" | "!=" | "=~" | "!~"; value: string };
export type SilencePreview = { silenceId?: string; matches: number; sample: string[]; broad: boolean; startsAt: string; endsAt: string };

/** Previews (dryRun) or creates a silence; the server validates matchers and times again. */
export async function submitSilence(body: { matchers: MatcherInput[]; startsAt: string; endsAt: string; comment: string; dryRun: boolean }): Promise<SilencePreview> {
  const res = await fetch("/api/v1/am/silences", {
    method: "POST",
    cache: "no-store",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(body),
  });
  const json = (await res.json().catch(() => ({}))) as { data?: SilencePreview; error?: string };
  if (!res.ok || !json.data) throw new Error(json.error ?? `HTTP ${res.status}`);
  return json.data;
}

export async function expireSilence(id: string): Promise<void> {
  const res = await fetch(`/api/v1/am/silences/${encodeURIComponent(id)}`, { method: "DELETE", cache: "no-store", headers: { Accept: "application/json" } });
  if (!res.ok) {
    const json = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(json.error ?? `HTTP ${res.status}`);
  }
}

export function matcherText(m: AMMatcher): string {
  const op = m.isRegex ? (m.isEqual ? "=~" : "!~") : m.isEqual ? "=" : "!=";
  return `${m.name}${op}"${m.value}"`;
}
