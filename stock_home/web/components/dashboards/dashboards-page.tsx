"use client";

import { ExternalLink, LayoutDashboard, Search } from "lucide-react";
import { useState } from "react";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { useLinkHref } from "@/components/shell/hooks";
import { Badge } from "@/components/ui/badge";
import { useSection } from "@/lib/api";

/** Dashboards page: the Grafana dashboards provisioned for stock observability, opened in Grafana. */
export function DashboardsPage() {
  const query = useSection("dashboards");
  const grafana = useLinkHref("grafana");
  const [filter, setFilter] = useState("");
  const term = filter.trim().toLowerCase();
  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Dashboards</h1>
          <p className="text-sm text-muted-foreground">Grafana dashboards for market and stack observability</p>
        </div>
        <div className="relative w-full max-w-sm">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <input
            type="search"
            aria-label="Filter dashboards"
            placeholder="Filter by title, folder or tag…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="h-9 w-full rounded-lg border bg-card pr-3 pl-9 text-sm"
          />
        </div>
      </div>
      <SectionCard id="grafana-dashboards" title="Grafana dashboards" icon={LayoutDashboard} query={query} href={grafana} hrefLabel="Open Grafana">
        {(d) => {
          const items = d.items.filter(
            (x) => !term || x.title.toLowerCase().includes(term) || x.folder?.toLowerCase().includes(term) || x.tags.some((t) => t.toLowerCase().includes(term)),
          );
          if (d.items.length === 0) return <Empty>Grafana has no dashboards visible to stock_home</Empty>;
          if (items.length === 0) return <Empty>No dashboards match “{filter}”</Empty>;
          return (
            <ul className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
              {items.map((x) => (
                <li key={x.uid}>
                  <a
                    href={x.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex h-full flex-col gap-2 rounded-lg border p-4 transition-colors hover:bg-accent"
                    title={`${x.title} (opens in a new tab)`}
                  >
                    <span className="flex items-start gap-2">
                      <LayoutDashboard className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden />
                      <span className="min-w-0 flex-1 font-medium">{x.title}</span>
                      <ExternalLink className="size-4 shrink-0 text-muted-foreground" aria-hidden />
                    </span>
                    {x.folder && <span className="text-xs text-muted-foreground">Folder: {x.folder}</span>}
                    {x.tags.length > 0 && (
                      <span className="flex flex-wrap gap-1">
                        {x.tags.map((t) => (
                          <Badge key={t} variant="outline">
                            {t}
                          </Badge>
                        ))}
                      </span>
                    )}
                  </a>
                </li>
              ))}
            </ul>
          );
        }}
      </SectionCard>
    </main>
  );
}
