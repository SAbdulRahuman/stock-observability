"use client";

import { ChevronDown, ChevronRight } from "lucide-react";
import { useTheme } from "next-themes";
import Link from "next/link";
import { useState } from "react";
import { useMounted } from "@/components/shell/hooks";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useSection } from "@/lib/api";
import { formatTimeIST } from "@/lib/format";
import { usePromStatus } from "@/lib/prom";
import { type Settings, useSettings } from "@/lib/settings";
import { cn } from "@/lib/utils";

export type Notify = (ok: boolean, text: string) => void;

export function SettingsCard({
  title,
  subtitle,
  icon: Icon,
  danger,
  children,
}: {
  title: string;
  subtitle: string;
  icon: React.ComponentType<{ className?: string }>;
  danger?: boolean;
  children: React.ReactNode;
}) {
  return (
    <Card aria-label={title} className={cn("min-w-0 p-5", danger && "border-critical/60")}>
      <header className="mb-4 flex items-start gap-3">
        <Icon className={cn("mt-0.5 size-5 shrink-0", danger ? "text-critical" : "text-primary")} aria-hidden />
        <div>
          <h2 className={cn("font-semibold", danger && "text-critical")}>{title}</h2>
          <p className="text-xs text-muted-foreground">{subtitle}</p>
        </div>
      </header>
      <div className="space-y-3 text-sm">{children}</div>
    </Card>
  );
}

