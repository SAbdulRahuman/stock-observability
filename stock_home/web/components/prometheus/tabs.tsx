"use client";

import type { UseQueryResult } from "@tanstack/react-query";
import { CircleAlert, X } from "lucide-react";
import { useState } from "react";
import { Empty } from "@/components/dashboard/section-card";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { formatDuration, relativeTime } from "@/lib/format";
import type { PromAlert, PromEnvelope, PromRule, PromTarget } from "@/lib/prom";
import { cn } from "@/lib/utils";

/** Loading and error handling for one Prometheus API panel. */
export function PromState<T>({ query, children }: { query: UseQueryResult<PromEnvelope<T>>; children: (d: T) => React.ReactNode }) {
  if (query.data) return <>{children(query.data.data)}</>;
  if (query.isError) {
    return (
      <p className="m-4 flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
        <CircleAlert className="size-4 text-warning" aria-hidden />
        Unavailable — {query.error.message}
      </p>
    );
  }
  return (
    <div className="space-y-2 p-4" aria-busy>
      <Skeleton className="h-6 w-full" />
      <Skeleton className="h-6 w-5/6" />
    </div>
  );
}

export function LabelList({ labels }: { labels: Record<string, string> | null | undefined }) {
  const entries = Object.entries(labels ?? {}).sort(([a], [b]) => a.localeCompare(b));
  if (entries.length === 0) return <span className="text-muted-foreground">—</span>;
  return (
    <span className="flex flex-wrap gap-1">
      {entries.map(([k, v]) => (
        <Badge key={k} variant="outline" className="max-w-full font-mono font-normal">
          <span className="truncate">
            {k}=&quot;{v}&quot;
          </span>
        </Badge>
      ))}
    </span>
  );
}

const stateVariant: Record<string, BadgeVariant> = { up: "ok", down: "critical", unknown: "default", firing: "critical", pending: "warning", inactive: "default", ok: "ok", err: "critical" };

export function StateBadge({ state }: { state: string }) {
  return <Badge variant={stateVariant[state] ?? "default"}>{state.toUpperCase()}</Badge>;
}

function Detail({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <Card aria-label={`${title} details`} className="min-w-0 p-4 text-sm">
      <div className="mb-3 flex items-center gap-2">
        <h3 className="min-w-0 flex-1 truncate font-semibold">{title}</h3>
        <button type="button" onClick={onClose} aria-label="Close details" className="rounded p-1 text-muted-foreground hover:text-foreground">
          <X className="size-4" />
        </button>
      </div>
      <dl className="space-y-3">{children}</dl>
    </Card>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 break-words">{children}</dd>
    </div>
  );
}

const select = "h-9 rounded-lg border bg-card px-2 text-sm";
const search = "h-9 min-w-48 flex-1 rounded-lg border bg-card px-3 text-sm";

