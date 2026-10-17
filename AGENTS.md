# AGENTS.md

Vukem Rifa: online raffles with PIX (AbacatePay) for a Brazilian audience. Go backend on the [vukem](https://vukem.vercel.app) framework, Inertia v3 + Vue 3 + TypeScript + Tailwind v4 frontend, PostgreSQL only.

## Start here

1. Read [PLAN.md](PLAN.md): briefing, decision log, architecture, then the first unchecked task of the current phase.
2. Read the `.claude/rules/*` that match the files you touch.
3. Framework problems: follow `.claude/skills/vukem-feedback` (log to `../framework-feedback/rifa.md`, local workaround tagged `// workaround(vukem vX.Y.Z): …`). Never edit `../vukem`.

## Commands

| Task | Command |
| --- | --- |
| Stack | `docker compose up -d --build` (app :3000, postgres :5432, mailpit :8025) |
| Dev | `docker compose up -d postgres mailpit`, `go run ./cmd/app migrate`, `bun run dev`, `go run ./cmd/app` |
| Go tests | `go test -race ./...`; integration `TEST_DATABASE_URL=postgres://rifa:rifa@localhost:5432/rifa?sslmode=disable go test -tags=integration -race ./...` |
| Go lint | `golangci-lint fmt` then `golangci-lint run ./...` |
| Frontend | `bun run lint`, `bun run typecheck`, `bun run build` |
| E2E | `go run ./cmd/app db:seed` once (demo drawn raffle `/r/demodraw01` for `e2e/draw.spec.ts`), then `bun run test:e2e` (stack running, `BASE_URL` default `http://localhost:3000`) |
| API types | `UPDATE_GOLDEN=1 go test ./tests -run TestCodegen_TypesUpToDate` regenerates `resources/js/generated/api.ts` (OpenAPI → TS + zod); the same test fails in CI when it is stale |
| Golden files | `UPDATE_GOLDEN=1 go test ./tests -run TestExport` rewrites `tests/testdata/*.golden` |
| Accessibility | `bunx playwright test e2e/a11y.spec.ts` (axe; serious/critical violations fail; part of `bun run test:e2e`). The admin payouts scan needs `E2E_ADMIN_EMAIL`/`E2E_ADMIN_PASSWORD` set for both `db:seed` and Playwright (CI does), otherwise it is skipped |
| Security suite | `go test ./tests -run 'Security\|OWASP'` + `TestPostgres_OWASP_*` (integration) + `bunx playwright test e2e/security.spec.ts`; category map in `docs/owasp-matrix.md` (add new abuse tests there) |
| Docs | `bun run lint:md` |
| AbacatePay sandbox | `ABACATEPAY_API_KEY=abc_dev_… go test -tags=abacatepay -p 1 -run AbacatePaySandbox ./...` (real dev mode, opt-in) |
| Admin | `go run ./cmd/app make:admin <email>` grants the payouts page (`/admin/payouts`) |
| CI status | `gh run list --limit 5`, `gh run view <id> --log-failed` |

## Rules

- Test first; done = `→ verify:` passes + CI green + `CHANGELOG.md` `[Unreleased]` updated + PLAN.md box ticked.
- Validate every input/output with vukem `z` (Go) and zod (forms). Money as integers (see `.claude/rules/money-and-payments.md`).
- Owner data always scoped by `user_id`; buyers never see other buyers' data; public pages show no personal data.
- Conventional Commits in English; SemVer; Keep a Changelog. Bun only for JS; exact stable versions; pinned Docker tags.
- Framework pin: bump vukem only when the coordinator announces a release (flow in PLAN.md §9).
- Payments: `ABACATEPAY_FAKE=true` (compose, e2e) uses an in-memory gateway plus `/dev/*` helpers and is refused in production; without `ABACATEPAY_API_KEY` the app boots and confirming a link answers 503. The webhook requires `?webhookSecret=` **and** a valid `X-Webhook-Signature`.
- No git worktrees without explicit owner confirmation.
- `.claude/settings.local.json` (`disableAllHooks: true`) belongs to the owner: git-ignored, never commit or edit it.

## Where things live

| Path | What |
| --- | --- |
| `bootstrap/` | explicit app wiring (Postgres driver, i18n, Inertia, providers) |
| `routes/` | HTTP routes (`web.go`), schedule |
| `app/` | controllers, jobs, listeners, mail, domain logic |
| `database/migrations/` | schema migrations (registry in `registry.go`) |
| `resources/js/` | Vue pages, layouts, components, composables |
| `resources/lang/` | `pt-BR.json`, `en.json` shared by Go and Vue |
| `tests/` | Go tests; `tests/pgtest` gives each integration test its own database |
| `docs/` | specs such as `draw-algorithm.md` (how a published draw is recomputed) |
| `e2e/` | Playwright specs |
| `.claude/skills/`, `.claude/rules/` | agent skills and path-scoped rules |
