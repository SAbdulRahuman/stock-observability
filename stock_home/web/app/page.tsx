import { Banners } from "@/components/dashboard/banners";
import { ActiveAlerts, FeedHealth, Portfolio } from "@/components/dashboard/feed-alerts-portfolio";
import { CircuitWatch, MarketOverview, TopMovers } from "@/components/dashboard/market";
import { AlertTimeline, QuickLinks, SystemHealth } from "@/components/dashboard/system";

export default function Home() {
  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-5 p-4 md:p-6">
      <Banners />
      <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
        <FeedHealth />
        <ActiveAlerts />
        <MarketOverview />
        <Portfolio />
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <TopMovers />
        <CircuitWatch />
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <AlertTimeline />
        <SystemHealth />
      </div>
      <QuickLinks />
    </main>
  );
}
