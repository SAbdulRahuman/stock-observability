"use client";

import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { navItem } from "@/components/shell/nav";
import { useLinkHref } from "@/components/shell/hooks";

/** "View All →" to an in-app page once it exists, else to the configured fallback tool. */
export function ViewAll({ page, fallbackLinkId, label }: { page: string; fallbackLinkId?: string; label: string }) {
  const item = navItem(page);
  const fallback = useLinkHref(fallbackLinkId);
  const cls = "inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline";
  const body = (
    <>
      View All <ArrowRight className="size-3.5" aria-hidden />
    </>
  );
  if (item?.ready && item.href) {
    return (
      <Link href={item.href} className={cls} aria-label={`View all ${label}`}>
        {body}
      </Link>
    );
  }
  if (!fallback) return null;
  return (
    <a href={fallback} className={cls} aria-label={`View all ${label}`} target="_blank" rel="noopener noreferrer">
      {body}
    </a>
  );
}
