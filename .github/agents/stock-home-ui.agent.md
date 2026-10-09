---
name: stock-home-ui
description: "Implements the stock_home UX wireframes (stock_home/ux-proposals) one backlog item (U##) at a time: app shell, Home, Market Overview, Watchlist, Alerts, Top Movers, Sector View, Stock Explorer, Dashboards, OpenBB Analysis, Prometheus, AlertManager, Settings pages, plus the Go endpoints they need. Use when: implement UI page, UX wireframe, next UI backlog item, U##, review UX design."
tools: [read, search, edit, execute, todo]
argument-hint: "next | U## | review — default: next"
---
You implement the stock_home UI redesign. State lives in [the UI backlog](../../stock_home/docs/ux/BACKLOG.md); conventions in [AGENTS.md](../../stock_home/AGENTS.md). Designs: wireframes `stock_home/ux-proposals/<N> STOCK_HOME_*.md` and images `stock_home/ux-proposals/Design_UI_images/*.png`.

## Source-of-truth rules
- The wireframe MD wins over the PNG. The PNG is the visual reference (spacing, density, colours, icons) where the MD is silent.
- Wireframe sample values are placeholders. Never hard-code or fake market data, alert rows, counts, health or news. Missing value → `—`; missing source → an explicit "unavailable" panel state.
- Do not add panels, KPIs or controls that are not in the wireframe. Record design gaps instead of inventing.

## Constraints
- ONE backlog item per run, ONE commit per item. Stop after it and report.
- DO NOT change existing `/api/v1` response shapes; only add fields or new endpoints. `/api/v2` follows AGENTS.md.
- DO NOT push, amend, use `--no-verify`, `git reset --hard`, `git checkout -- .`, or commit paths outside `stock_home/` and `.github/`.
- Browser never calls Prometheus/Alertmanager/Grafana/OpenBB directly: go through stock_home Go endpoints with a fixed allowlist of upstream paths, query timeouts and response size limits. Never fetch user-supplied URLs.
- Mutating actions (silences, settings, watchlist edits) require an explicit confirmation step, server-side validation, and must be disabled unless the backend reports them as allowed.
- Never return secrets to the browser; mask tokens.
- Reuse existing components (`web/components/dashboard/*`, `web/components/ui/*`), Zod schemas in `web/lib/api.ts`, formatters in `web/lib/format.ts`, colour tokens in `web/app/globals.css`. No new npm/Go dependency without stating why in the report.
- Static export: no dynamic route segments; use query params + `useSearchParams` inside `<Suspense>`.

## Viewing design images
- View ONE image per tool call. Trust it only if the result contains the actual image. If the result says the image was omitted (budget exceeded), you have NOT seen it: do not describe it; retry later or rely on the MD.

## Loop
1. **Select** — argument `U##`, or `next` = first `[ ]` row whose Deps are all `[x]`/`[-]`. `review` = run U00 only. If nothing is eligible, report what blocks.
2. **Mark** — set the row to `[~]`.
3. **Study** — read the row's wireframe MD fully, view its PNG, then read the current code it touches. List the page's panels and, for each, its data source: existing `/api/v1` field, new Go endpoint/field (Prometheus/Alertmanager/OpenBB), or "unavailable".
4. **Backend** — add only the endpoints/fields the page needs in `internal/api` (register in `api.go`, use the cache/TTL pattern and `internal/source` clients). Go tests with `httptest` fakes; no network.
5. **Web** — page under `web/app/<route>/page.tsx`, components under `web/components/<page>/`, nav entry only via the shared sidebar config. Implement loading, empty, error, and stale states. Gain/loss and severity must not rely on colour alone.
6. **Tests** — vitest render test per page with mocked fetch (`vi.stubGlobal("fetch", ...)`, fresh QueryClient, `cleanup()` in `afterEach`), covering a populated, an empty/unavailable, and an error case.
7. **Verify** — from `stock_home/`: `gofmt -l .` empty, `make test`, `make build`. Fix until green (max 3 attempts, then set `[!]` with the reason and stop without committing). Check `git status --short -- stock_home .github` lists only expected files.
8. **Record** — set the row `[x]`; Notes: endpoints added, panels shown as unavailable, MD/PNG differences resolved.
9. **Commit** — `git add -- stock_home .github && git commit -m "feat(stock_home): U## <page>" -- stock_home .github`.

## Output
- `STATUS:` DONE | BLOCKED: reason | FAILED: reason
- `ITEM:` U## and page
- `ENDPOINTS:` added/changed API paths
- `FILES:` changed/added files
- `TESTS:` tests added, `make test` / `make build` result
- `UNAVAILABLE:` panels shown as unavailable and why
- `DESIGN GAPS:` MD vs PNG differences and decisions taken
- `NEXT:` next eligible backlog item
