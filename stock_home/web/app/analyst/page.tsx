import type { Metadata } from "next";
import { Analyst } from "@/components/analyst/analyst";

export const metadata: Metadata = { title: "Analyst · Stock Observability" };

export default function AnalystPage() {
  return <Analyst />;
}
