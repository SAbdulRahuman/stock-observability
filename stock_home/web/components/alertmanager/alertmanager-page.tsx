"use client";

import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert, ExternalLink, RefreshCw } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { useNow } from "@/components/dashboard/section-card";
import { useLinkHref } from "@/components/shell/hooks";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton, StatusDot } from "@/components/ui/primitives";
import { ConfirmDialog } from "@/components/watchlist/dialogs";
import { type AMSilence, expireSilence, type MatcherInput, matcherText, useAM } from "@/lib/am";
import { useSection } from "@/lib/api";
import { formatCount, relativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import { SilenceForm } from "./silence-form";
import { AlertsTab, alertStatus, RoutesTab, SilencesTab, StatusTab } from "./tabs";

const TABS = [
  { id: "alerts", label: "Alerts" },
  { id: "silences", label: "Silences" },
  { id: "routes", label: "Receivers & Routes" },
  { id: "status", label: "Status" },
] as const;
type Tab = (typeof TABS)[number]["id"];

function Loading<T>({ q, children }: { q: { data?: { data: T }; isError: boolean; error: Error | null }; children: (d: T) => React.ReactNode }) {
  if (q.data) return <>{children(q.data.data)}</>;
  if (q.isError)
    return (
      <p className="flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
        <CircleAlert className="size-4 text-warning" aria-hidden />
        Unavailable — {q.error?.message}
      </p>
    );
  return <Skeleton className="h-40 w-full" />;
}

export function AlertmanagerPage() {
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname() ?? "/observability/alertmanager";
  const tab: Tab = TABS.find((t) => t.id === params?.get("tab"))?.id ?? "alerts";
  const now = useNow(10_000);
  const client = useQueryClient();
  const full = useLinkHref("alertmanager");
  const canSilence = useSection("info").data?.data.silencesAllowed === true;
  const groups = useAM("groups");
  const silences = useAM("silences");
  const status = useAM("status", 30_000);
  const [form, setForm] = useState<{ key: number; initial: MatcherInput[] } | null>(null);
  const [expiring, setExpiring] = useState<AMSilence | null>(null);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);

  const instances = new Map((groups.data?.data.groups ?? []).flatMap((g) => g.alerts).map((a) => [a.fingerprint, a]));
  const active = [...instances.values()].filter((a) => alertStatus(a) === "firing").length;
  const silenced = [...instances.values()].filter((a) => alertStatus(a) === "silenced").length;
  const pending = silences.data?.data.silences.filter((s) => s.state === "pending").length;
  const val = (v: number | undefined, err: boolean) => (v == null ? (err ? "unavailable" : "—") : formatCount(v));
  const cards = [
    { label: "Alert Groups", value: val(groups.data?.data.groups.length, groups.isError) },
    { label: "Active Alerts", value: val(groups.data ? active : undefined, groups.isError), tone: active ? "text-critical" : "" },
    { label: "Silenced Alerts", value: val(groups.data ? silenced : undefined, groups.isError) },
    { label: "Pending Silences", value: val(pending, silences.isError) },
  ];
  const lastOk = [groups, silences, status].map((q) => q.dataUpdatedAt).filter((t) => t > 0);
  const openForm = (initial: MatcherInput[]) => {
    setMessage(null);
    setForm({ key: Date.now(), initial });
  };

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Alertmanager</h1>
          <p className="text-sm text-muted-foreground">Review alert groups, silence notifications, and inspect routing status</p>
        </div>
        {full && (
          <Button variant="outline" asChild>
            <a href={full} target="_blank" rel="noopener noreferrer">
              <ExternalLink /> Open full Alertmanager UI
            </a>
          </Button>
        )}
      </div>

      <div role="tablist" aria-label="Alertmanager sections" className="flex flex-wrap gap-1 border-b">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={t.id === tab}
            onClick={() => router.replace(`${pathname}?tab=${t.id}`, { scroll: false })}
            className={cn(
              "-mb-px border-b-2 px-3 py-2 text-sm font-medium transition-colors",
              t.id === tab ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground",
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((c) => (
          <Card key={c.label} aria-label={c.label} className="p-4">
            <div className="text-sm text-muted-foreground">{c.label}</div>
            <div className={cn("mt-1 text-2xl font-semibold tabular-nums", c.tone)}>{c.value}</div>
          </Card>
        ))}
      </div>
      <p className="-mt-2 text-xs text-muted-foreground">Alert counts are distinct alert instances (by fingerprint), even when an alert appears in several groups.</p>

      <Card className="flex flex-wrap items-center gap-3 px-4 py-3 text-sm" aria-label="Alertmanager connection">
        <span className="text-muted-foreground">Alertmanager connection:</span>
        <span className="font-medium">configured server</span>
        <span className="flex items-center gap-2" role="status">
          <StatusDot ok={status.data ? status.data.data.ready : status.isError ? false : null} />
          {status.isError ? "Unreachable" : status.data ? (status.data.data.ready ? "Ready" : "Not ready") : "Checking…"}
        </span>
        {!canSilence && <span className="text-xs text-muted-foreground">Silence actions are disabled (start stock_home with --alertmanager.allow-silences)</span>}
        <span className="ml-auto text-xs text-muted-foreground">Last successful API check: {lastOk.length ? relativeTime(new Date(Math.max(...lastOk)), now) : "—"}</span>
        <Button variant="outline" size="sm" onClick={() => void client.invalidateQueries({ queryKey: ["am"] })}>
          <RefreshCw /> Refresh
        </Button>
      </Card>

      {message && (
        <p role="status" className={message.ok ? "text-sm text-ok" : "text-sm text-critical"}>
          {message.text}
        </p>
      )}

      <section role="tabpanel" aria-label={`${tab} tab`}>
        {tab === "alerts" && <Loading q={groups}>{(d) => <AlertsTab groups={d.groups} now={now} canSilence={canSilence} onSilence={openForm} />}</Loading>}
        {tab === "silences" && (
          <Loading q={silences}>{(d) => <SilencesTab silences={d.silences} canSilence={canSilence} onCreate={() => openForm([])} onExpire={setExpiring} />}</Loading>
        )}
        {tab === "routes" && <Loading q={status}>{(d) => <RoutesTab status={d} />}</Loading>}
        {tab === "status" && <Loading q={status}>{(d) => <StatusTab status={d} now={now} />}</Loading>}
      </section>

      {form && (
        <SilenceForm
          key={form.key}
          open
          initial={form.initial}
          onOpenChange={(o) => !o && setForm(null)}
          onCreated={(text) => setMessage({ ok: true, text })}
        />
      )}
      <ConfirmDialog
        open={expiring != null}
        onOpenChange={(o) => !o && setExpiring(null)}
        title="Expire silence?"
        description={expiring ? `Notifications resume for alerts matching ${expiring.matchers.map(matcherText).join(", ")}.` : ""}
        confirmLabel="Expire silence"
        onConfirm={() => {
          const s = expiring;
          if (!s) return;
          expireSilence(s.id)
            .then(() => setMessage({ ok: true, text: `Silence ${s.id} expired.` }))
            .catch((e: Error) => setMessage({ ok: false, text: `Could not expire silence: ${e.message}` }))
            .finally(() => void client.invalidateQueries({ queryKey: ["am"] }));
        }}
      />
    </main>
  );
}
