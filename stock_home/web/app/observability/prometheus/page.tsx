import type { Metadata } from "next";
import { Suspense } from "react";
import { PrometheusPage } from "@/components/prometheus/prometheus-page";

export const metadata: Metadata = { title: "Prometheus · stock_home" };

export default function Prometheus() {
  return (
    <Suspense>
      <PrometheusPage />
    </Suspense>
  );
}
