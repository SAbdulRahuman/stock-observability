"use client";

import { useQueryClient } from "@tanstack/react-query";
import { ExternalLink, RefreshCw } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useNow } from "@/components/dashboard/section-card";
import { useLinkHref } from "@/components/shell/hooks";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { StatusDot } from "@/components/ui/primitives";
import { formatCount, relativeTime } from "@/lib/format";
import { exampleQueries, type PromStatus, usePromSection, usePromStatus } from "@/lib/prom";
import { cn } from "@/lib/utils";
import { QueryTab } from "./query-tab";
import { AlertsTab, PromState, RulesTab, TargetsTab } from "./tabs";

const TABS = ["overview", "query", "targets", "rules", "alerts", "status"] as const;
type Tab = (typeof TABS)[number];

export function PrometheusPage() {
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname() ?? "/observability/prometheus";
  const tab: Tab = (TABS as readonly string[]).includes(params?.get("tab") ?? "") ? (params?.get("tab") as Tab) : "overview";
  const initialQuery = params?.get("query") ?? "";
  const now = useNow(10_000);
  const client = useQueryClient();
  const full = useLinkHref("prometheus");
  const alertmanager = useLinkHref("alertmanager");
  const status = usePromStatus();
  const targets = usePromSection("targets", tab === "overview" || tab === "targets");
  const rules = usePromSection("rules", tab === "overview" || tab === "rules");
  const alerts = usePromSection("alerts", tab === "overview" || tab === "alerts");
  const go = (next: Record<string, string>) => {
    const qs = new URLSearchParams(params?.toString() ?? "");
    for (const [k, v] of Object.entries(next)) {
      if (v) qs.set(k, v);
      else qs.delete(k);
    }
    router.replace(`${pathname}?${qs}`, { scroll: false });
  };
  const connected = status.data ? status.data.ready : null;
  const lastOk = [status, targets, rules, alerts].map((q) => q.dataUpdatedAt).filter((t) => t > 0);

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Prometheus</h1>
          <p className="text-sm text-muted-foreground">Query metrics, inspect targets, rules, and alert state</p>
        </div>
        {full && (
          <Button variant="outline" asChild>
            <a href={full} target="_blank" rel="noopener noreferrer">
              <ExternalLink /> Open full Prometheus UI
            </a>
          </Button>
        )}
      </div>

      <div role="tablist" aria-label="Prometheus sections" className="flex flex-wrap gap-1 border-b">
        {TABS.map((t) => (
          <button
            key={t}
            type="button"
            role="tab"
            aria-selected={t === tab}
            onClick={() => go({ tab: t })}
            className={cn(
              "-mb-px border-b-2 px-3 py-2 text-sm font-medium capitalize transition-colors",
              t === tab ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground",
            )}
          >
            {t}
          </button>
        ))}
      </div>

      <Card className="flex flex-wrap items-center gap-3 px-4 py-3 text-sm" aria-label="Prometheus connection">
        <span className="text-muted-foreground">Prometheus connection:</span>
        <span className="font-medium">configured server</span>
        <span className="flex items-center gap-2" role="status">
          <StatusDot ok={connected} />
          {status.isError ? "Unreachable" : connected == null ? "Checking…" : connected ? "Ready" : "Not ready"}
        </span>
        {status.data?.build.version && <span className="text-muted-foreground">v{status.data.build.version}</span>}
        <span className="ml-auto text-xs text-muted-foreground">Last successful API check: {lastOk.length ? relativeTime(new Date(Math.max(...lastOk)), now) : "—"}</span>
        <Button variant="outline" size="sm" onClick={() => void client.invalidateQueries({ queryKey: ["prom"] })}>
          <RefreshCw /> Refresh
        </Button>
      </Card>

      <section role="tabpanel" aria-label={`${tab} tab`}>
        {tab === "overview" && (
          <Overview
            targets={targets.data?.data.activeTargets}
            groups={rules.data?.data.groups}
            alerts={alerts.data?.data.alerts}
            failed={{ targets: targets.isError, rules: rules.isError, alerts: alerts.isError }}
            onExample={(q) => go({ tab: "query", query: q })}
          />
        )}
        {tab === "query" && <QueryTab key={initialQuery} initial={initialQuery} />}
        {tab === "targets" && <PromState query={targets}>{(d) => <TargetsTab targets={d.activeTargets} now={now} />}</PromState>}
        {tab === "rules" && <PromState query={rules}>{(d) => <RulesTab groups={d.groups} now={now} />}</PromState>}
        {tab === "alerts" && <PromState query={alerts}>{(d) => <AlertsTab alerts={d.alerts} now={now} alertmanagerHref={alertmanager} />}</PromState>}
        {tab === "status" && <StatusTab status={status.data} error={status.isError ? status.error.message : null} />}
      </section>
    </main>
  );
}

