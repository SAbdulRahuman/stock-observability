"use client";

import { Bell, BellRing, EllipsisVertical } from "lucide-react";
import { DropdownMenu } from "radix-ui";
import { Sparkline } from "@/components/dashboard/sparkline";
import type { Quote } from "@/lib/api";
import { formatChange, formatPercent, formatPrice, formatVolume, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";

export type Columns = { chart: boolean; volume: boolean; alerts: boolean };
export const columnLabels: Record<keyof Columns, string> = { chart: "Day Chart", volume: "Volume", alerts: "Alerts" };

type Common = {
  items: Quote[];
  selected: string | null;
  onSelect: (symbol: string) => void;
  alertSymbols: Set<string>;
  editable: boolean;
  onRemove: (symbol: string) => void;
};

export function WatchlistTable({
  items,
  selected,
  onSelect,
  alertSymbols,
  editable,
  onRemove,
  checked,
  onToggle,
  onToggleAll,
  columns,
}: Common & { checked: Set<string>; onToggle: (symbol: string) => void; onToggleAll: () => void; columns: Columns }) {
  const all = items.length > 0 && items.every((q) => checked.has(q.symbol));
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-xs text-muted-foreground">
          <th className="w-8 py-2 pl-3 text-left font-normal">
            <input type="checkbox" aria-label="Select all" checked={all} onChange={onToggleAll} disabled={!editable || items.length === 0} />
          </th>
          <th className="w-8 py-2 text-left font-normal">#</th>
          <th className="py-2 text-left font-normal">Symbol</th>
          <th className="py-2 text-right font-normal">LTP (₹)</th>
          <th className="py-2 text-right font-normal">Change</th>
          <th className="py-2 text-right font-normal">%</th>
          {columns.chart && <th className="py-2 pl-3 text-center font-normal">Day Chart</th>}
          {columns.volume && <th className="py-2 text-right font-normal">Volume</th>}
          {columns.alerts && <th className="py-2 text-center font-normal">Alerts</th>}
          <th className="w-10 py-2 pr-2 text-center font-normal">Actions</th>
        </tr>
      </thead>
      <tbody className="divide-y">
        {items.map((q, i) => {
          const active = q.symbol === selected;
          return (
            <tr key={q.symbol} data-selected={active || undefined} className={cn("cursor-pointer hover:bg-accent/60", active && "bg-primary/15")} onClick={() => onSelect(q.symbol)}>
              <td className="py-2 pl-3" onClick={(e) => e.stopPropagation()}>
                <input type="checkbox" aria-label={`Select ${q.symbol}`} checked={checked.has(q.symbol)} onChange={() => onToggle(q.symbol)} disabled={!editable} />
              </td>
              <td className="py-2 text-muted-foreground tabular-nums">{i + 1}</td>
              <td className="py-2 font-medium">
                <button type="button" className="text-left hover:underline" onClick={() => onSelect(q.symbol)} aria-label={`Show ${q.symbol} details`}>
                  {q.symbol}
                </button>
              </td>
              <td className="py-2 text-right tabular-nums">{formatPrice(q.price)}</td>
              <td className={cn("py-2 text-right tabular-nums", trendClass(q.change))}>{formatChange(q.change)}</td>
              <td className={cn("py-2 text-right font-medium tabular-nums", trendClass(q.changePercent))}>{formatPercent(q.changePercent)}</td>
              {columns.chart && (
                <td className="py-1 pl-3">
                  <Sparkline points={q.spark} className="mx-auto w-20" height={24} />
                </td>
              )}
              {columns.volume && <td className="py-2 text-right tabular-nums">{formatVolume(q.volume)}</td>}
              {columns.alerts && (
                <td className="py-2 text-center">
                  <AlertIndicator symbol={q.symbol} active={alertSymbols.has(q.symbol)} />
                </td>
              )}
              <td className="py-2 pr-2 text-center" onClick={(e) => e.stopPropagation()}>
                <RowActions quote={q} editable={editable} onSelect={onSelect} onRemove={onRemove} />
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

export function WatchlistCards({ items, selected, onSelect, alertSymbols, editable, onRemove }: Common) {
  return (
    <ul className="grid grid-cols-1 gap-3 p-3 sm:grid-cols-2 2xl:grid-cols-3">
      {items.map((q) => (
        <li key={q.symbol}>
          <div className={cn("rounded-lg border p-3", q.symbol === selected && "border-primary bg-primary/10")}>
            <div className="flex items-center justify-between gap-2">
              <button type="button" className="font-medium hover:underline" onClick={() => onSelect(q.symbol)} aria-label={`Show ${q.symbol} details`}>
                {q.symbol}
              </button>
              <span className="flex items-center gap-1">
                <AlertIndicator symbol={q.symbol} active={alertSymbols.has(q.symbol)} />
                <RowActions quote={q} editable={editable} onSelect={onSelect} onRemove={onRemove} />
              </span>
            </div>
            <div className="mt-1 flex items-end justify-between gap-2">
              <div>
                <div className="text-lg font-semibold tabular-nums">{formatPrice(q.price)}</div>
                <div className={cn("text-xs font-medium tabular-nums", trendClass(q.changePercent))}>
                  {formatChange(q.change)} ({formatPercent(q.changePercent)})
                </div>
              </div>
              <Sparkline points={q.spark} className="w-24" height={32} />
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
}

function AlertIndicator({ symbol, active }: { symbol: string; active: boolean }) {
  return active ? (
    <BellRing className="inline size-4 text-critical" role="img" aria-label={`${symbol}: alert firing`} />
  ) : (
    <Bell className="inline size-4 text-muted-foreground/60" role="img" aria-label={`${symbol}: no active alert`} />
  );
}

const menuItem = "flex cursor-pointer items-center rounded-md px-2 py-1.5 text-sm outline-none data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 data-[highlighted]:bg-accent";

export function RowActions({ quote, editable, onSelect, onRemove }: { quote: Quote; editable: boolean; onSelect: (s: string) => void; onRemove: (s: string) => void }) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground" aria-label={`Actions for ${quote.symbol}`}>
        <EllipsisVertical className="size-4" />
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content align="end" className="z-50 min-w-44 rounded-lg border bg-card p-1 shadow-lg">
          <DropdownMenu.Item className={menuItem} onSelect={() => onSelect(quote.symbol)}>
            Show details
          </DropdownMenu.Item>
          {quote.exploreUrl && (
            <DropdownMenu.Item className={menuItem} asChild>
              <a href={quote.exploreUrl} target="_blank" rel="noopener noreferrer">
                View in Grafana
              </a>
            </DropdownMenu.Item>
          )}
          <DropdownMenu.Item className={cn(menuItem, "text-critical")} disabled={!editable} onSelect={() => onRemove(quote.symbol)}>
            Remove from watchlist
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
