import type { MetadataRoute } from "next";
import { docEntries, posts } from "@/lib/content";
import { site } from "@/lib/site";

export const dynamic = "force-static";

export default function sitemap(): MetadataRoute.Sitemap {
  const pages = ["", "/demo", "/docs", "/blog", "/waitlist", "/privacy"];
  return [
    ...pages.map((p) => ({ url: `${site.url}${p || "/"}` })),
    ...docEntries().map((d) => ({ url: `${site.url}/docs/${d.slug}` })),
    ...posts().map((p) => ({ url: `${site.url}/blog/${p.slug}`, lastModified: p.date })),
  ];
}
