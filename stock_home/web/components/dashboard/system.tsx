"use client";

import {
  BarChart3,
  Bell,
  BookOpen,
  ExternalLink,
  GitBranch,
  History,
  LogIn,
  type LucideIcon,
  Server,
  Target,
} from "lucide-react";
import { Badge, severityVariant } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Meter, StatusDot } from "@/components/ui/primitives";
import { useSection } from "@/lib/api";
import { formatCount, formatDuration, formatTimeIST } from "@/lib/format";
import { cn } from "@/lib/utils";
import { Empty, SectionCard, useNow } from "./section-card";

/** Section 9. */
export function SystemHealth() {
  const query = useSection("system");
  return (
    <SectionCard id="system" title="System health" icon={Server} query={query} href="/prometheus/targets" hrefLabel="Prometheus targets">
      {(s) => (
        <div className="space-y-4">
          <div className="grid grid-cols-3 gap-2">
            {s.components.map((c) => (
              <a key={c.name} href={c.href} className="rounded-lg border px-3 py-2 hover:bg-accent">
                <div className="flex items-center gap-2 text-sm font-medium">
                  <StatusDot ok={c.healthy} />
                  {c.name}
                </div>
                <div className="text-xs text-muted-foreground tabular-nums">{c.healthy ? `${Math.round(c.latencyMs)} ms` : "unreachable"}</div>
              </a>
            ))}
          </div>
          {s.targets.length > 0 && (
            <table className="w-full text-sm">
              <caption className="sr-only">Scrape jobs</caption>
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th className="py-1 text-left font-normal">Job</th>
                  <th className="py-1 text-left font-normal">Scrape time / timeout</th>
                  <th className="py-1 text-left font-normal">Sample limit</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {s.targets.map((t) => {
                  const timeUse = t.durationSeconds != null && t.timeoutSeconds ? t.durationSeconds / t.timeoutSeconds : null;
                  return (
                    <tr key={t.job}>
                      <td className="py-1.5 pr-2">
                        <span className="flex items-center gap-2 font-medium">
                          <StatusDot ok={t.up} />
                          {t.job}
                        </span>
                      </td>
                      <td className="py-1.5 pr-2">
                        {timeUse != null ? (
                          <div title={`${(t.durationSeconds ?? 0).toFixed(3)}s of ${t.timeoutSeconds}s`}>
                            <Meter value={timeUse} />
                            <span className="text-xs text-muted-foreground tabular-nums">{Math.round(timeUse * 100)}%</span>
                          </div>
                        ) : (
                          "—"
                        )}
                      </td>
                      <td className="py-1.5">
                        {t.sampleUsage != null ? (
                          <div>
                            <Meter value={t.sampleUsage} />
                            <span className="text-xs text-muted-foreground tabular-nums">{Math.round(t.sampleUsage * 100)}%</span>
                          </div>
                        ) : (
                          <span className="text-xs text-muted-foreground">no limit</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
          <div className="flex flex-wrap items-center gap-1.5">
            {Object.entries(s.collectors)
              .sort()
              .map(([name, ok]) => (
                <Badge key={name} variant={ok ? "ok" : "critical"}>
                  {name}
                </Badge>
              ))}
            {s.headSeries != null && <span className="ml-auto text-xs text-muted-foreground">{formatCount(s.headSeries)} active series</span>}
          </div>
        </div>
      )}
    </SectionCard>
  );
}

/** Section 10: firing periods over the last 24 hours, drawn on a shared time axis. */
export function AlertTimeline() {
  const query = useSection("timeline");
  const now = useNow(60_000);
  return (
    <SectionCard id="timeline" title="Alert timeline · 24h" icon={History} query={query} href="/grafana/alerting/list" hrefLabel="Open Grafana Alerting">
      {(t) => {
        const from = new Date(t.from).getTime();
        const span = now.getTime() - from;
        const items = t.items.slice(0, 25);
        return (
          <div className="space-y-3">
            <div className="flex flex-wrap gap-1.5">
              {(["critical", "warning", "info"] as const).map((sev) => (
                <Badge key={sev} variant={severityVariant(sev)}>
                  {t.bySeverity[sev] ?? 0} {sev}
                </Badge>
              ))}
            </div>
            {items.length === 0 ? (
              <Empty>No alerts fired in the last 24 hours</Empty>
            ) : (
              <ul className="max-h-80 space-y-1.5 overflow-y-auto pr-1">
                {items.map((it) => {
                  const start = new Date(it.start).getTime();
                  const end = it.end ? new Date(it.end).getTime() : now.getTime();
                  const left = Math.max(0, ((start - from) / span) * 100);
                  const width = Math.max(0.6, ((end - start) / span) * 100);
                  return (
                    <li key={`${it.name}:${it.subject}:${it.start}`} className="grid grid-cols-[minmax(0,10rem)_1fr] items-center gap-3">
                      <div className="min-w-0">
                        <div className="truncate text-sm font-medium" title={it.name}>
                          {it.name}
                        </div>
                        <div className="truncate text-xs text-muted-foreground">
                          {it.subject || it.category} · {formatTimeIST(it.start)}
                          {it.end ? ` · ${formatDuration((end - start) / 1000)}` : " · firing"}
                        </div>
                      </div>
                      <div className="relative h-3 rounded-full bg-muted" aria-hidden>
                        <div
                          className={cn(
                            "absolute top-0 h-3 rounded-full",
                            { critical: "bg-critical", warning: "bg-warning", info: "bg-info" }[it.severity] ?? "bg-muted-foreground",
                            !it.end && "animate-pulse",
                          )}
                          style={{ left: `${left}%`, width: `${Math.min(width, 100 - left)}%` }}
                        />
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
            <div className="flex justify-between text-xs text-muted-foreground tabular-nums">
              <span>{formatTimeIST(t.from)}</span>
              <span>now</span>
            </div>
          </div>
        );
      }}
    </SectionCard>
  );
}

const linkIcons: Record<string, LucideIcon> = {
  "Kite login": LogIn,
  Grafana: BarChart3,
  "Grafana Alerting": Bell,
  "Prometheus targets": Target,
  "Prometheus rules": BookOpen,
  Alertmanager: GitBranch,
};

/** Section 5. */
export function QuickLinks() {
  const query = useSection("links");
  const links = query.data?.data ?? [];
  return (
    <nav aria-label="Quick links" className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
      {links.map((l) => {
        const Icon = linkIcons[l.label] ?? ExternalLink;
        return (
          <Card key={l.href} className="transition-colors hover:bg-accent">
            <a href={l.href} className="flex items-center gap-3 p-4">
              <Icon className="size-5 shrink-0 text-primary" aria-hidden />
              <span className="min-w-0">
                <span className="block truncate text-sm font-medium">{l.label}</span>
                {l.description && <span className="block truncate text-xs text-muted-foreground">{l.description}</span>}
              </span>
            </a>
          </Card>
        );
      })}
    </nav>
  );
}
