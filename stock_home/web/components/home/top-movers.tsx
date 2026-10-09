"use client";

import { TrendingUp } from "lucide-react";
import { useState } from "react";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { type Quote, useSection } from "@/lib/api";
import { formatChange, formatCount, formatPercent, formatPrice, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";

const tabs = [
  { key: "gainers", label: "Gainers", empty: "No gainers right now" },
  { key: "losers", label: "Losers", empty: "No losers right now" },
  { key: "highVolume", label: "High Volume", empty: "No traded volume yet" },
] as const;

type Tab = (typeof tabs)[number]["key"];

/** Movers come from the exporter's core tier (index constituents), not NIFTY 50 membership. */
export function TopMoversPanel() {
  const query = useSection("movers");
  const [tab, setTab] = useState<Tab>("gainers");
  const current = tabs.find((t) => t.key === tab) ?? tabs[0];
  return (
    <SectionCard id="top-movers" title="Top Movers (core equities)" icon={TrendingUp} query={query}>
      {(m) => (
        <div className="space-y-3">
          <div role="group" aria-label="Movers list" className="flex gap-2">
            {tabs.map((t) => (
              <button
                key={t.key}
                type="button"
                aria-pressed={t.key === tab}
                onClick={() => setTab(t.key)}
                className={cn(
                  "rounded-md border px-3 py-1 text-sm font-medium transition-colors",
                  t.key === tab ? "border-transparent bg-up text-background" : "text-muted-foreground hover:bg-accent hover:text-foreground",
                )}
              >
                {t.label}
              </button>
            ))}
          </div>
          <MoversTable items={m[tab].slice(0, 5)} empty={current.empty} />
        </div>
      )}
    </SectionCard>
  );
}

function MoversTable({ items, empty }: { items: Quote[]; empty: string }) {
  if (items.length === 0) return <Empty>{empty}</Empty>;
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-xs text-muted-foreground">
          <th className="w-8 py-1.5 text-left font-normal">#</th>
          <th className="py-1.5 text-left font-normal">Symbol</th>
          <th className="py-1.5 text-right font-normal">LTP (₹)</th>
          <th className="py-1.5 text-right font-normal">Change</th>
          <th className="py-1.5 text-right font-normal">%</th>
        </tr>
      </thead>
      <tbody className="divide-y">
        {items.map((q, i) => (
          <tr key={`${q.exchange}:${q.symbol}`} title={q.volume != null ? `Volume ${formatCount(q.volume)}` : q.sector}>
            <td className="py-2 text-muted-foreground tabular-nums">{i + 1}</td>
            <td className="py-2 font-medium">{q.symbol}</td>
            <td className="py-2 text-right tabular-nums">{formatPrice(q.price)}</td>
            <td className={cn("py-2 text-right tabular-nums", trendClass(q.change))}>{formatChange(q.change)}</td>
            <td className={cn("py-2 text-right font-medium tabular-nums", trendClass(q.changePercent))}>{formatPercent(q.changePercent)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
