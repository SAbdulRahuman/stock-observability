import fs from "node:fs";
import path from "node:path";
import matter from "gray-matter";

// Paths in the docs manifest are relative to website/; `next build` runs in website/web.
const websiteDir = path.join(process.cwd(), "..");
const contentDir = path.join(websiteDir, "content");

export type DocEntry = { slug: string; title: string; description: string; group: string; file: string };
export type Doc = DocEntry & { body: string; format: "md" | "mdx" };
export type Post = { slug: string; title: string; date: string; summary: string; author: string };

export function docEntries(): DocEntry[] {
  return JSON.parse(fs.readFileSync(path.join(contentDir, "docs", "manifest.json"), "utf8")) as DocEntry[];
}

export function docFileToSlug(): Map<string, string> {
  return new Map(docEntries().map((d) => [path.posix.normalize(d.file), d.slug]));
}

export function loadDoc(slug: string): Doc | undefined {
  const entry = docEntries().find((d) => d.slug === slug);
  if (!entry) return undefined;
  const { content } = matter(fs.readFileSync(path.join(websiteDir, entry.file), "utf8"));
  const format = entry.file.endsWith(".mdx") ? "mdx" : "md";
  // The page renders the title, so drop a leading H1 from repo markdown.
  const body = format === "md" ? content.replace(/^\s*# .*\n/, "") : content;
  return { ...entry, body, format };
}

function readPost(file: string): Post & { body: string; draft: boolean } {
  const { data, content } = matter(fs.readFileSync(path.join(contentDir, "blog", file), "utf8"));
  const date = data.date instanceof Date ? data.date.toISOString().slice(0, 10) : String(data.date ?? "");
  return {
    slug: file.replace(/\.mdx$/, ""),
    title: String(data.title ?? ""),
    date,
    summary: String(data.summary ?? ""),
    author: String(data.author ?? ""),
    draft: data.draft === true,
    body: content,
  };
}

/** Published posts, newest first. */
export function posts(): Post[] {
  return fs
    .readdirSync(path.join(contentDir, "blog"))
    .filter((f) => f.endsWith(".mdx"))
    .map(readPost)
    .filter((p) => !p.draft)
    .sort((a, b) => b.date.localeCompare(a.date))
    .map(({ body: _body, draft: _draft, ...p }) => p);
}

export function loadPost(slug: string): (Post & { body: string }) | undefined {
  if (!posts().some((p) => p.slug === slug)) return undefined;
  const { draft: _draft, ...p } = readPost(`${slug}.mdx`);
  return p;
}

export function formatDate(iso: string): string {
  return new Date(`${iso}T00:00:00Z`).toLocaleDateString("en-IN", { day: "numeric", month: "long", year: "numeric", timeZone: "UTC" });
}
