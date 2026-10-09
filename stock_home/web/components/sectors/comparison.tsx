"use client";

import { useQueries } from "@tanstack/react-query";
import { LineChart as LineIcon } from "lucide-react";
import { useState } from "react";
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Empty } from "@/components/dashboard/section-card";
import { RangePicker } from "@/components/home/intraday-chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { type ChartRange, fetchChart } from "@/lib/api";
import { formatDecimal, IST } from "@/lib/format";
import { cn } from "@/lib/utils";

/** One colour per position in the configured sector-index list, reused by every sector chart on the page. */
export const sectorPalette = ["#22c55e", "#ef4444", "#3b82f6", "#eab308", "#a855f7", "#14b8a6", "#f97316", "#ec4899", "#84cc16", "#06b6d4", "#94a3b8"];

const MAX_SELECTED = 4;
const ticks: Record<ChartRange, Intl.DateTimeFormat> = {
  "1D": new Intl.DateTimeFormat("en-IN", { timeZone: IST, hour: "2-digit", minute: "2-digit", hour12: false }),
  "1W": new Intl.DateTimeFormat("en-IN", { timeZone: IST, weekday: "short", day: "numeric" }),
  "1M": new Intl.DateTimeFormat("en-IN", { timeZone: IST, day: "numeric", month: "short" }),
  "3M": new Intl.DateTimeFormat("en-IN", { timeZone: IST, day: "numeric", month: "short" }),
  "1Y": new Intl.DateTimeFormat("en-IN", { timeZone: IST, month: "short", year: "2-digit" }),
  "5Y": new Intl.DateTimeFormat("en-IN", { timeZone: IST, month: "short", year: "numeric" }),
};

/** Each series is rebased to 100 at its first point so different index levels compare on one axis. */
export function rebase(series: { symbol: string; points: { t: number; v: number }[] }[]): Record<string, number>[] {
  const rows = new Map<number, Record<string, number>>();
  for (const s of series) {
    const base = s.points[0]?.v;
    if (!base) continue;
    for (const p of s.points) {
      const row = rows.get(p.t) ?? { t: p.t };
      row[s.symbol] = (p.v / base) * 100;
      rows.set(p.t, row);
    }
  }
  return [...rows.values()].sort((a, b) => a.t - b.t);
}

export function SectorComparison({ symbols }: { symbols: string[] }) {
  const [range, setRange] = useState<ChartRange>("1D");
  const [picked, setPicked] = useState<string[] | null>(null);
  const selected = (picked ?? symbols.slice(0, MAX_SELECTED)).filter((s) => symbols.includes(s));
  const charts = useQueries({
    queries: selected.map((symbol) => ({
      queryKey: ["chart", symbol, range],
      queryFn: ({ signal }: { signal: AbortSignal }) => fetchChart(symbol, range, signal),
      refetchInterval: range === "1D" ? 30_000 : 120_000,
    })),
  });
  const color = (s: string) => sectorPalette[symbols.indexOf(s) % sectorPalette.length];
  const toggle = (s: string) => {
    const cur = picked ?? symbols.slice(0, MAX_SELECTED);
    setPicked(cur.includes(s) ? cur.filter((x) => x !== s) : cur.length < MAX_SELECTED ? [...cur, s] : cur);
  };
  const loaded = charts.flatMap((c, i) => (c.data ? [{ symbol: selected[i], points: c.data.data.points }] : []));
  const rows = rebase(loaded);
  const failed = charts.filter((c) => c.isError).length;

  return (
    <Card aria-labelledby="sector-comparison-title" className="min-w-0">
      <CardHeader className="flex-wrap">
        <LineIcon className="size-4 text-muted-foreground" aria-hidden />
        <CardTitle id="sector-comparison-title">Sector Comparison</CardTitle>
        <div className="ml-auto">
          <RangePicker value={range} onChange={setRange} label="Comparison timeframe" />
        </div>
      </CardHeader>
      <CardContent>
        <div role="group" aria-label="Compared sectors" className="mb-3 flex flex-wrap gap-1.5">
          {symbols.map((s) => {
            const on = selected.includes(s);
            return (
              <button
                key={s}
                type="button"
                aria-pressed={on}
                onClick={() => toggle(s)}
                disabled={!on && selected.length >= MAX_SELECTED}
                className={cn("inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs disabled:opacity-40", on ? "border-primary bg-primary/10" : "text-muted-foreground")}
              >
                <span className="size-2 rounded-full" style={{ background: color(s) }} aria-hidden />
                {s.replace(/^NIFTY /, "")}
              </button>
            );
          })}
        </div>
        {symbols.length === 0 ? (
          <Empty>No sector indices configured</Empty>
        ) : selected.length === 0 ? (
          <Empty>Select up to {MAX_SELECTED} sectors to compare</Empty>
        ) : charts.some((c) => c.isPending) ? (
          <Skeleton className="h-56 w-full" />
        ) : rows.length < 2 ? (
          <Empty>{failed > 0 ? "Sector charts unavailable" : "No index history for this period"}</Empty>
        ) : (
          <figure
            className="h-56"
            aria-label={`Rebased to 100: ${loaded.map((l) => `${l.symbol} ${formatDecimal(rows.at(-1)?.[l.symbol])}`).join(", ")}`}
          >
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={rows} margin={{ top: 4, right: 8, bottom: 0, left: -10 }}>
                <CartesianGrid stroke="var(--color-border)" strokeDasharray="3 3" vertical={false} />
                <XAxis
                  dataKey="t"
                  type="number"
                  scale="time"
                  domain={["dataMin", "dataMax"]}
                  tickFormatter={(t: number) => ticks[range].format(new Date(t))}
                  tick={{ fill: "var(--color-muted-foreground)", fontSize: 11 }}
                  stroke="var(--color-border)"
                  minTickGap={30}
                />
                <YAxis domain={["auto", "auto"]} tick={{ fill: "var(--color-muted-foreground)", fontSize: 11 }} stroke="var(--color-border)" />
                <Tooltip
                  labelFormatter={(t) => `${ticks[range].format(new Date(Number(t)))} IST`}
                  formatter={(v) => formatDecimal(Number(v))}
                  contentStyle={{ background: "var(--color-card)", border: "1px solid var(--color-border)" }}
                />
                {loaded.map((l) => (
                  <Line key={l.symbol} dataKey={l.symbol} stroke={color(l.symbol)} dot={false} isAnimationActive={false} connectNulls />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </figure>
        )}
        <p className="mt-2 text-xs text-muted-foreground">Index values rebased to 100 at the start of the period</p>
      </CardContent>
    </Card>
  );
}
