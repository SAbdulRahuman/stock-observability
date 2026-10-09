import type { Metadata } from "next";
import { Suspense } from "react";
import { Analyst } from "@/components/analyst/analyst";

export const metadata: Metadata = { title: "Analyst · stock_home" };

export default function AnalystPage() {
  return (
    <Suspense>
      <Analyst />
    </Suspense>
  );
}
