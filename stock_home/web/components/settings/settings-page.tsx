"use client";

import { Archive, Bell, Database, LayoutDashboard, Monitor, Plug, Settings, TriangleAlert, TrendingUp } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useState } from "react";
import { cn } from "@/lib/utils";
import { BackupCard, DangerCard } from "./backup";
import { ApplicationCard, DashboardCard, DisplayCard, IntegrationsCard, MarketCard, type Notify, NotificationCard, RetentionCard } from "./cards";

type CardId = "application" | "market" | "display" | "notifications" | "dashboard" | "integrations" | "retention" | "backup" | "danger";
const ALL: CardId[] = ["application", "market", "display", "notifications", "dashboard", "integrations", "retention", "backup", "danger"];

/** The wireframe details only General; the other tabs show the General cards that belong to them. */
const TABS: { id: string; label: string; cards: CardId[] }[] = [
  { id: "general", label: "General", cards: ALL },
  { id: "data-sources", label: "Data Sources", cards: ["market", "integrations", "retention"] },
  { id: "alert-rules", label: "Alert Rules", cards: ["notifications", "retention", "backup"] },
  { id: "watchlist", label: "Watchlist", cards: ["backup"] },
  { id: "preferences", label: "User Preferences", cards: ["application", "display", "dashboard"] },
  { id: "integrations", label: "Integrations", cards: ["integrations"] },
  { id: "system", label: "System", cards: ["retention", "backup", "danger"] },
];

export function SettingsPage() {
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname() ?? "/settings";
  const tab = TABS.find((t) => t.id === params?.get("tab")) ?? TABS[0];
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);
  const notify: Notify = useCallback((ok, text) => setMessage({ ok, text }), []);

  const card: Record<CardId, React.ReactNode> = {
    application: <ApplicationCard key="application" icon={Settings} notify={notify} />,
    market: <MarketCard key="market" icon={TrendingUp} />,
    display: <DisplayCard key="display" icon={Monitor} notify={notify} />,
    notifications: <NotificationCard key="notifications" icon={Bell} />,
    dashboard: <DashboardCard key="dashboard" icon={LayoutDashboard} notify={notify} />,
    integrations: <IntegrationsCard key="integrations" icon={Plug} />,
    retention: <RetentionCard key="retention" icon={Database} />,
    backup: <BackupCard key="backup" icon={Archive} notify={notify} />,
    danger: <DangerCard key="danger" icon={TriangleAlert} notify={notify} />,
  };

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
            <Settings className="size-6 text-primary" aria-hidden />
            Settings
          </h1>
          <p className="text-sm text-muted-foreground">Configure Stock Home, data sources, alerts, and user preferences</p>
        </div>
        <p role="status" className={cn("text-sm", message ? (message.ok ? "text-ok" : "text-critical") : "text-muted-foreground")}>
          {message?.text ?? "Changes save automatically in this browser."}
        </p>
      </div>

      <div role="tablist" aria-label="Settings categories" className="flex flex-wrap gap-2">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={t.id === tab.id}
            onClick={() => router.replace(t.id === "general" ? pathname : `${pathname}?tab=${t.id}`, { scroll: false })}
            className={cn(
              "rounded-lg border px-3 py-2 text-sm font-medium transition-colors",
              t.id === tab.id ? "border-primary bg-primary text-primary-foreground" : "bg-card text-muted-foreground hover:text-foreground",
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <section role="tabpanel" aria-label={`${tab.label} settings`} className="grid items-start gap-4 md:grid-cols-2 xl:grid-cols-3">
        {tab.cards.map((id) => card[id])}
      </section>
    </main>
  );
}
