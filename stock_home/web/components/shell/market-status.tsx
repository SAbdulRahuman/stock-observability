"use client";

import { Clock3, Wifi } from "lucide-react";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/primitives";
import { useSection } from "@/lib/api";
import { feedSourceLabel, formatClock } from "@/lib/format";
import { cn } from "@/lib/utils";

const phases: Record<string, { label: string; variant: BadgeVariant }> = {
  open: { label: "OPEN", variant: "ok" },
  pre_open: { label: "PRE-OPEN", variant: "warning" },
  post_close: { label: "POST-CLOSE", variant: "warning" },
  closed: { label: "CLOSED", variant: "default" },
};

export function PhaseBadge({ phase }: { phase: string | undefined }) {
  const p = (phase && phases[phase]) || { label: "UNKNOWN", variant: "default" as const };
  return (
    <Badge variant={p.variant}>
      <span className="size-1.5 rounded-full bg-current" aria-hidden />
      {p.label}
    </Badge>
  );
}

export function dataSourceText(feedSource: string | undefined, open: boolean | undefined): string {
  return `${open && feedSource === "kite" ? "Live data" : "Data"} via ${feedSourceLabel(feedSource)}`;
}

/** Exchange phase, session hours and data source; used in the sidebar and the Home summary row. */
export function MarketStatusCard({ className }: { className?: string }) {
  const status = useSection("status");
  const info = useSection("info");
  const market = status.data?.data.market;
  const exchange = info.data?.data.exchange;
  return (
    <section aria-label="Market status" className={cn("rounded-xl border bg-card p-3 text-card-foreground", className)}>
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-sm font-medium">Market Status</h2>
        {market ? <PhaseBadge phase={market.phase} /> : status.isError ? <Badge>unavailable</Badge> : <Skeleton className="h-5 w-14" />}
      </div>
      <div className="mt-2 flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-xs text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <Clock3 className="size-3.5" aria-hidden />
          {exchange || "—"}
        </span>
        <span className="tabular-nums">
          {info.data ? `${formatClock(info.data.data.sessionOpen)} – ${formatClock(info.data.data.sessionClose)}` : "—"}
        </span>
      </div>
      <p className="mt-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
        <Wifi className="size-3.5 shrink-0" aria-hidden />
        <span className="truncate">{market ? dataSourceText(market.feedSource, market.open) : status.isError ? "Status unavailable" : "—"}</span>
      </p>
    </section>
  );
}
