import { MDXRemote } from "next-mdx-remote/rsc";
import Link from "next/link";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";
import { resolveDocHref } from "@/lib/links";

type Props = { source: string; format: "md" | "mdx"; file?: string; fileToSlug?: ReadonlyMap<string, string> };

/** Renders trusted repo markdown/MDX at build time; JS expressions stay blocked. */
export function Markdown({ source, format, file = "", fileToSlug = new Map() }: Props) {
  const components = {
    a: ({ href = "", children }: React.ComponentProps<"a">) => {
      const r = resolveDocHref(href, file, fileToSlug);
      if (!r) return <span>{children}</span>;
      if (r.external) {
        return (
          <a href={r.href} target="_blank" rel="noopener noreferrer">
            {children}
          </a>
        );
      }
      return r.href.startsWith("#") ? <a href={r.href}>{children}</a> : <Link href={r.href}>{children}</Link>;
    },
    img: ({ src, alt }: React.ComponentProps<"img">) =>
      typeof src === "string" && src.startsWith("/") ? <img src={src} alt={alt ?? ""} loading="lazy" /> : <span>{alt}</span>,
  };
  return (
    <div className="prose max-w-none dark:prose-invert prose-headings:scroll-mt-20 prose-pre:bg-card prose-pre:text-card-foreground prose-pre:border">
      <MDXRemote
        source={source}
        components={components}
        options={{ blockJS: true, mdxOptions: { format, remarkPlugins: [remarkGfm], rehypePlugins: [rehypeSlug] } }}
      />
    </div>
  );
}
