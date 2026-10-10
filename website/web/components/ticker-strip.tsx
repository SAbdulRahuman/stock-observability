"use client";

import { Activity } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { fetchTicker, formatPercent, formatPrice, type TickerQuote } from "@/lib/api";
import { usePoll } from "@/lib/use-poll";
import { cn } from "@/lib/utils";

function Item({ q }: { q: TickerQuote }) {
  const dir = q.changePercent == null ? "" : q.changePercent >= 0 ? "text-up" : "text-down";
  return (
    <li className="flex items-baseline gap-2 whitespace-nowrap">
      <span className="font-medium">{q.symbol}</span>
      <span className="font-mono text-muted-foreground">{formatPrice(q.price)}</span>
      <span className={cn("font-mono text-xs", dir)}>{formatPercent(q.changePercent)}</span>
    </li>
  );
}

/** Live strip of index levels and top movers from the demo stack. */
export function TickerStrip({ className }: { className?: string }) {
  const { data, error } = usePoll(fetchTicker, 10_000);
  const quotes = data?.available ? [...data.indices, ...data.gainers, ...data.losers] : [];
  return (
    <div className={cn("flex items-center gap-4 overflow-hidden rounded-xl border bg-card px-4 py-3 text-sm", className)} aria-live="polite">
      <span className="flex shrink-0 items-center gap-2 font-semibold">
        <Activity className="size-4 text-primary" aria-hidden />
        Live demo
      </span>
      {data?.simulated && <Badge variant="info">Simulated data</Badge>}
      {quotes.length > 0 ? (
        <ul className="flex min-w-0 gap-6 overflow-x-auto [scrollbar-width:none]" aria-label="Demo market ticker">
          {quotes.map((q, i) => (
            <Item key={`${q.symbol}-${i}`} q={q} />
          ))}
        </ul>
      ) : (
        <span className="text-muted-foreground">{data || error ? "Demo offline. Market data will be back shortly." : "Connecting to the demo…"}</span>
      )}
    </div>
  );
}
