export const IST = "Asia/Kolkata";

const inr = new Intl.NumberFormat("en-IN", { style: "currency", currency: "INR", maximumFractionDigits: 0 });
const inrCompact = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  notation: "compact",
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
});
const price = new Intl.NumberFormat("en-IN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const count = new Intl.NumberFormat("en-IN", { maximumFractionDigits: 0 });
const decimal = new Intl.NumberFormat("en-IN", { maximumFractionDigits: 1 });

const DASH = "—";

type Num = number | null | undefined;

const isNum = (v: Num): v is number => typeof v === "number" && Number.isFinite(v);

/** Compact (lakh/crore) notation is only used from one lakh upwards. */
export function formatINR(v: Num, opts: { compact?: boolean; signed?: boolean } = {}): string {
  if (!isNum(v)) return DASH;
  const s = (opts.compact && Math.abs(v) >= 100_000 ? inrCompact : inr).format(Math.abs(v));
  if (v < 0) return `-${s}`;
  return opts.signed && v > 0 ? `+${s}` : s;
}

export function formatPrice(v: Num): string {
  return isNum(v) ? price.format(v) : DASH;
}

/** Signed price difference such as "+126.40" or "-12.35". */
export function formatChange(v: Num): string {
  if (!isNum(v)) return DASH;
  const s = price.format(Math.abs(v));
  if (v < 0) return `-${s}`;
  return v > 0 ? `+${s}` : s;
}

export function formatCount(v: Num): string {
  return isNum(v) ? count.format(v) : DASH;
}

export function formatDecimal(v: Num): string {
  return isNum(v) ? decimal.format(v) : DASH;
}

export function formatPercent(v: Num, digits = 2, signed = true): string {
  if (!isNum(v)) return DASH;
  const s = `${Math.abs(v).toFixed(digits)}%`;
  if (v < 0) return `-${s}`;
  return signed && v > 0 ? `+${s}` : s;
}

/** Human duration such as "45s", "12m 05s", "2h 14m", "3d 4h". */
export function formatDuration(seconds: Num): string {
  if (!isNum(seconds)) return DASH;
  const s = Math.max(0, Math.round(seconds));
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${pad(m)}m`;
  if (m > 0) return `${m}m ${pad(sec)}s`;
  return `${sec}s`;
}

const timeFmt = new Intl.DateTimeFormat("en-IN", { timeZone: IST, hour: "2-digit", minute: "2-digit", hour12: false });
const timeSecFmt = new Intl.DateTimeFormat("en-IN", {
  timeZone: IST,
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  hour12: false,
});
const dayFmt = new Intl.DateTimeFormat("en-IN", { timeZone: IST, weekday: "short", day: "numeric", month: "short" });

export function formatTimeIST(d: Date | string | null | undefined, withSeconds = false): string {
  if (!d) return DASH;
  const date = typeof d === "string" ? new Date(d) : d;
  if (Number.isNaN(date.getTime())) return DASH;
  return (withSeconds ? timeSecFmt : timeFmt).format(date);
}

/** Formats an exchange calendar date ("2026-10-20") as "Tue, 20 Oct". */
export function formatDateIST(isoDate: string | null | undefined): string {
  if (!isoDate) return DASH;
  const date = new Date(`${isoDate}T12:00:00+05:30`);
  return Number.isNaN(date.getTime()) ? DASH : dayFmt.format(date);
}

const headerDateFmt = new Intl.DateTimeFormat("en-IN", { timeZone: IST, weekday: "short", day: "numeric", month: "short", year: "numeric" });
const clockFmt = new Intl.DateTimeFormat("en-IN", { timeZone: IST, hour: "2-digit", minute: "2-digit", hour12: true });

/** "Thu, 8 Oct 2026" and "09:42 am" parts of an instant, in IST. */
export function formatDateTimeIST(d: Date): { date: string; time: string } {
  return { date: headerDateFmt.format(d), time: clockFmt.format(d).toUpperCase() };
}

/** Formats a wall-clock "HH:MM" (24h) as "09:15 AM". */
export function formatClock(hhmm: string | null | undefined): string {
  const m = /^(\d{2}):(\d{2})$/.exec(hhmm ?? "");
  if (!m) return DASH;
  const h = Number(m[1]);
  return `${String(h % 12 || 12).padStart(2, "0")}:${m[2]} ${h < 12 ? "AM" : "PM"}`;
}

export function relativeTime(d: Date | string, now: Date = new Date()): string {
  const date = typeof d === "string" ? new Date(d) : d;
  const diff = Math.round((now.getTime() - date.getTime()) / 1000);
  if (diff < 5) return "just now";
  return `${formatDuration(diff)} ago`;
}

export function trendClass(v: Num): string {
  if (!isNum(v) || v === 0) return "text-muted-foreground";
  return v > 0 ? "text-up" : "text-down";
}

export function feedSourceLabel(source: string | null | undefined): string {
  if (source === "kite") return "Zerodha Kite API";
  if (source === "sim") return "simulated feed";
  return source || DASH;
}

export const phaseLabel: Record<string, string> = {
  open: "Market open",
  pre_open: "Pre-open",
  post_close: "Post-close",
  closed: "Market closed",
  unknown: "Market status unknown",
};
