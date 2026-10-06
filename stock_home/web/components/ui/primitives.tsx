import type * as React from "react";
import { cn } from "@/lib/utils";

export function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("animate-pulse rounded-md bg-muted", className)} {...props} />;
}

/** Horizontal usage bar; value is a 0..1 fraction. */
export function Meter({ value, className, tone }: { value: number; className?: string; tone?: "ok" | "warning" | "critical" }) {
  const pct = Math.min(100, Math.max(0, value * 100));
  const auto = pct >= 90 ? "critical" : pct >= 75 ? "warning" : "ok";
  const color = { ok: "bg-ok", warning: "bg-warning", critical: "bg-critical" }[tone ?? auto];
  return (
    <div className={cn("h-1.5 w-full overflow-hidden rounded-full bg-muted", className)} role="meter" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
      <div className={cn("h-full rounded-full transition-[width]", color)} style={{ width: `${pct}%` }} />
    </div>
  );
}

export function StatusDot({ ok, className }: { ok: boolean | null; className?: string }) {
  const color = ok == null ? "bg-muted-foreground" : ok ? "bg-ok" : "bg-critical";
  return (
    <span className={cn("relative inline-flex size-2.5 shrink-0", className)}>
      {ok === false && <span className={cn("absolute inline-flex size-full animate-ping rounded-full opacity-60", color)} />}
      <span className={cn("relative inline-flex size-2.5 rounded-full", color)} />
    </span>
  );
}
