import type { Metadata } from "next";
import { WaitlistForm } from "@/components/waitlist-form";
import { site } from "@/lib/site";

export const metadata: Metadata = {
  title: "Join the waitlist",
  description: `Get early access to ${site.name}, the ${site.category.toLowerCase()} for Indian markets.`,
};

export default function WaitlistPage() {
  return (
    <div className="mx-auto max-w-2xl px-4 py-16">
      <h1 className="text-4xl font-bold tracking-tight">Join the waitlist</h1>
      <p className="mt-4 mb-10 text-lg text-muted-foreground">
        Be first to try hosted {site.name}, get release notes, and help shape which markets and KPIs come next.
      </p>
      <WaitlistForm />
    </div>
  );
}
