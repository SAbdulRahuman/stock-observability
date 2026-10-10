import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Markdown } from "@/components/markdown";
import { formatDate, loadPost, posts } from "@/lib/content";

type Params = { params: Promise<{ slug: string }> };

export const dynamicParams = false;

export function generateStaticParams() {
  return posts().map((p) => ({ slug: p.slug }));
}

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const p = loadPost((await params).slug);
  return p ? { title: p.title, description: p.summary, openGraph: { type: "article", publishedTime: p.date } } : {};
}

export default async function PostPage({ params }: Params) {
  const p = loadPost((await params).slug);
  if (!p) notFound();
  return (
    <article className="mx-auto max-w-3xl px-4 py-16">
      <Link href="/blog" className="text-sm text-muted-foreground hover:text-foreground">
        ← All posts
      </Link>
      <h1 className="mt-6 text-4xl font-bold tracking-tight">{p.title}</h1>
      <p className="mt-3 mb-10 text-sm text-muted-foreground">
        <time dateTime={p.date}>{formatDate(p.date)}</time>
        {p.author && ` · ${p.author}`}
      </p>
      <Markdown source={p.body} format="mdx" />
    </article>
  );
}
