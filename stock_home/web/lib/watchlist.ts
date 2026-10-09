import { useEffect, useState } from "react";

/** Browser-storage watchlist until the SQLite store exists; shared with the Watchlist page. */
export const WATCHLIST_KEY = "stock_home.watchlist.v1";
const MAX = 20;
const symbolPattern = /^[A-Z0-9][A-Z0-9&.\- ]{0,31}$/;

export function readWatchlist(storage: Pick<Storage, "getItem"> | undefined = globalThis.localStorage): string[] {
  try {
    const raw: unknown = JSON.parse(storage?.getItem(WATCHLIST_KEY) ?? "[]");
    if (!Array.isArray(raw)) return [];
    const out: string[] = [];
    for (const s of raw) {
      if (typeof s === "string" && symbolPattern.test(s) && !out.includes(s)) out.push(s);
    }
    return out.slice(0, MAX);
  } catch {
    return [];
  }
}

/** Watchlist symbols; null until read on the client. */
export function useWatchlist(): string[] | null {
  const [symbols, setSymbols] = useState<string[] | null>(null);
  useEffect(() => {
    setSymbols(readWatchlist());
    const onStorage = (e: StorageEvent) => {
      if (e.key === WATCHLIST_KEY) setSymbols(readWatchlist());
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);
  return symbols;
}
