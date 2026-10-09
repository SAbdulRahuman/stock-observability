"use client";

import { Area, AreaChart, ResponsiveContainer, YAxis } from "recharts";
import { useSettings } from "@/lib/settings";

type Point = { t: number; v: number };

/** Minimal area chart without axes; colour follows the trend between first and last point. Hidden by the Settings preference. */
export function Sparkline({ points, className, height = 48 }: { points: Point[] | null | undefined; className?: string; height?: number }) {
  const [settings] = useSettings();
  if (!settings.showSparklines || !points || points.length < 2) return <div className={className} style={{ height }} />;
  const up = points[points.length - 1].v >= points[0].v;
  const color = up ? "var(--color-up)" : "var(--color-down)";
  const id = `spark-${up ? "up" : "down"}`;
  return (
    <div className={className} style={{ height }} aria-hidden>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={points} margin={{ top: 2, right: 0, bottom: 2, left: 0 }}>
          <defs>
            <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity={0.35} />
              <stop offset="100%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <YAxis hide domain={["dataMin", "dataMax"]} />
          <Area
            type="monotone"
            dataKey="v"
            stroke={color}
            strokeWidth={1.5}
            fill={`url(#${id})`}
            isAnimationActive={false}
            dot={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
