"use client";

import { Bell, CircleCheck, Flame } from "lucide-react";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { Badge } from "@/components/ui/badge";
import { useSection } from "@/lib/api";
import { formatTimeIST } from "@/lib/format";
import { cn } from "@/lib/utils";
import { ViewAll } from "./view-all";

const MAX_ROWS = 6;

type Row = { key: string; time: string; symbol?: string; alert: string; value?: string; severity: string; firing: boolean };

const severityDot: Record<string, string> = { critical: "bg-critical", warning: "bg-warning", info: "bg-info" };

/** Firing alerts first (newest first), then alerts resolved in the last 24h to fill the panel. */
export function ActiveAlertsPanel() {
  const alerts = useSection("alerts");
  const timeline = useSection("timeline");
  const count = alerts.data ? Object.values(alerts.data.data.bySeverity).reduce((n, v) => n + v, 0) : null;
  return (
    <SectionCard
      id="active-alerts"
      title="Active Alerts"
      icon={Bell}
      query={alerts}
      badge={
        count != null && (
          <Badge variant={count > 0 ? "critical" : "default"} aria-label={`${count} active alerts`}>
            {count}
          </Badge>
        )
      }
      actions={<ViewAll page="Alerts" fallbackLinkId="alertmanager" label="alerts" />}
    >
      {(a) => {
        const firing: Row[] = [...a.items]
          .sort((x, y) => y.startsAt.localeCompare(x.startsAt))
          .map((it) => ({
            key: it.fingerprint,
            time: it.startsAt,
            symbol: it.symbol,
            alert: it.summary || it.name,
            value: it.value,
            severity: it.severity,
            firing: true,
          }));
        const resolved: Row[] = (timeline.data?.data.items ?? [])
          .filter((it) => it.end)
          .sort((x, y) => (y.end ?? "").localeCompare(x.end ?? ""))
          .map((it) => ({
            key: `${it.name}:${it.subject}:${it.start}`,
            time: it.start,
            symbol: it.symbol,
            alert: it.name,
            severity: it.severity,
            firing: false,
          }));
        const rows = [...firing, ...resolved].slice(0, MAX_ROWS);
        if (rows.length === 0) return <Empty>No active alerts</Empty>;
        return (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th className="py-1.5 text-left font-normal">Time</th>
                <th className="py-1.5 text-left font-normal">Symbol</th>
                <th className="py-1.5 text-left font-normal">Alert</th>
                <th className="py-1.5 text-right font-normal">Value</th>
                <th className="py-1.5 pl-3 text-left font-normal">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map((r) => (
                <tr key={r.key}>
                  <td className="py-2 whitespace-nowrap text-muted-foreground tabular-nums">
                    <span className={cn("mr-2 inline-block size-2 rounded-full", r.firing ? (severityDot[r.severity] ?? "bg-muted-foreground") : "bg-ok")} aria-hidden />
                    <span className="sr-only">{r.severity} </span>
                    {formatTimeIST(r.time)}
                  </td>
                  <td className="py-2 pr-2 font-medium">{r.symbol || "—"}</td>
                  <td className="w-full max-w-0 truncate py-2 pr-2" title={r.alert}>
                    {r.alert}
                  </td>
                  <td className="py-2 text-right tabular-nums">{r.value || "—"}</td>
                  <td className="py-2 pl-3">
                    {r.firing ? (
                      <Badge variant="critical">
                        <Flame aria-hidden />
                        Firing
                      </Badge>
                    ) : (
                      <Badge variant="ok">
                        <CircleCheck aria-hidden />
                        Resolved
                      </Badge>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        );
      }}
    </SectionCard>
  );
}
