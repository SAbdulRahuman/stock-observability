"use client";

import { EllipsisVertical } from "lucide-react";
import { DropdownMenu } from "radix-ui";
import type { AlertRow, SilenceKind } from "@/lib/api";
import { formatTimeIST } from "@/lib/format";
import { cn } from "@/lib/utils";
import { ruleText, SeverityBadge, StatusBadge, silenceable } from "./shared";

const menuItem = "flex cursor-pointer items-center rounded-md px-2 py-1.5 text-sm outline-none data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 data-[highlighted]:bg-accent";

export function AlertsTable({
  rows,
  selected,
  onSelect,
  checked,
  onToggle,
  onToggleAll,
  canSilence,
  onAction,
}: {
  rows: AlertRow[];
  selected: string | null;
  onSelect: (key: string) => void;
  checked: Set<string>;
  onToggle: (key: string) => void;
  onToggleAll: () => void;
  canSilence: boolean;
  onAction: (kind: SilenceKind, row: AlertRow) => void;
}) {
  const selectable = rows.filter(silenceable);
  const all = selectable.length > 0 && selectable.every((r) => checked.has(r.key));
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-xs text-muted-foreground">
          <th className="w-8 py-2 pl-3 text-left font-normal">
            <input type="checkbox" aria-label="Select all firing alerts on this page" checked={all} onChange={onToggleAll} disabled={selectable.length === 0} />
          </th>
          <th className="py-2 text-left font-normal">Severity</th>
          <th className="py-2 text-left font-normal">Symbol</th>
          <th className="py-2 text-left font-normal">Rule</th>
          <th className="py-2 text-left font-normal">Triggered</th>
          <th className="py-2 text-right font-normal">Value</th>
          <th className="py-2 pl-3 text-left font-normal">Status</th>
          <th className="py-2 text-left font-normal">Source</th>
          <th className="w-10 py-2 pr-2 text-center font-normal">Actions</th>
        </tr>
      </thead>
      <tbody className="divide-y">
        {rows.map((r) => (
          <tr key={r.key} data-selected={r.key === selected || undefined} className={cn("cursor-pointer hover:bg-accent/60", r.key === selected && "bg-primary/15")} onClick={() => onSelect(r.key)}>
            <td className="py-2 pl-3" onClick={(e) => e.stopPropagation()}>
              <input
                type="checkbox"
                aria-label={`Select ${r.symbol || r.name}`}
                checked={checked.has(r.key)}
                onChange={() => onToggle(r.key)}
                disabled={!silenceable(r)}
              />
            </td>
            <td className="py-2">
              <SeverityBadge severity={r.severity} />
            </td>
            <td className="py-2 font-medium">{r.symbol || "—"}</td>
            <td className="max-w-0 truncate py-2 pr-2" style={{ width: "35%" }} title={r.name}>
              <button type="button" className="max-w-full truncate text-left hover:underline" onClick={() => onSelect(r.key)}>
                {ruleText(r)}
              </button>
            </td>
            <td className="py-2 whitespace-nowrap text-muted-foreground tabular-nums" title={new Date(r.startsAt).toLocaleString("en-IN")}>
              {formatTimeIST(r.startsAt, true)}
            </td>
            <td className="py-2 text-right tabular-nums">{r.value || "—"}</td>
            <td className="py-2 pl-3">
              <StatusBadge status={r.status} />
            </td>
            <td className="py-2 text-muted-foreground">{r.source}</td>
            <td className="py-2 pr-2 text-center" onClick={(e) => e.stopPropagation()}>
              <DropdownMenu.Root>
                <DropdownMenu.Trigger className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground" aria-label={`Actions for ${r.symbol || r.name}`}>
                  <EllipsisVertical className="size-4" />
                </DropdownMenu.Trigger>
                <DropdownMenu.Portal>
                  <DropdownMenu.Content align="end" className="z-50 min-w-44 rounded-lg border bg-card p-1 shadow-lg">
                    <DropdownMenu.Item className={menuItem} onSelect={() => onSelect(r.key)}>
                      Show details
                    </DropdownMenu.Item>
                    <DropdownMenu.Item className={menuItem} disabled={!canSilence || !silenceable(r)} onSelect={() => onAction("acknowledge", r)}>
                      Acknowledge
                    </DropdownMenu.Item>
                    <DropdownMenu.Item className={menuItem} disabled={!canSilence || !silenceable(r)} onSelect={() => onAction("silence", r)}>
                      Silence
                    </DropdownMenu.Item>
                    {r.exploreUrl && (
                      <DropdownMenu.Item className={menuItem} asChild>
                        <a href={r.exploreUrl} target="_blank" rel="noopener noreferrer">
                          Open in Grafana
                        </a>
                      </DropdownMenu.Item>
                    )}
                  </DropdownMenu.Content>
                </DropdownMenu.Portal>
              </DropdownMenu.Root>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export function Pagination({ page, pages, total, perPage, onPage }: { page: number; pages: number; total: number; perPage: number; onPage: (p: number) => void }) {
  const from = total === 0 ? 0 : (page - 1) * perPage + 1;
  const to = Math.min(total, page * perPage);
  const nums = Array.from({ length: pages }, (_, i) => i + 1).filter((p) => p === 1 || p === pages || Math.abs(p - page) <= 1);
  return (
    <nav aria-label="Pagination" className="flex flex-wrap items-center justify-between gap-2 border-t px-3 py-2 text-sm">
      <span className="text-muted-foreground">
        Showing {from}–{to} of {total}
      </span>
      <span className="flex items-center gap-1">
        <button type="button" className="rounded-md border px-2 py-1 disabled:opacity-40" onClick={() => onPage(page - 1)} disabled={page <= 1} aria-label="Previous page">
          ‹
        </button>
        {nums.map((p, i) => (
          <span key={p} className="flex items-center gap-1">
            {i > 0 && p - nums[i - 1] > 1 && <span className="px-1 text-muted-foreground">…</span>}
            <button
              type="button"
              aria-current={p === page ? "page" : undefined}
              onClick={() => onPage(p)}
              className={cn("min-w-8 rounded-md border px-2 py-1 tabular-nums", p === page && "border-primary bg-primary text-primary-foreground")}
            >
              {p}
            </button>
          </span>
        ))}
        <button type="button" className="rounded-md border px-2 py-1 disabled:opacity-40" onClick={() => onPage(page + 1)} disabled={page >= pages} aria-label="Next page">
          ›
        </button>
      </span>
    </nav>
  );
}
