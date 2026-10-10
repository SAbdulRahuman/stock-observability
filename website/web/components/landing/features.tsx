"use client";

import { BellRing, Calculator, Gauge, LayoutDashboard, LineChart, Radio } from "lucide-react";
import { useRef } from "react";
import { Card } from "@/components/ui/card";
import { gsap, MOTION, useGSAP } from "./motion";

const features = [
  { icon: Radio, title: "Every tick as a metric", body: "Last price, volume, circuit limits, five-level depth and order-book imbalance for every instrument on your watchlist." },
  { icon: Gauge, title: "NSE KPI catalogue", body: "Breadth, sector moves, pivot and reference levels, derivatives and risk KPIs, each with a tested contract." },
  { icon: BellRing, title: "Alerts that know the market", body: "Price, circuit, flow and portfolio alerts, muted outside the NSE session and on exchange holidays." },
  { icon: LayoutDashboard, title: "One product home", body: "Market overview, watchlist, movers, sectors, stock explorer and alerts. No PromQL needed." },
  { icon: LineChart, title: "Grafana when you need depth", body: "Provisioned dashboards for market, instruments and feed health, one click from any card." },
  { icon: Calculator, title: "Quant lab built in", body: "Implied volatility, Greeks, max pain and short-horizon forecasts, scraped like any other metric." },
];

const stats = [
  { value: 1, suffix: " s", label: "core scrape interval" },
  { value: 5, suffix: "", label: "levels of market depth" },
  { value: 13, suffix: "k", label: "series per core scrape" },
  { value: 0, suffix: "", label: "pages outside market hours" },
];

export function Features() {
  const root = useRef<HTMLElement>(null);
  useGSAP(
    () => {
      const mm = gsap.matchMedia();
      mm.add(MOTION, () => {
        gsap.from(".feat-head", { autoAlpha: 0, y: 32, duration: 0.8, ease: "power3.out", scrollTrigger: { trigger: ".feat-head", start: "top 85%" } });
        gsap.from(".feat-card", {
          autoAlpha: 0,
          y: 48,
          rotateX: -12,
          duration: 0.8,
          ease: "power3.out",
          stagger: { each: 0.08, grid: "auto", from: "start" },
          scrollTrigger: { trigger: ".feat-grid", start: "top 80%" },
        });
        gsap.utils.toArray<HTMLElement>(".stat-num").forEach((el) => {
          const target = Number(el.dataset.value);
          const n = { v: 0 };
          gsap.to(n, {
            v: target,
            duration: 1.4,
            ease: "power2.out",
            scrollTrigger: { trigger: el, start: "top 90%" },
            onUpdate: () => {
              el.textContent = String(Math.round(n.v));
            },
          });
        });
      });
    },
    { scope: root },
  );

  return (
    <section id="features" ref={root} className="mx-auto max-w-6xl scroll-mt-14 px-4 py-24">
      <div className="feat-head">
        <p className="font-mono text-xs tracking-widest text-primary uppercase">Features</p>
        <h2 className="mt-3 max-w-2xl text-3xl font-bold tracking-tight sm:text-5xl">Observability, applied to the market.</h2>
        <p className="mt-4 max-w-2xl text-muted-foreground">
          The tools that keep production systems healthy, pointed at prices, order books and alerts instead.
        </p>
      </div>
      <div className="feat-grid mt-12 grid gap-4 [perspective:1200px] sm:grid-cols-2 lg:grid-cols-3">
        {features.map((f) => (
          <Card key={f.title} className="feat-card group relative overflow-hidden p-6 transition-colors hover:border-primary/50">
            <div aria-hidden className="pointer-events-none absolute -top-16 -right-16 size-40 rounded-full bg-primary/10 blur-2xl transition-opacity group-hover:opacity-100 md:opacity-0" />
            <f.icon className="size-6 text-primary" aria-hidden />
            <h3 className="mt-4 font-semibold">{f.title}</h3>
            <p className="mt-2 text-sm text-muted-foreground">{f.body}</p>
          </Card>
        ))}
      </div>
      <dl className="mt-20 grid grid-cols-2 gap-8 border-y py-10 md:grid-cols-4">
        {stats.map((s) => (
          <div key={s.label} className="text-center">
            <dt className="sr-only">{s.label}</dt>
            <dd className="font-mono text-4xl font-bold tracking-tight sm:text-5xl">
              <span className="stat-num" data-value={s.value}>
                {s.value}
              </span>
              <span className="text-primary">{s.suffix}</span>
            </dd>
            <dd className="mt-2 text-sm text-muted-foreground">{s.label}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
