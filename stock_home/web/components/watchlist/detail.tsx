"use client";

import { BarChart3, Bell, ChartCandlestick, CircleAlert, Maximize2, Star } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { Empty } from "@/components/dashboard/section-card";
import { PriceChart, RangePicker } from "@/components/home/intraday-chart";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { type ChartRange, chartRanges, useChart, useInstrument } from "@/lib/api";
import { formatChange, formatPercent, formatPrice, formatVolume, trendClass } from "@/lib/format";
import { schemas, useCommand } from "@/lib/openbb";
import { cn } from "@/lib/utils";
import { RowActions } from "./table";

export function StockDetail({
  symbol,
  favourite,
  editable,
  onToggleFavourite,
  onSelect,
  onRemove,
}: {
  symbol: string | null;
  favourite: boolean;
  editable: boolean;
  onToggleFavourite: () => void;
  onSelect: (s: string) => void;
  onRemove: (s: string) => void;
}) {
  const [range, setRange] = useState<ChartRange>("1D");
  const inst = useInstrument(symbol);
  const chart = useChart(symbol ?? "", range, !!symbol);
  // Company name is optional: only shown when an OpenBB upstream answers.
  const profile = useCommand("equity/profile", { symbol: `${symbol}.NS`, provider: "yfinance" }, schemas.profile, { enabled: !!symbol });
  const name = profile.data?.results?.[0]?.name;

  if (!symbol) {
    return (
      <Card className="min-w-0 p-5">
        <Empty>Select a symbol to see its details</Empty>
      </Card>
    );
  }
  const d = inst.data?.data;
  return (
    <Card aria-label={`${symbol} details`} className="min-w-0 p-5">
      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <h2 className="flex flex-wrap items-baseline gap-x-3 text-xl font-semibold">
            {symbol}
            {name && <span className="truncate text-sm font-normal text-muted-foreground">{name}</span>}
          </h2>
          <div className="mt-1 flex flex-wrap gap-1.5">
            {d?.exchange && <Badge variant="outline">{d.exchange}</Badge>}
            {d?.sector && <Badge variant="outline">{d.sector}</Badge>}
          </div>
        </div>
        <button
          type="button"
          aria-pressed={favourite}
          aria-label={favourite ? "Remove from My Watchlist" : "Add to My Watchlist"}
          disabled={!editable}
          onClick={onToggleFavourite}
          className="rounded-md p-1 hover:bg-accent disabled:opacity-50"
        >
          <Star className={cn("size-5", favourite ? "fill-warning text-warning" : "text-muted-foreground")} />
        </button>
        {d && <RowActions quote={d} editable={editable} onSelect={onSelect} onRemove={onRemove} />}
      </div>

      {inst.isPending ? (
        <Skeleton className="mt-4 h-10 w-2/3" />
      ) : inst.isError ? (
        <p className="mt-4 flex items-center gap-2 text-sm text-muted-foreground">
          <CircleAlert className="size-4 text-warning" aria-hidden />
          {inst.error.message === "upstream unavailable" ? "Unavailable — upstream unavailable" : `No market data recorded for ${symbol}`}
        </p>
      ) : (
        d && (
          <div className="mt-3 flex flex-wrap items-end justify-between gap-4">
            <div>
              <div className="text-3xl font-semibold tabular-nums">{formatPrice(d.price)}</div>
              <div className={cn("text-sm font-medium tabular-nums", trendClass(d.changePercent))}>
                {formatChange(d.change)} ({formatPercent(d.changePercent)})
              </div>
            </div>
            <dl className="grid grid-cols-3 gap-4 text-sm">
              <Stat label="Day High" value={formatPrice(d.high)} />
              <Stat label="Day Low" value={formatPrice(d.low)} />
              <Stat label="Volume" value={formatVolume(d.volume)} />
            </dl>
          </div>
        )
      )}

      <div className="mt-4">
        <RangePicker value={range} onChange={setRange} ranges={chartRanges} label="Stock timeframe" />
        <div className="mt-2">
          {chart.data ? (
            chart.data.data.points.length < 2 ? (
              <Empty>No price history for this period</Empty>
            ) : (
              <PriceChart chart={chart.data.data} range={range} className="h-56" />
            )
          ) : chart.isError ? (
            <Empty>Chart unavailable — {chart.error.message}</Empty>
          ) : (
            <Skeleton className="h-56 w-full" />
          )}
        </div>
      </div>

      {d && (
        <dl className="mt-4 grid grid-cols-2 gap-x-6 gap-y-1.5 text-sm sm:grid-cols-3">
          <Row label="Open" value={formatPrice(d.open)} />
          <Row label="High" value={formatPrice(d.high)} className="text-up" />
          <Row label="Low" value={formatPrice(d.low)} className="text-down" />
          <Row label="Prev Close" value={formatPrice(d.prevClose)} />
          <Row label="Volume" value={formatVolume(d.volume)} />
          <Row label="Avg Vol (20D)" value={formatVolume(d.avgVolume20d)} />
        </dl>
      )}

      <div className="mt-4 grid grid-cols-2 gap-2 xl:grid-cols-4">
        <Button disabled title="Alert rules are defined in Prometheus and cannot be created here">
          <Bell /> Set Alert
        </Button>
        <ExternalButton href={d?.exploreUrl}>
          <BarChart3 /> View in Grafana
        </ExternalButton>
        <Button variant="outline" asChild>
          <Link href={`/openbb?symbol=${encodeURIComponent(`${symbol}.NS`)}`}>
            <ChartCandlestick /> OpenBB Analysis
          </Link>
        </Button>
        <ExternalButton href={d?.chartUrl}>
          <Maximize2 /> Full Chart
        </ExternalButton>
      </div>
    </Card>
  );
}

function ExternalButton({ href, children }: { href?: string; children: React.ReactNode }) {
  if (!href) {
    return (
      <Button variant="outline" disabled>
        {children}
      </Button>
    );
  }
  return (
    <Button variant="outline" asChild>
      <a href={href} target="_blank" rel="noopener noreferrer">
        {children}
      </a>
    </Button>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}

function Row({ label, value, className }: { label: string; value: string; className?: string }) {
  return (
    <div className="flex justify-between gap-2">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={cn("font-medium tabular-nums", className)}>{value}</dd>
    </div>
  );
}
