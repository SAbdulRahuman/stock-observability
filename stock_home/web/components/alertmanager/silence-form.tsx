"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, TriangleAlert } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { type MatcherInput, type SilencePreview, submitSilence } from "@/lib/am";

const durations: Record<string, number> = { "1h": 1, "2h": 2, "4h": 4, "8h": 8, "24h": 24, "7d": 168 };
const ops: MatcherInput["op"][] = ["=", "!=", "=~", "!~"];
const input = "h-9 rounded-md border bg-background px-2 text-sm";
const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
const fmt = (iso: string) => new Date(iso).toLocaleString("en-IN", { dateStyle: "medium", timeStyle: "short" });

/** Create Silence: edit matchers and window, preview the alerts they select, then confirm. */
export function SilenceForm({
  open,
  onOpenChange,
  initial,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initial: MatcherInput[];
  onCreated: (message: string) => void;
}) {
  const client = useQueryClient();
  const [matchers, setMatchers] = useState<MatcherInput[]>(initial.length ? initial : [{ name: "alertname", op: "=", value: "" }]);
  const [startAt, setStartAt] = useState("");
  const [duration, setDuration] = useState("2h");
  const [endAt, setEndAt] = useState("");
  const [comment, setComment] = useState("");
  const [preview, setPreview] = useState<SilencePreview | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const timeWindow = () => {
    const start = startAt ? new Date(startAt) : new Date();
    const end = duration === "custom" ? new Date(endAt) : new Date(start.getTime() + durations[duration] * 3_600_000);
    return { startsAt: startAt ? start.toISOString() : "", endsAt: Number.isNaN(end.getTime()) ? "" : end.toISOString() };
  };
  const edit = (i: number, patch: Partial<MatcherInput>) => {
    setPreview(null);
    setMatchers((ms) => ms.map((m, j) => (j === i ? { ...m, ...patch } : m)));
  };
  const call = async (dryRun: boolean) => {
    setBusy(true);
    setError(null);
    try {
      const res = await submitSilence({ matchers, ...timeWindow(), comment, dryRun });
      if (dryRun) {
        setPreview(res);
      } else {
        onCreated(`Silence ${res.silenceId ?? ""} created for ${res.matches} matching alert${res.matches === 1 ? "" : "s"}.`);
        void client.invalidateQueries({ queryKey: ["am"] });
        onOpenChange(false);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title="Create silence" description="Notifications for matching alerts stop during the window. Nothing is created until you confirm." className="w-[min(40rem,calc(100vw-2rem))]">
        <div className="space-y-4 text-sm">
          <fieldset className="space-y-2">
            <legend className="mb-1 font-medium">Matchers</legend>
            {matchers.map((m, i) => (
              <div key={i} className="flex gap-2">
                <input aria-label={`Matcher ${i + 1} label`} className={`${input} w-36`} value={m.name} onChange={(e) => edit(i, { name: e.target.value.trim() })} />
                <select aria-label={`Matcher ${i + 1} operator`} className={input} value={m.op} onChange={(e) => edit(i, { op: e.target.value as MatcherInput["op"] })}>
                  {ops.map((o) => (
                    <option key={o} value={o}>
                      {o}
                    </option>
                  ))}
                </select>
                <input aria-label={`Matcher ${i + 1} value`} className={`${input} min-w-0 flex-1`} value={m.value} onChange={(e) => edit(i, { value: e.target.value })} />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={`Remove matcher ${i + 1}`}
                  disabled={matchers.length === 1}
                  onClick={() => {
                    setPreview(null);
                    setMatchers((ms) => ms.filter((_, j) => j !== i));
                  }}
                >
                  <Trash2 />
                </Button>
              </div>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={() => setMatchers((ms) => [...ms, { name: "", op: "=", value: "" }])} disabled={matchers.length >= 20}>
              <Plus /> Add matcher
            </Button>
          </fieldset>
          <div className="grid grid-cols-[auto_1fr] items-center gap-2">
            <label htmlFor="silence-start">Starts</label>
            <input
              id="silence-start"
              type="datetime-local"
              className={input}
              value={startAt}
              onChange={(e) => {
                setPreview(null);
                setStartAt(e.target.value);
              }}
            />
            <label htmlFor="silence-duration">Duration</label>
            <div className="flex gap-2">
              <select
                id="silence-duration"
                className={input}
                value={duration}
                onChange={(e) => {
                  setPreview(null);
                  setDuration(e.target.value);
                }}
              >
                {Object.keys(durations).map((d) => (
                  <option key={d} value={d}>
                    {d}
                  </option>
                ))}
                <option value="custom">Until…</option>
              </select>
              {duration === "custom" && (
                <input aria-label="Ends" type="datetime-local" className={input} value={endAt} onChange={(e) => setEndAt(e.target.value)} />
              )}
            </div>
            <span className="text-muted-foreground">Creator</span>
            <span className="text-muted-foreground">stock_home (no user identity is available)</span>
            <label htmlFor="silence-comment">Comment</label>
            <input
              id="silence-comment"
              className={input}
              maxLength={200}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder="Required: why are these alerts silenced?"
            />
          </div>
          <p className="text-xs text-muted-foreground">Times are in your browser time zone ({tz}); leave Starts empty to start now.</p>

          {preview && (
            <div className="rounded-lg border p-3" role="status" aria-label="Silence preview">
              <p className="font-medium">
                Silences {preview.matches} active alert{preview.matches === 1 ? "" : "s"} from {fmt(preview.startsAt)} to {fmt(preview.endsAt)}
              </p>
              <p className="mt-1 font-mono text-xs">{matchers.map((m) => `${m.name}${m.op}"${m.value}"`).join(", ")}</p>
              {preview.sample.length > 0 && (
                <ul className="mt-2 max-h-24 list-disc overflow-y-auto pl-5 text-xs text-muted-foreground">
                  {preview.sample.map((s, i) => (
                    <li key={i}>{s}</li>
                  ))}
                </ul>
              )}
              {preview.broad && (
                <p className="mt-2 flex items-center gap-2 text-warning">
                  <TriangleAlert className="size-4 shrink-0" aria-hidden />
                  Broad silence: it matches many alerts or has no exact alertname matcher.
                </p>
              )}
            </div>
          )}
          {error && <p className="text-critical">Could not {preview ? "create" : "preview"} the silence: {error}</p>}
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          {preview ? (
            <Button variant="critical" onClick={() => void call(false)} disabled={busy}>
              {busy ? "Creating…" : "Confirm create silence"}
            </Button>
          ) : (
            <Button onClick={() => void call(true)} disabled={busy || !comment.trim()}>
              {busy ? "Checking…" : "Preview"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
