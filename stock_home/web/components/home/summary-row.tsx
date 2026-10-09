"use client";

import { CircleAlert } from "lucide-react";
import { Sparkline } from "@/components/dashboard/sparkline";
import { MarketStatusCard } from "@/components/shell/market-status";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { type Quote, useSection } from "@/lib/api";
import { formatChange, formatPercent, formatPrice, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";

/** Index symbols on the Home summary row with their display names. */
export const homeIndices = [
  { symbol: "NIFTY 50", name: "NIFTY 50" },
  { symbol: "NIFTY BANK", name: "BANK NIFTY" },
  { symbol: "SENSEX", name: "SENSEX" },
  { symbol: "INDIA VIX", name: "INDIA VIX" },
];

export function SummaryRow({ showIndices = true }: { showIndices?: boolean }) {
  const market = useSection("market");
  const env = market.data;
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
      {showIndices &&
        homeIndices.map(({ symbol, name }) => (
          <IndexCard
            key={symbol}
            name={name}
            quote={env?.data.indices.find((q) => q.symbol === symbol)}
            loading={!env && !market.isError}
            error={!env && market.isError}
            stale={!!env && (env.stale || market.isError)}
          />
        ))}
      <MarketStatusCard className="p-4" />
    </div>
  );
}

export function IndexCard({ name, quote, loading, error, stale }: { name: string; quote?: Quote; loading: boolean; error: boolean; stale: boolean }) {
  return (
    <Card aria-label={name} className="min-w-0 p-4">
      <div className="flex items-center justify-between gap-2">
        <h2 className="truncate text-sm text-muted-foreground">{name}</h2>
        {stale && <Badge variant="warning">stale</Badge>}
      </div>
      {loading ? (
        <div className="mt-2 space-y-2" aria-busy>
          <Skeleton className="h-7 w-28" />
          <Skeleton className="h-4 w-24" />
        </div>
      ) : error ? (
        <p className="mt-3 flex items-center gap-1.5 text-sm text-muted-foreground">
          <CircleAlert className="size-4 text-warning" aria-hidden />
          Unavailable
        </p>
      ) : (
        <div className="mt-1 flex items-end justify-between gap-2">
          <div className="min-w-0">
            <div className="text-2xl font-semibold tabular-nums">{formatPrice(quote?.price)}</div>
            <div className={cn("text-sm font-medium whitespace-nowrap tabular-nums", trendClass(quote?.changePercent))}>
              {formatChange(quote?.change)} ({formatPercent(quote?.changePercent)})
            </div>
          </div>
          <Sparkline points={quote?.spark} className="w-24 shrink-0" height={44} />
        </div>
      )}
    </Card>
  );
}
