"use client";

import type { UseQueryResult } from "@tanstack/react-query";
import { Activity, ArrowDownRight, ArrowUpRight, BarChart3, ChartColumn, Flame, Layers, TrendingDown, TrendingUp } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { Sparkline } from "@/components/dashboard/sparkline";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import type { Envelope, MoversScan, Quote, RangeItem } from "@/lib/api";
import { formatChange, formatCount, formatPercent, formatPrice, formatVolume, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";

type Q = UseQueryResult<Envelope<MoversScan>>;

function SymbolLink({ symbol }: { symbol: string }) {
  return (
    <Link href={`/explorer?symbol=${encodeURIComponent(symbol)}`} className="font-medium hover:underline" title={`${symbol} details`}>
      {symbol}
    </Link>
  );
}

export function SummaryCards({ query }: { query: Q }) {
  const d = query.data?.data;
  if (!d) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) =>
          query.isError ? (
            <Card key={i} className="p-4 text-sm text-muted-foreground">
              Unavailable
            </Card>
          ) : (
            <Skeleton key={i} className="h-24" />
          ),
        )}
      </div>
    );
  }
  const best = d.gainers[0];
  const worst = d.losers[0];
  const active = d.mostActive[0];
  const vol = d.volumeChange[0];
  const cards = [
    { label: "Top Gainers", icon: ArrowUpRight, tone: "text-up", value: formatCount(d.advancers), hint: `of ${formatCount(d.total)} · best ${best ? `${best.symbol} ${formatPercent(best.changePercent)}` : "—"}` },
    { label: "Top Losers", icon: ArrowDownRight, tone: "text-down", value: formatCount(d.decliners), hint: `of ${formatCount(d.total)} · worst ${worst ? `${worst.symbol} ${formatPercent(worst.changePercent)}` : "—"}` },
    { label: "Most Active", icon: Activity, tone: "text-info", value: active?.symbol ?? "—", hint: active ? `${formatVolume(active.volume)} shares traded` : "No traded volume yet" },
    { label: "Highest Volume Change", icon: Flame, tone: "text-info", value: vol?.symbol ?? "—", hint: vol ? `${vol.relativeVolume.toFixed(2)}x usual volume pace` : "No relative-volume data" },
  ];
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      {cards.map((c) => {
        const Icon = c.icon;
        return (
          <Card key={c.label} aria-label={`${c.label} summary`} className="p-4">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Icon className={cn("size-4", c.tone)} aria-hidden /> {c.label}
            </div>
            <div className={cn("mt-1 truncate text-2xl font-semibold tabular-nums", c.tone)}>{c.value}</div>
            <div className="truncate text-xs text-muted-foreground">{c.hint}</div>
          </Card>
        );
      })}
    </div>
  );
}

