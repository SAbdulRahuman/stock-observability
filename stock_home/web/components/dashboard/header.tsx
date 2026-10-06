"use client";

import { CalendarDays, Clock, Eye, EyeOff, LineChart, Moon, Radio, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import { usePrivacy } from "@/components/privacy";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/primitives";
import { useSection } from "@/lib/api";
import { formatDateIST, formatDuration, formatTimeIST, phaseLabel } from "@/lib/format";
import { useNow } from "./section-card";

/** Section 6: market clock, plus page title and display toggles. */
export function Header() {
  const status = useSection("status");
  const now = useNow(1_000);
  const market = status.data?.data.market;
  const fetchedAt = status.dataUpdatedAt;
  const elapsed = (now.getTime() - fetchedAt) / 1000;

  let countdown: string | null = null;
  if (market?.open && market.secondsToClose != null) {
    countdown = `closes in ${formatDuration(market.secondsToClose - elapsed)}`;
  } else if (market && !market.open && market.secondsToOpen != null && market.secondsToOpen > 0) {
    countdown = `opens in ${formatDuration(market.secondsToOpen - elapsed)}`;
  }
  const phaseVariant = market?.phase === "open" ? "ok" : market?.phase === "pre_open" || market?.phase === "post_close" ? "warning" : "default";

  return (
    <header className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
      <div className="flex items-center gap-3">
        <div className="flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
          <LineChart className="size-5" aria-hidden />
        </div>
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Stock Observability</h1>
          <p className="text-sm text-muted-foreground">Live feed, alerts, market and portfolio at a glance</p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2 text-sm">
        {market ? (
          <>
            <Badge variant={phaseVariant} className="text-sm">
              <Radio aria-hidden />
              {phaseLabel[market.phase] ?? market.phase}
            </Badge>
            {countdown && <span className="text-muted-foreground tabular-nums">{countdown}</span>}
            <span className="flex items-center gap-1 tabular-nums" title={market.timezone}>
              <Clock className="size-4 text-muted-foreground" aria-hidden />
              {formatTimeIST(now, true)} IST
            </span>
            {market.nextHoliday && (
              <span className="hidden items-center gap-1 text-muted-foreground lg:flex" title="Next exchange holiday">
                <CalendarDays className="size-4" aria-hidden />
                {formatDateIST(market.nextHoliday.date)} · {market.nextHoliday.name}
              </span>
            )}
            {market.feedSource && <Badge variant="outline">{market.feedSource === "sim" ? "simulated feed" : `feed: ${market.feedSource}`}</Badge>}
          </>
        ) : (
          <Skeleton className="h-6 w-56" />
        )}
        <PrivacyToggle />
        <ThemeToggle />
      </div>
    </header>
  );
}

function PrivacyToggle() {
  const { hidden, toggle } = usePrivacy();
  const label = hidden ? "Show amounts" : "Hide amounts";
  return (
    <Button variant="outline" size="icon" onClick={toggle} aria-label={label} title={label}>
      {hidden ? <EyeOff /> : <Eye />}
    </Button>
  );
}

function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const dark = mounted && resolvedTheme === "dark";
  const label = dark ? "Switch to light theme" : "Switch to dark theme";
  return (
    <Button variant="outline" size="icon" onClick={() => setTheme(dark ? "light" : "dark")} aria-label={label} title={label}>
      {dark ? <Sun /> : <Moon />}
    </Button>
  );
}
