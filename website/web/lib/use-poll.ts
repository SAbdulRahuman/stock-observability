"use client";

import { useEffect, useState } from "react";

/** Polls fn every intervalMs while the tab is visible; data stays at the last good value on errors. */
export function usePoll<T>(fn: (signal: AbortSignal) => Promise<T>, intervalMs: number) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState(false);
  useEffect(() => {
    let ctrl = new AbortController();
    const tick = () => {
      if (document.visibilityState === "hidden") return;
      ctrl.abort();
      ctrl = new AbortController();
      fn(ctrl.signal)
        .then((d) => {
          setData(d);
          setError(false);
        })
        .catch((e: unknown) => {
          if (!(e instanceof DOMException && e.name === "AbortError")) setError(true);
        });
    };
    tick();
    const id = setInterval(tick, intervalMs);
    return () => {
      clearInterval(id);
      ctrl.abort();
    };
  }, [fn, intervalMs]);
  return { data, error };
}
