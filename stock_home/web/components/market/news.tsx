"use client";

import { ArrowRight, CircleAlert, Newspaper } from "lucide-react";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { formatTimeIST } from "@/lib/format";
import { isNotConfigured, safeUrl, schemas, useCommand } from "@/lib/openbb";

/** Yahoo Finance news for the NIFTY 50 index through the stock_home OpenBB proxy (keyless provider). */
const NEWS = { symbol: "^NSEI", provider: "yfinance", limit: 6 };

export function MarketNews() {
  const news = useCommand("news/company", NEWS, schemas.news, { refetchInterval: 300_000 });
  const items = [...(news.data?.results ?? [])].sort((a, b) => b.date.localeCompare(a.date));
  return (
    <Card id="market-news" aria-labelledby="market-news-title" className="min-w-0">
      <CardHeader>
        <Newspaper className="size-4 text-muted-foreground" aria-hidden />
        <CardTitle id="market-news-title">Market News</CardTitle>
        <Link
          href={`/openbb?symbol=${encodeURIComponent(NEWS.symbol)}`}
          className="ml-auto inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
          aria-label="View all market news"
        >
          View All <ArrowRight className="size-3.5" aria-hidden />
        </Link>
      </CardHeader>
      <CardContent>
        {news.isPending ? (
          <div className="space-y-2" aria-busy>
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-4 w-4/6" />
          </div>
        ) : news.isError ? (
          <p className="flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
            <CircleAlert className="size-4 shrink-0 text-warning" aria-hidden />
            {isNotConfigured(news.error) ? "Market news unavailable — no OpenBB upstream configured" : `Market news unavailable — ${news.error.message}`}
          </p>
        ) : items.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">No market news</p>
        ) : (
          <ul className="space-y-2 text-sm">
            {items.map((n) => {
              const href = safeUrl(n.url);
              return (
                <li key={`${n.date}:${n.title}`} className="flex gap-3">
                  <span className="mt-1.5 size-2 shrink-0 rounded-full bg-info" aria-hidden />
                  <time dateTime={n.date} className="w-11 shrink-0 text-muted-foreground tabular-nums" title={new Date(n.date).toLocaleString("en-IN")}>
                    {formatTimeIST(n.date)}
                  </time>
                  {href ? (
                    <a href={href} target="_blank" rel="noopener noreferrer" className="min-w-0 truncate hover:underline" title={n.title}>
                      {n.title}
                    </a>
                  ) : (
                    <span className="min-w-0 truncate" title={n.title}>
                      {n.title}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
