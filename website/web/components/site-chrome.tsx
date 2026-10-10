import Link from "next/link";
import { ThemeToggle } from "@/components/providers";
import { Button } from "@/components/ui/button";
import { site } from "@/lib/site";

/** The "eye over a tick chart" mark; currentColor lets it follow the theme. */
export function Logo({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" className={className} aria-hidden fill="none" stroke="currentColor" strokeWidth={2.2} strokeLinecap="round" strokeLinejoin="round">
      <path d="M2 16s5-9 14-9 14 9 14 9-5 9-14 9S2 16 2 16Z" />
      <path d="M9 18l4-4 3 3 4-5 3 3" stroke="var(--primary)" />
    </svg>
  );
}

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/80 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-6xl items-center gap-6 px-4">
        <Link href="/" className="flex items-center gap-2 font-semibold tracking-tight">
          <Logo className="size-7" />
          {site.name}
        </Link>
        <nav aria-label="Main" className="hidden gap-5 text-sm text-muted-foreground md:flex">
          {site.nav.map((n) => (
            <Link key={n.href} href={n.href} className="hover:text-foreground">
              {n.label}
            </Link>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-2">
          <ThemeToggle />
          <Button asChild size="sm">
            <Link href="/waitlist">Join waitlist</Link>
          </Button>
        </div>
      </div>
      <nav aria-label="Main mobile" className="flex gap-4 overflow-x-auto border-t px-4 py-2 text-sm text-muted-foreground md:hidden">
        {site.nav.map((n) => (
          <Link key={n.href} href={n.href} className="whitespace-nowrap hover:text-foreground">
            {n.label}
          </Link>
        ))}
      </nav>
    </header>
  );
}

export function SiteFooter() {
  return (
    <footer className="mt-24 border-t">
      <div className="mx-auto grid max-w-6xl gap-6 px-4 py-10 text-sm text-muted-foreground md:grid-cols-[2fr_1fr_1fr]">
        <div className="space-y-2">
          <p className="flex items-center gap-2 font-semibold text-foreground">
            <Logo className="size-5" /> {site.name}
          </p>
          <p>{site.tagline}</p>
          <p className="text-xs">
            Demo data is simulated. Nothing on this site is investment advice. Not affiliated with NSE, BSE or any broker.
          </p>
        </div>
        <ul className="space-y-1.5">
          <li><Link href="/demo" className="hover:text-foreground">Live demo</Link></li>
          <li><Link href="/docs" className="hover:text-foreground">Docs</Link></li>
          <li><Link href="/blog" className="hover:text-foreground">Blog</Link></li>
        </ul>
        <ul className="space-y-1.5">
          <li><Link href="/waitlist" className="hover:text-foreground">Waitlist</Link></li>
          <li><Link href="/privacy" className="hover:text-foreground">Privacy</Link></li>
          <li><a href="/blog/feed.xml" className="hover:text-foreground">RSS</a></li>
        </ul>
      </div>
    </footer>
  );
}
