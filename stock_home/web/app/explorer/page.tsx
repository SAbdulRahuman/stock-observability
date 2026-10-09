import type { Metadata } from "next";
import { Suspense } from "react";
import { ExplorerPage } from "@/components/explorer/explorer-page";

export const metadata: Metadata = { title: "Stock Explorer · stock_home" };

export default function Explorer() {
  return (
    <Suspense>
      <ExplorerPage />
    </Suspense>
  );
}
