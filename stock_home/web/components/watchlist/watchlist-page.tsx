"use client";

import { Download, EllipsisVertical, PanelRight, Plus, Search, Settings2, Upload } from "lucide-react";
import { DropdownMenu, Popover } from "radix-ui";
import { useRef, useState } from "react";
import { Empty } from "@/components/dashboard/section-card";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { type Quote, useQuotes, useSection } from "@/lib/api";
import { cn } from "@/lib/utils";
import {
  addSymbols,
  createGroup,
  DEFAULT_GROUP_ID,
  deleteGroup,
  MAX_SYMBOLS,
  parseCsv,
  removeSymbols,
  toCsv,
  useWatchlistStore,
} from "@/lib/watchlist";
import { Performance, QuickActions, RecentAlerts } from "./bottom";
import { StockDetail } from "./detail";
import { AddSymbolsDialog, ConfirmDialog, CreateGroupDialog } from "./dialogs";
import { type Columns, columnLabels, WatchlistCards, WatchlistTable } from "./table";

type Filter = "all" | "gainers" | "losers" | "alerts";
type Confirm = { title: string; description: string; label: string; destructive?: boolean; action: () => void };

const MAX_IMPORT_BYTES = 64 * 1024;
const menuItem = "flex cursor-pointer items-center rounded-md px-2 py-1.5 text-sm outline-none data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 data-[highlighted]:bg-accent";

