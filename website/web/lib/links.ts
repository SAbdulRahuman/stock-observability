import path from "node:path";

export type ResolvedHref = { href: string; external: boolean };

/**
 * Maps a link inside a repo markdown file to the website: published docs become /docs/<slug>, site paths and
 * http(s)/mailto links are kept, and anything else (repo-only files, other schemes) is dropped.
 */
export function resolveDocHref(href: string, fromFile: string, fileToSlug: ReadonlyMap<string, string>): ResolvedHref | null {
  if (href.startsWith("#")) return { href, external: false };
  if (/^(https?:|mailto:)/i.test(href)) return { href, external: true };
  if (href.startsWith("//") || /^[a-z][a-z0-9+.-]*:/i.test(href)) return null;
  if (href.startsWith("/")) return { href, external: false };
  const [target, hash] = href.split("#", 2);
  const resolved = path.posix.normalize(path.posix.join(path.posix.dirname(fromFile), target));
  const slug = fileToSlug.get(resolved);
  if (!slug) return null;
  return { href: `/docs/${slug}${hash ? `#${hash}` : ""}`, external: false };
}
