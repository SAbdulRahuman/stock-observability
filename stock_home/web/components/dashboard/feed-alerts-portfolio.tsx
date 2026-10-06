"use client";

import { Activity, Bell, Briefcase, LogIn } from "lucide-react";
import { Money } from "@/components/privacy";
import { Badge, severityVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Meter, StatusDot } from "@/components/ui/primitives";
import { useSection } from "@/lib/api";
import { formatCount, formatDecimal, formatDuration, formatPercent, formatTimeIST, relativeTime, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";
import { Empty, Row, SectionCard, Stat, useNow } from "./section-card";
import { Sparkline } from "./sparkline";

/** Section 1. */
export function FeedHealth() {
  const query = useSection("feed");
  return (
    <SectionCard id="feed" title="Feed health" icon={Activity} query={query} href={query.data?.data.exploreUrl} hrefLabel="Tick rate in Grafana Explore">
      {(f) => {
        const kite = f.source === "kite";
        const tickAge = f.lastTickAgeSeconds;
        const tickOk = tickAge != null && tickAge < 60;
        const failures = Object.entries(f.apiOutcomes10m).filter(([o]) => o !== "success").reduce((n, [, v]) => n + v, 0);
        return (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <StatusDot ok={kite ? f.sessionValid : true} />
              <span className="font-medium">
                {kite ? (f.sessionValid ? "Kite session active" : "Kite session expired") : `Source: ${f.source || "unknown"}`}
              </span>
              {f.mode && <Badge variant="outline" className="ml-auto">{f.mode}</Badge>}
            </div>
            {kite && f.sessionValid && f.sessionExpiresAt && (
              <p className="text-xs text-muted-foreground">Valid until {formatTimeIST(f.sessionExpiresAt)} IST</p>
            )}
            {kite && !f.sessionValid && (
              <Button asChild size="sm" variant="critical" className="w-full">
                <a href="/exporter/login">
                  <LogIn /> Log in to Kite
                </a>
              </Button>
            )}
            <div className="grid grid-cols-2 gap-3">
              <Stat label="Ticks / s" value={formatDecimal(f.ticksPerSecond)} />
              <Stat
                label="Last tick"
                value={<span className={tickOk ? undefined : "text-warning"}>{tickAge == null ? "—" : formatDuration(tickAge)}</span>}
                hint={tickAge == null ? undefined : "ago"}
              />
            </div>
            <Sparkline points={f.tickRateSpark} height={40} />
            <div className="divide-y">
              <Row label="Connections">{formatCount(f.connections)}</Row>
              <Row label="Subscribed instruments">{formatCount(f.subscribed)}</Row>
              <Row label="Stale core instruments">
                <span className={(f.staleRatio ?? 0) > 0.1 ? "text-warning" : undefined}>{formatPercent((f.staleRatio ?? 0) * 100, 1, false)}</span>
              </Row>
              <Row label="Reconnects / errors (1h)">
                {formatCount(f.reconnects1h)} / {formatCount(f.errors1h)}
              </Row>
              <Row label="Kite API failures (10m)">
                <span className={failures > 0 ? "text-warning" : undefined}>{formatCount(failures)}</span>
              </Row>
            </div>
          </div>
        );
      }}
    </SectionCard>
  );
}

/** Section 2. */
export function ActiveAlerts() {
  const query = useSection("alerts");
  const now = useNow(10_000);
  return (
    <SectionCard id="alerts" title="Active alerts" icon={Bell} query={query} href="/grafana/alerting/list" hrefLabel="Open Grafana Alerting">
      {(a) => (
        <div className="space-y-3">
          <div className="grid grid-cols-3 gap-2">
            {(["critical", "warning", "info"] as const).map((sev) => (
              <a
                key={sev}
                href={`/alertmanager/#/alerts?silenced=false&inhibited=false&filter=%7Bseverity%3D%22${sev}%22%7D`}
                className={cn(
                  "rounded-lg border px-3 py-2 transition-colors hover:bg-accent",
                  (a.bySeverity[sev] ?? 0) > 0 && { critical: "border-critical/40 bg-critical/10", warning: "border-warning/40 bg-warning/10", info: "border-info/40 bg-info/10" }[sev],
                )}
              >
                <div className="text-xs capitalize text-muted-foreground">{sev}</div>
                <div className="text-2xl font-semibold tabular-nums">{a.bySeverity[sev] ?? 0}</div>
              </a>
            ))}
          </div>
          <div className="flex flex-wrap gap-1.5 text-xs">
            {Object.entries(a.byCategory)
              .sort(([, x], [, y]) => y - x)
              .map(([c, n]) => (
                <Badge key={c} variant="outline">
                  {c} · {n}
                </Badge>
              ))}
            <a href="/alertmanager/#/silences" className="ml-auto text-muted-foreground hover:text-foreground">
              {a.silenced} silenced · {a.inhibited} inhibited · {a.activeSilences} silences
            </a>
          </div>
          {a.items.length === 0 ? (
            <Empty>No active alerts</Empty>
          ) : (
            <ul className="max-h-72 space-y-1 overflow-y-auto pr-1">
              {a.items.map((it) => (
                <li key={it.fingerprint}>
                  <a
                    href={it.sourceUrl || "/alertmanager/"}
                    className="flex items-start gap-2 rounded-md px-2 py-1.5 hover:bg-accent"
                    title={it.summary}
                  >
                    <Badge variant={severityVariant(it.severity)} className="mt-0.5">
                      {it.severity}
                    </Badge>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium">
                        {it.name}
                        {it.symbol && <span className="text-muted-foreground"> · {it.symbol}</span>}
                      </span>
                      {it.summary && <span className="block truncate text-xs text-muted-foreground">{it.summary}</span>}
                    </span>
                    <span className="shrink-0 text-xs text-muted-foreground">{relativeTime(it.startsAt, now)}</span>
                  </a>
                </li>
              ))}
            </ul>
          )}
          {a.truncated && <p className="text-xs text-muted-foreground">Showing the 50 most important alerts.</p>}
        </div>
      )}
    </SectionCard>
  );
}

/** Section 4. */
export function Portfolio() {
  const query = useSection("portfolio");
  return (
    <SectionCard id="portfolio" title="Portfolio" icon={Briefcase} query={query}>
      {(p) =>
        !p.enabled ? (
          <Empty>
            Portfolio data is not available.
            <br />
            Enable the exporter&apos;s portfolio collector and log in to Kite.
          </Empty>
        ) : (
          <div className="space-y-3">
            <Stat label="Holdings value" value={<Money value={p.holdingValue} compact />} />
            <div className="grid grid-cols-2 gap-3">
              <Stat
                label="Day change"
                value={<Money value={p.dayChange} signed compact className={trendClass(p.dayChange)} />}
                hint={<span className={trendClass(p.dayChangePercent)}>{formatPercent(p.dayChangePercent)}</span>}
              />
              <Stat label="Holdings P&L" value={<Money value={p.holdingPnl} signed compact className={trendClass(p.holdingPnl)} />} />
              <Stat label="Positions P&L" value={<Money value={p.positionPnl} signed compact className={trendClass(p.positionPnl)} />} />
              <div>
                <div className="text-xs text-muted-foreground">Margin used</div>
                <div className="text-lg font-semibold tabular-nums">
                  {p.marginTotal ? formatPercent(((p.marginUsed ?? 0) / p.marginTotal) * 100, 0, false) : "—"}
                </div>
                {p.marginTotal ? <Meter value={(p.marginUsed ?? 0) / p.marginTotal} className="mt-1" /> : null}
              </div>
            </div>
            {p.topHoldings.length > 0 && (
              <table className="w-full text-sm">
                <caption className="sr-only">Holdings with the largest day change</caption>
                <thead>
                  <tr className="text-xs text-muted-foreground">
                    <th className="py-1 text-left font-normal">Holding</th>
                    <th className="py-1 text-right font-normal">Day</th>
                    <th className="py-1 text-right font-normal">P&amp;L</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {p.topHoldings.map((h) => (
                    <tr key={`${h.exchange}:${h.symbol}`}>
                      <td className="py-1.5 font-medium">{h.symbol}</td>
                      <td className="py-1.5 text-right">
                        <Money value={h.dayChange} signed className={trendClass(h.dayChange)} />
                      </td>
                      <td className="py-1.5 text-right">
                        <Money value={h.pnl} signed className={trendClass(h.pnl)} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            {p.updatedAt && <p className="text-xs text-muted-foreground">Refreshed from Kite at {formatTimeIST(p.updatedAt)} IST</p>}
          </div>
        )
      }
    </SectionCard>
  );
}