function ViewAllToggle({ open, onToggle, count }: { open: boolean; onToggle: () => void; count: number }) {
  if (count <= 5) return null;
  return (
    <button type="button" onClick={onToggle} className="text-xs font-medium text-primary hover:underline" aria-expanded={open}>
      {open ? "Show top 5" : `View All (${count})`}
    </button>
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
        {items.map((q, i) => (
          <tr key={`${q.exchange}:${q.symbol}`}>
            <td className="py-2 text-muted-foreground tabular-nums">{i + 1}</td>
            <td className="py-2">
              <SymbolLink symbol={q.symbol} />
            </td>
            <td className="py-2 text-right tabular-nums">{formatPrice(q.price)}</td>
            <td className={cn("py-2 text-right tabular-nums", trendClass(q.change))}>{formatChange(q.change)}</td>
            <td className={cn("py-2 text-right font-medium tabular-nums", trendClass(q.changePercent))}>{formatPercent(q.changePercent)}</td>
            <td className="py-1 pl-3">
              <Sparkline points={q.spark} className="ml-auto w-24" height={24} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export function GainersLosers({ query, kind }: { query: Q; kind: "gainers" | "losers" }) {
  const [all, setAll] = useState(false);
  const gain = kind === "gainers";
  const count = query.data?.data[kind].length ?? 0;
  return (
    <SectionCard
      id={`top-${kind}`}
      title={gain ? "Top Gainers" : "Top Losers"}
      icon={gain ? TrendingUp : TrendingDown}
      query={query}
      actions={<ViewAllToggle open={all} onToggle={() => setAll((v) => !v)} count={count} />}
    >
      {(d) => <MoversTable items={all ? d[kind] : d[kind].slice(0, 5)} empty={gain ? "No gainers for these filters" : "No losers for these filters"} />}
    </SectionCard>
  );
}

export function MostActiveByVolume({ query }: { query: Q }) {
  return (
    <SectionCard id="most-active-volume" title="Most Active by Volume" icon={BarChart3} query={query}>
      {(d) =>
        d.mostActive.length === 0 ? (
          <Empty>No traded volume yet</Empty>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th className="w-6 py-1.5 text-left font-normal">#</th>
                <th className="py-1.5 text-left font-normal">Symbol</th>
                <th className="py-1.5 text-right font-normal">Volume</th>
                <th className="py-1.5 text-right font-normal">LTP (₹)</th>
                <th className="py-1.5 text-right font-normal">%</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {d.mostActive.slice(0, 5).map((q, i) => (
                <tr key={q.symbol}>
                  <td className="py-2 text-muted-foreground tabular-nums">{i + 1}</td>
                  <td className="py-2">
                    <SymbolLink symbol={q.symbol} />
                  </td>
                  <td className="py-2 text-right tabular-nums">{formatVolume(q.volume)}</td>
                  <td className="py-2 text-right tabular-nums">{formatPrice(q.price)}</td>
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

/** 52-week lists cover instruments with reference data (exporter `reference` collector, core tier). */
export function Range52({ query, side }: { query: Q; side: "high" | "low" }) {
  const high = side === "high";
  return (
    <SectionCard id={`range-52w-${side}`} title={high ? "52-Week Highs" : "52-Week Lows"} icon={ChartColumn} query={query}>
      {(d) => {
        const items: RangeItem[] = high ? d.high52 : d.low52;
        if (items.length === 0) return <Empty>No 52-week range data</Empty>;
        return (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th className="py-1.5 text-left font-normal">Symbol</th>
                <th className="py-1.5 text-right font-normal">LTP (₹)</th>
                <th className="py-1.5 text-right font-normal">{high ? "52W High" : "52W Low"}</th>
                <th className="py-1.5 text-right font-normal">{high ? "From high" : "From low"}</th>
                <th className="w-20 py-1.5 pl-3 text-left font-normal">Range</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {items.map((r) => (
                <tr key={r.symbol}>
                  <td className="py-2">
                    <SymbolLink symbol={r.symbol} />
                  </td>
                  <td className="py-2 text-right tabular-nums">{formatPrice(r.price)}</td>
                  <td className="py-2 text-right tabular-nums">{formatPrice(high ? r.high52 : r.low52)}</td>
                  <td className={cn("py-2 text-right font-medium tabular-nums", trendClass(r.distancePercent))}>{formatPercent(r.distancePercent)}</td>
                  <td className="py-2 pl-3">
                    <div
                      className="h-1.5 w-full rounded-full bg-muted"
                      role="meter"
                      aria-label={`${r.symbol} position in 52-week range`}
                      aria-valuemin={0}
                      aria-valuemax={100}
                      aria-valuenow={Math.round(r.position * 100)}
                    >
                      <div className={cn("h-full rounded-full", high ? "bg-up" : "bg-down")} style={{ width: `${r.position * 100}%` }} />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        );
      }}
    </SectionCard>
  );
}

export function Distribution({ query }: { query: Q }) {
  return (
    <SectionCard id="change-distribution" title="Daily % Change Distribution" icon={ChartColumn} query={query}>
      {(d) => {
        const max = Math.max(1, ...d.distribution.map((b) => b.count));
        if (d.total === 0) return <Empty>No instruments for these filters</Empty>;
        return (
          <div>
            <ul className="flex h-40 items-end gap-1" aria-label="Day change distribution">
              {d.distribution.map((b) => {
                const zero = b.from === 0 && b.to === 0;
                const tone = zero ? "bg-muted-foreground" : b.to != null && b.to <= 0 ? "bg-down" : "bg-up";
                return (
                  <li key={b.label} className="flex h-full flex-1 flex-col items-center justify-end gap-1" title={`${b.label}: ${b.count}`}>
                    <span className="text-[10px] text-muted-foreground tabular-nums">{b.count}</span>
                    <span className={cn("w-full rounded-t-sm", tone)} style={{ height: `${(b.count / max) * 100}%`, minHeight: b.count > 0 ? 2 : 0 }} />
                  </li>
                );
              })}
            </ul>
            <div className="mt-1 flex gap-1 text-[10px] text-muted-foreground">
              {d.distribution.map((b) => (
                <span key={b.label} className="flex-1 truncate text-center" title={b.label}>
                  {b.label.replace(" to ", "…")}
                </span>
              ))}
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {formatCount(d.decliners)} down · {formatCount(d.unchanged)} unchanged · {formatCount(d.advancers)} up
            </p>
          </div>
        );
      }}
    </SectionCard>
  );
}

export function SectorRanking({ query }: { query: Q }) {
  return (
    <SectionCard id="sector-movers" title="Top Movers by Sector" icon={Layers} query={query}>
      {(d) => {
        if (d.sectors.length === 0) return <Empty>No sector data</Empty>;
        const maxAbs = Math.max(0.01, ...d.sectors.map((s) => Math.abs(s.changePercent)));
        return (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th className="py-1.5 text-left font-normal">Sector</th>
                <th className="py-1.5 text-right font-normal">Avg %</th>
                <th className="w-24 py-1.5 pl-3 font-normal" />
                <th className="py-1.5 pl-3 text-left font-normal">Top gainer</th>
                <th className="py-1.5 pl-3 text-left font-normal">Top loser</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {d.sectors.slice(0, 10).map((s) => (
                <tr key={s.name}>
                  <td className="max-w-0 truncate py-1.5 font-medium" style={{ width: "28%" }} title={`${s.name} · ${s.count} instruments`}>
                    {s.name}
                  </td>
                  <td className={cn("py-1.5 text-right tabular-nums", trendClass(s.changePercent))}>{formatPercent(s.changePercent)}</td>
                  <td className="py-1.5 pl-3">
                    <div className="h-1.5 rounded-full bg-muted" aria-hidden>
                      <div className={cn("h-full rounded-full", s.changePercent >= 0 ? "bg-up" : "bg-down")} style={{ width: `${(Math.abs(s.changePercent) / maxAbs) * 100}%` }} />
                    </div>
                  </td>
                  <td className="py-1.5 pl-3 whitespace-nowrap">
                    {s.gainer ? (
                      <>
                        {s.gainer.symbol} <span className="text-up tabular-nums">{formatPercent(s.gainer.changePercent)}</span>
                      </>
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className="py-1.5 pl-3 whitespace-nowrap">
                    {s.loser ? (
                      <>
                        {s.loser.symbol} <span className="text-down tabular-nums">{formatPercent(s.loser.changePercent)}</span>
                      </>
                    ) : (
                      "—"
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        );
      }}
    </SectionCard>
  );
}
