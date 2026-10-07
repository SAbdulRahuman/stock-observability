# stock_home — agent guide

Go 1.25 server (`main.go`, `internal/`) + Next.js 16 static export (`web/`) embedded via `ui/`. Module `github.com/seenimo/stock_home`.

## Layout

- `/api/v1/*` — existing NSE dashboard (Prometheus + Alertmanager). Do not change its contract.
- `/api/v2/*` — OpenBB-parity commands. Path = OpenBB command path, e.g. `equity.price.historical` → `GET /api/v2/equity/price/historical`. Provider-specific routers: `/api/v2/<provider>/<command>`.
- `internal/platform/` — command registry, OBBject envelope, `/api/v2` router, errors.
- `internal/provider/` — `Fetcher` interface, provider registry, shared HTTP client, credentials lookup. One sub-package per upstream: `internal/provider/<name>/` (`sec`, `nasdaq`, `cboe`, `fred`, …). Fixtures in `internal/provider/<name>/testdata/`.
- `internal/provider/providertest/` — fixture-replay httptest helpers.
- `internal/models/` — standard models (query params + data structs) mirroring `openbb_core/provider/standard_models/*.py`.
- `internal/analytics/{technical,quant,econometrics}/` — pure functions on slices; no I/O.
- `internal/store/` — SQLite (`modernc.org/sqlite`, pure Go, `CGO_ENABLED=0` must keep working).
- `web/app/<extension>/.../page.tsx` — one page per command group; `web/components/command/` shared `CommandView`; `web/lib/commands.ts` client.

## Conventions

- JSON field and query-param names on `/api/v2` use OpenBB snake_case names (`start_date`, `symbol`, `provider`), not the camelCase of `/api/v1`.
- Response envelope: `{"results": [...], "provider": "...", "warnings": [...], "chart": null, "extra": {...}}`.
- Every command declares: path, description, standard model, providers (first = default), params (name, type, required, default, choices, description). Metadata drives UI forms, export, `widgets.json`, MCP tools and the CLI — never hand-duplicate it.
- Static export forbids dynamic route segments: pass inputs as query params (`/equity/price/historical?symbol=AAPL`), read with `useSearchParams` inside `<Suspense>`.
- Upstream HTTP only through the shared client (User-Agent, timeout, retry, rate limit). SEC requires a contact User-Agent (`-sec.user-agent`).
- Credentials: env `STOCK_HOME_<PROVIDER>_<KEY>` > flag > SQLite settings. Never log, echo, or commit them. Missing credential → 401 with `{"error":"missing credential: fred.api_key"}`.
- Never fetch user-supplied URLs (SSRF). Only fixed upstream hosts or registry entries.
- Comments: one short line, only for what code cannot say.

## Testing

- `make test` — `go vet`, `go test -race`, `tsc --noEmit`, `vitest`. `make build` — web export + static Go binary. Both must pass before any commit.
- Provider tests replay recorded upstream responses from `testdata/` (strip keys/tokens before saving). No network in default tests.
- Live smoke tests: skip unless `STOCK_HOME_LIVE=1`.
- Analytics tests compare with golden outputs (`testdata/golden/*.json`, tolerance 1e-6) or textbook reference values.
- Each new page gets a vitest render test with a mocked fetch.

## Git

- Commit only `stock_home/` and `.github/` paths: `git add -- stock_home .github && git commit -m "feat(stock_home): F### <command paths>" -- stock_home .github`. The root index may hold unrelated staged work (e.g. `OpenBB/` submodules) — never include it.
- Never push, never `--no-verify`, never amend existing commits, never `git reset --hard` / `git checkout -- .`.