export function TargetsTab({ targets, now }: { targets: PromTarget[]; now: Date }) {
  const [state, setState] = useState("all");
  const [term, setTerm] = useState("");
  const [sel, setSel] = useState<string | null>(null);
  const t = term.trim().toLowerCase();
  const rows = targets.filter(
    (x) => (state === "all" || x.health === state) && (!t || x.scrapePool.toLowerCase().includes(t) || (x.labels.instance ?? "").toLowerCase().includes(t)),
  );
  const key = (x: PromTarget) => `${x.scrapePool}|${x.scrapeUrl}`;
  const picked = targets.find((x) => key(x) === sel);
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <select aria-label="Target state" className={select} value={state} onChange={(e) => setState(e.target.value)}>
          <option value="all">All states</option>
          <option value="up">Up</option>
          <option value="down">Down</option>
          <option value="unknown">Unknown</option>
        </select>
        <input aria-label="Search targets" placeholder="Search job or instance…" className={search} value={term} onChange={(e) => setTerm(e.target.value)} />
      </div>
      <div className={cn("grid gap-4", picked && "xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]")}>
        <Card className="min-w-0 overflow-x-auto">
          {rows.length === 0 ? (
            <Empty>No targets match the current filter</Empty>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th className="py-2 pl-4 text-left font-normal">State</th>
                  <th className="py-2 text-left font-normal">Job</th>
                  <th className="py-2 text-left font-normal">Instance</th>
                  <th className="py-2 text-left font-normal">Last scrape</th>
                  <th className="py-2 pr-4 text-left font-normal">Duration / Error</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {rows.map((x) => (
                  <tr key={key(x)} className={cn("cursor-pointer hover:bg-accent/60", sel === key(x) && "bg-primary/15")} onClick={() => setSel(key(x))}>
                    <td className="py-2 pl-4">
                      <StateBadge state={x.health} />
                    </td>
                    <td className="py-2 font-medium">{x.scrapePool}</td>
                    <td className="py-2 font-mono text-xs">{x.labels.instance ?? "—"}</td>
                    <td className="py-2 text-muted-foreground">{x.lastScrape.startsWith("0001") ? "never" : relativeTime(x.lastScrape, now)}</td>
                    <td className={cn("max-w-72 truncate py-2 pr-4", x.lastError && "text-critical")} title={x.lastError || undefined}>
                      {x.lastError || `${(x.lastScrapeDuration * 1000).toFixed(1)} ms`}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
        {picked && (
          <Detail title={`${picked.scrapePool} · ${picked.labels.instance ?? ""}`} onClose={() => setSel(null)}>
            <Field label="Target labels">
              <LabelList labels={picked.labels} />
            </Field>
            <Field label="Scrape pool">{picked.scrapePool}</Field>
            <Field label="Endpoint">
              <span className="font-mono text-xs">{picked.scrapeUrl}</span>
            </Field>
            <Field label="Interval / timeout">
              {picked.scrapeInterval ?? "—"} / {picked.scrapeTimeout ?? "—"}
            </Field>
            <Field label="Last scrape">
              {picked.lastScrape.startsWith("0001") ? "never" : new Date(picked.lastScrape).toLocaleString("en-IN")} · {(picked.lastScrapeDuration * 1000).toFixed(1)} ms
            </Field>
            <Field label="Last error">{picked.lastError || "—"}</Field>
          </Detail>
        )}
      </div>
    </div>
  );
}

type RuleRow = PromRule & { group: string };

export function RulesTab({ groups, now }: { groups: { name: string; interval?: number; lastEvaluation?: string; rules: PromRule[] }[]; now: Date }) {
  const [group, setGroup] = useState("");
  const [type, setType] = useState("all");
  const [term, setTerm] = useState("");
  const [sel, setSel] = useState<string | null>(null);
  const t = term.trim().toLowerCase();
  const shown = groups
    .filter((g) => !group || g.name === group)
    .map((g) => ({ ...g, rules: g.rules.filter((r) => (type === "all" || r.type === type) && (!t || r.name.toLowerCase().includes(t) || r.query.toLowerCase().includes(t))) }))
    .filter((g) => g.rules.length > 0);
  const all: RuleRow[] = groups.flatMap((g) => g.rules.map((r) => ({ ...r, group: g.name })));
  const picked = all.find((r) => `${r.group}|${r.name}` === sel);
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <select aria-label="Rule group" className={select} value={group} onChange={(e) => setGroup(e.target.value)}>
          <option value="">All groups</option>
          {groups.map((g) => (
            <option key={g.name} value={g.name}>
              {g.name}
            </option>
          ))}
        </select>
        <select aria-label="Rule type" className={select} value={type} onChange={(e) => setType(e.target.value)}>
          <option value="all">All types</option>
          <option value="alerting">Alerting</option>
          <option value="recording">Recording</option>
        </select>
        <input aria-label="Search rules" placeholder="Search name or expression…" className={search} value={term} onChange={(e) => setTerm(e.target.value)} />
      </div>
      <div className={cn("grid gap-4", picked && "xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]")}>
        <Card className="min-w-0 overflow-x-auto">
          {shown.length === 0 ? (
            <Empty>No rules match the current filter</Empty>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th className="py-2 pl-4 text-left font-normal">Type</th>
                  <th className="py-2 text-left font-normal">Rule name</th>
                  <th className="py-2 text-left font-normal">State</th>
                  <th className="py-2 text-left font-normal">Last evaluation</th>
                  <th className="py-2 pr-4 text-left font-normal">Health / Error</th>
                </tr>
              </thead>
              {shown.map((g) => (
                <tbody key={g.name} className="divide-y border-t">
                  <tr className="bg-muted/40 text-xs">
                    <th colSpan={5} scope="rowgroup" className="py-1.5 pl-4 text-left font-medium">
                      {g.name}
                      <span className="ml-3 font-normal text-muted-foreground">
                        every {g.interval != null ? formatDuration(g.interval) : "—"} · last {g.lastEvaluation ? relativeTime(g.lastEvaluation, now) : "—"}
                      </span>
                    </th>
                  </tr>
                  {g.rules.map((r) => (
                    <tr
                      key={r.name + r.query}
                      className={cn("cursor-pointer hover:bg-accent/60", sel === `${g.name}|${r.name}` && "bg-primary/15")}
                      onClick={() => setSel(`${g.name}|${r.name}`)}
                    >
                      <td className="py-2 pl-4 capitalize text-muted-foreground">{r.type}</td>
                      <td className="max-w-72 truncate py-2 font-mono text-xs">{r.name}</td>
                      <td className="py-2">{r.state ? <StateBadge state={r.state} /> : "—"}</td>
                      <td className="py-2 text-muted-foreground">{r.lastEvaluation && !r.lastEvaluation.startsWith("0001") ? relativeTime(r.lastEvaluation, now) : "—"}</td>
                      <td className={cn("max-w-64 truncate py-2 pr-4", r.lastError && "text-critical")} title={r.lastError || undefined}>
                        {r.lastError || <StateBadge state={r.health} />}
                      </td>
                    </tr>
                  ))}
                </tbody>
              ))}
            </table>
          )}
        </Card>
        {picked && (
          <Detail title={picked.name} onClose={() => setSel(null)}>
            <Field label="Expression (read-only)">
              <pre className="overflow-x-auto rounded-md bg-muted p-2 font-mono text-xs whitespace-pre-wrap">{picked.query}</pre>
            </Field>
            <Field label="Group">{picked.group}</Field>
            {picked.duration != null && picked.type === "alerting" && <Field label="For">{formatDuration(picked.duration)}</Field>}
            <Field label="Labels">
              <LabelList labels={picked.labels} />
            </Field>
            {picked.type === "alerting" && (
              <>
                <Field label="Annotations">
                  <LabelList labels={picked.annotations} />
                </Field>
                <Field label="Current instances">{picked.alerts?.length ?? 0}</Field>
              </>
            )}
            <Field label="Health">
              {picked.health}
              {picked.lastError && ` — ${picked.lastError}`}
            </Field>
          </Detail>
        )}
      </div>
    </div>
  );
}

export function AlertsTab({ alerts, now, alertmanagerHref }: { alerts: PromAlert[]; now: Date; alertmanagerHref?: string }) {
  const [state, setState] = useState("all");
  const [term, setTerm] = useState("");
  const [sel, setSel] = useState<number | null>(null);
  const t = term.trim().toLowerCase();
  const rows = alerts
    .map((a, i) => ({ a, i }))
    .filter(({ a }) => (state === "all" || a.state === state) && (!t || Object.entries(a.labels).some(([k, v]) => `${k}=${v}`.toLowerCase().includes(t))));
  const picked = sel != null ? alerts[sel] : undefined;
  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">
        Alert evaluation state from Prometheus. Grouping, silences and notifications are handled by AlertManager
        {alertmanagerHref && (
          <>
            {" "}
            —{" "}
            <a href={alertmanagerHref} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
              open AlertManager
            </a>
          </>
        )}
        .
      </p>
      <div className="flex flex-wrap gap-2">
        <select aria-label="Alert state" className={select} value={state} onChange={(e) => setState(e.target.value)}>
          <option value="all">All states</option>
          <option value="firing">Firing</option>
          <option value="pending">Pending</option>
        </select>
        <input aria-label="Search alerts" placeholder="Search labels (e.g. symbol=INFY)…" className={search} value={term} onChange={(e) => setTerm(e.target.value)} />
      </div>
      <div className={cn("grid gap-4", picked && "xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]")}>
        <Card className="min-w-0 overflow-x-auto">
          {rows.length === 0 ? (
            <Empty>No alerts match the current filter</Empty>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th className="py-2 pl-4 text-left font-normal">State</th>
                  <th className="py-2 text-left font-normal">Alert</th>
                  <th className="py-2 text-left font-normal">Active since</th>
                  <th className="py-2 text-left font-normal">Key labels</th>
                  <th className="py-2 pr-4 text-left font-normal">Summary</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {rows.map(({ a, i }) => (
                  <tr key={i} className={cn("cursor-pointer hover:bg-accent/60", sel === i && "bg-primary/15")} onClick={() => setSel(i)}>
                    <td className="py-2 pl-4">
                      <StateBadge state={a.state} />
                    </td>
                    <td className="py-2 font-medium">{a.labels.alertname}</td>
                    <td className="py-2 text-muted-foreground">{a.activeAt ? relativeTime(a.activeAt, now) : "—"}</td>
                    <td className="py-2 text-xs">
                      {["severity", "symbol", "job"]
                        .filter((k) => a.labels[k])
                        .map((k) => `${k}=${a.labels[k]}`)
                        .join(" · ") || "—"}
                    </td>
                    <td className="max-w-72 truncate py-2 pr-4" title={a.annotations.summary}>
                      {a.annotations.summary ?? "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
        {picked && (
          <Detail title={picked.labels.alertname ?? "Alert"} onClose={() => setSel(null)}>
            <Field label="State">
              <StateBadge state={picked.state} /> {picked.activeAt && `since ${new Date(picked.activeAt).toLocaleString("en-IN")}`}
            </Field>
            {picked.value && <Field label="Value">{picked.value}</Field>}
            <Field label="Labels">
              <LabelList labels={picked.labels} />
            </Field>
            <Field label="Annotations">
              <LabelList labels={picked.annotations} />
            </Field>
          </Detail>
        )}
      </div>
    </div>
  );
}
