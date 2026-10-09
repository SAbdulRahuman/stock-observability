import type { Metadata } from "next";
import { TopMoversPage } from "@/components/movers/top-movers-page";

export const metadata: Metadata = { title: "Top Movers · stock_home" };

export default function Movers() {
  return <TopMoversPage />;
}