export function WatchlistPage() {
  const [store, update] = useWatchlistStore();
  const info = useSection("info");
  const alerts = useSection("alerts");
  // Edits stay disabled until the backend answers and reports its symbol limit.
  const editable = !!info.data;
  const max = Math.min(info.data?.data.quoteSymbolsMax ?? MAX_SYMBOLS, MAX_SYMBOLS);

  const [activeId, setActiveId] = useState(DEFAULT_GROUP_ID);
  const [selected, setSelected] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const [view, setView] = useState<"table" | "cards">("table");
  const [columns, setColumns] = useState<Columns>({ chart: true, volume: true, alerts: true });
  const [showDetail, setShowDetail] = useState(true);
  const [checked, setChecked] = useState<Set<string>>(new Set());
  const [adding, setAdding] = useState<{ initial?: string[] } | null>(null);
  const [creating, setCreating] = useState(false);
  const [confirm, setConfirm] = useState<Confirm | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const groups = store?.groups ?? [];
  const group = groups.find((g) => g.id === activeId) ?? groups[0];
  const symbols = group?.symbols ?? [];
  const quotes = useQuotes(symbols);
  const bySymbol = new Map((quotes.data?.data.items ?? []).map((q) => [q.symbol, q]));
  const rows: Quote[] = symbols.map((s) => bySymbol.get(s) ?? { symbol: s, exchange: "", exploreUrl: "", price: null, change: null, changePercent: null });
  const alertSymbols = new Set((alerts.data?.data.items ?? []).flatMap((a) => (a.symbol ? [a.symbol] : [])));
  const term = search.trim().toUpperCase();
  const visible = rows.filter(
    (q) =>
      (!term || q.symbol.includes(term)) &&
      (filter === "all" ||
        (filter === "gainers" && (q.changePercent ?? 0) > 0) ||
        (filter === "losers" && (q.changePercent ?? 0) < 0) ||
        (filter === "alerts" && alertSymbols.has(q.symbol))),
  );
  const current = selected && symbols.includes(selected) ? selected : (symbols[0] ?? null);
  const favourite = !!current && !!store?.groups[0].symbols.includes(current);

  const switchGroup = (id: string) => {
    setActiveId(id);
    setChecked(new Set());
    setSelected(null);
  };
  const askRemove = (syms: string[]) =>
    group &&
    setConfirm({
      title: syms.length === 1 ? `Remove ${syms[0]}?` : `Remove ${syms.length} symbols?`,
      description: `${syms.join(", ")} will be removed from ${group.name}. Watchlists are stored in this browser.`,
      label: "Remove",
      action: () => {
        update((s) => removeSymbols(s, group.id, syms));
        setChecked(new Set());
      },
    });
  const exportCsv = () => {
    if (!group) return;
    const url = URL.createObjectURL(new Blob([toCsv(group)], { type: "text/csv" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = `${group.name.replace(/[^A-Za-z0-9-]+/g, "_")}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };
  const importCsv = async (file: File | undefined) => {
    if (!file) return;
    if (file.size > MAX_IMPORT_BYTES) {
      setImportError("CSV file is larger than 64 KB");
      return;
    }
    setImportError(null);
    setAdding({ initial: parseCsv(await file.text()) });
  };

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Watchlist</h1>
          <p className="text-sm text-muted-foreground">Track your favourite stocks, set alerts, and monitor real-time performance</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button onClick={() => setAdding({})} disabled={!editable || !group || symbols.length >= max} title={editable ? undefined : "Watchlist editing needs the stock_home API"}>
            <Plus /> Add Symbol
          </Button>
          <Button variant="outline" onClick={() => fileRef.current?.click()} disabled={!editable}>
            <Download /> Import
          </Button>
          <input
            ref={fileRef}
            type="file"
            accept=".csv,text/csv"
            className="hidden"
            data-testid="import-file"
            onChange={(e) => {
              void importCsv(e.target.files?.[0]);
              e.target.value = "";
            }}
          />
          <Button variant="outline" onClick={exportCsv} disabled={!group || symbols.length === 0}>
            <Upload /> Export
          </Button>
          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild>
              <Button variant="outline" size="icon" aria-label="More watchlist actions">
                <EllipsisVertical />
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content align="end" className="z-50 min-w-48 rounded-lg border bg-card p-1 shadow-lg">
                <DropdownMenu.Item className={menuItem} disabled={!editable} onSelect={() => setCreating(true)}>
                  Create new watchlist
                </DropdownMenu.Item>
                <DropdownMenu.Item
                  className={cn(menuItem, "text-critical")}
                  disabled={!editable || !group || group.id === DEFAULT_GROUP_ID}
                  onSelect={() =>
                    group &&
                    setConfirm({
                      title: `Delete ${group.name}?`,
                      description: `The watchlist and its ${group.symbols.length} symbols will be deleted from this browser.`,
                      label: "Delete",
                      action: () => {
                        update((s) => deleteGroup(s, group.id));
                        switchGroup(DEFAULT_GROUP_ID);
                      },
                    })
                  }
                >
                  Delete this watchlist
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
          <Button
            variant={showDetail ? "default" : "outline"}
            size="icon"
            aria-pressed={showDetail}
            aria-label="Show details panel"
            onClick={() => setShowDetail((v) => !v)}
          >
            <PanelRight />
          </Button>
        </div>
      </div>
      {importError && <p className="text-sm text-critical">{importError}</p>}

      <div role="group" aria-label="Watchlists" className="flex flex-wrap gap-2">
        {store == null ? (
          <Skeleton className="h-9 w-40" />
        ) : (
          groups.map((g) => (
            <button
              key={g.id}
              type="button"
              aria-pressed={g.id === group?.id}
              onClick={() => switchGroup(g.id)}
              className={cn(
                "rounded-lg border px-4 py-2 text-sm font-medium transition-colors",
                g.id === group?.id ? "border-primary bg-primary/15 text-foreground" : "text-muted-foreground hover:bg-accent hover:text-foreground",
              )}
            >
              {g.name} ({g.symbols.length})
            </button>
          ))
        )}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <input
            type="search"
            aria-label="Search in watchlist"
            placeholder="Search in watchlist..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-9 w-full max-w-md rounded-lg border bg-card pr-3 pl-9 text-sm"
          />
        </div>
        <select aria-label="Filter" value={filter} onChange={(e) => setFilter(e.target.value as Filter)} className="h-9 rounded-lg border bg-card px-2 text-sm">
          <option value="all">All</option>
          <option value="gainers">Gainers</option>
          <option value="losers">Losers</option>
          <option value="alerts">With active alerts</option>
        </select>
        <div role="group" aria-label="View" className="flex rounded-lg border p-0.5">
          {(["table", "cards"] as const).map((v) => (
            <button
              key={v}
              type="button"
              aria-pressed={view === v}
              onClick={() => setView(v)}
              className={cn("rounded-md px-3 py-1 text-sm capitalize", view === v ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground")}
            >
              {v}
            </button>
          ))}
        </div>
        <Popover.Root>
          <Popover.Trigger asChild>
            <Button variant="outline">
              <Settings2 /> Customize
            </Button>
          </Popover.Trigger>
          <Popover.Portal>
            <Popover.Content align="end" className="z-50 w-48 space-y-2 rounded-lg border bg-card p-3 text-sm shadow-lg">
              <p className="text-xs text-muted-foreground">Table columns</p>
              {(Object.keys(columnLabels) as (keyof Columns)[]).map((k) => (
                <label key={k} className="flex items-center gap-2">
                  <input type="checkbox" checked={columns[k]} onChange={() => setColumns((c) => ({ ...c, [k]: !c[k] }))} />
                  {columnLabels[k]}
                </label>
              ))}
            </Popover.Content>
          </Popover.Portal>
        </Popover.Root>
        {checked.size > 0 && (
          <Button variant="critical" size="sm" onClick={() => askRemove([...checked])}>
            Remove selected ({checked.size})
          </Button>
        )}
      </div>

      <div className={cn("grid gap-4", showDetail && "xl:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)]")}>
        <Card aria-label={`${group?.name ?? "Watchlist"} symbols`} className="min-w-0 overflow-x-auto">
          {store == null ? (
            <Skeleton className="m-4 h-40" />
          ) : symbols.length === 0 ? (
            <Empty>
              {group?.name} is empty.
              <br />
              Use Add Symbol or Import to add NSE symbols.
            </Empty>
          ) : quotes.isError && !quotes.data ? (
            <Empty>Unavailable — {quotes.error.message}</Empty>
          ) : visible.length === 0 ? (
            <Empty>No symbols match the search or filter</Empty>
          ) : view === "table" ? (
            <WatchlistTable
              items={visible}
              selected={current}
              onSelect={setSelected}
              alertSymbols={alertSymbols}
              editable={editable}
              onRemove={(s) => askRemove([s])}
              checked={checked}
              onToggle={(s) =>
                setChecked((c) => {
                  const n = new Set(c);
                  if (n.has(s)) n.delete(s);
                  else n.add(s);
                  return n;
                })
              }
              onToggleAll={() => setChecked((c) => (visible.every((q) => c.has(q.symbol)) ? new Set() : new Set(visible.map((q) => q.symbol))))}
              columns={columns}
            />
          ) : (
            <WatchlistCards items={visible} selected={current} onSelect={setSelected} alertSymbols={alertSymbols} editable={editable} onRemove={(s) => askRemove([s])} />
          )}
        </Card>
        {showDetail && (
          <StockDetail
            symbol={current}
            favourite={favourite}
            editable={editable}
            onSelect={setSelected}
            onRemove={(s) => askRemove([s])}
            onToggleFavourite={() =>
              current &&
              setConfirm(
                favourite
                  ? {
                      title: `Remove ${current} from My Watchlist?`,
                      description: "The symbol stays in other watchlists.",
                      label: "Remove",
                      action: () => update((s) => removeSymbols(s, DEFAULT_GROUP_ID, [current])),
                    }
                  : {
                      title: `Add ${current} to My Watchlist?`,
                      description: "My Watchlist also feeds the Home page snapshot.",
                      label: "Add",
                      destructive: false,
                      action: () => update((s) => addSymbols(s, DEFAULT_GROUP_ID, [current]).store),
                    },
              )
            }
          />
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Performance total={symbols.length} quotes={quotes.data?.data.items ?? []} />
        <RecentAlerts symbols={symbols} />
        <QuickActions editable={editable} onCreate={() => setCreating(true)} onImport={() => fileRef.current?.click()} onExport={exportCsv} />
      </div>

      {group && (
        <AddSymbolsDialog
          open={adding != null}
          onOpenChange={(o) => !o && setAdding(null)}
          groupName={group.name}
          existing={symbols}
          max={max}
          initial={adding?.initial}
          onConfirm={(syms) => update((s) => addSymbols(s, group.id, syms).store)}
        />
      )}
      <CreateGroupDialog
        open={creating}
        onOpenChange={setCreating}
        existingNames={groups.map((g) => g.name)}
        onConfirm={(name) => {
          const res = createGroup(store ?? { groups: [] }, name);
          if (res) {
            update((s) => createGroup(s, name)?.store ?? s);
            switchGroup(res.id);
          }
        }}
      />
      <ConfirmDialog
        open={confirm != null}
        onOpenChange={(o) => !o && setConfirm(null)}
        title={confirm?.title ?? ""}
        description={confirm?.description ?? ""}
        confirmLabel={confirm?.label ?? "Confirm"}
        destructive={confirm?.destructive ?? true}
        onConfirm={() => confirm?.action()}
      />
    </main>
  );
}
