"use client";

import type { UseQueryResult } from "@tanstack/react-query";
import { CircleAlert, ExternalLink, PlugZap } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Empty, Row, Stat } from "@/components/dashboard/section-card";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { formatCount, formatPercent, formatPrice } from "@/lib/format";
import { type Bar, type NewsItem, type Profile, type Quote, isNotConfigured, safeUrl } from "@/lib/openbb";
import { cn } from "@/lib/utils";

type Result<T> = { results?: T[] | null; provider?: string | null; warnings?: { message: string }[] | null };

const dateFmt = new Intl.DateTimeFormat("en-IN", { day: "numeric", month: "short", year: "numeric" });
const shortDateFmt = new Intl.DateTimeFormat("en-IN", { day: "numeric", month: "short" });

function fmtDate(s: string, fmt = dateFmt): string {
  const d = new Date(s);
  return Number.isNaN(d.getTime()) ? s : fmt.format(d);
}

/** Card for one OpenBB command: loading, error, not-configured and empty states. */
export function CommandCard<T>({
  id,
  title,
  icon: Icon,
  query,
  className,
  children,
}: {
  id: string;
  title: string;
  icon: LucideIcon;
  query: UseQueryResult<Result<T>>;
  className?: string;
  children: (rows: T[]) => React.ReactNode;
}) {
  const data = query.data;
  const warnings = data?.warnings ?? [];
  return (
    <Card id={id} className={cn("min-w-0", className)} aria-labelledby={`${id}-title`}>
      <CardHeader>
        <Icon className="size-4 text-muted-foreground" aria-hidden />
        <CardTitle id={`${id}-title`}>{title}</CardTitle>
        <div className="ml-auto flex items-center gap-2">
          {warnings.length > 0 && (
            <Badge variant="warning" title={warnings.map((w) => w.message).join("\n")}>
              {warnings.length} warning{warnings.length > 1 ? "s" : ""}
            </Badge>
          )}
          {data?.provider && <Badge variant="outline">{data.provider}</Badge>}
        </div>
      </CardHeader>
      <CardContent>
        {query.isError ? (
          <CommandFailure error={query.error} />
        ) : data ? (
          data.results?.length ? (
            children(data.results)
          ) : (
            <Empty>No data</Empty>
          )
        ) : (
          <div className="space-y-3" aria-busy>
            <Skeleton className="h-6 w-2/3" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function CommandFailure({ error }: { error: Error }) {
  if (isNotConfigured(error)) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
        <PlugZap className="size-4 text-warning" aria-hidden />
        OpenBB is not connected — start stock_home with --openbb.url.
      </div>
    );
  }
  return (
    <div className="flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
      <CircleAlert className="size-4 shrink-0 text-warning" aria-hidden />
      <span className="min-w-0 break-words">Unavailable — {error.message}</span>
    </div>
  );
}

export function QuoteView({ quote }: { quote: Quote }) {
  // yfinance quotes carry prev_close but no change fields.
  const change = quote.change ?? (quote.last_price != null && quote.prev_close != null ? quote.last_price - quote.prev_close : null);
  const pct =
    quote.change_percent != null
      ? quote.change_percent * 100
      : change != null && quote.prev_close
        ? (change / quote.prev_close) * 100
        : null;
  const up = (change ?? pct ?? 0) >= 0;
  return (
    <div className="space-y-4">
      <div>
        <div className="truncate text-sm text-muted-foreground">
          {quote.name ?? quote.symbol}
          {quote.exchange && ` · ${quote.exchange}`}
        </div>
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <span className="text-3xl font-semibold tabular-nums" data-testid="last-price">
            {formatPrice(quote.last_price)}
          </span>
          {quote.currency && <span className="text-sm text-muted-foreground">{quote.currency}</span>}
          <Badge variant={up ? "up" : "down"} className="text-sm tabular-nums">
            {change != null && `${change >= 0 ? "+" : ""}${formatPrice(change)} `}
            ({formatPercent(pct)})
          </Badge>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-x-6">
        <Row label="Open">{formatPrice(quote.open)}</Row>
        <Row label="Prev close">{formatPrice(quote.prev_close)}</Row>
        <Row label="Day high">{formatPrice(quote.high)}</Row>
        <Row label="Day low">{formatPrice(quote.low)}</Row>
        <Row label="52w high">{formatPrice(quote.year_high)}</Row>
        <Row label="52w low">{formatPrice(quote.year_low)}</Row>
        <Row label="Volume">{formatCount(quote.volume)}</Row>
      </div>
    </div>
  );
}

export function PriceChart({ bars }: { bars: Bar[] }) {
  const first = bars[0];
  const last = bars[bars.length - 1];
  const change = first.close ? ((last.close - first.close) / first.close) * 100 : null;
  const high = Math.max(...bars.map((b) => b.high));
  const low = Math.min(...bars.map((b) => b.low));
  const color = (change ?? 0) >= 0 ? "var(--color-up)" : "var(--color-down)";
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Stat label="Period change" value={formatPercent(change)} hint={`${fmtDate(first.date)} – ${fmtDate(last.date)}`} />
        <Stat label="Last close" value={formatPrice(last.close)} />
        <Stat label="Period high" value={formatPrice(high)} />
        <Stat label="Period low" value={formatPrice(low)} hint={`${formatCount(bars.length)} sessions`} />
      </div>
      <div className="h-72" role="img" aria-label="Closing price chart">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={bars} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="openbb-close" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity={0.3} />
                <stop offset="100%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" vertical={false} />
            <XAxis
              dataKey="date"
              tickFormatter={(d: string) => fmtDate(d, shortDateFmt)}
              minTickGap={32}
              tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }}
            />
            <YAxis
              domain={["auto", "auto"]}
              width={64}
              tickFormatter={(v: number) => formatPrice(v)}
              tick={{ fontSize: 12, fill: "var(--color-muted-foreground)" }}
            />
            <Tooltip
              labelFormatter={(d) => fmtDate(String(d))}
              formatter={(v) => [formatPrice(Number(v)), "Close"]}
              contentStyle={{ background: "var(--color-card)", border: "1px solid var(--color-border)", borderRadius: 8 }}
            />
            <Area type="monotone" dataKey="close" stroke={color} strokeWidth={1.5} fill="url(#openbb-close)" isAnimationActive={false} dot={false} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export function ProfileView({ profile }: { profile: Profile }) {
  const site = safeUrl(profile.company_url);
  const about = profile.long_description ?? profile.short_description;
  return (
    <div className="space-y-3">
      <div className="font-medium">{profile.name ?? profile.symbol}</div>
      <div>
        <Row label="Sector">{profile.sector ?? "—"}</Row>
        <Row label="Industry">{profile.industry_category ?? "—"}</Row>
        <Row label="Exchange">{profile.stock_exchange ?? "—"}</Row>
        <Row label="Country">{profile.hq_country ?? "—"}</Row>
        <Row label="Employees">{formatCount(profile.employees)}</Row>
      </div>
      {about && <p className="line-clamp-6 text-sm text-muted-foreground">{about}</p>}
      {site && (
        <a href={site} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 text-sm text-info hover:underline">
          Website <ExternalLink className="size-3" aria-hidden />
        </a>
      )}
    </div>
  );
}

export function NewsList({ items }: { items: NewsItem[] }) {
  return (
    <ul className="divide-y">
      {items.map((n) => {
        const href = safeUrl(n.url);
        return (
          <li key={`${n.url}-${n.date}`} className="py-2.5 first:pt-0 last:pb-0">
            {href ? (
              <a href={href} target="_blank" rel="noopener noreferrer" className="font-medium hover:underline">
                {n.title}
              </a>
            ) : (
              <span className="font-medium">{n.title}</span>
            )}
            <div className="text-xs text-muted-foreground">
              {fmtDate(n.date)}
              {(n.source ?? n.author) && ` · ${n.source ?? n.author}`}
            </div>
            {n.excerpt && <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{n.excerpt}</p>}
          </li>
        );
      })}
    </ul>
  );
}
