import type { Metadata } from "next";
import { AlertsPage } from "@/components/alerts/alerts-page";

export const metadata: Metadata = { title: "Alerts · stock_home" };

export default function Alerts() {
  return <AlertsPage />;
}
