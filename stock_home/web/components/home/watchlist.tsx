"use client";

import { Star } from "lucide-react";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { Sparkline } from "@/components/dashboard/sparkline";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { useQuotes } from "@/lib/api";
import { formatChange, formatPercent, formatPrice, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";
import { useWatchlist } from "@/lib/watchlist";
import { ViewAll } from "./view-all";

const ROWS = 5;

export function WatchlistPanel() {
  const symbols = useWatchlist();
  const shown = (symbols ?? []).slice(0, ROWS);
  const query = useQuotes(shown);
  const actions = <ViewAll page="Watchlist" label="watchlist" />;
  if (shown.length === 0) {
    return (
      <Card id="watchlist" aria-labelledby="watchlist-title" className="min-w-0">
        <CardHeader>
          <Star className="size-4 text-warning" aria-hidden />
          <CardTitle id="watchlist-title">Watchlist</CardTitle>
          <div className="ml-auto">{actions}</div>
        </CardHeader>
        <CardContent>
          {symbols == null ? (
            <Skeleton className="h-24 w-full" />
          ) : (
            <Empty>
              Your watchlist is empty.
              <br />
              Symbols added on the Watchlist page appear here.
            </Empty>
          )}
        </CardContent>
      </Card>
    );
  }
  return (
    <SectionCard id="watchlist" title="Watchlist" icon={Star} query={query} actions={actions}>
      {(d) => (
        <table className="w-full text-sm">
          <thead>
            <tr className="text-xs text-muted-foreground">
              <th className="py-1.5 text-left font-normal">Symbol</th>
              <th className="py-1.5 text-right font-normal">LTP (₹)</th>
              <th className="py-1.5 text-right font-normal">Change</th>
              <th className="py-1.5 text-right font-normal">%</th>
              <th className="py-1.5 pl-4 text-right font-normal">Mini Chart</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {d.items.map((q) => (
              <tr key={q.symbol}>
                <td className="py-2 font-medium">{q.symbol}</td>
                <td className="py-2 text-right tabular-nums">{formatPrice(q.price)}</td>
                <td className={cn("py-2 text-right tabular-nums", trendClass(q.change))}>{formatChange(q.change)}</td>
                <td className={cn("py-2 text-right font-medium tabular-nums", trendClass(q.changePercent))}>{formatPercent(q.changePercent)}</td>
                <td className="py-1 pl-4">
                  <Sparkline points={q.spark} className="ml-auto w-24" height={28} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </SectionCard>
  );
}
