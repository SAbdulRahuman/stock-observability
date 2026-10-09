"use client";

import { Banners } from "@/components/dashboard/banners";
import { useSettings } from "@/lib/settings";
import { cn } from "@/lib/utils";
import { ActiveAlertsPanel } from "./active-alerts";
import { IntradayChart } from "./intraday-chart";
import { QuickLinksPanel } from "./quick-links";
import { SummaryRow } from "./summary-row";
import { TopMoversPanel } from "./top-movers";
import { WatchlistPanel } from "./watchlist";

export function HomePage() {
  const [{ home }] = useSettings();
  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <Banners />
      <SummaryRow showIndices={home.indexCards} />
      <div className={cn("grid gap-4", home.topMovers && "xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]")}>
        <IntradayChart />
        {home.topMovers && <TopMoversPanel />}
      </div>
      {(home.recentAlerts || home.watchlistSnapshot) && (
        <div className={cn("grid gap-4", home.recentAlerts && home.watchlistSnapshot && "lg:grid-cols-2")}>
          {home.recentAlerts && <ActiveAlertsPanel />}
          {home.watchlistSnapshot && <WatchlistPanel />}
        </div>
      )}
      <QuickLinksPanel />
    </main>
  );
}
