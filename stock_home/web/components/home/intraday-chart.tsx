"use client";

import { ChartLine } from "lucide-react";
import { useState } from "react";
import { Area, Bar, CartesianGrid, ComposedChart, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { type Chart, type ChartRange, chartRanges, useChart } from "@/lib/api";
import { formatCount, formatPrice, IST } from "@/lib/format";
import { cn } from "@/lib/utils";

const SYMBOL = "NIFTY 50";

const tickFormats: Record<ChartRange, Intl.DateTimeFormat> = {
  "1D": new Intl.DateTimeFormat("en-IN", { timeZone: IST, hour: "2-digit", minute: "2-digit", hour12: false }),
  "1W": new Intl.DateTimeFormat("en-IN", { timeZone: IST, weekday: "short", day: "numeric" }),
  "1M": new Intl.DateTimeFormat("en-IN", { timeZone: IST, day: "numeric", month: "short" }),
  "3M": new Intl.DateTimeFormat("en-IN", { timeZone: IST, day: "numeric", month: "short" }),
  "1Y": new Intl.DateTimeFormat("en-IN", { timeZone: IST, month: "short", year: "2-digit" }),
};
const tooltipTime = new Intl.DateTimeFormat("en-IN", { timeZone: IST, day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", hour12: false });
const axisPrice = new Intl.NumberFormat("en-IN", { maximumFractionDigits: 0 });

type Row = { t: number; v: number; vol?: number };

/** Range is internal unless the page controls it (Market Overview syncs it with its page-level selector). */
export function IntradayChart({ range: controlled, onRangeChange }: { range?: ChartRange; onRangeChange?: (r: ChartRange) => void } = {}) {
  const [own, setOwn] = useState<ChartRange>("1D");
  const range = controlled ?? own;
  const setRange = onRangeChange ?? setOwn;
  const query = useChart(SYMBOL, range);
  return (
    <SectionCard
      id="nifty-chart"
      title={`${SYMBOL} Intraday Chart`}
      icon={ChartLine}
      query={query}
      actions={<RangePicker value={range} onChange={setRange} />}
    >
      {(c) => (c.points.length < 2 ? <Empty>No {SYMBOL} price data for this period</Empty> : <PriceChart chart={c} range={range} />)}
    </SectionCard>
  );
}

export function RangePicker({ value, onChange, label = "Timeframe" }: { value: ChartRange; onChange: (r: ChartRange) => void; label?: string }) {
  return (
    <div role="group" aria-label={label} className="flex rounded-lg bg-muted p-0.5">
      {chartRanges.map((r) => (
        <button
          key={r}
          type="button"
          aria-pressed={r === value}
          onClick={() => onChange(r)}
          className={cn(
            "rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
            r === value ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground",
          )}
        >
          {r}
        </button>
      ))}
    </div>
  );
}

function PriceChart({ chart, range }: { chart: Chart; range: ChartRange }) {
  const volume = new Map(chart.volume.map((p) => [p.t, p.v]));
  const rows: Row[] = chart.points.map((p) => ({ t: p.t, v: p.v, vol: volume.get(p.t) }));
  const first = rows[0].v;
  const last = chart.last ?? rows[rows.length - 1].v;
  const up = last >= first;
  const color = up ? "var(--color-up)" : "var(--color-down)";
  const maxVol = Math.max(0, ...chart.volume.map((p) => p.v));
  const fmt = tickFormats[range];
  return (
    <figure aria-label={`${chart.symbol} ${range}: last ${formatPrice(last)}, ${up ? "up" : "down"} from ${formatPrice(first)}`}>
      <div className="h-72">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={rows} margin={{ top: 8, right: 64, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="nifty-area" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity={0.35} />
                <stop offset="100%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid stroke="var(--color-border)" strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="t"
              type="number"
              scale="time"
              domain={["dataMin", "dataMax"]}
              tickFormatter={(t: number) => fmt.format(new Date(t))}
              tick={{ fill: "var(--color-muted-foreground)", fontSize: 12 }}
              stroke="var(--color-border)"
              minTickGap={40}
            />
            <YAxis
              yAxisId="price"
              domain={["auto", "auto"]}
              tickFormatter={(v: number) => axisPrice.format(v)}
              tick={{ fill: "var(--color-muted-foreground)", fontSize: 12 }}
              stroke="var(--color-border)"
              width={60}
            />
            {/* Volume bars use the bottom quarter of the plot. */}
            <YAxis yAxisId="vol" hide domain={[0, maxVol > 0 ? maxVol * 4 : 1]} />
            <Bar yAxisId="vol" dataKey="vol" fill="var(--color-muted-foreground)" fillOpacity={0.35} isAnimationActive={false} />
            <Area
              yAxisId="price"
              type="monotone"
              dataKey="v"
              stroke={color}
              strokeWidth={2}
              fill="url(#nifty-area)"
              dot={false}
              isAnimationActive={false}
            />
            <ReferenceLine
              yAxisId="price"
              y={last}
              stroke={color}
              strokeDasharray="4 4"
              label={{ value: formatPrice(last), position: "right", fill: color, fontSize: 12 }}
            />
            <Tooltip
              content={({ active, payload }) => {
                const row = active ? (payload?.[0]?.payload as Row | undefined) : undefined;
                if (!row) return null;
                return (
                  <div className="rounded-md border bg-card px-3 py-2 text-xs shadow-sm">
                    <div className="text-muted-foreground">{tooltipTime.format(new Date(row.t))} IST</div>
                    <div className="font-medium tabular-nums">{formatPrice(row.v)}</div>
                    <div className="text-muted-foreground tabular-nums">Vol {formatCount(row.vol)}</div>
                  </div>
                );
              }}
            />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      <figcaption className="mt-1 text-xs text-muted-foreground">Bars: {chart.volumeLabel.toLowerCase()} · times in IST</figcaption>
    </figure>
  );
}
