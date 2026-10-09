import type { Metadata } from "next";
import { Suspense } from "react";
import { TopMoversPage } from "@/components/movers/top-movers-page";

export const metadata: Metadata = { title: "Top Movers · stock_home" };

export default function Movers() {
  return (
    <Suspense>
      <TopMoversPage />
    </Suspense>
  );
}
