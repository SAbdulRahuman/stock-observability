"use client";

import { Bell, Download, Plus, Settings, Upload } from "lucide-react";
import { Area, AreaChart, ResponsiveContainer, XAxis, YAxis } from "recharts";
import { Empty } from "@/components/dashboard/section-card";
import { ViewAll } from "@/components/home/view-all";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { type Quote, useSection } from "@/lib/api";
import { formatCount, formatPercent, formatTimeIST, IST } from "@/lib/format";
import { cn } from "@/lib/utils";

const tick = new Intl.DateTimeFormat("en-IN", { timeZone: IST, hour: "2-digit", minute: "2-digit", hour12: false });

/** Equal-weight average % change versus previous close at each sparkline timestamp. */
export function averageChange(quotes: Quote[]): { t: number; v: number }[] {
  const sums = new Map<number, { s: number; n: number }>();
  for (const q of quotes) {
    if (q.price == null || q.change == null) continue;
    const prev = q.price - q.change;
    if (prev <= 0) continue;
    for (const p of q.spark ?? []) {
      const e = sums.get(p.t) ?? { s: 0, n: 0 };
      e.s += (p.v / prev - 1) * 100;
      e.n += 1;
      sums.set(p.t, e);
    }
  }
  return [...sums.entries()].sort(([a], [b]) => a - b).map(([t, e]) => ({ t, v: e.s / e.n }));
}

