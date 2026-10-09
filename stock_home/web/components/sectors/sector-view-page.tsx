"use client";

import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { ArrowRight, BarChart3, LayoutGrid, RefreshCw, TrendingDown, TrendingUp, Upload } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { Empty, SectionCard } from "@/components/dashboard/section-card";
import { IndexCard } from "@/components/home/summary-row";
import { DayOnly, HeatmapTiles, PerformanceBars } from "@/components/market/sectors";
import { Button } from "@/components/ui/button";
import { type MoversScan, useMoversScan, useSection } from "@/lib/api";
import { formatPercent, trendClass } from "@/lib/format";
import { cn } from "@/lib/utils";
import { SectorComparison } from "./comparison";

const CARDS = 5;
const PERF_ROWS = 8;
const LEADERS = 5;
const universeLabels: Record<string, string> = { core: "Core (index constituents)", all: "All NSE equities" };

type Sector = MoversScan["sectors"][number];

function csvCell(v: string | number | null | undefined): string {
  const s = v == null ? "" : String(v);
  // Leading =,+,-,@ would be evaluated as formulas by spreadsheet apps.
  const safe = typeof v === "string" && /^[=+\-@]/.test(s) ? `'${s}` : s;
  return /[",\n]/.test(safe) ? `"${safe.replace(/"/g, '""')}"` : safe;
}

export function SectorViewPage() {
  const [universe, setUniverse] = useState("core");
  const [sector, setSector] = useState("");
  const [sizing, setSizing] = useState<"constituents" | "equal">("constituents");
  const [allPerf, setAllPerf] = useState(false);
  const indices = useSection("sectors");
  const scan = useMoversScan({ universe, sector, price: "all" });
  const exchange = useSection("info").data?.data.exchange;
  const client = useQueryClient();
  const fetching = useIsFetching() > 0;
  const env = indices.data;
  const symbols = env?.data.indices.map((q) => q.symbol) ?? [];
  const names = [...new Set([...(scan.data?.data.sectorNames ?? []), ...(sector ? [sector] : [])])].sort();
  const select = "h-9 rounded-lg border bg-card px-2 text-sm disabled:opacity-80";

  const exportCsv = () => {
    const lines = [["sector", "avg_change_percent", "instruments", "advancers", "decliners", "top_gainer", "top_gainer_percent", "top_loser", "top_loser_percent"].join(",")];
    for (const s of scan.data?.data.sectors ?? []) {
      lines.push([s.name, s.changePercent, s.count, s.advancers, s.decliners, s.gainer?.symbol, s.gainer?.changePercent, s.loser?.symbol, s.loser?.changePercent].map(csvCell).join(","));
    }
    lines.push("", ["index", "ltp", "change", "change_percent"].join(","));
    for (const q of env?.data.indices ?? []) lines.push([q.symbol, q.price, q.change, q.changePercent].map(csvCell).join(","));
    const url = URL.createObjectURL(new Blob([`${lines.join("\n")}\n`], { type: "text/csv" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = "sectors.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Sector View</h1>
        <p className="text-sm text-muted-foreground">Compare {exchange || "NSE"} sector performance and explore market rotation</p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <select aria-label="Exchange" className={select} value={exchange ?? ""} disabled title="Only the configured exchange is available">
          <option value={exchange ?? ""}>{exchange || "—"}</option>
        </select>
        <select aria-label="Time range" className={select} value="1D" disabled title="Sector change versus previous close is the only period recorded">
          <option value="1D">1D</option>
        </select>
        <select aria-label="Universe" className={select} value={universe} onChange={(e) => setUniverse(e.target.value)}>
          {Object.entries(universeLabels).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </select>
        <select aria-label="Sector" className={select} value={sector} onChange={(e) => setSector(e.target.value)}>
          <option value="">All Sectors</option>
          {names.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        <Button variant="outline" onClick={() => void client.invalidateQueries()} aria-busy={fetching}>
          <RefreshCw className={cn(fetching && "animate-spin")} /> Refresh
        </Button>
        <Button variant="outline" onClick={exportCsv} disabled={!scan.data && !env}>
          <Upload /> Export
        </Button>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {(env ? env.data.indices.slice(0, CARDS) : Array.from({ length: CARDS }, () => undefined)).map((q, i) => (
          <IndexCard
            key={q?.symbol ?? i}
            name={q?.symbol ?? "Sector index"}
            quote={q}
            loading={!env && !indices.isError}
            error={!env && indices.isError}
            stale={!!env && (env.stale || indices.isError)}
          />
        ))}
        {env && env.data.indices.length === 0 && <p className="col-span-full text-sm text-muted-foreground">No sector index data recorded</p>}
      </div>

      <SectionCard
        id="sector-heatmap-full"
        title={`${exchange || "NSE"} Sector Heatmap`}
        icon={LayoutGrid}
        query={scan}
        actions={
          <>
            <DayOnly />
            <select aria-label="Tile size" className="h-7 rounded-md border bg-card px-1 text-xs" value={sizing} onChange={(e) => setSizing(e.target.value as typeof sizing)}>
              <option value="constituents">Size: constituents</option>
              <option value="equal">Size: equal</option>
            </select>
          </>
        }
      >
        {(d) =>
          d.sectors.length === 0 ? (
            <Empty>No sector data for these filters</Empty>
          ) : (
            <HeatmapTiles
              large
              tiles={d.sectors.map((s) => ({
                name: s.name,
                changePercent: s.changePercent,
                size: sizing === "equal" ? 1 : s.count,
                title: `${s.name}: ${formatPercent(s.changePercent)} · ${s.count} instruments · ${s.advancers} up / ${s.decliners} down`,
              }))}
            />
          )
        }
      </SectionCard>

      <div className="grid gap-4 lg:grid-cols-2">
        <SectionCard
          id="sector-performance-full"
          title="Sector Performance"
          icon={BarChart3}
          query={scan}
          actions={
            (scan.data?.data.sectors.length ?? 0) > PERF_ROWS && (
              <button type="button" className="text-xs font-medium text-primary hover:underline" aria-expanded={allPerf} onClick={() => setAllPerf((v) => !v)}>
                {allPerf ? "Show top" : `View All (${scan.data?.data.sectors.length})`}
              </button>
            )
          }
        >
          {(d) =>
            d.sectors.length === 0 ? <Empty>No sector data for these filters</Empty> : <PerformanceBars sectors={allPerf ? d.sectors : d.sectors.slice(0, PERF_ROWS)} />
          }
        </SectionCard>
        <SectorComparison symbols={symbols} />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <SectionCard id="sector-gainers" title="Top Sector Gainers" icon={TrendingUp} query={scan}>
          {(d) => <LeadersTable rows={d.sectors.filter((s) => s.changePercent > 0).slice(0, LEADERS)} lead="gainer" universe={universe} empty="No sector is up" />}
        </SectionCard>
        <SectionCard id="sector-losers" title="Top Sector Losers" icon={TrendingDown} query={scan}>
          {(d) => (
            <LeadersTable
              rows={d.sectors.filter((s) => s.changePercent < 0).reverse().slice(0, LEADERS)}
              lead="loser"
              universe={universe}
              empty="No sector is down"
            />
          )}
        </SectionCard>
      </div>
    </main>
  );
}

function LeadersTable({ rows, lead, universe, empty }: { rows: Sector[]; lead: "gainer" | "loser"; universe: string; empty: string }) {
  if (rows.length === 0) return <Empty>{empty}</Empty>;
  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="text-xs text-muted-foreground">
          <th className="py-1.5 text-left font-normal">Sector</th>
          <th className="py-1.5 text-left font-normal">Leading stock</th>
          <th className="py-1.5 text-right font-normal">Stock %</th>
          <th className="py-1.5 text-right font-normal">Sector %</th>
          <th className="py-1.5 pl-3 text-right font-normal">
            <span className="sr-only">Action</span>
          </th>
        </tr>
      </thead>
      <tbody className="divide-y">
        {rows.map((s) => {
          const q = s[lead];
          return (
            <tr key={s.name}>
              <td className="max-w-0 truncate py-2 font-medium" style={{ width: "32%" }} title={s.name}>
                {s.name}
              </td>
              <td className="py-2">{q?.symbol ?? "—"}</td>
              <td className={cn("py-2 text-right tabular-nums", trendClass(q?.changePercent))}>{formatPercent(q?.changePercent)}</td>
              <td className={cn("py-2 text-right font-medium tabular-nums", trendClass(s.changePercent))}>{formatPercent(s.changePercent)}</td>
              <td className="py-2 pl-3 text-right whitespace-nowrap">
                <Link
                  href={`/movers?${new URLSearchParams({ universe, sector: s.name })}`}
                  className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
                  aria-label={`View sector ${s.name}`}
                >
                  View Sector <ArrowRight className="size-3.5" aria-hidden />
                </Link>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
