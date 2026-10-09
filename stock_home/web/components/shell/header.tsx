"use client";

import { Bell, Menu, Search, Settings, UserRound } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { StatusDot } from "@/components/ui/primitives";
import { useSection } from "@/lib/api";
import { formatDateTimeIST, phaseLabel } from "@/lib/format";
import { cn } from "@/lib/utils";
import { useNow } from "@/components/dashboard/section-card";
import { useActiveAlertCount, useLinkHref, useMounted } from "./hooks";
import { navItem } from "./nav";

const symbolPattern = /^[A-Z0-9][A-Z0-9&.\- ]{0,31}$/;
const iconButton = "relative inline-flex size-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground";

export function Header({ collapsed, onToggleSidebar }: { collapsed: boolean; onToggleSidebar: () => void }) {
  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-3 border-b bg-background/90 px-4 backdrop-blur md:px-6">
      <button
        type="button"
        className={iconButton}
        onClick={onToggleSidebar}
        aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        aria-expanded={!collapsed}
      >
        <Menu className="size-5" />
      </button>
      <StockSearch />
      <div className="ml-auto flex items-center gap-2 md:gap-4">
        <Clock />
        <LiveMarket />
        <SettingsButton />
        <AlertsBell />
        <span className="inline-flex size-9 items-center justify-center rounded-full bg-primary text-primary-foreground" role="img" aria-label="User">
          <UserRound className="size-5" aria-hidden />
        </span>
      </div>
    </header>
  );
}

/** Opens the analyst view until Stock Explorer exists. */
function StockSearch() {
  const router = useRouter();
  const [value, setValue] = useState("");
  const [invalid, setInvalid] = useState(false);
  return (
    <form
      role="search"
      className="relative w-full max-w-xl"
      onSubmit={(e) => {
        e.preventDefault();
        const symbol = value.trim().toUpperCase();
        if (!symbolPattern.test(symbol)) {
          setInvalid(true);
          return;
        }
        setInvalid(false);
        router.push(`/analyst?symbol=${encodeURIComponent(symbol)}`);
      }}
    >
      <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
      <input
        type="search"
        aria-label="Search stocks"
        aria-invalid={invalid || undefined}
        placeholder="Search stocks (e.g. RELIANCE, TCS, INFY)..."
        value={value}
        onChange={(e) => {
          setValue(e.target.value);
          setInvalid(false);
        }}
        className={cn(
          "h-10 w-full rounded-lg border bg-card pr-3 pl-9 text-sm placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
          invalid && "border-critical",
        )}
      />
    </form>
  );
}

function Clock() {
  const now = useNow(1_000);
  const mounted = useMounted();
  if (!mounted) return <span className="hidden w-44 lg:block" />;
  const { date, time } = formatDateTimeIST(now);
  return (
    <time dateTime={now.toISOString()} className="hidden text-sm whitespace-nowrap tabular-nums lg:block" title="India Standard Time">
      {date}
      <span className="ml-2">{time}</span>
    </time>
  );
}

function LiveMarket() {
  const status = useSection("status");
  const market = status.data?.data.market;
  let label = "Status unavailable";
  let ok: boolean | null = null;
  if (market) {
    ok = market.open ? true : null;
    label = market.open ? "Live Market" : (phaseLabel[market.phase] ?? market.phase);
  } else if (!status.isError) {
    label = "—";
  }
  return (
    <span className={cn("hidden items-center gap-2 text-sm whitespace-nowrap sm:flex", market?.open ? "text-ok" : "text-muted-foreground")} role="status">
      <StatusDot ok={ok} />
      {label}
    </span>
  );
}

function SettingsButton() {
  const settings = navItem("Settings");
  if (settings?.ready && settings.href) {
    return (
      <Link href={settings.href} className={iconButton} aria-label="Settings">
        <Settings className="size-5" />
      </Link>
    );
  }
  return (
    <span className={cn(iconButton, "cursor-not-allowed opacity-50 hover:bg-transparent")} aria-disabled="true" title="Settings — coming soon" role="img" aria-label="Settings (coming soon)">
      <Settings className="size-5" aria-hidden />
    </span>
  );
}

function AlertsBell() {
  const count = useActiveAlertCount();
  const alerts = navItem("Alerts");
  const alertmanager = useLinkHref("alertmanager");
  const href = alerts?.ready && alerts.href ? alerts.href : alertmanager;
  const label = count == null ? "Alerts" : `Alerts: ${count} active`;
  const body = (
    <>
      <Bell className="size-5" aria-hidden />
      {count != null && count > 0 && (
        <span className="absolute -top-0.5 -right-0.5 min-w-4 rounded-full bg-critical px-1 text-center text-[10px] leading-4 font-semibold text-white tabular-nums" aria-hidden>
          {count > 99 ? "99+" : count}
        </span>
      )}
    </>
  );
  if (!href) {
    return (
      <span className={iconButton} role="img" aria-label={label}>
        {body}
      </span>
    );
  }
  return (
    <a href={href} className={iconButton} aria-label={label} title={label}>
      {body}
    </a>
  );
}
