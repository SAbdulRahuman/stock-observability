import type { Metadata } from "next";
import Link from "next/link";
import { formatDate, posts } from "@/lib/content";

export const metadata: Metadata = { title: "Blog", description: "Release notes and market observability write-ups from the iSee Ticks team." };

export default function BlogIndex() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-16">
      <h1 className="text-4xl font-bold tracking-tight">Blog</h1>
      <p className="mt-4 text-lg text-muted-foreground">
        Release notes and write-ups. <a href="/blog/feed.xml" className="text-primary hover:underline">RSS</a>
      </p>
      <ul className="mt-12 grid gap-8">
        {posts().map((p) => (
          <li key={p.slug}>
            <article>
              <time dateTime={p.date} className="text-sm text-muted-foreground">
                {formatDate(p.date)}
              </time>
              <h2 className="mt-1 text-xl font-semibold">
                <Link href={`/blog/${p.slug}`} className="hover:text-primary">
                  {p.title}
                </Link>
              </h2>
              <p className="mt-2 text-muted-foreground">{p.summary}</p>
            </article>
          </li>
        ))}
      </ul>
    </div>
  );
}
