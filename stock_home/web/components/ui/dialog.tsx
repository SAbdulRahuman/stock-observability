"use client";

import { Dialog as D } from "radix-ui";
import type * as React from "react";
import { cn } from "@/lib/utils";

export const Dialog = D.Root;
export const DialogClose = D.Close;

export function DialogContent({ title, description, className, children }: { title: string; description?: React.ReactNode; className?: string; children: React.ReactNode }) {
  return (
    <D.Portal>
      <D.Overlay className="fixed inset-0 z-40 bg-black/60" />
      <D.Content
        className={cn(
          "fixed top-1/2 left-1/2 z-50 w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border bg-card p-5 text-card-foreground shadow-lg",
          className,
        )}
      >
        <D.Title className="text-base font-semibold">{title}</D.Title>
        {description ? <D.Description className="mt-1 text-sm text-muted-foreground">{description}</D.Description> : <D.Description className="sr-only">{title}</D.Description>}
        <div className="mt-4">{children}</div>
      </D.Content>
    </D.Portal>
  );
}

export function DialogFooter({ children }: { children: React.ReactNode }) {
  return <div className="mt-5 flex justify-end gap-2">{children}</div>;
}
