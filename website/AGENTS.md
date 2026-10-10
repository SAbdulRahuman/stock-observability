# website (iSee Ticks)

Marketing site + read-only public demo proxy, one Go binary (`stock_website`) with the Next.js static export embedded (`ui/dist`, same pattern as stock_home).

- `make test` (go vet, go test -race, tsc, vitest), `make build`, `make dist` (linux/amd64 tarball for `deploy/vm/install-website.sh`).
- Brand text lives in `web/lib/site.ts`; showcase screens and captions in `web/lib/screens.json`.
- Landing motion: GSAP + ScrollTrigger (`web/components/landing/motion.ts`), three.js/R3F hero (`hero-scene.tsx`, lazy-loaded). Every animation sits behind `gsap.matchMedia` so reduced-motion and the static HTML show the full content; pinned sections only on `DESKTOP_MOTION`.
- No external fetches from the browser: CSP is `connect-src 'self'`, so avoid drei helpers that load CDN assets (`Text`, `Environment`, `useGLTF` from URLs).
- Screenshots are real stock_home screens on the simulator: start the sim stack + stock_home, then `make screenshots HOME_URL=http://localhost:8081`.
- Docs pages come from `content/docs/manifest.json` (repo markdown is rendered with JS blocked; links to unpublished repo files become plain text). Blog posts are `content/blog/*.mdx`.
- Demo proxy (`internal/demo`) allows GET/HEAD only, plus Grafana's data-query POSTs; keep Grafana admin/login paths blocked.
