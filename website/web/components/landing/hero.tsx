"use client";

import { ArrowDown, ArrowRight } from "lucide-react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRef } from "react";
import { DemoLink } from "@/components/demo-link";
import { TickerStrip } from "@/components/ticker-strip";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { site } from "@/lib/site";
import { gsap, MOBILE, MOTION, REDUCED, ScrollTrigger, useGSAP, useMediaQuery } from "./motion";

// three.js is only downloaded for the hero, after hydration.
const HeroScene = dynamic(() => import("./hero-scene"), { ssr: false });

export function Hero() {
  const root = useRef<HTMLElement>(null);
  const progress = useRef(0);
  const reduced = useMediaQuery(REDUCED);
  const lite = useMediaQuery(MOBILE);

  useGSAP(
    () => {
      const mm = gsap.matchMedia();
      mm.add(MOTION, () => {
        gsap.from(".hero-word", { yPercent: 110, duration: 0.9, ease: "power3.out", stagger: 0.06, delay: 0.1 });
        gsap.from(".hero-fade", { y: 24, autoAlpha: 0, duration: 0.8, ease: "power3.out", stagger: 0.1, delay: 0.5 });
        gsap.from(".hero-scene", { autoAlpha: 0, duration: 1.6, ease: "power2.out" });
        const scroll = { trigger: root.current, start: "top top", end: "bottom top", scrub: true };
        ScrollTrigger.create({ ...scroll, onUpdate: (s) => (progress.current = s.progress) });
        gsap.to(".hero-copy", { yPercent: -30, autoAlpha: 0, ease: "none", scrollTrigger: scroll });
        gsap.to(".hero-cue", { y: 8, repeat: -1, yoyo: true, duration: 0.9, ease: "sine.inOut" });
      });
    },
    { scope: root },
  );

  return (
    <section ref={root} className="dark relative isolate flex min-h-[100svh] flex-col overflow-hidden bg-background text-foreground">
      <div className="hero-scene absolute inset-0 -z-10">
        <HeroScene progress={progress} reduced={reduced} lite={lite} />
      </div>
      <div aria-hidden className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(70%_55%_at_50%_0%,color-mix(in_oklch,var(--primary)_28%,transparent),transparent)]" />
      <div aria-hidden className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-b from-background/70 via-background/20 to-transparent md:bg-gradient-to-r md:from-background/85 md:via-background/40" />
      <div aria-hidden className="pointer-events-none absolute inset-x-0 bottom-0 -z-10 h-1/3 bg-gradient-to-b from-transparent to-background" />

      <div className="hero-copy mx-auto w-full max-w-6xl flex-1 px-4 pt-20 text-center sm:pt-28 md:pt-32 md:text-left">
        <Badge variant="outline" className="hero-fade mb-6 border-primary/40 bg-background/40 backdrop-blur">
          {site.category}
        </Badge>
        <h1 className="mx-auto max-w-3xl text-5xl font-bold tracking-tight text-balance sm:text-7xl md:mx-0">
          {site.tagline.split(" ").map((w, i) => (
            <span key={`${w}-${i}`} className="inline-block overflow-hidden pb-1 align-bottom">
              <span className="hero-word inline-block">{w}&nbsp;</span>
            </span>
          ))}
        </h1>
        <p className="hero-fade mx-auto mt-6 max-w-xl text-lg text-pretty text-muted-foreground md:mx-0">{site.description}</p>
        <div className="hero-fade mt-8 flex flex-wrap justify-center gap-3 md:justify-start">
          <DemoLink />
          <Button size="lg" variant="outline" className="bg-background/40 backdrop-blur" asChild>
            <Link href="/waitlist">
              Join the waitlist <ArrowRight aria-hidden />
            </Link>
          </Button>
        </div>
      </div>

      <div className="hero-fade mx-auto w-full max-w-5xl px-4 pb-10">
        <TickerStrip className="bg-card/70 backdrop-blur" />
        <a href="#pipeline" className="hero-cue mx-auto mt-6 flex w-fit items-center gap-2 text-xs text-muted-foreground hover:text-foreground">
          <ArrowDown className="size-4" aria-hidden /> Scroll to see how it works
        </a>
      </div>
    </section>
  );
}
