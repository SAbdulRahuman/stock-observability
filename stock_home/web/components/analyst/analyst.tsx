"use client";

import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { z } from "zod";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const analysisSchema = z.object({
  symbol: z.string(),
  marketRegime: z.string().optional(),
  regimes: z.record(z.string(), z.string()),
  summary: z.array(z.string()),
  evidence: z.array(
    z.object({
      kpi: z.string(),
      label: z.string(),
      value: z.number(),
      unit: z.string(),
      quality: z.string().optional(),
      method: z.string().optional(),
    }),
  ),
  conditions: z.array(z.object({ alert: z.string(), direction: z.string().optional(), hitRate: z.number().optional() })),
  disclaimer: z.string(),
});

export type Analysis = z.infer<typeof analysisSchema>;

async function fetchAnalysis(symbol: string, signal?: AbortSignal): Promise<Analysis> {
  const res = await fetch(`/api/v1/analyst?symbol=${encodeURIComponent(symbol)}`, { signal, headers: { Accept: "application/json" } });
  const body = (await res.json().catch(() => ({}))) as { data?: unknown; error?: string };
  if (!res.ok) throw new Error(body.error ?? `HTTP ${res.status}`);
  return analysisSchema.parse(body.data);
}

const qualityVariant: Record<string, BadgeVariant> = { exact: "ok", approximation: "warning", heuristic: "critical", proxy: "warning" };

export function Analyst() {
  const initial = (useSearchParams()?.get("symbol") ?? "").trim().toUpperCase();
  const [input, setInput] = useState(initial);
  const [symbol, setSymbol] = useState(initial);
  useEffect(() => {
    if (initial) {
      setInput(initial);
      setSymbol(initial);
    }
  }, [initial]);
  const query = useQuery({
    queryKey: ["analyst", symbol],
    queryFn: ({ signal }) => fetchAnalysis(symbol, signal),
    enabled: symbol !== "",
    refetchInterval: 30_000,
  });

  return (
    <main className="mx-auto flex max-w-[1100px] flex-col gap-4 p-4 md:p-6">
      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          setSymbol(input.trim().toUpperCase());
        }}
      >
        <input
          aria-label="Symbol"
          className="w-64 rounded-md border bg-background px-3 py-1.5 text-sm"
          placeholder="NSE symbol, e.g. RELIANCE"
          value={input}
          onChange={(e) => setInput(e.target.value)}
        />
        <button type="submit" className="rounded-md bg-secondary px-3 py-1.5 text-sm font-medium">
          Explain
        </button>
      </form>

      {query.isError && <p className="text-sm text-critical">{(query.error as Error).message}</p>}
      {query.data && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Why is {query.data.symbol} moving?</CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="list-disc space-y-1 pl-5 text-sm">
                {query.data.summary.map((s) => (
                  <li key={s}>{s}</li>
                ))}
              </ul>
              <p className="mt-3 text-xs text-muted-foreground">{query.data.disclaimer}</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Evidence</CardTitle>
            </CardHeader>
            <CardContent>
              <table className="w-full text-sm">
                <tbody>
                  {query.data.evidence.map((e) => (
                    <tr key={e.kpi} className="border-b last:border-0">
                      <td className="py-1.5" title={e.kpi}>
                        {e.label}
                      </td>
                      <td className="py-1.5 text-right font-mono tabular-nums">
                        {e.value.toFixed(2)}
                        {e.unit && ` ${e.unit}`}
                      </td>
                      <td className="py-1.5 pl-3 text-right">
                        {e.quality && (
                          <Badge variant={qualityVariant[e.quality] ?? "outline"} title={e.method}>
                            {e.quality}
                          </Badge>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </CardContent>
          </Card>
        </>
      )}
    </main>
  );
}
