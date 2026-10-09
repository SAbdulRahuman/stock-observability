import type { Metadata } from "next";
import { MarketOverviewPage } from "@/components/market/market-overview";

export const metadata: Metadata = { title: "Market Overview · stock_home" };

export default function MarketPage() {
  return <MarketOverviewPage />;
}
