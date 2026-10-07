import type { Metadata } from "next";
import { Suspense } from "react";
import { Workbench } from "@/components/openbb/workbench";

export const metadata: Metadata = { title: "OpenBB · Stock Observability" };

export default function OpenbbPage() {
  return (
    <Suspense>
      <Workbench />
    </Suspense>
  );
}
