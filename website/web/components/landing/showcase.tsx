"use client";

import { Check } from "lucide-react";
import { useRef } from "react";
import screens from "@/lib/screens.json";
import { site } from "@/lib/site";
import { cn } from "@/lib/utils";
import { DESKTOP_MOTION, gsap, ScrollTrigger, useGSAP, useMediaQuery } from "./motion";

type Screen = (typeof screens)[number];

const W = 1600;
const H = 1000;

function BrowserFrame({ label, children, className }: { label: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={cn("overflow-hidden rounded-xl border bg-card shadow-[0_40px_120px_-40px_color-mix(in_oklch,var(--primary)_55%,transparent)]", className)}>
      <div className="flex items-center gap-2 border-b bg-muted/60 px-4 py-2.5">
        <span className="size-2.5 rounded-full bg-down/70" />
        <span className="size-2.5 rounded-full bg-warning/70" />
        <span className="size-2.5 rounded-full bg-up/70" />
        <span className="mx-auto truncate rounded-md bg-background/70 px-3 py-0.5 font-mono text-xs text-muted-foreground">
          {label}
        </span>
      </div>
      <div className="relative aspect-[16/10]">{children}</div>
    </div>
  );
}

function Shot({ s, className, eager }: { s: Screen; className?: string; eager?: boolean }) {
  return (
    <img
      src={`/screenshots/${s.id}.webp`}
      alt={`${site.name} ${s.title} screen, running on simulated market data`}
      width={W}
      height={H}
      loading={eager ? "eager" : "lazy"}
      decoding="async"
      className={cn("absolute inset-0 size-full object-cover object-top", className)}
    />
  );
}

function Caption({ s, i, className }: { s: Screen; i: number; className?: string }) {
  return (
    <div className={className}>
      <p className="font-mono text-xs tracking-widest text-primary uppercase">
        0{i + 1} · {s.title}
      </p>
      <h3 className="mt-2 text-2xl font-bold tracking-tight sm:text-3xl">{s.kicker}</h3>
      <ul className="mt-4 grid gap-2 text-muted-foreground">
        {s.points.map((p) => (
          <li key={p} className="flex gap-2">
            <Check className="mt-1 size-4 shrink-0 text-up" aria-hidden />
            {p}
          </li>
        ))}
      </ul>
    </div>
  );
}

/** Pinned on desktop: screens replace each other while the captions follow. */
function PinnedShowcase() {
  const root = useRef<HTMLDivElement>(null);
  useGSAP(
    () => {
      const shots = gsap.utils.toArray<HTMLElement>(".show-shot");
      const caps = gsap.utils.toArray<HTMLElement>(".show-cap");
      const bars = gsap.utils.toArray<HTMLElement>(".show-bar");
      gsap.set(shots.slice(1), { clipPath: "inset(100% 0% 0% 0%)" });
      gsap.set(caps.slice(1), { autoAlpha: 0, y: 24 });
      gsap.set(bars, { scaleX: 0 });
      gsap.set(bars[0], { scaleX: 1 });

      gsap.fromTo(
        ".show-frame",
        { rotateX: 24, scale: 0.86, y: 80 },
        { rotateX: 0, scale: 1, y: 0, ease: "none", scrollTrigger: { trigger: root.current, start: "top bottom", end: "top top", scrub: true } },
      );
      const tl = gsap.timeline({
        defaults: { ease: "power2.inOut", duration: 1 },
        scrollTrigger: { trigger: root.current, start: "top top", end: `+=${(shots.length - 1) * 90}%`, scrub: 0.6, pin: true, refreshPriority: 1 },
      });
      for (let i = 1; i < shots.length; i++) {
        const at = i - 1;
        tl.to(shots[i], { clipPath: "inset(0% 0% 0% 0%)" }, at)
          .to(shots[i - 1], { filter: "brightness(0.35)" }, at)
          .to(caps[i - 1], { autoAlpha: 0, y: -24, duration: 0.5 }, at)
          .to(caps[i], { autoAlpha: 1, y: 0, duration: 0.5 }, at + 0.5)
          .to(bars[i], { scaleX: 1, ease: "none" }, at);
      }
      // Earlier sections were measured before this pin existed.
      ScrollTrigger.sort();
      ScrollTrigger.refresh();
    },
    { scope: root },
  );

  return (
    <div ref={root} className="flex min-h-[100svh] items-center">
      <div className="mx-auto grid w-full max-w-7xl items-center gap-12 px-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.8fr)]">
        <div>
          <div className="mb-8 flex gap-2" aria-hidden>
            {screens.map((s) => (
              <span key={s.id} className="h-1 flex-1 overflow-hidden rounded-full bg-border">
                <span className="show-bar block h-full origin-left rounded-full bg-primary" />
              </span>
            ))}
          </div>
          <div className="grid">
            {screens.map((s, i) => (
              <Caption key={s.id} s={s} i={i} className="show-cap [grid-area:1/1]" />
            ))}
          </div>
        </div>
        <div className="[perspective:1600px]">
          <BrowserFrame label="Live demo · simulated data" className="show-frame origin-bottom">
            {screens.map((s, i) => (
              <Shot key={s.id} s={s} eager={i === 0} className="show-shot" />
            ))}
          </BrowserFrame>
        </div>
      </div>
    </div>
  );
}

/** Mobile, reduced motion and the pre-hydration HTML: a plain list of screens. */
function ListShowcase() {
  return (
    <div className="mx-auto grid max-w-3xl gap-16 px-4 py-8">
      {screens.map((s, i) => (
        <div key={s.id} className="grid gap-6">
          <Caption s={s} i={i} />
          <BrowserFrame label={`demo${s.path}`}>
            <Shot s={s} />
          </BrowserFrame>
        </div>
      ))}
    </div>
  );
}

export function Showcase() {
  const pinned = useMediaQuery(DESKTOP_MOTION);
  return (
    <section id="product" className="scroll-mt-14 border-b py-20 md:py-0">
      <div className="mx-auto max-w-7xl px-4 md:pt-24">
        <p className="font-mono text-xs tracking-widest text-primary uppercase">The product</p>
        <h2 className="mt-3 max-w-2xl text-3xl font-bold tracking-tight sm:text-5xl">Your market, observed like production.</h2>
        <p className="mt-4 max-w-2xl text-muted-foreground">
          Real screens from the live demo, captured on the built-in market simulator.
        </p>
      </div>
      {pinned ? <PinnedShowcase /> : <ListShowcase />}
    </section>
  );
}