function Row({ label, hint, children }: { label: string; hint?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div>
      <div className="flex items-center justify-between gap-3">
        <span>{label}</span>
        <div className="flex min-w-0 items-center justify-end gap-2 text-right">{children}</div>
      </div>
      {hint && <p className="mt-0.5 text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}

/** A value this page cannot change; `source` says who owns it. */
function Fixed({ label, value, source, hint }: { label: string; value: React.ReactNode; source: string; hint?: React.ReactNode }) {
  return (
    <Row label={label} hint={hint}>
      <span className="truncate font-medium">{value}</span>
      <Badge>{source}</Badge>
    </Row>
  );
}

function Unavailable({ label, hint }: { label: string; hint?: string }) {
  return (
    <Row label={label} hint={hint}>
      <Badge variant="outline" className="text-muted-foreground">
        Unavailable
      </Badge>
    </Row>
  );
}

function Switch({ label, checked, onChange, hint }: { label: string; checked: boolean; onChange: (v: boolean) => void; hint?: string }) {
  return (
    <Row label={label} hint={hint}>
      <span className="text-xs text-muted-foreground">{checked ? "On" : "Off"}</span>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        onClick={() => onChange(!checked)}
        className={cn(
          "relative h-6 w-11 shrink-0 rounded-full border transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
          checked ? "bg-primary" : "bg-muted",
        )}
      >
        <span className={cn("absolute top-0.5 size-4.5 rounded-full bg-background shadow transition-all", checked ? "left-5.5" : "left-0.5")} />
      </button>
    </Row>
  );
}

function useSetting(notify: Notify) {
  const [settings, update] = useSettings();
  const set = (fn: (s: Settings) => Settings) => {
    update(fn);
    notify(true, "Saved in this browser.");
  };
  return [settings, set] as const;
}

export function ApplicationCard({ icon, notify }: { icon: React.ComponentType<{ className?: string }>; notify: Notify }) {
  const { theme, setTheme } = useTheme();
  const mounted = useMounted();
  const info = useSection("info").data?.data;
  return (
    <SettingsCard title="Application Settings" subtitle="General application configuration" icon={icon}>
      <Fixed label="Application Name" value="stock_home" source="build" />
      <Row label="Theme" hint="Stored in this browser; System follows your OS setting.">
        <div role="radiogroup" aria-label="Theme" className="flex rounded-lg border p-0.5">
          {(["light", "dark", "system"] as const).map((t) => (
            <button
              key={t}
              type="button"
              role="radio"
              aria-checked={mounted && theme === t}
              onClick={() => {
                setTheme(t);
                notify(true, "Saved in this browser.");
              }}
              className={cn("rounded-md px-2.5 py-1 text-xs capitalize", mounted && theme === t ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground")}
            >
              {t}
            </button>
          ))}
        </div>
      </Row>
      <Fixed label="Language" value="English" source="only option" />
      <Fixed label="Time Zone" value={info?.timezone ?? "—"} source="server" hint="Exchange time zone (--market.timezone); market times are shown in it." />
      <Fixed label="Refresh Interval" value="Automatic" source="server" hint="Each panel refreshes at its server cache interval (5–60 s)." />
    </SettingsCard>
  );
}

export function MarketCard({ icon }: { icon: React.ComponentType<{ className?: string }> }) {
  const info = useSection("info");
  const d = info.data?.data;
  const holidays = d?.holidayCount;
  return (
    <SettingsCard title="Market Settings" subtitle="Default market and display preferences" icon={icon}>
      {info.isError && <p className="text-xs text-warning">Server configuration is unavailable: {info.error.message}</p>}
      <Fixed label="Default Exchange" value={d?.exchange || "—"} source="server" hint="--market.exchange" />
      <Fixed label="Default Index" value={d?.indices?.[0] ?? "—"} source="server" hint="First entry of --market.indices" />
      <Fixed label="Market Hours" value={d ? `${d.sessionOpen} – ${d.sessionClose}` : "—"} source="server" hint={d ? `${d.timezone} (--market.session-hours)` : undefined} />
      <Fixed
        label="Holiday Calendar"
        value={holidays == null ? "—" : holidays > 0 ? `${holidays} holidays loaded` : "Not configured"}
        source="server"
        hint="--market.holidays-file"
      />
      <p className="text-xs text-muted-foreground">These are stock_home server flags: change them in the deployment and restart stock_home.</p>
    </SettingsCard>
  );
}

export function DisplayCard({ icon, notify }: { icon: React.ComponentType<{ className?: string }>; notify: Notify }) {
  const [settings, set] = useSetting(notify);
  return (
    <SettingsCard title="Data Display Settings" subtitle="Configure how data is shown" icon={icon}>
      <Fixed label="Price Format" value="₹ (INR)" source="fixed" />
      <Fixed label="Number Format" value="1,23,456.78" source="fixed" hint="Indian digit grouping" />
      <Unavailable label="Change Display" hint="Pages show absolute and percentage change together." />
      <Fixed label="Decimal Places" value="2" source="fixed" />
      <Unavailable label="Table Rows per Page" hint="Tables are sized per page." />
      <Switch label="Show Sparkline" checked={settings.showSparklines} onChange={(v) => set((s) => ({ ...s, showSparklines: v }))} hint="Mini price charts on every page." />
      <Fixed label="Highlight Price Change" value="On" source="fixed" hint="Gains and losses always use colour plus a sign." />
    </SettingsCard>
  );
}

export function NotificationCard({ icon }: { icon: React.ComponentType<{ className?: string }> }) {
  return (
    <SettingsCard title="Notification Settings" subtitle="Browser and in-app notifications" icon={icon}>
      {["Enable Notifications", "Price Alerts", "Rule Alerts", "Market Open/Close", "Daily Summary", "Notification Sound"].map((l) => (
        <Unavailable key={l} label={l} />
      ))}
      <p className="text-xs text-muted-foreground">
        stock_home does not send browser or in-app notifications. Alert notifications are delivered by Alertmanager receivers; see{" "}
        <Link href="/observability/alertmanager?tab=routes" className="text-primary underline-offset-2 hover:underline">
          Alertmanager › Receivers &amp; Routes
        </Link>
        .
      </p>
    </SettingsCard>
  );
}

export function DashboardCard({ icon, notify }: { icon: React.ComponentType<{ className?: string }>; notify: Notify }) {
  const [settings, set] = useSetting(notify);
  const toggle = (key: keyof Settings["home"], label: string) => (
    <Switch label={label} checked={settings.home[key]} onChange={(v) => set((s) => ({ ...s, home: { ...s.home, [key]: v } }))} />
  );
  return (
    <SettingsCard title="Dashboard Settings" subtitle="Configure dashboard layout and widgets" icon={icon}>
      <Fixed label="Default Dashboard View" value="Home" source="fixed" />
      <Unavailable label="Widget Layout" hint="The Home layout is fixed." />
      {toggle("indexCards", "Show Index Cards")}
      {toggle("recentAlerts", "Show Recent Alerts")}
      {toggle("topMovers", "Show Top Movers")}
      {toggle("watchlistSnapshot", "Show Watchlist Snapshot")}
      <p className="text-xs text-muted-foreground">These switches change the Home page only.</p>
    </SettingsCard>
  );
}

type TestResult = { status: string; message: string; latencyMs: number; checkedAt: string };
const testBadge: Record<string, { label: string; variant: BadgeVariant }> = {
  connected: { label: "Connected", variant: "ok" },
  error: { label: "Error", variant: "critical" },
  not_configured: { label: "Not configured", variant: "default" },
};

async function runTest(id: string): Promise<TestResult> {
  const res = await fetch(`/api/v1/integrations/${encodeURIComponent(id)}/test`, { method: "POST", cache: "no-store", headers: { Accept: "application/json" } });
  const body = (await res.json().catch(() => ({}))) as { data?: TestResult; error?: string };
  if (!res.ok || !body.data) throw new Error(body.error ?? `HTTP ${res.status}`);
  return body.data;
}

export function IntegrationsCard({ icon }: { icon: React.ComponentType<{ className?: string }> }) {
  const list = useSection("integrations");
  const [results, setResults] = useState<Record<string, TestResult | { error: string } | "running">>({});
  const [open, setOpen] = useState<string | null>(null);
  const test = (id: string) => {
    setResults((r) => ({ ...r, [id]: "running" }));
    runTest(id)
      .then((res) => setResults((r) => ({ ...r, [id]: res })))
      .catch((e: Error) => setResults((r) => ({ ...r, [id]: { error: e.message } })));
  };
  return (
    <SettingsCard title="API & Integration Settings" subtitle="Configure external services and APIs" icon={icon}>
      {list.isError ? (
        <p className="text-muted-foreground">Integration settings are unavailable: {list.error.message}</p>
      ) : !list.data ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : (
        <ul className="divide-y">
          {list.data.data.filter((i) => i.id !== "kite").map((i) => {
            const r = results[i.id];
            const done = r && r !== "running" && !("error" in r) ? r : undefined;
            const badge = done ? testBadge[done.status] : i.configured ? { label: "Configured", variant: "info" as BadgeVariant } : testBadge.not_configured;
            return (
              <li key={i.id} className="py-2">
                <div className="flex items-center gap-2">
                  <span className="flex-1 font-medium">{i.name}</span>
                  <Badge variant={badge.variant}>{badge.label}</Badge>
                  <Button size="sm" variant="outline" onClick={() => test(i.id)} disabled={r === "running"} aria-label={`Test ${i.name}`}>
                    {r === "running" ? "Testing…" : "Test"}
                  </Button>
                  <button
                    type="button"
                    aria-expanded={open === i.id}
                    aria-label={`${i.name} details`}
                    onClick={() => setOpen(open === i.id ? null : i.id)}
                    className="rounded p-1 text-muted-foreground hover:text-foreground"
                  >
                    {open === i.id ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
                  </button>
                </div>
                {r && r !== "running" && (
                  <p role="status" className={cn("mt-1 text-xs", "error" in r || r.status === "error" ? "text-critical" : "text-muted-foreground")}>
                    {"error" in r ? `Test failed: ${r.error}` : `${r.message} · ${r.latencyMs.toFixed(0)} ms · checked ${formatTimeIST(r.checkedAt, true)} IST`}
                  </p>
                )}
                {open === i.id && (
                  <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 rounded-md bg-muted/40 p-2 text-xs">
                    <dt className="text-muted-foreground">Endpoint</dt>
                    <dd className="font-mono break-all">{i.endpoint || "—"}</dd>
                    <dt className="text-muted-foreground">Credentials</dt>
                    <dd>{i.auth}</dd>
                    {i.note && (
                      <>
                        <dt className="text-muted-foreground">Note</dt>
                        <dd>{i.note}</dd>
                      </>
                    )}
                  </dl>
                )}
              </li>
            );
          })}
        </ul>
      )}
      <p className="text-xs text-muted-foreground">
        Kite Connect is configured under <Link href="/settings?tab=data-sources&source=kite" className="text-primary underline-offset-2 hover:underline">Data Sources</Link>.
      </p>
      <p className="text-xs text-muted-foreground">Endpoints and credentials are set in the stock_home deployment; secrets are never sent to the browser.</p>
    </SettingsCard>
  );
}

export function RetentionCard({ icon }: { icon: React.ComponentType<{ className?: string }> }) {
  const status = usePromStatus();
  const retention = status.data?.runtime.storageRetention;
  return (
    <SettingsCard title="Data Retention" subtitle="Configure data storage and retention policies" icon={icon}>
      <Fixed
        label="Prometheus Retention"
        value={typeof retention === "string" && retention ? retention : status.isError ? "unavailable" : "—"}
        source="Prometheus"
        hint="Owned by Prometheus (--storage.tsdb.retention.time). Change it in the deployment; it applies after Prometheus restarts."
      />
      <Fixed
        label="Alert History Retention"
        value="Not applicable"
        source="Prometheus"
        hint="stock_home stores no alert history; the Alerts timeline reads Prometheus ALERTS series and follows Prometheus retention."
      />
    </SettingsCard>
  );
}
