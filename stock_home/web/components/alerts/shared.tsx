"use client";

import { BellOff, CheckCheck, CircleCheck, Flame, Info, OctagonAlert, ShieldMinus, TriangleAlert } from "lucide-react";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import type { AlertRow } from "@/lib/api";

const severities: Record<string, { label: string; variant: BadgeVariant; icon: typeof Info }> = {
  critical: { label: "Critical", variant: "critical", icon: OctagonAlert },
  warning: { label: "Warning", variant: "warning", icon: TriangleAlert },
  info: { label: "Info", variant: "info", icon: Info },
};

export function SeverityBadge({ severity }: { severity: string }) {
  const s = severities[severity] ?? { label: severity || "none", variant: "default" as const, icon: Info };
  const Icon = s.icon;
  return (
    <Badge variant={s.variant}>
      <Icon aria-hidden />
      {s.label}
    </Badge>
  );
}

const statuses: Record<AlertRow["status"], { label: string; variant: BadgeVariant; icon: typeof Info }> = {
  firing: { label: "Firing", variant: "critical", icon: Flame },
  acknowledged: { label: "Acknowledged", variant: "info", icon: CheckCheck },
  silenced: { label: "Silenced", variant: "default", icon: BellOff },
  inhibited: { label: "Inhibited", variant: "default", icon: ShieldMinus },
  resolved: { label: "Resolved", variant: "ok", icon: CircleCheck },
};

export const statusLabel = (s: AlertRow["status"]) => statuses[s].label;

export function StatusBadge({ status }: { status: AlertRow["status"] }) {
  const s = statuses[status];
  const Icon = s.icon;
  return (
    <Badge variant={s.variant}>
      <Icon aria-hidden />
      {s.label}
    </Badge>
  );
}

/** Only firing alerts can be acknowledged or silenced; the rest are already suppressed or over. */
export const silenceable = (r: AlertRow) => r.status === "firing" && !!r.fingerprint;

export type Filters = { search: string; severity: string; status: string; source: string; range: string };
export const defaultFilters: Filters = { search: "", severity: "all", status: "all", source: "all", range: "all" };
export const timeRanges: Record<string, { label: string; ms: number | null }> = {
  "1h": { label: "Last 1 hour", ms: 3_600_000 },
  "6h": { label: "Last 6 hours", ms: 6 * 3_600_000 },
  "24h": { label: "Last 24 hours", ms: 24 * 3_600_000 },
  all: { label: "All time", ms: null },
};

export function applyFilters(rows: AlertRow[], f: Filters, now: number): AlertRow[] {
  const term = f.search.trim().toLowerCase();
  const window = timeRanges[f.range]?.ms ?? null;
  return rows.filter(
    (r) =>
      (!term || [r.name, r.summary, r.symbol, r.description].some((v) => v?.toLowerCase().includes(term))) &&
      (f.severity === "all" || r.severity === f.severity) &&
      (f.status === "all" || r.status === f.status) &&
      (f.source === "all" || r.source === f.source) &&
      (window == null || now - new Date(r.startsAt).getTime() <= window),
  );
}

export const ruleText = (r: AlertRow) => r.summary || r.name;
