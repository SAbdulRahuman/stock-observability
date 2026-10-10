export type TickerQuote = { symbol: string; price: number | null; changePercent: number | null };

export type Ticker = {
  available: boolean;
  simulated: boolean;
  updatedAt: string;
  indices: TickerQuote[];
  gainers: TickerQuote[];
  losers: TickerQuote[];
  advancers: number | null;
  decliners: number | null;
};

export type SiteConfig = { demoUrl: string; version: string };

export type WaitlistInput = { email: string; name: string; interest: string; message: string; website: string };

export type WaitlistResult = { ok: true } | { ok: false; error: string; field?: string };

export const interests = [
  { value: "", label: "Select one (optional)" },
  { value: "investor", label: "Long-term investor" },
  { value: "trader", label: "Active trader" },
  { value: "developer", label: "Developer / quant" },
  { value: "other", label: "Something else" },
] as const;

async function getJSON<T>(url: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(url, { signal, headers: { Accept: "application/json" } });
  if (!res.ok) throw new Error(`${url}: HTTP ${res.status}`);
  return (await res.json()) as T;
}

export const fetchTicker = (signal?: AbortSignal) => getJSON<Ticker>("/api/site/ticker", signal);

export const fetchConfig = (signal?: AbortSignal) => getJSON<SiteConfig>("/api/site/config", signal);

export async function joinWaitlist(input: WaitlistInput): Promise<WaitlistResult> {
  let res: Response;
  try {
    res = await fetch("/api/site/waitlist", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify(input),
    });
  } catch {
    return { ok: false, error: "Network error. Please check your connection and try again." };
  }
  if (res.ok) return { ok: true };
  const body = (await res.json().catch(() => ({}))) as { error?: string; field?: string };
  return { ok: false, error: body.error ?? `Sign-up failed (HTTP ${res.status}).`, field: body.field };
}

const priceFmt = new Intl.NumberFormat("en-IN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export function formatPrice(v: number | null): string {
  return v == null ? "—" : priceFmt.format(v);
}

export function formatPercent(v: number | null): string {
  if (v == null) return "—";
  return `${v > 0 ? "+" : ""}${v.toFixed(2)}%`;
}
