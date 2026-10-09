"use client";

import { StatusDot } from "@/components/ui/primitives";
import { useSection } from "@/lib/api";
import { feedSourceLabel, phaseLabel } from "@/lib/format";

export function Footer() {
  const info = useSection("info").data?.data;
  const market = useSection("status").data?.data.market;
  return (
    <footer className="flex flex-wrap items-center justify-between gap-2 border-t px-4 py-3 text-xs text-muted-foreground md:px-6">
      <p className="flex flex-wrap items-center gap-2">
        <span className="font-medium text-foreground">stock_home</span>
        <span aria-hidden>|</span>
        <span>Stock Observability Platform</span>
        <span aria-hidden>|</span>
        <span>{info ? `v${info.version}` : "—"}</span>
      </p>
      <p className="flex flex-wrap items-center gap-2">
        <span>Data via {feedSourceLabel(market?.feedSource)}</span>
        <span aria-hidden>|</span>
        <span>{info?.exchange || "—"}</span>
        <span aria-hidden>|</span>
        <span className={market?.open ? "flex items-center gap-1.5 text-ok" : "flex items-center gap-1.5"}>
          <StatusDot ok={market?.open ? true : null} className="size-2" />
          {market ? (market.open ? "Live" : (phaseLabel[market.phase] ?? market.phase)) : "—"}
        </span>
      </p>
    </footer>
  );
}
