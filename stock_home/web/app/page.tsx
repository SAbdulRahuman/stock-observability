import { Banners } from "@/components/dashboard/banners";
import { ActiveAlerts, FeedHealth, Portfolio } from "@/components/dashboard/feed-alerts-portfolio";
import { Header } from "@/components/dashboard/header";
import { CircuitWatch, MarketOverview, TopMovers } from "@/components/dashboard/market";
import { AlertTimeline, QuickLinks, SystemHealth } from "@/components/dashboard/system";

export default function Home() {
  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-5 p-4 md:p-6">
      <Header />
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
      <footer className="pb-2 text-center text-xs text-muted-foreground">
        Read-only view · data from Prometheus and Alertmanager · times in IST
      </footer>
    </main>
  );
}
