"use client";

import { Activity, ArrowUpRight, Database, KeyRound, Radio, Search, Server, ShieldCheck } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback } from "react";
import { KiteCard } from "@/components/kite/kite-card";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useSection } from "@/lib/api";
import { type KiteSetup, useKiteSetup } from "@/lib/kite";
import { cn } from "@/lib/utils";

type Integration = { id: string; name: string; configured: boolean; endpoint?: string; auth: string; note?: string };
type Source = { id: string; name: string; category: string; icon: typeof Activity; description: string };
const SOURCES: Source[] = [
  { id: "kite", name: "Zerodha Kite Connect", category: "Market data", icon: Radio, description: "Live NSE prices, depth and account data from your Kite Connect app." },
  { id: "prometheus", name: "Prometheus", category: "Metrics database", icon: Activity, description: "Metrics storage and the rules that power market indicators and alerts." },
  { id: "alertmanager", name: "Alertmanager", category: "Alerting", icon: ShieldCheck, description: "Alert grouping, silences and notification routing." },
  { id: "grafana", name: "Grafana", category: "Dashboards", icon: Database, description: "Explore detailed dashboards and Prometheus queries." },
  { id: "openbb", name: "OpenBB", category: "Research", icon: Search, description: "Company profiles, fundamentals, news and additional research data." },
];

function kiteStatus(k?: KiteSetup): { label: string; variant: BadgeVariant } {
  if (!k?.available) return { label: "Unavailable", variant: "default" };
  if (!k.configured) return { label: "Not configured", variant: "warning" };
  if (!k.sessionValid) return { label: "Login required", variant: "critical" };
  return { label: "Connected", variant: "ok" };
}

/** Grafana-style source catalog with a selected source configuration panel. */
export function DataSourcesPage({ notify }: { notify: (ok: boolean, text: string) => void }) {
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname() ?? "/settings";
  const selected = SOURCES.some((s) => s.id === params?.get("source")) ? params?.get("source") ?? "kite" : "kite";
  const integrations = useSection("integrations");
  const kite = useKiteSetup(30_000);
  const integrationByID = new Map((integrations.data?.data ?? []).map((i: Integration) => [i.id, i]));

  const select = useCallback((id: string) => {
    const next = new URLSearchParams(params?.toString());
    next.set("tab", "data-sources");
    next.set("source", id);
    router.replace(`${pathname}?${next.toString()}`, { scroll: false });
  }, [params, pathname, router]);

  return (
    <div className="grid min-w-0 items-start gap-4 xl:grid-cols-[minmax(17rem,0.8fr)_minmax(0,1.6fr)]">
      <Card className="overflow-hidden">
        <div className="border-b px-4 py-3">
          <h2 className="font-semibold">Data sources</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">Connections used by this installation</p>
        </div>
        <ul aria-label="Configured data sources" className="divide-y">
          {SOURCES.map((source) => {
            const Icon = source.icon;
            const integration = integrationByID.get(source.id);
            const status = source.id === "kite" ? kiteStatus(kite.data) : integration?.configured
              ? { label: "Configured", variant: "info" as const }
              : { label: "Not configured", variant: "default" as const };
            return (
              <li key={source.id}>
                <button
                  type="button"
                  aria-current={selected === source.id ? "page" : undefined}
                  onClick={() => select(source.id)}
                  className={cn(
                    "flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50 focus-visible:outline-2 focus-visible:outline-ring",
                    selected === source.id && "bg-primary/5",
                  )}
                >
                  <Icon className={cn("mt-0.5 size-4 shrink-0", selected === source.id ? "text-primary" : "text-muted-foreground")} aria-hidden />
                  <span className="min-w-0 flex-1">
                    <span className="flex flex-wrap items-center justify-between gap-2">
                      <span className="font-medium">{source.name}</span>
                      <Badge variant={status.variant}>{status.label}</Badge>
                    </span>
                    <span className="mt-0.5 block text-xs text-muted-foreground">{source.category}</span>
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      </Card>

      {selected === "kite" ? (
        <KiteCard icon={Radio} notify={notify} />
      ) : (
        <SourceDetails source={SOURCES.find((s) => s.id === selected)!} integration={integrationByID.get(selected)} loading={integrations.isLoading} />
      )}
    </div>
  );
}

function SourceDetails({ source, integration, loading }: { source: Source; integration?: Integration; loading: boolean }) {
  const Icon = source.icon;
  const endpoint = integration?.endpoint;
  return (
    <Card className="min-w-0">
      <header className="flex flex-wrap items-start gap-3 border-b px-5 py-4">
        <Icon className="mt-0.5 size-5 text-primary" aria-hidden />
        <div className="min-w-0 flex-1">
          <h2 className="font-semibold">{source.name}</h2>
          <p className="text-sm text-muted-foreground">{source.description}</p>
        </div>
        <Badge variant={integration?.configured ? "ok" : "default"}>{loading ? "Checking…" : integration?.configured ? "Connected" : "Not configured"}</Badge>
      </header>
      <div className="space-y-4 p-5 text-sm">
        <p>{integration?.note ?? "This connection is configured by the deployment and is read-only here."}</p>
        <dl className="grid grid-cols-[8rem_minmax(0,1fr)] gap-x-3 gap-y-2">
          <dt className="text-muted-foreground">Endpoint</dt>
          <dd className="min-w-0 break-all font-mono text-xs">{endpoint || "Set by deployment"}</dd>
          <dt className="text-muted-foreground">Authentication</dt>
          <dd>{integration?.auth ?? "Managed by deployment"}</dd>
        </dl>
        {endpoint && <Button asChild size="sm" variant="outline"><a href={endpoint} target="_blank" rel="noreferrer">Open endpoint <ArrowUpRight /></a></Button>}
        <p className="flex items-center gap-2 border-t pt-3 text-xs text-muted-foreground"><Server className="size-3.5" aria-hidden /> To change this source, update the stock_home deployment configuration.</p>
      </div>
    </Card>
  );
}
