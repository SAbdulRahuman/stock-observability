"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { ALERT_LIST_KEY, type AlertRow, createSilences, type SilenceKind, type SilenceResult, silenceDurations } from "@/lib/api";
import { ruleText } from "./shared";

export type Outcome = { ok: boolean; message: string };

/** Confirmation step for Acknowledge/Silence; Alertmanager has no acknowledge state, so both create a silence. */
export function SilenceDialog({
  kind,
  targets,
  onOpenChange,
  onDone,
}: {
  kind: SilenceKind | null;
  targets: AlertRow[];
  onOpenChange: (open: boolean) => void;
  onDone: (o: Outcome) => void;
}) {
  const client = useQueryClient();
  const [duration, setDuration] = useState("1h");
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (kind) {
      setDuration(kind === "acknowledge" ? "1h" : "2h");
      setComment("");
    }
  }, [kind]);

  const verb = kind === "acknowledge" ? "Acknowledge" : "Silence";
  const submit = async () => {
    setBusy(true);
    try {
      const res: SilenceResult = await createSilences({
        fingerprints: targets.flatMap((t) => (t.fingerprint ? [t.fingerprint] : [])),
        kind: kind ?? "silence",
        duration,
        comment,
      });
      const failed = res.failed.length ? ` ${res.failed.length} failed: ${[...new Set(res.failed.map((f) => f.error))].join("; ")}.` : "";
      onDone({ ok: res.created.length > 0, message: `${verb}d ${res.created.length} alert${res.created.length === 1 ? "" : "s"} for ${duration}.${failed}` });
    } catch (e) {
      onDone({ ok: false, message: `${verb} failed: ${(e as Error).message}` });
    } finally {
      setBusy(false);
      onOpenChange(false);
      void client.invalidateQueries({ queryKey: ALERT_LIST_KEY });
      void client.invalidateQueries({ queryKey: ["section", "alerts"] });
    }
  };

  return (
    <Dialog open={kind != null} onOpenChange={onOpenChange}>
      <DialogContent
        title={`${verb} ${targets.length} alert${targets.length === 1 ? "" : "s"}?`}
        description={
          kind === "acknowledge"
            ? "Creates an Alertmanager silence marked as an acknowledgement. Notifications stop until it expires."
            : "Creates an Alertmanager silence matching exactly these alerts. Notifications stop until it expires."
        }
      >
        <ul className="max-h-32 space-y-0.5 overflow-y-auto text-sm">
          {targets.map((t) => (
            <li key={t.key} className="truncate">
              <span className="font-medium">{t.symbol || "—"}</span> · {ruleText(t)}
            </li>
          ))}
        </ul>
        <div className="mt-3 grid grid-cols-[auto_1fr] items-center gap-2 text-sm">
          <label htmlFor="silence-duration">Duration</label>
          <select id="silence-duration" value={duration} onChange={(e) => setDuration(e.target.value)} className="h-9 rounded-md border bg-background px-2">
            {silenceDurations.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </select>
          <label htmlFor="silence-comment">Comment</label>
          <input
            id="silence-comment"
            value={comment}
            maxLength={200}
            onChange={(e) => setComment(e.target.value)}
            className="h-9 rounded-md border bg-background px-3"
            placeholder="Optional"
          />
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button onClick={() => void submit()} disabled={busy || targets.length === 0}>
            {busy ? "Working…" : `Confirm ${verb.toLowerCase()}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
