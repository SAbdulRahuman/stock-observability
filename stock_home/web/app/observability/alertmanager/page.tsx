import type { Metadata } from "next";
import { Suspense } from "react";
import { AlertmanagerPage } from "@/components/alertmanager/alertmanager-page";

export const metadata: Metadata = { title: "Alertmanager · stock_home" };

export default function Alertmanager() {
  return (
    <Suspense>
      <AlertmanagerPage />
    </Suspense>
  );
}
