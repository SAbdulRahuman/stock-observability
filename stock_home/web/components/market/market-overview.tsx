"use client";

import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { useState } from "react";
import { Banners } from "@/components/dashboard/banners";
import { IntradayChart, RangePicker } from "@/components/home/intraday-chart";
import { IndexCard } from "@/components/home/summary-row";
import { type ChartRange, useSection } from "@/lib/api";
import { cn } from "@/lib/utils";
import { MarketBreadth } from "./breadth";
import { MostActive, TopGainers, TopLosers } from "./movers";
import { MarketNews } from "./news";
import { SectorHeatmap, SectorPerformance } from "./sectors";

const indices = [
  { symbol: "NIFTY 50", name: "NIFTY 50" },
  { symbol: "NIFTY BANK", name: "BANK NIFTY" },
  { symbol: "SENSEX", name: "SENSEX" },
  { symbol: "NIFTY NEXT 50", name: "NIFTY NEXT 50" },
  { symbol: "INDIA VIX", name: "INDIA VIX" },
];

export function MarketOverviewPage() {
  const [range, setRange] = useState<ChartRange>("1D");
  const exchange = useSection("info").data?.data.exchange;
  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Market Overview</h1>
          <p className="text-sm text-muted-foreground">Live market data, key indices, sectors, and market breadth</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <RangePicker value={range} onChange={setRange} label="Page timeframe" />
          <select
            aria-label="Exchange"
            className="h-8 rounded-lg border bg-card px-2 text-sm disabled:opacity-80"
            value={exchange ?? ""}
            disabled
            title="Only the configured exchange is available"
          >
            <option value={exchange ?? ""}>{exchange || "—"}</option>
          </select>
          <RefreshButton />
        </div>
      </div>
      <Banners />
      <IndexRow />
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.8fr)_minmax(0,1fr)]">
        <IntradayChart range={range} onRangeChange={setRange} />
        <MarketBreadth exchange={exchange} />
      </div>
      <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
        <TopGainers />
        <TopLosers />
        <SectorPerformance exchange={exchange} />
      </div>
      <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
        <SectorHeatmap exchange={exchange} />
        <MostActive />
        <MarketNews />
      </div>
    </main>
  );
}

function IndexRow() {
  const market = useSection("market");
  const env = market.data;
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
      {indices.map(({ symbol, name }) => (
        <IndexCard
          key={symbol}
          name={name}
          quote={env?.data.indices.find((q) => q.symbol === symbol)}
          loading={!env && !market.isError}
          error={!env && market.isError}
          stale={!!env && (env.stale || market.isError)}
        />
      ))}
    </div>
  );
}

function RefreshButton() {
  const client = useQueryClient();
  const fetching = useIsFetching() > 0;
  return (
    <button
      type="button"
      onClick={() => void client.invalidateQueries()}
      aria-busy={fetching}
      className="inline-flex h-8 items-center gap-1.5 rounded-lg border bg-card px-3 text-sm font-medium text-primary hover:bg-accent"
    >
      <RefreshCw className={cn("size-4", fetching && "animate-spin")} aria-hidden />
      Refresh
    </button>
  );
}
