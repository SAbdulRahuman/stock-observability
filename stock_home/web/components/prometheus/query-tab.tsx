"use client";

import { useMutation } from "@tanstack/react-query";
import { CircleAlert, Copy, Play, X } from "lucide-react";
import { useState } from "react";
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Empty } from "@/components/dashboard/section-card";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { IST } from "@/lib/format";
import { exampleQueries, fetchProm, promSchemas, type PromQueryData, useMetricNames } from "@/lib/prom";
import { cn } from "@/lib/utils";
import { LabelList } from "./tabs";

const ranges: Record<string, number> = { "15m": 900, "1h": 3600, "6h": 21600, "24h": 86400, "7d": 604800 };
const MAX_SERIES = 20;
const palette = ["#22c55e", "#3b82f6", "#ef4444", "#eab308", "#a855f7", "#14b8a6", "#f97316", "#ec4899"];
const tick = new Intl.DateTimeFormat("en-IN", { timeZone: IST, hour: "2-digit", minute: "2-digit", hour12: false });

/** About 250 points per series, rounded to whole seconds. */
export function autoStep(rangeSeconds: number): number {
  return Math.max(1, Math.round(rangeSeconds / 250));
}

const seriesName = (m: Record<string, string>) => {
  const { __name__, ...rest } = m;
  const inner = Object.entries(rest)
    .map(([k, v]) => `${k}="${v}"`)
    .join(", ");
  return `${__name__ ?? ""}{${inner}}`;
};

export function QueryTab({ initial }: { initial: string }) {
  const [expr, setExpr] = useState(initial);
  const [mode, setMode] = useState<"table" | "graph">("table");
  const [range, setRange] = useState("1h");
  const [stepText, setStepText] = useState("");
  const [evalAt, setEvalAt] = useState("");
  const [copied, setCopied] = useState("");
  const names = useMetricNames(true);

  const run = useMutation({
    mutationFn: async () => {
      const q = expr.trim();
      const end = evalAt ? new Date(evalAt).getTime() / 1000 : Date.now() / 1000;
      if (mode === "table") {
        return fetchProm("query", promSchemas.query, { query: q, ...(evalAt ? { time: String(end) } : {}) });
      }
      const span = ranges[range];
      const step = stepText ? Number(stepText) : autoStep(span);
      return fetchProm("query_range", promSchemas.query, { query: q, start: String(Math.floor(end - span)), end: String(Math.floor(end)), step: String(step) });
    },
  });
  const res = run.data;
  const copy = async (what: "query" | "results") => {
    const text = what === "query" ? expr : JSON.stringify(res?.data ?? null, null, 2);
    try {
      await navigator.clipboard.writeText(text);
      setCopied(what);
    } catch {
      setCopied("");
    }
  };

  return (
    <div className="space-y-3">
      <Card className="space-y-3 p-4">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (expr.trim()) run.mutate();
          }}
          className="space-y-3"
        >
          <textarea
            aria-label="PromQL expression"
            rows={3}
            value={expr}
            onChange={(e) => setExpr(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.ctrlKey || e.metaKey) && expr.trim()) run.mutate();
            }}
            placeholder="Enter a PromQL expression, e.g. up"
            className="w-full rounded-lg border bg-background p-3 font-mono text-sm"
          />
          <input aria-label="Metric name lookup" list="metric-names" placeholder={names.data ? `Browse ${names.data.data.length} metric names…` : "Metric names loading…"} className="h-9 w-full max-w-md rounded-lg border bg-card px-3 text-sm" onChange={(e) => names.data?.data.includes(e.target.value) && setExpr(e.target.value)} />
          <datalist id="metric-names">
            {names.data?.data.slice(0, 2000).map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>
          <div className="flex flex-wrap items-center gap-2">
            <Button type="submit" disabled={!expr.trim() || run.isPending}>
              <Play /> {run.isPending ? "Running…" : "Execute"}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setExpr("");
                run.reset();
              }}
            >
              <X /> Clear
            </Button>
            <select aria-label="Use example" className="h-9 rounded-lg border bg-card px-2 text-sm" value="" onChange={(e) => e.target.value && setExpr(e.target.value)}>
              <option value="">Use example…</option>
              {exampleQueries.map((x) => (
                <option key={x.query} value={x.query}>
                  {x.label}
                </option>
              ))}
            </select>
            <label className="flex items-center gap-2 text-sm text-muted-foreground">
              Evaluation time
              <input aria-label="Evaluation time" type="datetime-local" value={evalAt} onChange={(e) => setEvalAt(e.target.value)} className="h-9 rounded-lg border bg-card px-2 text-sm text-foreground" />
            </label>
            {evalAt && (
              <Button type="button" variant="ghost" size="sm" onClick={() => setEvalAt("")}>
                Now
              </Button>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <div role="group" aria-label="Result view" className="flex rounded-lg border p-0.5">
              {(["table", "graph"] as const).map((m) => (
                <button
                  key={m}
                  type="button"
                  aria-pressed={mode === m}
                  onClick={() => setMode(m)}
                  className={cn("rounded-md px-3 py-1 capitalize", mode === m ? "bg-primary text-primary-foreground" : "text-muted-foreground")}
                >
                  {m}
                </button>
              ))}
            </div>
            {mode === "graph" && (
              <>
                <select aria-label="Range" className="h-8 rounded-lg border bg-card px-2" value={range} onChange={(e) => setRange(e.target.value)}>
                  {Object.keys(ranges).map((r) => (
                    <option key={r} value={r}>
                      Last {r}
                    </option>
                  ))}
                </select>
                <input
                  aria-label="Step seconds"
                  inputMode="numeric"
                  placeholder={`step auto (${autoStep(ranges[range])}s)`}
                  value={stepText}
                  onChange={(e) => setStepText(e.target.value.replace(/[^0-9]/g, ""))}
                  className="h-8 w-40 rounded-lg border bg-card px-2"
                />
              </>
            )}
          </div>
        </form>
      </Card>

      <Card className="min-w-0 p-4" aria-label="Query results">
        {run.isError ? (
          <p className="flex items-start gap-2 text-sm text-critical">
            <CircleAlert className="mt-0.5 size-4 shrink-0" aria-hidden />
            <span className="font-mono break-all">{run.error.message}</span>
          </p>
        ) : !res ? (
          <Empty>{run.isPending ? "Running query…" : "Run a query to see results"}</Empty>
        ) : (
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
              <span>
                {res.data.resultType} · {Array.isArray(res.data.result) && res.data.resultType !== "scalar" && res.data.resultType !== "string" ? `${res.data.result.length} series` : "1 value"} ·{" "}
                {res.durationMs.toFixed(0)} ms
              </span>
              {res.warnings.map((w) => (
                <span key={w} className="text-warning">
                  {w}
                </span>
              ))}
              <span className="ml-auto flex gap-1">
                <Button variant="ghost" size="sm" onClick={() => void copy("query")}>
                  <Copy /> {copied === "query" ? "Copied" : "Copy query"}
                </Button>
                <Button variant="ghost" size="sm" onClick={() => void copy("results")}>
                  <Copy /> {copied === "results" ? "Copied" : "Copy results"}
                </Button>
              </span>
            </div>
            <QueryResult data={res.data} />
          </div>
        )}
      </Card>
    </div>
  );
}

