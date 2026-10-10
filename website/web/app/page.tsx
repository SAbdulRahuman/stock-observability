import { DemoLink } from "@/components/demo-link";
import { Features } from "@/components/landing/features";
import { Hero } from "@/components/landing/hero";
import { Pipeline } from "@/components/landing/pipeline";
import { Showcase } from "@/components/landing/showcase";
import { Card } from "@/components/ui/card";
import { WaitlistForm } from "@/components/waitlist-form";
import { site } from "@/lib/site";

export default function Landing() {
  return (
    <>
      <Hero />
      <Pipeline />
      <Showcase />
      <Features />
      <section id="waitlist" className="mx-auto grid max-w-6xl scroll-mt-14 gap-8 px-4 lg:grid-cols-[1fr_1.2fr] lg:items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight sm:text-5xl">See it live, then get early access.</h2>
          <p className="mt-4 text-muted-foreground">
            The demo is read-only and runs on simulated data. {site.name} is in MVP: join the waitlist for hosted access and the
            first public build.
          </p>
          <div className="mt-6">
            <DemoLink />
          </div>
        </div>
        <Card className="p-8">
          <WaitlistForm />
        </Card>
      </section>
    </>
  );
}
