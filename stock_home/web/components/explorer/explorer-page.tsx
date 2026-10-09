"use client";

import { useQueryClient } from "@tanstack/react-query";
import { ArrowRight, CircleAlert, RefreshCw, Search, Star, X } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useNow } from "@/components/dashboard/section-card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import { ConfirmDialog } from "@/components/watchlist/dialogs";
import { StockDetail } from "@/components/watchlist/detail";
import { type StockQuery, useSection, useStocks } from "@/lib/api";
import { formatChange, formatPercent, formatPrice, formatVolume, relativeTime, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";
import { addSymbols, DEFAULT_GROUP_ID, removeSymbols, useWatchlistStore } from "@/lib/watchlist";

const priceLabels: Record<string, string> = {
  all: "All prices",
  lt100: "Under ₹100",
  "100-500": "₹100 – ₹500",
  "500-2000": "₹500 – ₹2,000",
  gt2000: "Above ₹2,000",
};
const changeLabels: Record<string, string> = {
  all: "Any change",
  up: "Up today",
  down: "Down today",
  unchanged: "Unchanged",
  gt2: "Up 2% or more",
  "lt-2": "Down 2% or more",
};
const sortLabels: Record<string, string> = {
  "symbol:asc": "Symbol (A–Z)",
  "change:desc": "Change (high to low)",
  "change:asc": "Change (low to high)",
  "price:desc": "Price (high to low)",
  "price:asc": "Price (low to high)",
  "volume:desc": "Volume (high to low)",
};
const defaults: StockQuery = { q: "", sector: "", price: "all", change: "all", sort: "symbol", dir: "asc", page: 1, per: 25 };
const queryPattern = /^[A-Z0-9&.\- ]{0,32}$/;

type Confirm = { title: string; description: string; label: string; destructive: boolean; action: () => void };

export function ExplorerPage() {
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname() ?? "/explorer";
  const client = useQueryClient();
  const now = useNow(5_000);
  const urlQ = (params?.get("q") ?? "").trim().toUpperCase();
  const symbol = (params?.get("symbol") ?? "").trim().toUpperCase() || null;
  const [input, setInput] = useState(urlQ);
  const [query, setQuery] = useState<StockQuery>({ ...defaults, q: queryPattern.test(urlQ) ? urlQ : "" });
  const [invalid, setInvalid] = useState(false);
  const [confirm, setConfirm] = useState<Confirm | null>(null);
  // The header search navigates here with ?q=; follow it while the page is open.
  useEffect(() => {
    if (queryPattern.test(urlQ)) {
      setInput(urlQ);
      setQuery((cur) => (cur.q === urlQ ? cur : { ...cur, q: urlQ, page: 1 }));
    }
  }, [urlQ]);
  const stocks = useStocks(query);
  const info = useSection("info");
  const [store, update] = useWatchlistStore();
  const editable = !!info.data && store != null;
  const mine = new Set(store?.groups[0].symbols ?? []);
  const d = stocks.data?.data;
  const sectors = [...new Set([...(d?.sectorNames ?? []), ...(query.sector ? [query.sector] : [])])].sort();
  const filtered = query.q !== "" || query.sector !== "" || query.price !== "all" || query.change !== "all";

  const set = (patch: Partial<StockQuery>) => setQuery((cur) => ({ ...cur, page: 1, ...patch }));
  const go = (next: Record<string, string | null>) => {
    const sp = new URLSearchParams(params?.toString() ?? "");
    for (const [k, v] of Object.entries(next)) {
      if (v) sp.set(k, v);
      else sp.delete(k);
    }
    const qs = sp.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname);
  };
  const search = (text: string) => {
    const q = text.trim().toUpperCase();
    if (!queryPattern.test(q)) {
      setInvalid(true);
      return;
    }
    setInvalid(false);
    set({ q });
    go({ q });
  };
  const clear = () => {
    setInput("");
    setInvalid(false);
    setQuery(defaults);
    go({ q: null });
  };
  const toggleWatch = (sym: string) =>
    setConfirm(
      mine.has(sym)
        ? {
            title: `Remove ${sym} from My Watchlist?`,
            description: "The symbol stays in other watchlists.",
            label: "Remove",
            destructive: true,
            action: () => update((s) => removeSymbols(s, DEFAULT_GROUP_ID, [sym])),
          }
        : {
            title: `Add ${sym} to My Watchlist?`,
            description: "My Watchlist is stored in this browser and feeds the Home page.",
            label: "Add",
            destructive: false,
            action: () => update((s) => addSymbols(s, DEFAULT_GROUP_ID, [sym]).store),
          },
    );

  const from = d && d.total > 0 ? (d.page - 1) * d.per + 1 : 0;
  const to = d ? Math.min(d.total, d.page * d.per) : 0;
  const select = "h-9 rounded-lg border bg-card px-2 text-sm disabled:opacity-80";

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Stock Explorer</h1>
        <p className="text-sm text-muted-foreground">Find and inspect {info.data?.data.exchange || "NSE"} stocks</p>
      </div>

      <form
        role="search"
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          search(input);
        }}
      >
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <input
            aria-label="Search stocks by symbol"
            aria-invalid={invalid || undefined}
            placeholder="Search stocks by symbol (e.g. RELIANCE, TCS, INFY)…"
            value={input}
            onChange={(e) => {
              setInput(e.target.value);
              setInvalid(false);
            }}
            className={cn("h-10 w-full rounded-lg border bg-card pr-9 pl-9 text-sm", invalid && "border-critical")}
          />
          {input && (
            <button
              type="button"
              aria-label="Clear search"
              onClick={() => {
                setInput("");
                search("");
              }}
              className="absolute top-1/2 right-2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:text-foreground"
            >
              <X className="size-4" />
            </button>
          )}
        </div>
        <Button type="submit">Search</Button>
      </form>
      {invalid && <p className="text-sm text-critical">Use letters, digits, &amp;, ., - or spaces (up to 32 characters).</p>}

      <div className="flex flex-wrap items-center gap-2">
        <select aria-label="Exchange" className={select} value={info.data?.data.exchange ?? ""} disabled title="Only the configured exchange is available">
          <option value={info.data?.data.exchange ?? ""}>Exchange: {info.data?.data.exchange || "—"}</option>
        </select>
        <select aria-label="Sector" className={select} value={query.sector} onChange={(e) => set({ sector: e.target.value })}>
          <option value="">Sector: All</option>
          {sectors.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        <select aria-label="Price range" className={select} value={query.price} onChange={(e) => set({ price: e.target.value })}>
          {Object.entries(priceLabels).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </select>
        <select aria-label="Change" className={select} value={query.change} onChange={(e) => set({ change: e.target.value })}>
          {Object.entries(changeLabels).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </select>
        <Button variant="outline" onClick={clear} disabled={!filtered}>
          Clear
        </Button>
      </div>

      <div className={cn("grid gap-4", symbol && "xl:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]")}>
        <Card aria-label="Stock results" className="min-w-0">
          <div className="flex flex-wrap items-center gap-2 border-b px-4 py-3 text-sm">
            <span className="font-medium">Results: {d ? `${d.total.toLocaleString("en-IN")} stocks` : "—"}</span>
            {stocks.data?.stale && <Badge variant="warning">stale</Badge>}
            {stocks.data && <span className="text-xs text-muted-foreground">Updated {relativeTime(stocks.data.updatedAt, now)}</span>}
            <span className="ml-auto flex items-center gap-2">
              <select
                aria-label="Sort"
                className="h-8 rounded-lg border bg-card px-2 text-sm"
                value={`${query.sort}:${query.dir}`}
                onChange={(e) => {
                  const [sort, dir] = e.target.value.split(":");
                  set({ sort, dir });
                }}
              >
                {Object.entries(sortLabels).map(([k, v]) => (
                  <option key={k} value={k}>
                    Sort: {v}
                  </option>
                ))}
              </select>
              <Button variant="outline" size="sm" onClick={() => void client.invalidateQueries({ queryKey: ["stocks"] })} aria-busy={stocks.isFetching}>
                <RefreshCw className={cn(stocks.isFetching && "animate-spin")} /> Refresh
              </Button>
            </span>
          </div>
          {stocks.isPending ? (
            <div className="space-y-2 p-4" aria-busy>
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-6 w-5/6" />
            </div>
          ) : stocks.isError && !d ? (
            <p className="m-4 flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
              <CircleAlert className="size-4 text-warning" aria-hidden />
              Unavailable — {stocks.error.message}
            </p>
          ) : !d || d.items.length === 0 ? (
            <div className="p-6 text-center text-sm text-muted-foreground">
              {filtered ? (
                <>
                  No stocks match your search or filters.{" "}
                  <button type="button" className="text-primary hover:underline" onClick={clear}>
                    Clear search and filters
                  </button>
                </>
              ) : (
                "No stock data recorded yet. Search by NSE trading symbol once the exporter publishes prices."
              )}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-xs text-muted-foreground">
                    <th className="py-2 pl-4 text-left font-normal">Symbol</th>
                    <th className="py-2 text-left font-normal">Company</th>
                    <th className="py-2 text-left font-normal">Sector</th>
                    <th className="py-2 text-right font-normal">Price (₹)</th>
                    <th className="py-2 text-right font-normal">Change</th>
                    <th className="py-2 text-right font-normal">Volume</th>
                    <th className="py-2 pr-4 text-right font-normal">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {d.items.map((q) => (
                    <tr
                      key={`${q.exchange}:${q.symbol}`}
                      data-selected={q.symbol === symbol || undefined}
                      className={cn("cursor-pointer hover:bg-accent/60", q.symbol === symbol && "bg-primary/15")}
                      onClick={() => go({ symbol: q.symbol })}
                    >
                      <td className="py-2 pl-4 font-medium">{q.symbol}</td>
                      <td className="py-2 text-muted-foreground" title="Company names are not part of the recorded market data">
                        —
                      </td>
                      <td className="py-2 text-muted-foreground">{q.sector || "—"}</td>
                      <td className="py-2 text-right tabular-nums">{formatPrice(q.price)}</td>
                      <td className={cn("py-2 text-right whitespace-nowrap tabular-nums", trendClass(q.changePercent))}>
                        {formatChange(q.change)} ({formatPercent(q.changePercent)})
                      </td>
                      <td className="py-2 text-right tabular-nums">{formatVolume(q.volume)}</td>
                      <td className="py-2 pr-4 text-right whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                        <Button variant="ghost" size="sm" onClick={() => go({ symbol: q.symbol })} aria-label={`View ${q.symbol}`}>
                          View
                        </Button>
                        <button
                          type="button"
                          aria-pressed={mine.has(q.symbol)}
                          aria-label={mine.has(q.symbol) ? `Remove ${q.symbol} from My Watchlist` : `Add ${q.symbol} to My Watchlist`}
                          disabled={!editable}
                          onClick={() => toggleWatch(q.symbol)}
                          className="rounded-md p-1 align-middle hover:bg-accent disabled:opacity-50"
                        >
                          <Star className={cn("size-4", mine.has(q.symbol) ? "fill-warning text-warning" : "text-muted-foreground")} />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {d && d.total > 0 && (
            <nav aria-label="Pagination" className="flex flex-wrap items-center justify-between gap-2 border-t px-4 py-2 text-sm">
              <span className="text-muted-foreground">
                Showing {from}–{to} of {d.total.toLocaleString("en-IN")} stocks
              </span>
              <span className="flex items-center gap-2">
                <select aria-label="Rows per page" className="h-8 rounded-lg border bg-card px-2 text-sm" value={query.per} onChange={(e) => set({ per: Number(e.target.value) })}>
                  {[10, 25, 50].map((n) => (
                    <option key={n} value={n}>
                      {n} per page
                    </option>
                  ))}
                </select>
                <Button variant="outline" size="sm" disabled={d.page <= 1} onClick={() => setQuery((c) => ({ ...c, page: c.page - 1 }))}>
                  ‹ Previous
                </Button>
                <Button variant="outline" size="sm" disabled={to >= d.total} onClick={() => setQuery((c) => ({ ...c, page: c.page + 1 }))}>
                  Next ›
                </Button>
              </span>
            </nav>
          )}
        </Card>

        {symbol && (
          <div className="flex min-w-0 flex-col gap-2">
            <div className="flex items-center gap-3 text-sm">
              <Link href={`/analyst?symbol=${encodeURIComponent(symbol)}`} className="inline-flex items-center gap-1 font-medium text-primary hover:underline">
                Explain {symbol} in Analyst <ArrowRight className="size-3.5" aria-hidden />
              </Link>
              <button type="button" className="ml-auto inline-flex items-center gap-1 text-muted-foreground hover:text-foreground" onClick={() => go({ symbol: null })}>
                <X className="size-4" aria-hidden /> Close details
              </button>
            </div>
            <StockDetail
              symbol={symbol}
              favourite={mine.has(symbol)}
              editable={editable}
              onToggleFavourite={() => toggleWatch(symbol)}
              onSelect={(s) => go({ symbol: s })}
              onRemove={(s) => mine.has(s) && toggleWatch(s)}
            />
          </div>
        )}
      </div>

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
