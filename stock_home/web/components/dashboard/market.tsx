"use client";

import { ArrowDownRight, ArrowUpRight, Gauge, TrendingUp } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { type SectionData, useSection } from "@/lib/api";
import { formatDecimal, formatPercent, formatPrice, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";
import { Empty, SectionCard } from "./section-card";
import { Sparkline } from "./sparkline";

type Quote = SectionData<"movers">["gainers"][number];

/** Section 3. */
export function MarketOverview() {
  const query = useSection("market");
  return (
    <SectionCard id="market" title="Market overview" icon={TrendingUp} query={query} href="/grafana/" hrefLabel="Open Grafana">
      {(m) => {
        const adv = m.advancers ?? 0;
        const dec = m.decliners ?? 0;
        const breadth = adv + dec > 0 ? adv / (adv + dec) : null;
        return (
          <div className="space-y-4">
            {m.indices.length === 0 ? (
              <Empty>No index data yet</Empty>
            ) : (
              <ul className="divide-y">
                {m.indices.map((q) => (
                  <li key={q.symbol}>
                    <a href={q.exploreUrl} className="-mx-2 flex items-center gap-3 rounded-md px-2 py-1.5 hover:bg-accent">
                      <span className="min-w-0 flex-1 truncate text-sm font-medium">{q.symbol}</span>
                      <span className="text-sm tabular-nums">{formatPrice(q.price)}</span>
                      <Change value={q.changePercent} />
                    </a>
                  </li>
                ))}
              </ul>
            )}
            {m.spark && (
              <div>
                <div className="mb-1 text-xs text-muted-foreground">{m.spark.symbol} · today</div>
                <Sparkline points={m.spark.points} height={56} />
              </div>
            )}
            {breadth != null && (
              <div>
                <div className="mb-1 flex justify-between text-xs">
                  <span className="text-up">{adv} advancing</span>
                  <span className="text-muted-foreground">A/D {formatDecimal(m.advanceDecline)}</span>
                  <span className="text-down">{dec} declining</span>
                </div>
                <div className="flex h-2 overflow-hidden rounded-full bg-muted" role="img" aria-label={`${adv} advancing, ${dec} declining`}>
                  <div className="bg-up" style={{ width: `${breadth * 100}%` }} />
                  <div className="flex-1 bg-down" />
                </div>
              </div>
            )}
            {m.sectors.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {m.sectors.map((s) => (
                  <span
                    key={s.name}
                    className={cn("rounded-md px-2 py-0.5 text-xs tabular-nums", s.changePercent >= 0 ? "bg-up/10 text-up" : "bg-down/10 text-down")}
                    title={`${s.name}: ${formatPercent(s.changePercent)}`}
                  >
                    {s.name} {formatPercent(s.changePercent, 1)}
                  </span>
                ))}
              </div>
            )}
          </div>
        );
      }}
    </SectionCard>
  );
}

/** Section 7. */
export function TopMovers() {
  const query = useSection("movers");
  return (
    <SectionCard id="movers" title="Top movers · core equities" icon={TrendingUp} query={query}>
      {(m) => (
        <div className="grid gap-4 sm:grid-cols-2">
          <QuoteList title="Gainers" icon={<ArrowUpRight className="size-4 text-up" />} items={m.gainers} />
          <QuoteList title="Losers" icon={<ArrowDownRight className="size-4 text-down" />} items={m.losers} />
        </div>
      )}
    </SectionCard>
  );
}

function QuoteList({ title, icon, items }: { title: string; icon: React.ReactNode; items: Quote[] }) {
  return (
    <div>
      <div className="mb-1 flex items-center gap-1 text-xs font-medium text-muted-foreground">
        {icon}
        {title}
      </div>
      {items.length === 0 ? (
        <Empty>None</Empty>
      ) : (
        <ul className="divide-y">
          {items.map((q) => (
            <li key={`${q.exchange}:${q.symbol}`}>
              <a href={q.exploreUrl} className="-mx-2 flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-accent">
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium">{q.symbol}</span>
                  {q.sector && <span className="block truncate text-xs text-muted-foreground">{q.sector}</span>}
                </span>
                <span className="text-sm tabular-nums">{formatPrice(q.price)}</span>
                <Change value={q.changePercent} />
              </a>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/** Section 8. */
export function CircuitWatch() {
  const query = useSection("circuits");
  return (
    <SectionCard id="circuits" title="Circuit watch" icon={Gauge} query={query}>
      {(c) => (
        <div className="space-y-3">
          {c.hits.length > 0 && (
            <ul className="space-y-1">
              {c.hits.map((it) => (
                <li key={`${it.side}:${it.symbol}`}>
                  <a href={it.exploreUrl} className="flex items-center gap-2 rounded-md border border-critical/40 bg-critical/10 px-2 py-1.5">
                    <Badge variant="critical">{it.side} circuit hit</Badge>
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">{it.symbol}</span>
                    <span className="text-sm tabular-nums">{formatPrice(it.price)}</span>
                  </a>
                </li>
              ))}
            </ul>
          )}
          {c.near.length === 0 && c.hits.length === 0 ? (
            <Empty>No instrument within {c.thresholdPercent}% of a circuit limit</Empty>
          ) : (
            c.near.length > 0 && (
              <ul className="divide-y">
                {c.near.map((it) => (
                  <li key={`${it.side}:${it.symbol}`}>
                    <a href={it.exploreUrl} className="-mx-2 flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-accent">
                      <Badge variant={it.side === "upper" ? "up" : "down"}>{it.side}</Badge>
                      <span className="min-w-0 flex-1 truncate text-sm font-medium">{it.symbol}</span>
                      <span className="text-sm tabular-nums">{formatPrice(it.price)}</span>
                      <span className="w-20 text-right text-xs text-warning tabular-nums">{formatPercent(it.distancePercent, 2, false)} away</span>
                    </a>
                  </li>
                ))}
              </ul>
            )
          )}
        </div>
      )}
    </SectionCard>
  );
}

function Change({ value }: { value: number | null | undefined }) {
  return <span className={cn("w-20 text-right text-sm font-medium tabular-nums", trendClass(value))}>{formatPercent(value)}</span>;
}
