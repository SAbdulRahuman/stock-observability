"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useTheme } from "next-themes";
import { useRef, useState } from "react";
import { usePrivacy } from "@/components/privacy";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/watchlist/dialogs";
import { fetchProm, promSchemas } from "@/lib/prom";
import { SETTINGS_KEY, writeSettings } from "@/lib/settings";
import { clearStore, parseStore, readStore, type Store, WATCHLIST_KEY, writeStore } from "@/lib/watchlist";
import { type Notify, SettingsCard } from "./cards";

const MAX_IMPORT_BYTES = 256 << 10;

function today() {
  return new Date().toISOString().slice(0, 10);
}

function download(name: string, data: unknown) {
  const url = URL.createObjectURL(new Blob([`${JSON.stringify(data, null, 2)}\n`], { type: "application/json" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

type ImportPlan = { store: Store; symbols: number; dropped: number; fileName: string };

/** Validates a watchlist export; malformed groups and symbols are dropped and counted. */
export function planImport(text: string, fileName: string): ImportPlan | string {
  let raw: unknown;
  try {
    raw = JSON.parse(text);
  } catch {
    return "The file is not valid JSON.";
  }
  let given = 0;
  if (Array.isArray(raw)) given = raw.length;
  else if (raw && typeof raw === "object" && Array.isArray((raw as { groups?: unknown }).groups)) {
    for (const g of (raw as { groups: unknown[] }).groups) {
      const syms = (g as { symbols?: unknown } | null)?.symbols;
      given += Array.isArray(syms) ? syms.length : 0;
    }
  } else return "The file is not a stock_home watchlist export (expected {\"groups\": [...]}).";
  const store = parseStore(text);
  const symbols = store.groups.reduce((n, g) => n + g.symbols.length, 0);
  return { store, symbols, dropped: Math.max(0, given - symbols), fileName };
}

export function BackupCard({ icon, notify }: { icon: React.ComponentType<{ className?: string }>; notify: Notify }) {
  const { theme } = useTheme();
  const file = useRef<HTMLInputElement>(null);
  const [plan, setPlan] = useState<ImportPlan | null>(null);
  const [busy, setBusy] = useState(false);

  const exportJSON = (name: string, data: unknown) => {
    try {
      download(name, data);
      notify(true, `Exported ${name}.`);
    } catch (e) {
      notify(false, `Export failed: ${(e as Error).message}`);
    }
  };
  const exportRules = async () => {
    setBusy(true);
    try {
      const rules = await fetchProm("rules", promSchemas.rules);
      const groups = rules.data.groups
        .map((g) => ({
          name: g.name,
          rules: g.rules
            .filter((r) => r.type === "alerting")
            .map((r) => ({ alert: r.name, expr: r.query, for: r.duration, labels: r.labels ?? {}, annotations: r.annotations ?? {} })),
        }))
        .filter((g) => g.rules.length > 0);
      exportJSON(`stock_home-alert-rules-${today()}.json`, { source: "Prometheus rules loaded at export time (read-only)", groups });
    } catch (e) {
      notify(false, `Could not export alert rules: ${(e as Error).message}`);
    } finally {
      setBusy(false);
    }
  };
  const pick = async (f: File | undefined) => {
    if (file.current) file.current.value = "";
    if (!f) return;
    if (f.size > MAX_IMPORT_BYTES) {
      notify(false, "Import rejected: the file is larger than 256 KB.");
      return;
    }
    const res = planImport(await f.text(), f.name);
    if (typeof res === "string") notify(false, `Import rejected: ${res}`);
    else setPlan(res);
  };

  return (
    <SettingsCard title="Backup & Export" subtitle="Export and back up application data" icon={icon}>
      <div className="grid gap-2">
        <Button variant="outline" onClick={() => exportJSON(`stock_home-watchlists-${today()}.json`, readStore())}>
          Export Watchlists
        </Button>
        <Button variant="outline" onClick={() => void exportRules()} disabled={busy}>
          {busy ? "Exporting…" : "Export Alert Rules"}
        </Button>
        <Button
          variant="outline"
          onClick={() => exportJSON(`stock_home-settings-${today()}.json`, { settings: JSON.parse(localStorage.getItem(SETTINGS_KEY) ?? "{}"), theme: theme ?? null })}
        >
          Export Settings
        </Button>
        <Button variant="outline" onClick={() => file.current?.click()}>
          Import Watchlists
        </Button>
        <input ref={file} type="file" accept="application/json,.json" aria-label="Watchlist JSON file" className="hidden" onChange={(e) => void pick(e.target.files?.[0])} />
      </div>
      <p className="text-xs text-muted-foreground">
        Files are JSON. Watchlists and settings come from this browser; alert rules are a read-only copy of the rules Prometheus has loaded.
      </p>
      <Dialog open={plan != null} onOpenChange={(o) => !o && setPlan(null)}>
        {plan && (
          <DialogContent title="Replace watchlists?" description={`Importing ${plan.fileName} replaces all ${readStore().groups.length} watchlist group(s) in this browser.`}>
            <ul className="max-h-48 space-y-1 overflow-y-auto text-sm" aria-label="Import preview">
              {plan.store.groups.map((g) => (
                <li key={g.id} className="flex justify-between gap-3">
                  <span>{g.name}</span>
                  <span className="text-muted-foreground">
                    {g.symbols.length} symbol{g.symbols.length === 1 ? "" : "s"}
                  </span>
                </li>
              ))}
            </ul>
            {plan.dropped > 0 && <p className="text-sm text-warning">{plan.dropped} invalid or duplicate symbol(s) will be skipped.</p>}
            <DialogFooter>
              <DialogClose asChild>
                <Button variant="outline">Cancel</Button>
              </DialogClose>
              <Button
                variant="critical"
                onClick={() => {
                  writeStore(plan.store);
                  notify(true, `Imported ${plan.store.groups.length} group(s) with ${plan.symbols} symbol(s).`);
                  setPlan(null);
                }}
              >
                Replace watchlists
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </SettingsCard>
  );
}

export function DangerCard({ icon, notify }: { icon: React.ComponentType<{ className?: string }>; notify: Notify }) {
  const client = useQueryClient();
  const { setTheme } = useTheme();
  const privacy = usePrivacy();
  const [confirm, setConfirm] = useState<"reset" | "clear" | null>(null);
  return (
    <SettingsCard title="Danger Zone" subtitle="These actions are irreversible" icon={icon} danger>
      <div className="grid gap-2">
        <Button variant="outline" className="border-critical/60 text-critical" onClick={() => setConfirm("reset")}>
          Reset All Settings
        </Button>
        <Button variant="outline" className="border-critical/60 text-critical" onClick={() => setConfirm("clear")}>
          Clear Cache &amp; Local Data
        </Button>
      </div>
      <ConfirmDialog
        open={confirm === "reset"}
        onOpenChange={(o) => !o && setConfirm(null)}
        title="Reset all settings?"
        description="Restores this browser's preferences to their defaults: Dark theme, sparklines on and every Home widget shown. Watchlists are kept and nothing on the server changes."
        confirmLabel="Reset settings"
        onConfirm={() => {
          writeSettings(null);
          setTheme("dark");
          notify(true, "Settings were reset to their defaults.");
        }}
      />
      <ConfirmDialog
        open={confirm === "clear"}
        onOpenChange={(o) => !o && setConfirm(null)}
        title="Clear cache and local data?"
        description={`Deletes everything stock_home keeps in this browser: all watchlists (${WATCHLIST_KEY}), preferences (${SETTINGS_KEY}), the theme and the show-amounts choice, and drops cached API responses, which then reload. Export your watchlists first if you need them. Server data, Prometheus and Alertmanager are not affected.`}
        confirmLabel="Delete local data"
        onConfirm={() => {
          clearStore();
          writeSettings(null);
          setTheme("dark");
          if (!privacy.hidden) privacy.toggle();
          void client.resetQueries();
          notify(true, "Local data and cached responses were cleared.");
        }}
      />
    </SettingsCard>
  );
}
