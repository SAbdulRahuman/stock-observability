"use client";

import { Building2, LineChart, Newspaper, Search, TrendingUp } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { DEFAULTS, PROVIDERS, RANGES, type Range, rangeStart, schemas, useCommand } from "@/lib/openbb";
import { cn } from "@/lib/utils";
import { CommandCard, NewsList, PriceChart, ProfileView, QuoteView } from "./panels";

const inputClass = "h-9 rounded-md border bg-card px-3 text-sm focus-visible:outline-2 focus-visible:outline-offset-2";

/** OpenBB tab: one symbol across quote, price history, profile and news; state lives in the URL. */
export function Workbench() {
  const params = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const symbol = (params.get("symbol") || DEFAULTS.symbol).toUpperCase();
  const provider = params.get("provider") || DEFAULTS.provider;
  const rangeParam = params.get("range");
  const range: Range = rangeParam && rangeParam in RANGES ? (rangeParam as Range) : DEFAULTS.range;

  const update = (next: Record<string, string>) => {
    const qs = new URLSearchParams(params.toString());
    for (const [k, v] of Object.entries(next)) qs.set(k, v);
    router.replace(`${pathname}?${qs}`, { scroll: false });
  };

  const quote = useCommand("equity/price/quote", { symbol, provider }, schemas.quote, { refetchInterval: 30_000 });
  const history = useCommand("equity/price/historical", { symbol, provider, start_date: rangeStart(range) }, schemas.historical);
  const profile = useCommand("equity/profile", { symbol, provider }, schemas.profile);
  const news = useCommand("news/company", { symbol, provider, limit: 10 }, schemas.news);

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-5 p-4 md:p-6">
      <header>
        <h1 className="text-xl font-semibold tracking-tight">OpenBB</h1>
        <p className="text-sm text-muted-foreground">Quotes, price history, company profile and news from the OpenBB Platform</p>
      </header>

      <Toolbar key={symbol} symbol={symbol} provider={provider} range={range} onChange={update} />

      <div className="grid gap-5 lg:grid-cols-3">
        <CommandCard id="quote" title={`Quote · ${symbol}`} icon={TrendingUp} query={quote}>
          {(rows) => <QuoteView quote={rows[0]} />}
        </CommandCard>
        <CommandCard id="history" title="Price history" icon={LineChart} query={history} className="lg:col-span-2">
          {(rows) => <PriceChart bars={rows} />}
        </CommandCard>
      </div>
      <div className="grid gap-5 lg:grid-cols-3">
        <CommandCard id="profile" title="Company profile" icon={Building2} query={profile}>
          {(rows) => <ProfileView profile={rows[0]} />}
        </CommandCard>
        <CommandCard id="news" title="Company news" icon={Newspaper} query={news} className="lg:col-span-2">
          {(rows) => <NewsList items={rows} />}
        </CommandCard>
      </div>

      <footer className="pb-2 text-center text-xs text-muted-foreground">
        Data via OpenBB Platform · provider {provider} · delayed, for information only
      </footer>
    </main>
  );
}

function Toolbar({
  symbol,
  provider,
  range,
  onChange,
}: {
  symbol: string;
  provider: string;
  range: Range;
  onChange: (next: Record<string, string>) => void;
}) {
  const [draft, setDraft] = useState(symbol);
  const [query, setQuery] = useState("");
  // equity/search has no yfinance provider; yfinance ships its own search command.
  const searchPath = provider === "yfinance" ? "yfinance/search" : "equity/search";
  const searchParams = provider === "yfinance" ? { query } : { query, provider };
  const search = useCommand(searchPath, searchParams, schemas.search, { enabled: query !== "" });

  const choose = (s: string) => {
    setDraft(s);
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
              const s = draft.trim().toUpperCase();
              if (s) choose(s);
            }}
          >
            <label className="flex flex-col gap-1 text-xs text-muted-foreground">
              Symbol
              <input
                className={cn(inputClass, "w-40 uppercase text-foreground")}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="RELIANCE.NS"
                aria-label="Symbol"
              />
            </label>
            <Button type="submit" variant="outline">
              Go
            </Button>
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

          <fieldset className="flex flex-col gap-1">
            <legend className="mb-1 text-xs text-muted-foreground">Range</legend>
            <div className="flex gap-1">
              {(Object.keys(RANGES) as Range[]).map((r) => (
                <Button key={r} type="button" size="sm" variant={r === range ? "default" : "ghost"} aria-pressed={r === range} onClick={() => onChange({ range: r })}>
                  {r}
                </Button>
              ))}
            </div>
          </fieldset>
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
