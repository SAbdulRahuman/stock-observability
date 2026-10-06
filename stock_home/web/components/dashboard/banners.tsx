"use client";

import { CircleAlert, Info, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { type Banner, useSection } from "@/lib/api";
import { cn } from "@/lib/utils";

const styles = {
  critical: { box: "border-critical/40 bg-critical/10", icon: CircleAlert, text: "text-critical" },
  warning: { box: "border-warning/40 bg-warning/10", icon: TriangleAlert, text: "text-warning" },
  info: { box: "border-info/40 bg-info/10", icon: Info, text: "text-info" },
} as const;

/** Section 11: things that need attention now. */
export function Banners() {
  const status = useSection("status");
  const banners = status.data?.data.banners ?? [];
  if (status.isError && !status.data) {
    return <BannerItem banner={{ level: "warning", title: "Home page API unreachable", message: "Retrying…" }} />;
  }
  if (banners.length === 0) return null;
  return (
    <div className="space-y-2" role="region" aria-label="Attention required">
      {banners.map((b) => (
        <BannerItem key={b.title} banner={b} />
      ))}
    </div>
  );
}

export function BannerItem({ banner }: { banner: Banner }) {
  const s = styles[banner.level];
  const Icon = s.icon;
  return (
    <div role="alert" className={cn("flex flex-col gap-3 rounded-xl border px-4 py-3 sm:flex-row sm:items-center", s.box)}>
      <Icon className={cn("size-5 shrink-0", s.text)} aria-hidden />
      <div className="min-w-0 flex-1">
        <p className={cn("font-semibold", s.text)}>{banner.title}</p>
        <p className="text-sm text-muted-foreground">{banner.message}</p>
      </div>
      {banner.action && (
        <Button asChild size="sm" variant={banner.level === "critical" ? "critical" : "outline"}>
          <a href={banner.action.href}>{banner.action.label}</a>
        </Button>
      )}
    </div>
  );
}
