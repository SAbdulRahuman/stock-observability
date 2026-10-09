"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { type Quote, fetchQuotes } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { cleanName, symbolPattern } from "@/lib/watchlist";

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  destructive = true,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: React.ReactNode;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title={title} description={description}>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button
            variant={destructive ? "critical" : "default"}
            onClick={() => {
              onConfirm();
              onOpenChange(false);
            }}
          >
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type Check = { valid: Quote[]; unknown: string[]; invalid: string[]; duplicate: string[]; overLimit: string[] };

export function splitSymbols(text: string): string[] {
  const out: string[] = [];
  for (const s of text.split(/[\s,;]+/)) {
    const sym = s.trim().toUpperCase();
    if (sym && !out.includes(sym)) out.push(sym);
  }
  return out;
}

/** Validates symbols against stock_home (they must have a recorded price) before the user confirms the add. */
export function AddSymbolsDialog({
  open,
  onOpenChange,
  groupName,
  existing,
  max,
  initial,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  groupName: string;
  existing: string[];
  max: number;
  initial?: string[];
  onConfirm: (symbols: string[]) => void;
}) {
  const [text, setText] = useState("");
  const [check, setCheck] = useState<Check | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const run = async (input: string) => {
    setBusy(true);
    setError(null);
    setCheck(null);
    const c: Check = { valid: [], unknown: [], invalid: [], duplicate: [], overLimit: [] };
    const candidates: string[] = [];
    for (const s of splitSymbols(input)) {
      if (!symbolPattern.test(s)) c.invalid.push(s);
      else if (existing.includes(s)) c.duplicate.push(s);
      else if (existing.length + candidates.length >= max) c.overLimit.push(s);
      else candidates.push(s);
    }
    try {
      if (candidates.length > 0) {
        for (const q of await fetchQuotes(candidates)) {
          if (q.price != null) c.valid.push(q);
          else c.unknown.push(q.symbol);
        }
      }
      setCheck(c);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    if (!open) return;
    const t = (initial ?? []).join(", ");
    setText(t);
    setCheck(null);
    setError(null);
    if (t) void run(t);
    // Reset and re-check only when the dialog opens.
  }, [open]);

  const n = check?.valid.length ?? 0;
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title={`Add symbols to ${groupName}`} description={`NSE trading symbols, separated by commas or spaces. A watchlist holds up to ${max} symbols.`}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void run(text);
          }}
          className="flex gap-2"
        >
          <input
            aria-label="Symbols"
            value={text}
            onChange={(e) => {
              setText(e.target.value);
              setCheck(null);
            }}
            placeholder="RELIANCE, TCS, INFY"
            className="h-9 min-w-0 flex-1 rounded-md border bg-background px-3 text-sm"
          />
          <Button type="submit" variant="outline" disabled={busy || !text.trim()}>
            {busy ? "Checking…" : "Check"}
          </Button>
        </form>
        {error && <p className="mt-3 text-sm text-critical">Could not validate symbols: {error}</p>}
        {check && (
          <ul className="mt-3 space-y-1 text-sm">
            {check.valid.map((q) => (
              <li key={q.symbol} className="flex justify-between">
                <span>✓ {q.symbol}</span>
                <span className="tabular-nums text-muted-foreground">{formatPrice(q.price)}</span>
              </li>
            ))}
            {check.unknown.length > 0 && <li className="text-warning">No market data: {check.unknown.join(", ")}</li>}
            {check.invalid.length > 0 && <li className="text-critical">Not a trading symbol: {check.invalid.join(", ")}</li>}
            {check.duplicate.length > 0 && <li className="text-muted-foreground">Already listed: {check.duplicate.join(", ")}</li>}
            {check.overLimit.length > 0 && <li className="text-warning">Over the {max}-symbol limit: {check.overLimit.join(", ")}</li>}
          </ul>
        )}
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button
            disabled={n === 0}
            onClick={() => {
              onConfirm(check?.valid.map((q) => q.symbol) ?? []);
              onOpenChange(false);
            }}
          >
            {n > 0 ? `Add ${n} symbol${n === 1 ? "" : "s"}` : "Add"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function CreateGroupDialog({
  open,
  onOpenChange,
  existingNames,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  existingNames: string[];
  onConfirm: (name: string) => void;
}) {
  const [name, setName] = useState("");
  useEffect(() => {
    if (open) setName("");
  }, [open]);
  const clean = cleanName(name);
  const taken = existingNames.some((n) => n.toLowerCase() === clean.toLowerCase());
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title="Create new watchlist" description="Watchlists are stored in this browser.">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (clean && !taken) {
              onConfirm(clean);
              onOpenChange(false);
            }
          }}
        >
          <input
            aria-label="Watchlist name"
            value={name}
            maxLength={40}
            onChange={(e) => setName(e.target.value)}
            className="h-9 w-full rounded-md border bg-background px-3 text-sm"
          />
          {taken && <p className="mt-2 text-sm text-warning">A watchlist with this name exists.</p>}
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={!clean || taken}>
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
