import type { Metadata } from "next";
import { DemoLink } from "@/components/demo-link";
import { TickerStrip } from "@/components/ticker-strip";
import { Card } from "@/components/ui/card";

export const metadata: Metadata = {
  title: "Live demo",
  description: "Explore the iSee Ticks product home and Grafana dashboards on a live NSE market feed.",
};

const tour = [
  { title: "Home", body: "Index cards, breadth, top movers and active alerts update every few seconds." },
  { title: "Top movers & sectors", body: "Scan gainers, losers and volume leaders, then compare sector indices." },
  { title: "Stock explorer", body: "Open any symbol for price history, depth and the KPIs computed for it." },
  { title: "Alerts", body: "See which rules fire on the market feed and how Alertmanager groups them." },
  { title: "Grafana", body: "When enabled by the deployment owner, follow a dashboard link to the underlying time series." },
];

export default function DemoPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-16">
      <h1 className="text-4xl font-bold tracking-tight">Live demo</h1>
      <p className="mt-4 text-lg text-muted-foreground">
        The admin deployment runs stock_exporter, Prometheus, Alertmanager and the product home behind one HTTPS entry point.
        Grafana is optional and may be enabled by the deployment owner.
      </p>
      <div className="mt-8 flex flex-wrap items-center gap-4">
        <DemoLink />
        <span className="text-sm text-muted-foreground">Read-only. No sign-in needed.</span>
      </div>
      <TickerStrip className="mt-10" />
      <Card className="mt-6 border-warning/30 bg-warning/10 p-5 text-sm">
        <p className="font-semibold">Live market data</p>
        <p className="mt-1 text-muted-foreground">
          Data depends on the Kite session and exchange hours; the feed can be delayed or unavailable. This read-only demo is
          for observability education, not investment advice.
        </p>
      </Card>
      <h2 className="mt-14 text-2xl font-bold tracking-tight">What to try</h2>
      <ol className="mt-6 grid gap-3">
        {tour.map((t, i) => (
          <li key={t.title} className="flex gap-4 rounded-xl border bg-card p-4">
            <span className="font-mono text-sm text-muted-foreground">{i + 1}</span>
            <div>
              <p className="font-semibold">{t.title}</p>
              <p className="text-sm text-muted-foreground">{t.body}</p>
            </div>
          </li>
        ))}
      </ol>
    </div>
  );
}
