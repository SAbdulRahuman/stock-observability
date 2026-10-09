import type { Metadata } from "next";
import { SectorViewPage } from "@/components/sectors/sector-view-page";

export const metadata: Metadata = { title: "Sector View · stock_home" };

export default function Sectors() {
  return <SectorViewPage />;
}
