import type { Metadata } from "next";
import Link from "next/link";
import { docEntries } from "@/lib/content";

export const metadata: Metadata = { title: "Docs", description: "Quickstart, architecture and design documents for iSee Ticks." };

export default function DocsIndex() {
  const groups = Map.groupBy(docEntries(), (d) => d.group);
  return (
    <div className="mx-auto max-w-4xl px-4 py-16">
      <h1 className="text-4xl font-bold tracking-tight">Docs</h1>
      <p className="mt-4 text-lg text-muted-foreground">Run the stack yourself, and read how it is designed.</p>
      {[...groups].map(([group, entries]) => (
        <section key={group} className="mt-12">
          <h2 className="text-sm font-semibold tracking-wide text-muted-foreground uppercase">{group}</h2>
          <ul className="mt-4 grid gap-3 sm:grid-cols-2">
            {entries.map((d) => (
              <li key={d.slug}>
                <Link href={`/docs/${d.slug}`} className="block h-full rounded-xl border bg-card p-5 transition-colors hover:bg-accent">
                  <p className="font-semibold">{d.title}</p>
                  <p className="mt-1 text-sm text-muted-foreground">{d.description}</p>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
