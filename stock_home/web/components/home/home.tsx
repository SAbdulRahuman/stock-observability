"use client";

import { Banners } from "@/components/dashboard/banners";
import { ActiveAlertsPanel } from "./active-alerts";
import { IntradayChart } from "./intraday-chart";
import { QuickLinksPanel } from "./quick-links";
import { SummaryRow } from "./summary-row";
import { TopMoversPanel } from "./top-movers";
import { WatchlistPanel } from "./watchlist";

export function HomePage() {
  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <Banners />
      <SummaryRow />
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
        <IntradayChart />
        <TopMoversPanel />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <ActiveAlertsPanel />
        <WatchlistPanel />
      </div>
      <QuickLinksPanel />
    </main>
  );
}
