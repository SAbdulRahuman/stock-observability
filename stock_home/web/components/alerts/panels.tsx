"use client";

import { Bell, Info, OctagonAlert, TriangleAlert } from "lucide-react";
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Empty } from "@/components/dashboard/section-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { AlertRow, TimelineItem } from "@/lib/api";
import { formatCount, formatTimeIST, IST } from "@/lib/format";
import { cn } from "@/lib/utils";
import { ruleText, timeRanges } from "./shared";

const HOUR = 3_600_000;

export function SummaryCards({ rows, timeline, now }: { rows: AlertRow[]; timeline: TimelineItem[]; now: number }) {
  const active = rows.filter((r) => r.status !== "resolved");
  const fired24h = timeline.filter((t) => now - new Date(t.start).getTime() <= 24 * HOUR).length;
  const isNew = (r: AlertRow) => now - new Date(r.startsAt).getTime() <= HOUR;
  const cards = [
    { key: "critical", label: "Critical", icon: OctagonAlert, tone: "text-critical", ring: "border-critical/40" },
    { key: "warning", label: "Warning", icon: TriangleAlert, tone: "text-warning", ring: "border-warning/40" },
    { key: "info", label: "Info", icon: Info, tone: "text-info", ring: "border-info/40" },
  ];
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <Card aria-label="Total Alerts" className="p-4">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Bell className="size-4 text-critical" aria-hidden /> Total Alerts
        </div>
        <div className="mt-1 text-3xl font-semibold tabular-nums">{formatCount(active.length)}</div>
        <div className="text-xs text-muted-foreground">{fired24h} fired in the last 24h</div>
      </Card>
      {cards.map((c) => {
        const list = active.filter((r) => r.severity === c.key);
        const Icon = c.icon;
        return (
          <Card key={c.key} aria-label={c.label} className={cn("p-4", list.length > 0 && c.ring)}>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Icon className={cn("size-4", c.tone)} aria-hidden /> {c.label}
            </div>
            <div className={cn("mt-1 text-3xl font-semibold tabular-nums", list.length > 0 && c.tone)}>{formatCount(list.length)}</div>
            <div className="text-xs text-muted-foreground">{list.filter(isNew).length} new in the last hour</div>
          </Card>
        );
      })}
    </div>
  );
}

type Event = { key: string; at: string; symbol?: string; text: string; tone: string };

export function AlertActivity({ rows }: { rows: AlertRow[] }) {
  const events: Event[] = [];
  for (const r of rows) {
    events.push({ key: `${r.key}:start`, at: r.startsAt, symbol: r.symbol, text: `${ruleText(r)} fired`, tone: "bg-critical" });
    if (r.endsAt) events.push({ key: `${r.key}:end`, at: r.endsAt, symbol: r.symbol, text: `${r.name} resolved`, tone: "bg-ok" });
  }
  events.sort((a, b) => b.at.localeCompare(a.at));
  return (
    <Card aria-labelledby="alert-activity-title" className="min-w-0">
      <CardHeader>
        <CardTitle id="alert-activity-title">Alert Activity</CardTitle>
      </CardHeader>
      <CardContent>
        {events.length === 0 ? (
          <Empty>No alert activity</Empty>
        ) : (
          <ul className="space-y-1.5 text-sm">
            {events.slice(0, 8).map((e) => (
              <li key={e.key} className="flex items-center gap-3">
                <span className={cn("size-2 shrink-0 rounded-full", e.tone)} aria-hidden />
                <span className="w-11 shrink-0 text-muted-foreground tabular-nums">{formatTimeIST(e.at)}</span>
                <span className="w-24 shrink-0 truncate font-medium">{e.symbol || "—"}</span>
                <span className="min-w-0 truncate">{e.text}</span>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

const severityLines = [
  { key: "critical", label: "Critical", color: "var(--color-critical)" },
  { key: "warning", label: "Warning", color: "var(--color-warning)" },
  { key: "info", label: "Info", color: "var(--color-info)" },
] as const;

/** Firing alert count per severity at evenly spaced instants, from the 24h firing history. */
export function countsOverTime(timeline: TimelineItem[], range: string, now: number, buckets = 24) {
  const span = timeRanges[range]?.ms ?? 24 * HOUR;
  const step = span / buckets;
  const runs = timeline.map((t) => ({ sev: t.severity, start: new Date(t.start).getTime(), end: t.end ? new Date(t.end).getTime() : Number.POSITIVE_INFINITY }));
  return Array.from({ length: buckets + 1 }, (_, i) => {
    const t = now - span + i * step;
    const row: Record<string, number> = { t, critical: 0, warning: 0, info: 0 };
    for (const r of runs) {
      if (r.start <= t && t < r.end && r.sev in row) row[r.sev]++;
    }
    return row as { t: number; critical: number; warning: number; info: number };
  });
}

const tick = new Intl.DateTimeFormat("en-IN", { timeZone: IST, hour: "2-digit", minute: "2-digit", hour12: false });

export function AlertCounts({ timeline, range, now }: { timeline: TimelineItem[]; range: string; now: number }) {
  const data = countsOverTime(timeline, range, now);
  const peak = Math.max(0, ...data.map((d) => d.critical + d.warning + d.info));
  return (
    <Card aria-labelledby="alert-counts-title" className="min-w-0">
      <CardHeader>
        <CardTitle id="alert-counts-title">Alert Counts Over Time</CardTitle>
        <span className="ml-auto text-xs text-muted-foreground">{timeRanges[range === "all" ? "24h" : range]?.label}</span>
      </CardHeader>
      <CardContent>
        {timeline.length === 0 ? (
          <Empty>No alerts fired in this period</Empty>
        ) : (
          <figure aria-label={`Peak of ${peak} firing alerts`} className="h-44">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -20 }}>
                <CartesianGrid stroke="var(--color-border)" strokeDasharray="3 3" vertical={false} />
                <XAxis
                  dataKey="t"
                  type="number"
                  domain={["dataMin", "dataMax"]}
                  tickFormatter={(t: number) => tick.format(new Date(t))}
                  tick={{ fill: "var(--color-muted-foreground)", fontSize: 11 }}
                  stroke="var(--color-border)"
                  minTickGap={30}
                />
                <YAxis allowDecimals={false} tick={{ fill: "var(--color-muted-foreground)", fontSize: 11 }} stroke="var(--color-border)" />
                <Tooltip labelFormatter={(t) => `${tick.format(new Date(Number(t)))} IST`} contentStyle={{ background: "var(--color-card)", border: "1px solid var(--color-border)" }} />
                <Legend wrapperStyle={{ fontSize: 12 }} />
                {severityLines.map((s) => (
                  <Line key={s.key} type="stepAfter" dataKey={s.key} name={s.label} stroke={s.color} dot={false} isAnimationActive={false} />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </figure>
        )}
      </CardContent>
    </Card>
  );
}
