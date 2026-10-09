"use client";

import { BellRing, ChartNoAxesCombined, ExternalLink, Flame, Gauge, Link2, type LucideIcon } from "lucide-react";
import { useLinkHref } from "@/components/shell/hooks";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

const cards: { id: string; title: string; subtitle: string; icon: LucideIcon; tone: string }[] = [
  { id: "grafana", title: "Grafana Dashboards", subtitle: "Visualize stock metrics", icon: Gauge, tone: "text-warning" },
  { id: "prometheus", title: "Prometheus", subtitle: "Metrics & Rules", icon: Flame, tone: "text-critical" },
  { id: "alertmanager", title: "AlertManager", subtitle: "Manage alerts", icon: BellRing, tone: "text-critical" },
  { id: "openbb", title: "OpenBB", subtitle: "Fundamental & Technical Analysis", icon: ChartNoAxesCombined, tone: "text-foreground" },
];

export function QuickLinksPanel() {
  return (
    <Card aria-labelledby="quick-links-title">
      <CardHeader>
        <Link2 className="size-4 text-primary" aria-hidden />
        <CardTitle id="quick-links-title">Quick Links</CardTitle>
      </CardHeader>
      <CardContent>
        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {cards.map((c) => (
            <li key={c.id}>
              <QuickLink {...c} />
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}

function QuickLink({ id, title, subtitle, icon: Icon, tone }: (typeof cards)[number]) {
  const href = useLinkHref(id);
  const body = (
    <>
      <Icon className={cn("size-8 shrink-0", tone)} aria-hidden />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{title}</span>
        <span className="block truncate text-xs text-muted-foreground">{href ? subtitle : "Not configured"}</span>
      </span>
      <ExternalLink className="size-4 shrink-0 text-muted-foreground" aria-hidden />
    </>
  );
  const cls = "flex items-center gap-3 rounded-lg border p-3";
  if (!href) {
    return (
      <span aria-disabled="true" className={cn(cls, "opacity-60")}>
        {body}
      </span>
    );
  }
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className={cn(cls, "transition-colors hover:bg-accent")} title={`${title} (opens in a new tab)`}>
      {body}
    </a>
  );
}
