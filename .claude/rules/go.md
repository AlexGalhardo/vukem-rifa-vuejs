---
paths:
  - "**/*.go"
---

# Go rules (vukem app)

- Format and lint: `golangci-lint fmt` then `golangci-lint run ./...` (config `.golangci.yml`); tabs.
- `context.Context` first in every function doing I/O; never stored in structs.
- Wrap errors with context (`fmt.Errorf("reserve number %d: %w", n, err)`); sentinel `ErrX` for conditions.
- No globals, no `init()` registration: everything is wired in `bootstrap/` (vukem style).
- Use vukem packages before writing your own: `z` (validation), `orm`, `schema`, `queue`, `schedule`, `mail`, `auth`, `social`, `csrf`, `signed`, `webhook`, `ratelimit`, `i18n`, `openapi`, `apidocs`.
- Validate every input and every external response (AbacatePay, Mailtrap) with `z` schemas; never trust client amounts.
- SQL: Postgres only; parameterized queries; state changes on numbers use one transaction with `SELECT … FOR UPDATE` or a guarded `UPDATE … WHERE status = …` and check rows affected.
- Time comes from `app.Now()` (fake clock in tests), never `time.Now()` in domain code.
- Randomness from `crypto/rand` only.
- Comment only the *why*; known framework gaps use `// workaround(vukem vX.Y.Z): <feedback title>` so a bump can find and remove them.
