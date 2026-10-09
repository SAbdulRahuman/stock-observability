"use client";

import { Activity, TrendingDown, TrendingUp } from "lucide-react";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { Sparkline } from "@/components/dashboard/sparkline";
import { ViewAll } from "@/components/home/view-all";
import { type Quote, useSection } from "@/lib/api";
import { formatChange, formatPercent, formatPrice, formatVolume, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";

const ROWS = 5;

/** Universe is the exporter core tier (index constituents); there is no NIFTY 50 membership label. */
export function TopGainers() {
  const query = useSection("movers");
  return (
    <SectionCard id="top-gainers" title="Top Gainers (core equities)" icon={TrendingUp} query={query} actions={<ViewAll page="Top Movers" label="gainers" />}>
      {(m) => <MoversTable items={m.gainers} empty="No gainers right now" />}
    </SectionCard>
  );
}

export function TopLosers() {
  const query = useSection("movers");
  return (
    <SectionCard id="top-losers" title="Top Losers (core equities)" icon={TrendingDown} query={query} actions={<ViewAll page="Top Movers" label="losers" />}>
      {(m) => <MoversTable items={m.losers} empty="No losers right now" />}
    </SectionCard>
  );
}

function MoversTable({ items, empty }: { items: Quote[]; empty: string }) {
  if (items.length === 0) return <Empty>{empty}</Empty>;
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-xs text-muted-foreground">
          <th className="w-6 py-1.5 text-left font-normal">#</th>
          <th className="py-1.5 text-left font-normal">Symbol</th>
          <th className="py-1.5 text-right font-normal">LTP (₹)</th>
          <th className="py-1.5 text-right font-normal">Change</th>
          <th className="py-1.5 text-right font-normal">% Change</th>
          <th className="py-1.5 pl-3 text-right font-normal">Day Chart</th>
        </tr>
      </thead>
      <tbody className="divide-y">
        {items.slice(0, ROWS).map((q, i) => (
          <tr key={`${q.exchange}:${q.symbol}`}>
            <td className="py-2 text-muted-foreground tabular-nums">{i + 1}</td>
            <td className="py-2 font-medium">{q.symbol}</td>
            <td className="py-2 text-right tabular-nums">{formatPrice(q.price)}</td>
            <td className={cn("py-2 text-right tabular-nums", trendClass(q.change))}>{formatChange(q.change)}</td>
            <td className={cn("py-2 text-right font-medium tabular-nums", trendClass(q.changePercent))}>{formatPercent(q.changePercent)}</td>
            <td className="py-1 pl-3">
              <Sparkline points={q.spark} className="ml-auto w-20" height={24} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export function MostActive() {
  const query = useSection("movers");
  return (
    <SectionCard id="most-active" title="Most Active (By Volume)" icon={Activity} query={query} actions={<ViewAll page="Top Movers" label="most active" />}>
      {(m) =>
        m.highVolume.length === 0 ? (
          <Empty>No traded volume yet</Empty>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th className="w-6 py-1.5 text-left font-normal">#</th>
                <th className="py-1.5 text-left font-normal">Symbol</th>
                <th className="py-1.5 text-right font-normal">LTP (₹)</th>
                <th className="py-1.5 text-right font-normal">Volume</th>
                <th className="py-1.5 text-right font-normal">Change</th>
                <th className="py-1.5 text-right font-normal">%</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {m.highVolume.slice(0, ROWS).map((q, i) => (
                <tr key={`${q.exchange}:${q.symbol}`}>
                  <td className="py-2 text-muted-foreground tabular-nums">{i + 1}</td>
                  <td className="py-2 font-medium">{q.symbol}</td>
                  <td className="py-2 text-right tabular-nums">{formatPrice(q.price)}</td>
                  <td className="py-2 text-right tabular-nums">{formatVolume(q.volume)}</td>
                  <td className={cn("py-2 text-right tabular-nums", trendClass(q.change))}>{formatChange(q.change)}</td>
                  <td className={cn("py-2 text-right font-medium tabular-nums", trendClass(q.changePercent))}>{formatPercent(q.changePercent)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )
      }
    </SectionCard>
  );
}