export function QueryResult({ data }: { data: PromQueryData }) {
  if (data.resultType === "scalar" || data.resultType === "string") {
    return <p className="font-mono text-lg">{data.result[1]}</p>;
  }
  if (data.result.length === 0) return <Empty>Empty query result</Empty>;
  if (data.resultType === "vector") {
    return (
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-xs text-muted-foreground">
              <th className="py-1.5 text-left font-normal">Series</th>
              <th className="py-1.5 text-right font-normal">Value</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {data.result.slice(0, 500).map((r) => (
              <tr key={seriesName(r.metric)}>
                <td className="py-1.5 pr-3">
                  {r.metric.__name__ && <span className="mr-2 font-mono font-medium">{r.metric.__name__}</span>}
                  <LabelList labels={Object.fromEntries(Object.entries(r.metric).filter(([k]) => k !== "__name__"))} />
                </td>
                <td className="py-1.5 text-right font-mono tabular-nums">{r.value[1]}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {data.result.length > 500 && <p className="mt-2 text-xs text-muted-foreground">Showing the first 500 of {data.result.length} series.</p>}
      </div>
    );
  }
  const series = data.result.slice(0, MAX_SERIES);
  const rows = new Map<number, Record<string, number>>();
  series.forEach((s, i) => {
    for (const [t, v] of s.values) {
      const row = rows.get(t) ?? { t: t * 1000 };
      row[`s${i}`] = Number(v);
      rows.set(t, row);
    }
  });
  const points = [...rows.values()].sort((a, b) => a.t - b.t);
  return (
    <figure aria-label={`Graph of ${series.length} series`}>
      <div className="h-72">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={points} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="var(--color-border)" strokeDasharray="3 3" vertical={false} />
            <XAxis dataKey="t" type="number" domain={["dataMin", "dataMax"]} tickFormatter={(t: number) => tick.format(new Date(t))} tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} />
            <YAxis tick={{ fontSize: 11, fill: "var(--color-muted-foreground)" }} width={64} />
            <Tooltip labelFormatter={(t) => `${tick.format(new Date(Number(t)))} IST`} contentStyle={{ background: "var(--color-card)", border: "1px solid var(--color-border)" }} />
            {series.map((s, i) => (
              <Line key={i} dataKey={`s${i}`} name={seriesName(s.metric)} stroke={palette[i % palette.length]} dot={false} isAnimationActive={false} connectNulls={false} />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
      <figcaption className="mt-2 space-y-0.5 text-xs">
        {series.map((s, i) => (
          <div key={i} className="flex items-center gap-2 font-mono">
            <span className="size-2 shrink-0 rounded-full" style={{ background: palette[i % palette.length] }} aria-hidden />
            <span className="truncate">{seriesName(s.metric)}</span>
          </div>
        ))}
        {data.result.length > MAX_SERIES && <p className="text-muted-foreground">Showing {MAX_SERIES} of {data.result.length} series.</p>}
      </figcaption>
    </figure>
  );
}
