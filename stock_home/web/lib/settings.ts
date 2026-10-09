import { useCallback, useSyncExternalStore } from "react";
import { z } from "zod";

/** Browser-only preferences; nothing here is sent to the server. */
export const SETTINGS_KEY = "stock_home.settings.v1";
const CHANGE_EVENT = "stock_home:settings";

const homeDefaults = { indexCards: true, recentAlerts: true, topMovers: true, watchlistSnapshot: true };
const schema = z.object({
  showSparklines: z.boolean().catch(true),
  home: z
    .object({
      indexCards: z.boolean().catch(true),
      recentAlerts: z.boolean().catch(true),
      topMovers: z.boolean().catch(true),
      watchlistSnapshot: z.boolean().catch(true),
    })
    .catch(homeDefaults),
});

export type Settings = z.infer<typeof schema>;
export const defaultSettings: Settings = schema.parse({});

/** Unknown or malformed fields fall back to their defaults. */
export function parseSettings(json: string | null): Settings {
  try {
    const parsed = schema.safeParse(JSON.parse(json ?? "{}"));
    return parsed.success ? parsed.data : defaultSettings;
  } catch {
    return defaultSettings;
  }
}

let cachedRaw: string | null | undefined;
let cached = defaultSettings;

function snapshot(): Settings {
  const raw = localStorage.getItem(SETTINGS_KEY);
  if (raw !== cachedRaw) {
    cachedRaw = raw;
    cached = parseSettings(raw);
  }
  return cached;
}

function subscribe(onChange: () => void) {
  const onStorage = (e: StorageEvent) => {
    if (e.key === SETTINGS_KEY || e.key === null) onChange();
  };
  window.addEventListener("storage", onStorage);
  window.addEventListener(CHANGE_EVENT, onChange);
  return () => {
    window.removeEventListener("storage", onStorage);
    window.removeEventListener(CHANGE_EVENT, onChange);
  };
}

export function writeSettings(s: Settings | null) {
  if (s) localStorage.setItem(SETTINGS_KEY, JSON.stringify(s));
  else localStorage.removeItem(SETTINGS_KEY);
  window.dispatchEvent(new Event(CHANGE_EVENT));
}

/** Current settings (defaults during prerender) and a persisting updater shared by every component. */
export function useSettings(): [Settings, (update: (s: Settings) => Settings) => void] {
  const settings = useSyncExternalStore(subscribe, snapshot, () => defaultSettings);
  const update = useCallback((fn: (s: Settings) => Settings) => writeSettings(fn(snapshot())), []);
  return [settings, update];
}
