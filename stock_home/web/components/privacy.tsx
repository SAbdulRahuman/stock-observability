"use client";

import { createContext, useCallback, useContext, useEffect, useState } from "react";
import { formatINR } from "@/lib/format";
import { cn } from "@/lib/utils";

const STORAGE_KEY = "stock-home:hide-amounts";

type Privacy = { hidden: boolean; toggle: () => void };

const PrivacyContext = createContext<Privacy>({ hidden: true, toggle: () => {} });

/** Portfolio amounts are hidden by default because the portal may be reachable without login. */
export function PrivacyProvider({ children }: { children: React.ReactNode }) {
  const [hidden, setHidden] = useState(true);

  useEffect(() => {
    if (window.localStorage.getItem(STORAGE_KEY) === "false") setHidden(false);
  }, []);

  const toggle = useCallback(() => {
    setHidden((h) => {
      window.localStorage.setItem(STORAGE_KEY, String(!h));
      return !h;
    });
  }, []);

  return <PrivacyContext.Provider value={{ hidden, toggle }}>{children}</PrivacyContext.Provider>;
}

export function usePrivacy() {
  return useContext(PrivacyContext);
}

export function Money({
  value,
  signed,
  compact,
  className,
}: {
  value: number | null | undefined;
  signed?: boolean;
  compact?: boolean;
  className?: string;
}) {
  const { hidden } = usePrivacy();
  return (
    <span className={cn("tabular-nums", className)} aria-label={hidden ? "hidden amount" : undefined}>
      {hidden ? "₹ ••••" : formatINR(value, { signed, compact })}
    </span>
  );
}
