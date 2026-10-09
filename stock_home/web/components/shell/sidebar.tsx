"use client";

import { ChartCandlestick, ExternalLink } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useActiveAlertCount, useLinkHref } from "./hooks";
import { MarketStatusCard } from "./market-status";
import { isActive, type NavItem, navItems } from "./nav";

const itemClass = "flex h-10 items-center gap-3 rounded-lg px-3 text-sm font-medium transition-colors";
const labelClass = "min-w-0 flex-1 truncate group-data-[collapsed=true]/sidebar:hidden max-md:hidden";

export function Sidebar({ collapsed }: { collapsed: boolean }) {
  const pathname = usePathname() ?? "/";
  return (
    <aside
      data-collapsed={collapsed}
      className="group/sidebar sticky top-0 flex h-dvh w-64 shrink-0 flex-col border-r bg-card/40 data-[collapsed=true]:w-16 max-md:w-16"
    >
      <Link href="/" className="flex h-16 items-center gap-3 px-4" aria-label="stock_home home">
        <ChartCandlestick className="size-7 shrink-0 text-primary" aria-hidden />
        <span className={cn(labelClass, "leading-tight")}>
          <span className="block text-lg font-semibold tracking-tight">stock_home</span>
          <span className="block text-xs text-muted-foreground">Stock Observability Platform</span>
        </span>
      </Link>
      <nav aria-label="Main" className="flex-1 overflow-y-auto px-2 py-3">
        <ul className="space-y-1">
          {navItems.map((item) => (
            <li key={item.label}>
              <SidebarLink item={item} active={isActive(item, pathname)} />
            </li>
          ))}
        </ul>
      </nav>
      <div className="p-3 group-data-[collapsed=true]/sidebar:hidden max-md:hidden">
        <MarketStatusCard />
      </div>
    </aside>
  );
}

function SidebarLink({ item, active }: { item: NavItem; active: boolean }) {
  const Icon = item.icon;
  const externalHref = useLinkHref(item.linkId);
  const alertCount = useActiveAlertCount();
  const badge =
    item.badge === "alerts" && alertCount != null && alertCount > 0 ? (
      <span
        className="ml-auto rounded-md bg-critical px-1.5 text-xs font-semibold text-white tabular-nums group-data-[collapsed=true]/sidebar:hidden max-md:hidden"
        aria-label={`${alertCount} active alerts`}
      >
        {alertCount}
      </span>
    ) : null;
  const body = (
    <>
      <Icon className="size-5 shrink-0" aria-hidden />
      <span className={labelClass}>{item.label}</span>
      {badge}
    </>
  );

  if (item.linkId) {
    if (!externalHref) {
      return (
        <span aria-disabled="true" title={`${item.label} link not configured`} className={cn(itemClass, "cursor-not-allowed text-muted-foreground/60")}>
          {body}
        </span>
      );
    }
    return (
      <a href={externalHref} target="_blank" rel="noopener noreferrer" title={`${item.label} (opens in a new tab)`} className={cn(itemClass, "text-muted-foreground hover:bg-accent hover:text-foreground")}>
        {body}
        <ExternalLink className="ml-auto size-4 shrink-0 group-data-[collapsed=true]/sidebar:hidden max-md:hidden" aria-hidden />
      </a>
    );
  }
  if (!item.ready || !item.href) {
    return (
      <span aria-disabled="true" title={`${item.label} — coming soon`} className={cn(itemClass, "cursor-not-allowed text-muted-foreground/60")}>
        {body}
      </span>
    );
  }
  return (
    <Link
      href={item.href}
      aria-current={active ? "page" : undefined}
      title={item.label}
      className={cn(itemClass, active ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-accent hover:text-foreground")}
    >
      {body}
    </Link>
  );
}
