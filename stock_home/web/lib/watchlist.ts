import { useCallback, useEffect, useState } from "react";

/** Browser-storage watchlists until the SQLite store exists; the first group feeds the Home snapshot. */
export const WATCHLIST_KEY = "stock_home.watchlist.v1";
const CHANGE_EVENT = "stock_home:watchlist";
export const DEFAULT_GROUP_ID = "default";
export const DEFAULT_GROUP_NAME = "My Watchlist";
const MAX_GROUPS = 20;
/** Matches /api/v1/quotes; the backend reports the same limit as info.quoteSymbolsMax. */
export const MAX_SYMBOLS = 20;
export const symbolPattern = /^[A-Z0-9][A-Z0-9&.\- ]{0,31}$/;

export type Group = { id: string; name: string; symbols: string[] };
export type Store = { groups: Group[] };

function cleanSymbols(raw: unknown): string[] {
  if (!Array.isArray(raw)) return [];
  const out: string[] = [];
  for (const s of raw) {
    if (typeof s === "string" && symbolPattern.test(s) && !out.includes(s)) out.push(s);
  }
  return out.slice(0, MAX_SYMBOLS);
}

export function cleanName(name: string): string {
  return name.replace(/\s+/g, " ").trim().slice(0, 40);
}

/** Accepts the legacy symbol array and drops anything malformed. */
export function parseStore(json: string | null): Store {
  let raw: unknown = null;
  try {
    raw = JSON.parse(json ?? "null");
  } catch {
    raw = null;
  }
  const groups: Group[] = [];
  if (Array.isArray(raw)) {
    groups.push({ id: DEFAULT_GROUP_ID, name: DEFAULT_GROUP_NAME, symbols: cleanSymbols(raw) });
  } else if (raw && typeof raw === "object" && Array.isArray((raw as Store).groups)) {
    for (const g of (raw as Store).groups) {
      if (!g || typeof g.id !== "string" || typeof g.name !== "string" || groups.some((x) => x.id === g.id)) continue;
      const name = cleanName(g.name);
      if (name) groups.push({ id: g.id, name, symbols: cleanSymbols(g.symbols) });
    }
  }
  if (!groups.some((g) => g.id === DEFAULT_GROUP_ID)) groups.unshift({ id: DEFAULT_GROUP_ID, name: DEFAULT_GROUP_NAME, symbols: [] });
  groups.sort((a, b) => Number(b.id === DEFAULT_GROUP_ID) - Number(a.id === DEFAULT_GROUP_ID));
  return { groups: groups.slice(0, MAX_GROUPS) };
}

export function readStore(storage: Pick<Storage, "getItem"> | undefined = globalThis.localStorage): Store {
  return parseStore(storage?.getItem(WATCHLIST_KEY) ?? null);
}

export function writeStore(store: Store) {
  localStorage.setItem(WATCHLIST_KEY, JSON.stringify(store));
  window.dispatchEvent(new Event(CHANGE_EVENT));
}

export function readWatchlist(storage?: Pick<Storage, "getItem">): string[] {
  return readStore(storage).groups[0].symbols;
}

export function addSymbols(store: Store, groupId: string, symbols: string[]): { store: Store; added: string[] } {
  const added: string[] = [];
  const groups = store.groups.map((g) => {
    if (g.id !== groupId) return g;
    const next = [...g.symbols];
    for (const s of symbols) {
      if (next.length >= MAX_SYMBOLS) break;
      if (symbolPattern.test(s) && !next.includes(s)) {
        next.push(s);
        added.push(s);
      }
    }
    return { ...g, symbols: next };
  });
  return { store: { groups }, added };
}

export function removeSymbols(store: Store, groupId: string, symbols: string[]): Store {
  return { groups: store.groups.map((g) => (g.id === groupId ? { ...g, symbols: g.symbols.filter((s) => !symbols.includes(s)) } : g)) };
}

export function createGroup(store: Store, name: string): { store: Store; id: string } | null {
  const n = cleanName(name);
  if (!n || store.groups.length >= MAX_GROUPS || store.groups.some((g) => g.name.toLowerCase() === n.toLowerCase())) return null;
  const base = n.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "group";
  let id = base;
  for (let i = 2; store.groups.some((g) => g.id === id) || id === DEFAULT_GROUP_ID; i++) id = `${base}-${i}`;
  return { store: { groups: [...store.groups, { id, name: n, symbols: [] }] }, id };
}

export function deleteGroup(store: Store, groupId: string): Store {
  if (groupId === DEFAULT_GROUP_ID) return store;
  return { groups: store.groups.filter((g) => g.id !== groupId) };
}

/** Symbols from the first CSV column; a "symbol" header row is skipped. */
export function parseCsv(text: string): string[] {
  const out: string[] = [];
  for (const line of text.split(/\r?\n/)) {
    const cell = line.split(",")[0]?.replace(/^"|"$/g, "").trim().toUpperCase() ?? "";
    if (!cell || cell === "SYMBOL" || out.includes(cell)) continue;
    out.push(cell);
  }
  return out;
}

export function toCsv(group: Group): string {
  return `${["symbol", ...group.symbols].join("\n")}\n`;
}

/** Store and an updater that persists; null until read on the client. Syncs across tabs and components. */
export function useWatchlistStore(): [Store | null, (update: (s: Store) => Store) => void] {
  const [store, setStore] = useState<Store | null>(null);
  useEffect(() => {
    const reload = () => setStore(readStore());
    reload();
    const onStorage = (e: StorageEvent) => {
      if (e.key === WATCHLIST_KEY) reload();
    };
    window.addEventListener("storage", onStorage);
    window.addEventListener(CHANGE_EVENT, reload);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener(CHANGE_EVENT, reload);
    };
  }, []);
  const update = useCallback((fn: (s: Store) => Store) => writeStore(fn(readStore())), []);
  return [store, update];
}

/** Symbols of the default group; null until read on the client. */
export function useWatchlist(): string[] | null {
  const [store] = useWatchlistStore();
  return store ? store.groups[0].symbols : null;
}
