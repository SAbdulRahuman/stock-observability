"use client";

import { Building2, FileSpreadsheet, Gauge, LineChart, Newspaper, Search, TrendingUp } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { formatTimeIST } from "@/lib/format";
import { DEFAULTS, PROVIDERS, RANGES, type Range, rangeStart, schemas, useCommand } from "@/lib/openbb";
import { cn } from "@/lib/utils";
import { CommandCard, FieldTable, KeyMetrics, NewsList, PriceChart, ProfileView, QuoteHeadline, StatementTable } from "./panels";

const inputClass = "h-9 rounded-md border bg-card px-3 text-sm focus-visible:outline-2 focus-visible:outline-offset-2";

export const TABS = [
  { id: "overview", label: "Overview" },
  { id: "price", label: "Price & Volume" },
  { id: "fundamentals", label: "Fundamentals" },
  { id: "financials", label: "Financials" },
  { id: "news", label: "News" },
] as const;
type Tab = (typeof TABS)[number]["id"];

/** yfinance tickers carry the listing exchange as a suffix; other providers take plain tickers. */
const EXCHANGES = [
  { id: "NSE", suffix: ".NS" },
  { id: "BSE", suffix: ".BO" },
];

function splitSymbol(symbol: string): { base: string; exchange: string } {
  const ex = EXCHANGES.find((e) => symbol.endsWith(e.suffix));
  return ex ? { base: symbol.slice(0, -ex.suffix.length), exchange: ex.id } : { base: symbol, exchange: "" };
}

