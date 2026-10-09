"use client";

import { BellOff, CheckCheck, CircleAlert, EllipsisVertical, Plus, RotateCcw, Search, Upload } from "lucide-react";
import { DropdownMenu } from "radix-ui";
import { useState } from "react";
import { Empty, useNow } from "@/components/dashboard/section-card";
import { useLinkHref } from "@/components/shell/hooks";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { type AlertRow, type SilenceKind, useAlertList, useSection } from "@/lib/api";
import { AlertDetail } from "./detail";
import { AlertActivity, AlertCounts, SummaryCards } from "./panels";
import { applyFilters, defaultFilters, type Filters, silenceable, statusLabel, timeRanges } from "./shared";
import { type Outcome, SilenceDialog } from "./silence-dialog";
import { AlertsTable, Pagination } from "./table";

const PER_PAGE = 10;
const menuItem = "flex cursor-pointer items-center rounded-md px-2 py-1.5 text-sm outline-none data-[highlighted]:bg-accent";

function csvCell(v: string | undefined | null): string {
  const s = v ?? "";
  // Leading =,+,-,@ would be evaluated as formulas by spreadsheet apps.
  const safe = /^[=+\-@]/.test(s) ? `'${s}` : s;
  return /[",\n]/.test(safe) ? `"${safe.replace(/"/g, '""')}"` : safe;
}

export function AlertsPage() {
  const list = useAlertList();
  const info = useSection("info");
  const alertmanager = useLinkHref("alertmanager");
  const rules = useLinkHref("prometheus-rules");
  const now = useNow(10_000).getTime();
  const canSilence = info.data?.data.silencesAllowed === true;

  const [filters, setFilters] = useState<Filters>(defaultFilters);
  const [page, setPage] = useState(1);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [checked, setChecked] = useState<Set<string>>(new Set());
  const [dialog, setDialog] = useState<{ kind: SilenceKind; targets: AlertRow[] } | null>(null);
  const [outcome, setOutcome] = useState<Outcome | null>(null);

  const rows = list.data?.data.items ?? [];
  const timeline = list.data?.data.timeline ?? [];
  const filtered = applyFilters(rows, filters, now);
  const pages = Math.max(1, Math.ceil(filtered.length / PER_PAGE));
  const current = Math.min(page, pages);
  const pageRows = filtered.slice((current - 1) * PER_PAGE, current * PER_PAGE);
  const selected = rows.find((r) => r.key === selectedKey) ?? filtered[0] ?? null;
  const checkedRows = rows.filter((r) => checked.has(r.key) && silenceable(r));
  const sources = [...new Set(rows.map((r) => r.source))].sort();

  const setFilter = (k: keyof Filters, v: string) => {
    setFilters((f) => ({ ...f, [k]: v }));
    setPage(1);
  };
  const exportCsv = () => {
    const header = ["severity", "symbol", "rule", "summary", "triggered", "value", "status", "source"];
    const lines = filtered.map((r) => [r.severity, r.symbol, r.name, r.summary, r.startsAt, r.value, statusLabel(r.status), r.source].map(csvCell).join(","));
    const url = URL.createObjectURL(new Blob([`${[header.join(","), ...lines].join("\n")}\n`], { type: "text/csv" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = "alerts.csv";
    a.click();
    URL.revokeObjectURL(url);
  };
  const open = (kind: SilenceKind, targets: AlertRow[]) => {
    setOutcome(null);
    setDialog({ kind, targets });
  };
  const select = "h-9 rounded-lg border bg-card px-2 text-sm";

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Alerts</h1>
          <p className="text-sm text-muted-foreground">Monitor market alerts, investigate triggers, and manage alert rules</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button disabled title="Alert rules are defined in Prometheus rule files and cannot be created here">
            <Plus /> Create Alert
          </Button>
          <Button variant="outline" onClick={exportCsv} disabled={filtered.length === 0}>
            <Upload /> Export
          </Button>
          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild>
              <Button variant="outline" size="icon" aria-label="More alert actions">
                <EllipsisVertical />
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content align="end" className="z-50 min-w-52 rounded-lg border bg-card p-1 shadow-lg">
                {alertmanager && (
                  <DropdownMenu.Item className={menuItem} asChild>
                    <a href={alertmanager} target="_blank" rel="noopener noreferrer">
                      Open AlertManager
                    </a>
                  </DropdownMenu.Item>
                )}
                {rules && (
                  <DropdownMenu.Item className={menuItem} asChild>
                    <a href={rules} target="_blank" rel="noopener noreferrer">
                      View alert rules in Prometheus
                    </a>
                  </DropdownMenu.Item>
                )}
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        </div>
      </div>

      {list.data ? (
        <SummaryCards rows={rows} timeline={timeline} now={now} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <input
            type="search"
            aria-label="Search alerts"
            placeholder="Search alerts or symbols..."
            value={filters.search}
            onChange={(e) => setFilter("search", e.target.value)}
            className="h-9 w-full max-w-md rounded-lg border bg-card pr-3 pl-9 text-sm"
          />
        </div>
        <select aria-label="Severity" className={select} value={filters.severity} onChange={(e) => setFilter("severity", e.target.value)}>
          <option value="all">All severities</option>
          <option value="critical">Critical</option>
          <option value="warning">Warning</option>
          <option value="info">Info</option>
        </select>
        <select aria-label="Status" className={select} value={filters.status} onChange={(e) => setFilter("status", e.target.value)}>
          <option value="all">All statuses</option>
          {(["firing", "acknowledged", "silenced", "inhibited", "resolved"] as const).map((s) => (
            <option key={s} value={s}>
              {statusLabel(s)}
            </option>
          ))}
        </select>
        <select aria-label="Source" className={select} value={filters.source} onChange={(e) => setFilter("source", e.target.value)}>
          <option value="all">All sources</option>
          {sources.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        <select aria-label="Time range" className={select} value={filters.range} onChange={(e) => setFilter("range", e.target.value)}>
          {Object.entries(timeRanges).map(([k, v]) => (
            <option key={k} value={k}>
              {v.label}
            </option>
          ))}
        </select>
        <Button
          variant="outline"
          onClick={() => {
            setFilters(defaultFilters);
            setPage(1);
          }}
        >
          <RotateCcw /> Reset
        </Button>
      </div>
      {outcome && (
        <p role="status" className={outcome.ok ? "text-sm text-ok" : "text-sm text-critical"}>
          {outcome.message}
        </p>
      )}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)]">
        <Card aria-label="Alerts list" className="min-w-0 overflow-x-auto">
          {list.isPending ? (
            <Skeleton className="m-4 h-48" />
          ) : list.isError && !list.data ? (
            <p className="m-4 flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
              <CircleAlert className="size-4 text-warning" aria-hidden />
              Unavailable — {list.error.message}
            </p>
          ) : filtered.length === 0 ? (
            <Empty>{rows.length === 0 ? "No alerts firing or resolved in the last 24 hours" : "No alerts match the filters"}</Empty>
          ) : (
            <>
              <AlertsTable
                rows={pageRows}
                selected={selected?.key ?? null}
                onSelect={setSelectedKey}
                checked={checked}
                onToggle={(k) =>
                  setChecked((c) => {
                    const n = new Set(c);
                    if (n.has(k)) n.delete(k);
                    else n.add(k);
                    return n;
                  })
                }
                onToggleAll={() => {
                  const keys = pageRows.filter(silenceable).map((r) => r.key);
                  setChecked((c) => (keys.every((k) => c.has(k)) ? new Set() : new Set(keys)));
                }}
                canSilence={canSilence}
                onAction={(kind, r) => open(kind, [r])}
              />
              {checkedRows.length > 0 && (
                <div className="flex flex-wrap items-center gap-2 border-t px-3 py-2 text-sm">
                  <span className="text-muted-foreground">{checkedRows.length} selected</span>
                  <Button size="sm" onClick={() => open("acknowledge", checkedRows)} disabled={!canSilence} title={canSilence ? undefined : "Silences are disabled on this stock_home"}>
                    <CheckCheck /> Acknowledge
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => open("silence", checkedRows)} disabled={!canSilence} title={canSilence ? undefined : "Silences are disabled on this stock_home"}>
                    <BellOff /> Silence
                  </Button>
                </div>
              )}
              <Pagination page={current} pages={pages} total={filtered.length} perPage={PER_PAGE} onPage={setPage} />
            </>
          )}
        </Card>
        <AlertDetail row={selected} canSilence={canSilence} onAction={(kind, r) => open(kind, [r])} />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <AlertActivity rows={rows} />
        <AlertCounts timeline={timeline} range={filters.range} now={now} />
      </div>

      <SilenceDialog
        kind={dialog?.kind ?? null}
        targets={dialog?.targets ?? []}
        onOpenChange={(o) => !o && setDialog(null)}
        onDone={(o) => {
          setOutcome(o);
          setChecked(new Set());
        }}
      />
    </main>
  );
}