function Overview({
  targets,
  groups,
  alerts,
  failed,
  onExample,
}: {
  targets?: { health: string; scrapePool: string }[];
  groups?: { rules: { type: string }[] }[];
  alerts?: { state: string }[];
  failed: { targets: boolean; rules: boolean; alerts: boolean };
  onExample: (q: string) => void;
}) {
  const up = targets?.filter((t) => t.health === "up").length;
  const down = targets?.filter((t) => t.health === "down").length;
  const all = groups?.flatMap((g) => g.rules);
  const alerting = all?.filter((r) => r.type === "alerting").length;
  const firing = alerts?.filter((a) => a.state === "firing").length;
  const pending = alerts?.filter((a) => a.state === "pending").length;
  const val = (v: number | undefined, err: boolean) => (v == null ? (err ? "unavailable" : "—") : formatCount(v));
  const byJob = new Map<string, { up: number; total: number }>();
  for (const t of targets ?? []) {
    const e = byJob.get(t.scrapePool) ?? { up: 0, total: 0 };
    e.total++;
    if (t.health === "up") e.up++;
    byJob.set(t.scrapePool, e);
  }
  const cards = [
    { label: "Targets UP", value: targets ? `${up} / ${targets.length}` : val(undefined, failed.targets), tone: "text-ok" },
    { label: "Targets DOWN", value: val(down, failed.targets), tone: down ? "text-critical" : "" },
    { label: "Rules", value: val(all?.length, failed.rules), hint: all ? `${alerting} alerting · ${all.length - (alerting ?? 0)} recording` : undefined },
    { label: "Firing Alerts", value: val(firing, failed.alerts), tone: firing ? "text-critical" : "", hint: alerts ? `${pending} pending` : undefined },
  ];
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((c) => (
          <Card key={c.label} aria-label={c.label} className="p-4">
            <div className="text-sm text-muted-foreground">{c.label}</div>
            <div className={cn("mt-1 text-2xl font-semibold tabular-nums", c.tone)}>{c.value}</div>
            {c.hint && <div className="text-xs text-muted-foreground">{c.hint}</div>}
          </Card>
        ))}
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <Card className="p-4" aria-label="Target health by job">
          <h2 className="mb-2 text-sm font-semibold">Target health by job</h2>
          {byJob.size === 0 ? (
            <p className="text-sm text-muted-foreground">{failed.targets ? "Targets unavailable" : "No targets"}</p>
          ) : (
            <ul className="space-y-1.5 text-sm">
              {[...byJob.entries()].sort().map(([job, e]) => (
                <li key={job} className="flex items-center gap-2">
                  <StatusDot ok={e.up === e.total} />
                  <span className="flex-1 font-medium">{job}</span>
                  <span className="tabular-nums text-muted-foreground">
                    {e.up}/{e.total} up
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Card>
        <Card className="p-4" aria-label="Example queries">
          <h2 className="mb-2 text-sm font-semibold">Example queries</h2>
          <p className="mb-2 text-xs text-muted-foreground">Metrics published by this stack&apos;s exporter and recording rules; results depend on what is currently scraped.</p>
          <ul className="space-y-1 text-sm">
            {exampleQueries.map((x) => (
              <li key={x.query}>
                <button type="button" className="text-left hover:underline" onClick={() => onExample(x.query)}>
                  <span className="font-medium">{x.label}</span> <code className="text-xs text-muted-foreground">{x.query}</code>
                </button>
              </li>
            ))}
          </ul>
        </Card>
      </div>
    </div>
  );
}

const runtimeLabels: Record<string, string> = {
  startTime: "Start time",
  reloadConfigSuccess: "Last config reload successful",
  lastConfigTime: "Last config reload",
  goroutineCount: "Goroutines",
  storageRetention: "Storage retention",
  corruptionCount: "WAL corruptions",
};

function StatusTab({ status, error }: { status?: PromStatus; error: string | null }) {
  if (!status) return <p className="text-sm text-muted-foreground">{error ? `Unavailable — ${error}` : "Loading…"}</p>;
  const row = (k: string, v: unknown) => (
    <div key={k} className="flex justify-between gap-3 py-1.5 text-sm">
      <dt className="text-muted-foreground">{k}</dt>
      <dd className="text-right font-mono">{v == null || v === "" ? "—" : String(v)}</dd>
    </div>
  );
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card className="p-4" aria-label="Build information">
        <h2 className="mb-2 text-sm font-semibold">Build information</h2>
        <dl className="divide-y">{["version", "revision", "branch", "buildDate", "goVersion"].map((k) => row(k, status.build[k]))}</dl>
      </Card>
      <Card className="p-4" aria-label="Runtime information">
        <h2 className="mb-2 text-sm font-semibold">Runtime and health</h2>
        <dl className="divide-y">
          {row("Ready", status.ready ? "yes" : "no")}
          {row("Healthy", status.healthy ? "yes" : "no")}
          {Object.entries(runtimeLabels).map(([k, label]) => row(label, status.runtime[k]))}
        </dl>
        <p className="mt-2 text-xs text-muted-foreground">Configuration and command-line flags are intentionally not shown.</p>
      </Card>
    </div>
  );
}