/** OpenBB Analysis: one symbol across summary, price history, metrics, financials and news; state lives in the URL. */
export function Workbench() {
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const symbol = (params.get("symbol") || DEFAULTS.symbol).toUpperCase();
  const provider = params.get("provider") || DEFAULTS.provider;
  const rangeParam = params.get("range");
  const range: Range = rangeParam && rangeParam in RANGES ? (rangeParam as Range) : DEFAULTS.range;
  const tab: Tab = TABS.find((t) => t.id === params.get("tab"))?.id ?? "overview";
  const period = params.get("period") === "quarter" ? "quarter" : "annual";

  const update = (next: Record<string, string>) => {
    const qs = new URLSearchParams(params.toString());
    for (const [k, v] of Object.entries(next)) qs.set(k, v);
    router.replace(`${pathname}?${qs}`, { scroll: false });
  };

  const on = (...tabs: Tab[]) => tabs.includes(tab);
  const quote = useCommand("equity/price/quote", { symbol, provider }, schemas.quote, { refetchInterval: 30_000 });
  const metrics = useCommand("equity/fundamental/metrics", { symbol, provider }, schemas.metrics);
  const history = useCommand("equity/price/historical", { symbol, provider, start_date: rangeStart(range) }, schemas.historical, { enabled: on("overview", "price") });
  const profile = useCommand("equity/profile", { symbol, provider }, schemas.profile, { enabled: on("overview", "fundamentals") });
  const news = useCommand("news/company", { symbol, provider, limit: 10 }, schemas.news, { enabled: on("overview", "news") });
  const income = useCommand("equity/fundamental/income", { symbol, provider, period, limit: 4 }, schemas.income, { enabled: on("financials") });
  const marketCap = metrics.data?.results?.[0]?.market_cap;

  const rangeButtons = (
    <div className="flex gap-1" role="group" aria-label="Range">
      {(Object.keys(RANGES) as Range[]).map((r) => (
        <Button key={r} type="button" size="sm" variant={r === range ? "default" : "ghost"} aria-pressed={r === range} onClick={() => update({ range: r })}>
          {r}
        </Button>
      ))}
    </div>
  );
  const priceCard = (
    <CommandCard id="history" title="Price & Volume" icon={LineChart} query={history} className="lg:col-span-2" actions={rangeButtons}>
      {(rows) => <PriceChart bars={rows} />}
    </CommandCard>
  );
  const keyMetrics = (
    <CommandCard id="key-metrics" title="Key Metrics" icon={Gauge} query={quote}>
      {(rows) => <KeyMetrics quote={rows[0]} />}
    </CommandCard>
  );

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-5 p-4 md:p-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">OpenBB Analysis</h1>
        <p className="text-sm text-muted-foreground">Research and inspect a selected stock</p>
      </header>

      <Toolbar key={`${symbol}|${provider}`} symbol={symbol} provider={provider} onChange={update} />

      <div role="tablist" aria-label="Analysis sections" className="flex flex-wrap gap-1 border-b">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            id={`tab-${t.id}`}
            aria-selected={t.id === tab}
            aria-controls="analysis-panel"
            onClick={() => update({ tab: t.id })}
            className={cn(
              "-mb-px border-b-2 px-3 py-2 text-sm font-medium transition-colors",
              t.id === tab ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground",
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <CommandCard id="selected" title={`Selected Stock · ${symbol}`} icon={TrendingUp} query={quote}>
        {(rows) => (
          <div className="space-y-2">
            <QuoteHeadline quote={rows[0]} marketCap={typeof marketCap === "number" ? marketCap : null} />
            <p className="text-xs text-muted-foreground">
              Data status: delayed · fetched {formatTimeIST(new Date(quote.dataUpdatedAt))} IST via OpenBB ({quote.data?.provider ?? provider})
            </p>
          </div>
        )}
      </CommandCard>

      <div id="analysis-panel" role="tabpanel" aria-labelledby={`tab-${tab}`} className="flex flex-col gap-5">
        {tab === "overview" && (
          <>
            <div className="grid gap-5 lg:grid-cols-3">
              {priceCard}
              {keyMetrics}
            </div>
            <div className="grid gap-5 lg:grid-cols-3">
              <CommandCard id="profile" title="Fundamental Summary" icon={Building2} query={profile} className="lg:col-span-2">
                {(rows) => <ProfileView profile={rows[0]} />}
              </CommandCard>
              <CommandCard
                id="news"
                title="Related News"
                icon={Newspaper}
                query={news}
                actions={
                  <Button size="sm" variant="ghost" onClick={() => update({ tab: "news" })}>
                    View news
                  </Button>
                }
              >
                {(rows) => <NewsList items={rows.slice(0, 5)} />}
              </CommandCard>
            </div>
          </>
        )}
        {tab === "price" && (
          <div className="grid gap-5 lg:grid-cols-3">
            {priceCard}
            {keyMetrics}
          </div>
        )}
        {tab === "fundamentals" && (
          <div className="grid gap-5 lg:grid-cols-3">
            <CommandCard id="metrics" title="Fundamental Metrics" icon={Gauge} query={metrics} className="lg:col-span-2">
              {(rows) => <FieldTable row={rows[0]} />}
            </CommandCard>
            <CommandCard id="profile" title="Company Profile" icon={Building2} query={profile}>
              {(rows) => <ProfileView profile={rows[0]} />}
            </CommandCard>
          </div>
        )}
        {tab === "financials" && (
          <CommandCard
            id="income"
            title={`Income Statement (${period === "quarter" ? "quarterly" : "annual"})`}
            icon={FileSpreadsheet}
            query={income}
            actions={
              <div className="flex gap-1" role="group" aria-label="Reporting period">
                {(["annual", "quarter"] as const).map((p) => (
                  <Button key={p} size="sm" variant={p === period ? "default" : "ghost"} aria-pressed={p === period} onClick={() => update({ period: p })}>
                    {p === "annual" ? "Annual" : "Quarterly"}
                  </Button>
                ))}
              </div>
            }
          >
            {(rows) => (
              <div className="space-y-2">
                <StatementTable rows={rows} />
                <p className="text-xs text-muted-foreground">Amounts in the reporting currency as returned by the provider; periods are not mixed.</p>
              </div>
            )}
          </CommandCard>
        )}
        {tab === "news" && (
          <CommandCard id="news" title="Related News" icon={Newspaper} query={news}>
            {(rows) => <NewsList items={rows} />}
          </CommandCard>
        )}
      </div>

      <p className="text-center text-xs text-muted-foreground">Data via OpenBB Platform · provider {provider} · delayed, for information only</p>
    </main>
  );
}

function Toolbar({ symbol, provider, onChange }: { symbol: string; provider: string; onChange: (next: Record<string, string>) => void }) {
  const split = splitSymbol(symbol);
  const suffixed = provider === "yfinance";
  const [draft, setDraft] = useState(suffixed ? split.base : symbol);
  const [exchange, setExchange] = useState(suffixed ? split.exchange : "");
  const [query, setQuery] = useState("");
  // equity/search has no yfinance provider; yfinance ships its own search command.
  const searchPath = provider === "yfinance" ? "yfinance/search" : "equity/search";
  const searchParams = provider === "yfinance" ? { query } : { query, provider };
  const search = useCommand(searchPath, searchParams, schemas.search, { enabled: query !== "" });

  const choose = (s: string) => {
    setQuery("");
    onChange({ symbol: s });
  };

  return (
    <Card>
      <CardContent className="flex flex-col gap-3 pt-5">
        <div className="flex flex-wrap items-end gap-3">
          <form
            className="flex items-end gap-2"
            role="search"
            onSubmit={(e) => {
              e.preventDefault();
              const base = draft.trim().toUpperCase();
              if (!base) return;
              const ex = EXCHANGES.find((x) => x.id === exchange);
              choose(ex && !base.includes(".") ? base + ex.suffix : base);
            }}
          >
            <label className="flex flex-col gap-1 text-xs text-muted-foreground">
              Symbol
              <input
                className={cn(inputClass, "w-40 uppercase text-foreground")}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder={suffixed ? "RELIANCE" : "AAPL"}
                aria-label="Symbol"
              />
            </label>
            <label className="flex flex-col gap-1 text-xs text-muted-foreground">
              Exchange
              <select
                className={cn(inputClass, "text-foreground")}
                value={exchange}
                onChange={(e) => setExchange(e.target.value)}
                aria-label="Exchange"
                disabled={!suffixed}
                title={suffixed ? undefined : "Exchange suffixes apply to the yfinance provider only"}
              >
                {EXCHANGES.map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.id}
                  </option>
                ))}
                <option value="">As typed</option>
              </select>
            </label>
            <Button type="submit">Analyze</Button>
          </form>

          <form
            className="flex items-end gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              const q = new FormData(e.currentTarget).get("q");
              setQuery(typeof q === "string" ? q.trim() : "");
            }}
          >
            <label className="flex flex-col gap-1 text-xs text-muted-foreground">
              Find company
              <input name="q" className={cn(inputClass, "w-56 text-foreground")} placeholder="Name or ticker" aria-label="Find company" />
            </label>
            <Button type="submit" variant="outline" size="icon" aria-label="Search" title="Search">
              <Search />
            </Button>
          </form>

          <label className="flex flex-col gap-1 text-xs text-muted-foreground">
            Provider
            <select className={cn(inputClass, "text-foreground")} value={provider} onChange={(e) => onChange({ provider: e.target.value })} aria-label="Provider">
              {(PROVIDERS as readonly string[]).includes(provider) ? null : <option value={provider}>{provider}</option>}
              {PROVIDERS.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </label>
        </div>

        {query !== "" && (
          <div className="rounded-lg border p-2 text-sm" aria-live="polite">
            {search.isError ? (
              <p className="p-2 text-muted-foreground">Search unavailable — {search.error.message}</p>
            ) : !search.data ? (
              <p className="p-2 text-muted-foreground">Searching…</p>
            ) : !search.data.results?.length ? (
              <p className="p-2 text-muted-foreground">No matches for “{query}”</p>
            ) : (
              <ul className="max-h-64 overflow-y-auto" aria-label="Search results">
                {search.data.results
                  .filter((r) => r.symbol)
                  .slice(0, 20)
                  .map((r) => (
                    <li key={r.symbol}>
                      <button
                        type="button"
                        className="flex w-full items-center justify-between gap-3 rounded-md px-2 py-1.5 text-left hover:bg-accent"
                        onClick={() => choose(String(r.symbol).toUpperCase())}
                      >
                        <span className="font-medium">{r.symbol}</span>
                        <span className="truncate text-muted-foreground">
                          {r.name}
                          {r.exchange && ` · ${r.exchange}`}
                        </span>
                      </button>
                    </li>
                  ))}
              </ul>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
