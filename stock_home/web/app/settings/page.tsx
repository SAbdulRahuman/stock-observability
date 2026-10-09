import type { Metadata } from "next";
import { Suspense } from "react";
import { SettingsPage } from "@/components/settings/settings-page";

export const metadata: Metadata = { title: "Settings · stock_home" };

export default function Settings() {
  return (
    <Suspense>
      <SettingsPage />
    </Suspense>
  );
}
