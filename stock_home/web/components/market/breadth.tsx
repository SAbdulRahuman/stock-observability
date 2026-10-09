"use client";

import { PieChart as PieIcon } from "lucide-react";
import { Cell, Pie, PieChart } from "recharts";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { useSection } from "@/lib/api";
import { formatCount, formatPercent } from "@/lib/format";

const parts = [
  { key: "advancers", label: "Advances", color: "var(--color-up)", text: "text-up" },
  { key: "decliners", label: "Declines", color: "var(--color-down)", text: "text-down" },
  { key: "unchanged", label: "Unchanged", color: "var(--color-muted-foreground)", text: "text-muted-foreground" },
] as const;

/** Advancing, declining and unchanged equities from the market:* breadth rules. */
export function MarketBreadth({ exchange }: { exchange?: string }) {
  const query = useSection("market");
  return (
    <SectionCard id="market-breadth" title={`Market Breadth (${exchange || "—"})`} icon={PieIcon} query={query}>
      {(m) => {
        const rows = parts.map((p) => ({ ...p, value: m[p.key] ?? 0 }));
        const total = rows.reduce((n, r) => n + r.value, 0);
        if (m.advancers == null && m.decliners == null) return <Empty>Breadth data unavailable</Empty>;
        return (
          <div className="flex flex-wrap items-center justify-center gap-6">
            <div className="relative size-44 shrink-0">
              <PieChart width={176} height={176}>
                <Pie data={rows} dataKey="value" nameKey="label" innerRadius={58} outerRadius={84} stroke="none" isAnimationActive={false}>
                  {rows.map((r) => (
                    <Cell key={r.key} fill={r.color} />
                  ))}
                </Pie>
              </PieChart>
              <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
                <span className="text-2xl font-semibold tabular-nums">{formatCount(total)}</span>
                <span className="text-xs text-muted-foreground">Total</span>
              </div>
            </div>
            <ul className="space-y-3 text-sm">
              {rows.map((r) => (
                <li key={r.key} className="flex items-start gap-2">
                  <span className="mt-1.5 size-2.5 shrink-0 rounded-full" style={{ background: r.color }} aria-hidden />
                  <span>
                    <span className="block text-muted-foreground">{r.label}</span>
                    <span className={`block font-medium tabular-nums ${r.text}`}>
                      {formatCount(r.value)} ({total > 0 ? formatPercent((r.value / total) * 100, 1, false) : "—"})
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        );
      }}
    </SectionCard>
  );
}