export function Performance({ total, quotes }: { total: number; quotes: Quote[] }) {
  const known = quotes.filter((q) => q.changePercent != null);
  const gainers = known.filter((q) => (q.changePercent ?? 0) > 0).length;
  const losers = known.filter((q) => (q.changePercent ?? 0) < 0).length;
  const unchanged = known.length - gainers - losers;
  const pct = (n: number) => (known.length > 0 ? formatPercent((n / known.length) * 100, 1, false) : "—");
  const series = averageChange(quotes);
  const last = series.at(-1)?.v;
  const up = (last ?? 0) >= 0;
  return (
    <Card aria-labelledby="wl-performance-title" className="min-w-0">
      <CardHeader>
        <CardTitle id="wl-performance-title">Watchlist Performance</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          <Tile label="Total Symbols" value={formatCount(total)} />
          <Tile label="Gainers" value={`${gainers} (${pct(gainers)})`} className="border-up/40 bg-up/10 text-up" />
          <Tile label="Losers" value={`${losers} (${pct(losers)})`} className="border-down/40 bg-down/10 text-down" />
          <Tile label="Unchanged" value={`${unchanged} (${pct(unchanged)})`} />
        </div>
        {series.length < 2 ? (
          <p className="text-xs text-muted-foreground">No intraday history for these symbols yet</p>
        ) : (
          <figure aria-label={`Average change across the watchlist: ${formatPercent(last)}`} className="h-20">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={series} margin={{ top: 4, right: 4, bottom: 0, left: 4 }}>
                <XAxis
                  dataKey="t"
                  type="number"
                  scale="time"
                  domain={["dataMin", "dataMax"]}
                  tickFormatter={(t: number) => tick.format(new Date(t))}
                  tick={{ fill: "var(--color-muted-foreground)", fontSize: 10 }}
                  stroke="var(--color-border)"
                  minTickGap={30}
                />
                <YAxis hide domain={["auto", "auto"]} />
                <Area
                  type="monotone"
                  dataKey="v"
                  stroke={up ? "var(--color-up)" : "var(--color-down)"}
                  fill={up ? "var(--color-up)" : "var(--color-down)"}
                  fillOpacity={0.15}
                  dot={false}
                  isAnimationActive={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          </figure>
        )}
      </CardContent>
    </Card>
  );
}

function Tile({ label, value, className }: { label: string; value: string; className?: string }) {
  return (
    <div className={cn("rounded-lg border px-3 py-2", className)}>
      <div className="text-xs opacity-80">{label}</div>
      <div className="text-lg font-semibold tabular-nums">{value}</div>
    </div>
  );
}

const MAX_ALERTS = 5;

/** Firing alerts and alerts resolved in the last 24h whose symbol is in the active watchlist. */
export function RecentAlerts({ symbols }: { symbols: string[] }) {
  const alerts = useSection("alerts");
  const timeline = useSection("timeline");
  const set = new Set(symbols);
  const firing = (alerts.data?.data.items ?? [])
    .filter((a) => a.symbol && set.has(a.symbol))
    .map((a) => ({ key: a.fingerprint, time: a.startsAt, symbol: a.symbol ?? "", rule: a.summary || a.name, value: a.value, severity: a.severity, firing: true }));
  const resolved = (timeline.data?.data.items ?? [])
    .filter((a) => a.end && a.symbol && set.has(a.symbol))
    .map((a) => ({ key: `${a.name}:${a.subject}:${a.start}`, time: a.start, symbol: a.symbol ?? "", rule: a.name, value: undefined, severity: a.severity, firing: false }));
  const rows = [...firing, ...resolved].sort((a, b) => b.time.localeCompare(a.time)).slice(0, MAX_ALERTS);
  return (
    <Card aria-labelledby="wl-alerts-title" className="min-w-0">
      <CardHeader>
        <CardTitle id="wl-alerts-title">Recent Alerts (Watchlist)</CardTitle>
        <div className="ml-auto">
          <ViewAll page="Alerts" fallbackLinkId="alertmanager" label="watchlist alerts" />
        </div>
      </CardHeader>
      <CardContent>
        {alerts.isError && !alerts.data ? (
          <Empty>Alerts unavailable — {alerts.error.message}</Empty>
        ) : rows.length === 0 ? (
          <Empty>No alerts for these symbols</Empty>
        ) : (
          <table className="w-full text-sm">
            <tbody className="divide-y">
              {rows.map((r) => (
                <tr key={r.key}>
                  <td className="py-1.5 pr-2 whitespace-nowrap text-muted-foreground tabular-nums">
                    <span
                      className={cn("mr-2 inline-block size-2 rounded-full", r.firing ? ({ critical: "bg-critical", warning: "bg-warning" }[r.severity] ?? "bg-info") : "bg-ok")}
                      aria-hidden
                    />
                    <span className="sr-only">{r.firing ? `${r.severity} firing` : "resolved"} </span>
                    {formatTimeIST(r.time)}
                  </td>
                  <td className="py-1.5 pr-2 font-medium">{r.symbol}</td>
                  <td className="w-full max-w-0 truncate py-1.5 pr-2" title={r.rule}>
                    {r.rule}
                  </td>
                  <td className="py-1.5 text-right tabular-nums">{r.value || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  );
}

export function QuickActions({
  editable,
  onCreate,
  onImport,
  onExport,
}: {
  editable: boolean;
  onCreate: () => void;
  onImport: () => void;
  onExport: () => void;
}) {
  const item = "flex w-full items-center gap-2 rounded-md border px-3 py-2 text-left text-sm hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:bg-transparent";
  return (
    <Card aria-labelledby="wl-actions-title" className="min-w-0">
      <CardHeader>
        <CardTitle id="wl-actions-title">Quick Actions</CardTitle>
      </CardHeader>
      <CardContent>
        <ul className="space-y-1.5">
          <li>
            <button type="button" className={item} onClick={onCreate} disabled={!editable}>
              <Plus className="size-4" aria-hidden /> Create New Watchlist
            </button>
          </li>
          <li>
            <button type="button" className={item} onClick={onImport} disabled={!editable}>
              <Download className="size-4" aria-hidden /> Import from CSV
            </button>
          </li>
          <li>
            <button type="button" className={item} onClick={onExport}>
              <Upload className="size-4" aria-hidden /> Export Watchlist
            </button>
          </li>
          <li>
            <button type="button" className={item} disabled title="Alert rules are defined in Prometheus and cannot be created here">
              <Bell className="size-4" aria-hidden /> Set Global Alerts
            </button>
          </li>
          <li>
            <button type="button" className={item} disabled title="Settings page coming soon">
              <Settings className="size-4" aria-hidden /> Watchlist Settings
            </button>
          </li>
        </ul>
      </CardContent>
    </Card>
  );
}
