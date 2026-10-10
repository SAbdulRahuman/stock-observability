"use client";

import { BellRing, Cpu, Database, Eye, Radio } from "lucide-react";
import { useRef } from "react";
import { site } from "@/lib/site";
import { DESKTOP_MOTION, gsap, MOBILE, useGSAP } from "./motion";

const steps = [
  { icon: Radio, name: "Market feed", body: "Broker WebSocket ticks, or the built-in simulator for demos and tests." },
  { icon: Cpu, name: "Exporter", body: "Every tick becomes a metric: price, volume, five-level depth, circuits." },
  { icon: Database, name: "Prometheus", body: "Recording rules compute breadth, levels and risk KPIs every second." },
  { icon: BellRing, name: "Alertmanager", body: "Groups and routes alerts, and stays quiet outside NSE hours." },
  { icon: Eye, name: site.name, body: "One home for market, movers, explorer, alerts and Grafana." },
];

/** Feed-to-insight pipeline: pinned on desktop while a tick travels along the line. */
export function Pipeline() {
  const root = useRef<HTMLElement>(null);
  useGSAP(
    () => {
      const mm = gsap.matchMedia();
      mm.add(DESKTOP_MOTION, () => {
        const nodes = gsap.utils.toArray<HTMLElement>(".pipe-body");
        const marks = gsap.utils.toArray<HTMLElement>(".pipe-mark");
        gsap.set(nodes, { autoAlpha: 0.25, y: 16 });
        gsap.set(marks, { backgroundColor: "var(--background)", scale: 1 });
        gsap.set(".pipe-fill", { scaleX: 0 });
        const tl = gsap.timeline({
          defaults: { ease: "none" },
          scrollTrigger: { trigger: root.current, start: "top top", end: "+=160%", scrub: 0.6, pin: true, refreshPriority: 2 },
        });
        const span = nodes.length - 1;
        tl.to(".pipe-fill", { scaleX: 1, duration: span }, 0).fromTo(".pipe-dot", { left: "0%" }, { left: "100%", duration: span }, 0);
        nodes.forEach((n, i) => {
          const at = Math.max(0, i - 0.15);
          tl.to(n, { autoAlpha: 1, y: 0, duration: 0.25, ease: "power2.out" }, at);
          tl.to(marks[i], { backgroundColor: "var(--primary)", scale: 1.4, duration: 0.15 }, at);
        });
      });
      mm.add(MOBILE, () => {
        gsap.from(".pipe-node", { autoAlpha: 0, y: 24, stagger: 0.12, duration: 0.6, ease: "power2.out", scrollTrigger: { trigger: root.current, start: "top 75%" } });
      });
    },
    { scope: root },
  );

  return (
    <section id="pipeline" ref={root} className="flex min-h-[100svh] scroll-mt-14 flex-col justify-center border-b py-20">
      <div className="mx-auto w-full max-w-6xl px-4">
        <p className="font-mono text-xs tracking-widest text-primary uppercase">How it works</p>
        <h2 className="mt-3 max-w-2xl text-3xl font-bold tracking-tight sm:text-5xl">From tick to insight in about a second.</h2>
        <p className="mt-4 max-w-2xl text-muted-foreground">
          Proven open-source building blocks, wired for Indian markets and shipped as single binaries.
        </p>

        {/* Columns have no gap, so node centres sit at 10%, 30%, … 90%: the line spans 10%–90%. */}
        <div className="relative mx-[10%] mt-16 hidden md:block" aria-hidden>
          <div className="h-px w-full bg-border" />
          <div className="pipe-fill absolute inset-x-0 top-0 h-px origin-left bg-gradient-to-r from-primary/40 via-primary to-up" />
          <div className="pipe-dot absolute top-0 left-full size-3 -translate-1/2 rounded-full bg-primary shadow-[0_0_24px_6px_var(--primary)]" />
        </div>

        <ol className="mt-10 grid gap-6 md:mt-0 md:grid-cols-5 md:gap-0">
          {steps.map((s, i) => (
            <li key={s.name} className="pipe-node relative md:px-3 md:pt-10 md:text-center">
              <span aria-hidden className="pipe-mark absolute -top-1.5 left-1/2 hidden size-3 -translate-x-1/2 rounded-full border-2 border-primary bg-primary md:block" />
              <div className="pipe-body">
              <div className="flex items-center gap-3 md:block">
                <span className="grid size-11 shrink-0 place-items-center rounded-xl border bg-card text-primary md:mx-auto md:mb-4">
                  <s.icon className="size-5" aria-hidden />
                </span>
                <p className="font-semibold">
                  <span className="mr-2 font-mono text-xs text-muted-foreground">0{i + 1}</span>
                  {s.name}
                </p>
              </div>
              <p className="mt-2 text-sm text-muted-foreground">{s.body}</p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
