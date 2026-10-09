"use client";

import { useEffect, useState } from "react";
import { useSection } from "@/lib/api";

/** Active (not silenced or inhibited) alert count; null until known. */
export function useActiveAlertCount(): number | null {
  const alerts = useSection("alerts");
  const bySeverity = alerts.data?.data.bySeverity;
  if (!bySeverity) return null;
  return Object.values(bySeverity).reduce((n, v) => n + v, 0);
}

/** Same-origin href of a configured link from /api/v1/links. */
export function useLinkHref(id: string | undefined): string | undefined {
  const links = useSection("links");
  if (!id) return undefined;
  return links.data?.data.find((l) => l.id === id)?.href;
}

/** False during the static prerender and first client render, so clocks do not mismatch on hydration. */
export function useMounted(): boolean {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  return mounted;
}
