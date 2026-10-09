import type { Metadata } from "next";
import { WatchlistPage } from "@/components/watchlist/watchlist-page";

export const metadata: Metadata = { title: "Watchlist · stock_home" };

export default function Watchlist() {
  return <WatchlistPage />;
}
