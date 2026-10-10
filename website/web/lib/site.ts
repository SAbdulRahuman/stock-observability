export const site = {
  name: "iSee Ticks",
  tagline: "See Every Tick. Observe Every Move.",
  category: "Market Observability Platform",
  description:
    "iSee Ticks turns live NSE market data into metrics, KPIs and alerts, with Prometheus, Alertmanager and Grafana under one product home.",
  // Absolute URL for sitemap, RSS and Open Graph; set SITE_URL at build time once a domain exists.
  url: (process.env.SITE_URL ?? "http://localhost:8090").replace(/\/$/, ""),
  nav: [
    { href: "/#features", label: "Features" },
    { href: "/demo", label: "Live demo" },
    { href: "/docs", label: "Docs" },
    { href: "/blog", label: "Blog" },
  ],
} as const;
