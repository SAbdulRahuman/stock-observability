import { posts } from "@/lib/content";
import { site } from "@/lib/site";

export const dynamic = "force-static";

const esc = (s: string) => s.replace(/[<>&'"]/g, (c) => `&#${c.charCodeAt(0)};`);

export function GET() {
  const items = posts()
    .map(
      (p) => `<item>
  <title>${esc(p.title)}</title>
  <link>${site.url}/blog/${p.slug}</link>
  <guid>${site.url}/blog/${p.slug}</guid>
  <pubDate>${new Date(`${p.date}T00:00:00Z`).toUTCString()}</pubDate>
  <description>${esc(p.summary)}</description>
</item>`,
    )
    .join("\n");
  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>${esc(site.name)} blog</title>
<link>${site.url}/blog</link>
<description>${esc(site.tagline)}</description>
${items}
</channel>
</rss>
`;
  return new Response(xml, { headers: { "Content-Type": "application/rss+xml; charset=utf-8" } });
}
