"use client";

import { BarChart3, LayoutGrid } from "lucide-react";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { ViewAll } from "@/components/home/view-all";
import { Badge } from "@/components/ui/badge";
import { type SectionData, useSection } from "@/lib/api";
import { formatPercent, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";

type Sector = SectionData<"market">["sectors"][number];

/** Sector averages exist for the current session only, so the period is fixed at 1D. */
function DayOnly() {
  return (
    <Badge variant="outline" title="Sector change is available for the current session only">
      1D
    </Badge>
  );
}

export function SectorPerformance({ exchange }: { exchange?: string }) {
  const query = useSection("market");
  return (
    <SectionCard
      id="sector-performance"
      title={`Sector Performance (${exchange || "—"})`}
      icon={BarChart3}
      query={query}
      actions={
        <>
          <DayOnly />
          <ViewAll page="Sector View" label="sectors" />
        </>
      }
    >
      {(m) => (m.sectors.length === 0 ? <Empty>No sector data yet</Empty> : <PerformanceBars sectors={m.sectors} />)}
    </SectionCard>
  );
}

function PerformanceBars({ sectors }: { sectors: Sector[] }) {
  const maxAbs = Math.max(...sectors.map((s) => Math.abs(s.changePercent)), 0.01);
  return (
    <ul className="space-y-1.5 text-sm">
      {sectors.map((s) => {
        const width = `${(Math.abs(s.changePercent) / maxAbs) * 50}%`;
        const up = s.changePercent >= 0;
        return (
          <li key={s.name} className="grid grid-cols-[minmax(0,8rem)_1fr_4.5rem] items-center gap-2">
            <span className="flex min-w-0 items-center gap-2">
              <span className={cn("size-2 shrink-0 rounded-full", up ? "bg-up" : "bg-down")} aria-hidden />
              <span className="truncate">{s.name}</span>
            </span>
            <span className="relative h-3" aria-hidden>
              <span className="absolute inset-y-0 left-1/2 w-px bg-border" />
              <span className={cn("absolute inset-y-0 rounded-sm", up ? "left-1/2 bg-up" : "right-1/2 bg-down")} style={{ width }} />
            </span>
            <span className={cn("text-right font-medium tabular-nums", trendClass(s.changePercent))}>{formatPercent(s.changePercent)}</span>
          </li>
        );
      })}
    </ul>
  );
}

/** Tile area follows the number of core constituents per sector; there is no market-cap source. */
export function SectorHeatmap({ exchange }: { exchange?: string }) {
  const query = useSection("market");
  return (
    <SectionCard id="sector-heatmap" title={`Sector Heatmap (${exchange || "—"})`} icon={LayoutGrid} query={query} actions={<DayOnly />}>
      {(m) =>
        m.sectors.length === 0 ? (
          <Empty>No sector data yet</Empty>
        ) : (
          <ul className="flex flex-wrap gap-1" aria-label="Sector heatmap">
            {m.sectors.map((s) => {
              const n = Math.max(1, s.constituents ?? 1);
              const strength = Math.round(25 + Math.min(Math.abs(s.changePercent) / 3, 1) * 55);
              const tone = s.changePercent >= 0 ? "var(--color-up)" : "var(--color-down)";
              return (
                <li
                  key={s.name}
                  className="flex min-h-16 min-w-20 flex-col justify-center rounded-md p-2 text-foreground"
                  style={{ flexGrow: n, flexBasis: `${n * 12}px`, background: `color-mix(in oklch, ${tone} ${strength}%, transparent)` }}
                  title={`${s.name}: ${formatPercent(s.changePercent)}${s.constituents ? ` · ${s.constituents} constituents` : ""}`}
                >
                  <span className="truncate text-xs font-semibold uppercase">{s.name}</span>
                  <span className="text-xs tabular-nums">{formatPercent(s.changePercent)}</span>
                </li>
              );
            })}
          </ul>
        )
      }
    </SectionCard>
  );
}
