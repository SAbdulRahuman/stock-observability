import type { Metadata } from "next";
import { DashboardsPage } from "@/components/dashboards/dashboards-page";

export const metadata: Metadata = { title: "Dashboards · stock_home" };

export default function Dashboards() {
  return <DashboardsPage />;
}
