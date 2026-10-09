"use client";

import { BarChart3, BellOff, CheckCheck, ExternalLink } from "lucide-react";
import { Empty, useNow } from "@/components/dashboard/section-card";
import { PriceChart } from "@/components/home/intraday-chart";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { type AlertRow, type SilenceKind, useChart, useQuotes } from "@/lib/api";
import { formatDuration, formatPrice, formatTimeIST } from "@/lib/format";
import { ruleText, SeverityBadge, StatusBadge, silenceable } from "./shared";

const dateFmt = new Intl.DateTimeFormat("en-IN", { timeZone: "Asia/Kolkata", day: "numeric", month: "short" });

/** A numeric "threshold" annotation becomes the chart's trigger line; anything else is shown as text. */
function parseThreshold(t: string | undefined): number | null {
  if (!t) return null;
  const n = Number(t.replace(/[₹,\s]/g, ""));
  return Number.isFinite(n) ? n : null;
}

export function AlertDetail({
  row,
  canSilence,
  onAction,
}: {
  row: AlertRow | null;
  canSilence: boolean;
  onAction: (kind: SilenceKind, row: AlertRow) => void;
}) {
  const now = useNow(1_000);
  const symbol = row?.symbol ?? "";
  const quote = useQuotes(symbol ? [symbol] : []);
  const chart = useChart(symbol, "1D", !!symbol);
  if (!row) {
    return (
      <Card className="min-w-0 p-5">
        <Empty>Select an alert to see its details</Empty>
      </Card>
    );
  }
  const threshold = parseThreshold(row.threshold);
  const started = new Date(row.startsAt);
  const end = row.endsAt ? new Date(row.endsAt) : now;
  const price = quote.data?.data.items[0]?.price;
  const disabledReason = canSilence ? undefined : "Silences are disabled on this stock_home (--alertmanager.allow-silences)";
  return (
    <Card aria-label="Selected alert" className="min-w-0 p-5">
      <div className="flex flex-wrap items-center gap-2">
        <SeverityBadge severity={row.severity} />
        <StatusBadge status={row.status} />
      </div>
      <h2 className="mt-2 text-xl font-semibold">{row.symbol || "—"}</h2>
      <p className="font-medium">{ruleText(row)}</p>
      {row.description && <p className="mt-1 text-sm text-muted-foreground">{row.description}</p>}

      <dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1.5 text-sm">
        <dt className="text-muted-foreground">Current Price</dt>
        <dd className="text-right font-medium tabular-nums">{symbol ? (quote.isPending ? "…" : price != null ? `₹${formatPrice(price)}` : "—") : "—"}</dd>
        <dt className="text-muted-foreground">Trigger Level</dt>
        <dd className="text-right font-medium tabular-nums">{threshold != null ? `₹${formatPrice(threshold)}` : row.threshold || "—"}</dd>
        <dt className="text-muted-foreground">Value</dt>
        <dd className="text-right tabular-nums">{row.value || "—"}</dd>
        <dt className="text-muted-foreground">Started</dt>
        <dd className="text-right tabular-nums">
          {dateFmt.format(started)} {formatTimeIST(started, true)}
        </dd>
        <dt className="text-muted-foreground">Duration</dt>
        <dd className="text-right tabular-nums">{formatDuration((end.getTime() - started.getTime()) / 1000)}</dd>
        {row.silencedUntil && (
          <>
            <dt className="text-muted-foreground">Silenced until</dt>
            <dd className="text-right tabular-nums">{formatTimeIST(row.silencedUntil)}</dd>
          </>
        )}
        <dt className="text-muted-foreground">Source</dt>
        <dd className="text-right">
          {row.sourceUrl?.startsWith("http") ? (
            <a href={row.sourceUrl} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 hover:underline">
              {row.source} <ExternalLink className="size-3" aria-hidden />
            </a>
          ) : (
            row.source
          )}
        </dd>
        <dt className="text-muted-foreground">Rule</dt>
        <dd className="truncate text-right font-mono text-xs" title={row.name}>
          {row.name}
        </dd>
      </dl>

      <div className="mt-4">
        <h3 className="mb-1 text-sm font-medium">Price Chart</h3>
        {!symbol ? (
          <Empty>No symbol on this alert</Empty>
        ) : chart.data ? (
          chart.data.data.points.length < 2 ? (
            <Empty>No price history for {symbol} today</Empty>
          ) : (
            <PriceChart chart={chart.data.data} range="1D" className="h-48" threshold={threshold} />
          )
        ) : chart.isError ? (
          <Empty>Chart unavailable — {chart.error.message}</Empty>
        ) : (
          <Skeleton className="h-48 w-full" />
        )}
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        <Button onClick={() => onAction("acknowledge", row)} disabled={!canSilence || !silenceable(row)} title={disabledReason}>
          <CheckCheck /> Acknowledge
        </Button>
        <Button variant="outline" onClick={() => onAction("silence", row)} disabled={!canSilence || !silenceable(row)} title={disabledReason}>
          <BellOff /> Silence
        </Button>
        {row.exploreUrl ? (
          <Button variant="outline" asChild>
            <a href={row.exploreUrl} target="_blank" rel="noopener noreferrer">
              <BarChart3 /> Open in Grafana
            </a>
          </Button>
        ) : (
          <Button variant="outline" disabled>
            <BarChart3 /> Open in Grafana
          </Button>
        )}
      </div>
    </Card>
  );
}
