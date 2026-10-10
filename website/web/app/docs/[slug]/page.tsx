import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Markdown } from "@/components/markdown";
import { docEntries, docFileToSlug, loadDoc } from "@/lib/content";

type Params = { params: Promise<{ slug: string }> };

export const dynamicParams = false;

export function generateStaticParams() {
  return docEntries().map((d) => ({ slug: d.slug }));
}

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const doc = loadDoc((await params).slug);
  return doc ? { title: doc.title, description: doc.description } : {};
}

export default async function DocPage({ params }: Params) {
  const doc = loadDoc((await params).slug);
  if (!doc) notFound();
  const entries = docEntries();
  return (
    <div className="mx-auto grid max-w-6xl gap-10 px-4 py-12 lg:grid-cols-[220px_1fr]">
      <nav aria-label="Docs" className="text-sm lg:sticky lg:top-20 lg:self-start">
        <ul className="flex flex-wrap gap-x-4 gap-y-2 lg:flex-col">
          {entries.map((d) => (
            <li key={d.slug}>
              <Link
                href={`/docs/${d.slug}`}
                aria-current={d.slug === doc.slug ? "page" : undefined}
                className="text-muted-foreground hover:text-foreground aria-[current=page]:font-semibold aria-[current=page]:text-foreground"
              >
                {d.title}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      <article className="min-w-0">
        <p className="text-sm text-muted-foreground">{doc.group}</p>
        <h1 className="mt-1 mb-8 text-4xl font-bold tracking-tight">{doc.title}</h1>
        <Markdown source={doc.body} format={doc.format} file={doc.file} fileToSlug={docFileToSlug()} />
      </article>
    </div>
  );
}
