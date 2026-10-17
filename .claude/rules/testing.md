---
paths:
  - "tests/**"
  - "**/*_test.go"
  - "e2e/**"
---

# Testing rules

- Test first (see `skills/tdd`). Table-driven Go tests, `t.Parallel()`, stdlib assertions + `vukemtest`.
- Unit/HTTP tests: `vukemtest.New(t, bootstrap.New)` (in-memory SQLite, dialect-neutral code only).
- Integration tests: `//go:build integration`, Postgres via `tests/pgtest.NewDatabase(t)` (one database per test); run `go test -tags=integration -race ./...` with `docker compose up -d postgres`.
- Concurrency tests (reserve the same number in parallel) run on Postgres with `-race`.
- No network in unit tests: fake AbacatePay with `httptest.Server`; Mailtrap Sandbox API only in e2e/integration tagged `mailtrap`, skipped when `MAILTRAP_API_TOKEN` is unset.
- E2E: Playwright against a running stack (`docker compose up -d --build`, then `bun run test:e2e`).
