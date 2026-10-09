import { describe, expect, it } from "vitest";
import {
  formatClock,
  formatDateIST,
  formatDateTimeIST,
  formatDuration,
  formatINR,
  formatPercent,
  formatTimeIST,
  feedSourceLabel,
  relativeTime,
  trendClass,
} from "./format";

describe("formatINR", () => {
  it("uses Indian grouping and rupee symbol", () => {
    expect(formatINR(1240000)).toBe("₹12,40,000");
  });
  it("signs values on request", () => {
    expect(formatINR(8210, { signed: true })).toBe("+₹8,210");
    expect(formatINR(-1150, { signed: true })).toBe("-₹1,150");
  });
  it("uses lakh/crore compact notation", () => {
    expect(formatINR(1240000, { compact: true })).toMatch(/₹12\.4\s?L/);
  });
  it("renders a dash for missing values", () => {
    expect(formatINR(null)).toBe("—");
    expect(formatINR(Number.NaN)).toBe("—");
  });
});

describe("formatPercent", () => {
  it("signs positive and negative values", () => {
    expect(formatPercent(0.8)).toBe("+0.80%");
    expect(formatPercent(-2)).toBe("-2.00%");
    expect(formatPercent(0)).toBe("0.00%");
  });
});

describe("formatDuration", () => {
  it.each([
    [45, "45s"],
    [725, "12m 05s"],
    [8040, "2h 14m"],
    [273600, "3d 4h"],
    [-5, "0s"],
  ])("%d seconds -> %s", (s, want) => {
    expect(formatDuration(s)).toBe(want);
  });
});

describe("IST formatting", () => {
  it("formats times in Asia/Kolkata", () => {
    expect(formatTimeIST("2026-10-06T03:45:00Z")).toBe("09:15");
  });
  it("formats exchange dates", () => {
    expect(formatDateIST("2026-10-20")).toBe("Tue, 20 Oct");
  });
  it("formats the header clock and session hours", () => {
    const { date, time } = formatDateTimeIST(new Date("2026-10-08T04:12:00Z"));
    expect(date).toMatch(/Thu.*8.*Oct.*2026/);
    expect(time).toMatch(/^09:42\sAM$/);
    expect(formatClock("09:15")).toBe("09:15 AM");
    expect(formatClock("15:30")).toBe("03:30 PM");
    expect(formatClock("bad")).toBe("—");
  });
  it("names the feed source", () => {
    expect(feedSourceLabel("kite")).toBe("Zerodha Kite API");
    expect(feedSourceLabel("")).toBe("—");
  });
  it("describes elapsed time", () => {
    const now = new Date("2026-10-06T10:00:00Z");
    expect(relativeTime("2026-10-06T09:59:58Z", now)).toBe("just now");
    expect(relativeTime("2026-10-06T09:58:00Z", now)).toBe("2m 00s ago");
  });
});

describe("trendClass", () => {
  it("maps sign to colour", () => {
    expect(trendClass(1)).toBe("text-up");
    expect(trendClass(-1)).toBe("text-down");
    expect(trendClass(null)).toBe("text-muted-foreground");
  });
});
