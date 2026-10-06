import { cva, type VariantProps } from "class-variance-authority";
import type * as React from "react";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 whitespace-nowrap rounded-md border px-2 py-0.5 text-xs font-medium [&>svg]:size-3",
  {
    variants: {
      variant: {
        default: "border-transparent bg-secondary text-secondary-foreground",
        outline: "text-foreground",
        critical: "border-critical/30 bg-critical/10 text-critical",
        warning: "border-warning/30 bg-warning/10 text-warning",
        info: "border-info/30 bg-info/10 text-info",
        ok: "border-ok/30 bg-ok/10 text-ok",
        up: "border-transparent bg-up/10 text-up",
        down: "border-transparent bg-down/10 text-down",
      },
    },
    defaultVariants: { variant: "default" },
  },
);

export type BadgeVariant = NonNullable<VariantProps<typeof badgeVariants>["variant"]>;

export function Badge({
  className,
  variant,
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof badgeVariants>) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export function severityVariant(severity: string): BadgeVariant {
  if (severity === "critical" || severity === "warning" || severity === "info") return severity;
  return "default";
}
