"use client";

import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { RefreshCw, Upload } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { type MoversFilters, type Quote, useMoversScan, useSection } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Distribution, GainersLosers, MostActiveByVolume, Range52, SectorRanking, SummaryCards } from "./panels";

const universeLabels: Record<string, string> = { core: "Core (index constituents)", all: "All NSE equities" };
const priceLabels: Record<string, string> = {
  all: "All Prices",
  lt100: "Under ₹100",
  "100-500": "₹100 – ₹500",
  "500-2000": "₹500 – ₹2,000",
  gt2000: "Above ₹2,000",
};

function csvCell(v: string | number | null | undefined): string {
  const s = v == null ? "" : String(v);
  // Leading =,+,-,@ would be evaluated as formulas by spreadsheet apps; numbers are written unquoted.
  const safe = typeof v === "string" && /^[=+\-@]/.test(s) ? `'${s}` : s;
  return /[",\n]/.test(safe) ? `"${safe.replace(/"/g, '""')}"` : safe;
}

export function TopMoversPage() {
  const params = useSearchParams();
  const [filters, setFilters] = useState<MoversFilters>(() => ({
    universe: params?.get("universe") === "all" ? "all" : "core",
    sector: params?.get("sector") ?? "",
    price: (params?.get("price") ?? "") in priceLabels ? (params?.get("price") as string) : "all",
  }));
  const query = useMoversScan(filters);
  const exchange = useSection("info").data?.data.exchange;
  const client = useQueryClient();
  const fetching = useIsFetching({ queryKey: ["movers-scan"] }) > 0;
  const d = query.data?.data;
  const sectors = [...new Set([...(d?.sectorNames ?? []), ...(filters.sector ? [filters.sector] : [])])].sort();
  const select = "h-9 rounded-lg border bg-card px-2 text-sm disabled:opacity-80";

  const exportCsv = () => {
    if (!d) return;
    const lines = [["list", "rank", "symbol", "sector", "ltp", "change", "change_percent", "volume"].join(",")];
    const add = (list: string, qs: Quote[]) =>
      qs.forEach((q, i) => lines.push([list, i + 1, q.symbol, q.sector, q.price, q.change, q.changePercent, q.volume].map(csvCell).join(",")));
    add("gainers", d.gainers);
    add("losers", d.losers);
    add("most_active", d.mostActive);
    add("52w_high", d.high52);
    add("52w_low", d.low52);
    const url = URL.createObjectURL(new Blob([`${lines.join("\n")}\n`], { type: "text/csv" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = "top-movers.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <main className="mx-auto flex max-w-[1600px] flex-col gap-4 p-4 md:p-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Top Movers</h1>
        <p className="text-sm text-muted-foreground">Find the biggest price movers across the {exchange || "NSE"} market</p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <select aria-label="Exchange" className={select} value={exchange ?? ""} disabled title="Only the configured exchange is available">
          <option value={exchange ?? ""}>{exchange || "—"}</option>
        </select>
        <select aria-label="Universe" className={select} value={filters.universe} onChange={(e) => setFilters((f) => ({ ...f, universe: e.target.value }))}>
          {Object.entries(universeLabels).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </select>
        <select aria-label="Time range" className={select} value="1D" disabled title="Day change versus previous close is the only period recorded">
          <option value="1D">1D</option>
        </select>
        <select aria-label="Sector" className={select} value={filters.sector} onChange={(e) => setFilters((f) => ({ ...f, sector: e.target.value }))}>
          <option value="">All Sectors</option>
          {sectors.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        <select aria-label="Price" className={select} value={filters.price} onChange={(e) => setFilters((f) => ({ ...f, price: e.target.value }))}>
          {Object.entries(priceLabels).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </select>
        <Button variant="outline" onClick={() => void client.invalidateQueries({ queryKey: ["movers-scan"] })} aria-busy={fetching}>
          <RefreshCw className={cn(fetching && "animate-spin")} /> Refresh
        </Button>
        <Button variant="outline" onClick={exportCsv} disabled={!d}>
          <Upload /> Export
        </Button>
      </div>

      <SummaryCards query={query} />
      <div className="grid gap-4 lg:grid-cols-2">
        <GainersLosers query={query} kind="gainers" />
        <GainersLosers query={query} kind="losers" />
      </div>
      <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
        <MostActiveByVolume query={query} />
        <Range52 query={query} side="high" />
        <Range52 query={query} side="low" />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <Distribution query={query} />
        <SectorRanking query={query} />
      </div>
    </main>
  );
}
