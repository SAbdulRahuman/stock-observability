"use client";

import type { UseQueryResult } from "@tanstack/react-query";
import { ArrowUpRight, CircleAlert } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/primitives";
import type { Envelope } from "@/lib/api";
import { relativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";

/** Card wrapper that handles loading, error and stale states for one API section. */
export function SectionCard<T>({
  id,
  title,
  icon: Icon,
  href,
  hrefLabel,
  query,
  className,
  children,
}: {
  id: string;
  title: string;
  icon: LucideIcon;
  href?: string;
  hrefLabel?: string;
  query: UseQueryResult<Envelope<T>>;
  className?: string;
  children: (data: T) => React.ReactNode;
}) {
  const env = query.data;
  return (
    <Card id={id} className={cn("min-w-0", className)} aria-labelledby={`${id}-title`}>
      <CardHeader>
        <Icon className="size-4 text-muted-foreground" aria-hidden />
        <CardTitle id={`${id}-title`}>{title}</CardTitle>
        <div className="ml-auto flex items-center gap-2">
          {(env?.stale || (query.isError && env)) && <Badge variant="warning">stale</Badge>}
          {env && <Updated at={env.updatedAt} />}
          {href && (
            <a
              href={href}
              className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
              aria-label={hrefLabel ?? `Open ${title}`}
              title={hrefLabel ?? `Open ${title}`}
            >
              <ArrowUpRight className="size-4" />
            </a>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {env ? (
          children(env.data)
        ) : query.isError ? (
          <div className="flex items-center gap-2 rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
            <CircleAlert className="size-4 text-warning" aria-hidden />
            Unavailable — {query.error.message}
          </div>
        ) : (
          <div className="space-y-3" aria-busy>
            <Skeleton className="h-6 w-2/3" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function Updated({ at }: { at: string }) {
  const now = useNow(5_000);
  return (
    <span className="hidden text-xs text-muted-foreground sm:inline" title={new Date(at).toLocaleString("en-IN")}>
      {relativeTime(at, now)}
    </span>
  );
}

export function useNow(intervalMs = 1_000) {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), intervalMs);
    return () => clearInterval(t);
  }, [intervalMs]);
  return now;
}

export function Stat({
  label,
  value,
  hint,
  className,
}: {
  label: string;
  value: React.ReactNode;
  hint?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("min-w-0", className)}>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="truncate text-lg font-semibold tabular-nums">{value}</div>
      {hint && <div className="truncate text-xs text-muted-foreground">{hint}</div>}
    </div>
  );
}

export function Row({ label, children }: { label: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 py-1.5 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-medium tabular-nums">{children}</span>
    </div>
  );
}

export function Empty({ children }: { children: React.ReactNode }) {
  return <p className="py-6 text-center text-sm text-muted-foreground">{children}</p>;
}
