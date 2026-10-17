# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Payment links: owners fill buyer name, email and BR phone (normalized to E.164) per number; a 256-bit link (stored as SHA-256) reserves the number for 10 minutes, shown once with copy, WhatsApp share and countdown; owner cancel; 30/min per owner. Expiry by a delayed job plus an every-minute sweeper; concurrent reservations of one number yield exactly one link.
- Buyer checkout `/pay/{token}`: masked buyer data, countdown, PIX QR modal with copy-paste code, status polling, and the expired-link page ("Link expirado. Peça ao dono da rifa…", HTTP 410).
- AbacatePay v2 PIX client (create, check, simulate; z-validated, timeouts) and a fake gateway for local/CI that production refuses.
- Verified AbacatePay webhook (`webhookSecret` + HMAC, dedupe, `payment_events`): number paid, raffle locked, 90/10 ledger, `refund_due` for late or mismatched payments, queued pt-BR/en emails to buyer and owner, sold-out draw.
- Append-only ledger (Postgres trigger), owner balance on the dashboard, admin payouts page and `make:admin` command.
- Auditable draw v1 (crypto/rand seed, HMAC-SHA256 rejection sampling, participants hash) at the deadline or when sold out, exactly once; public result with "how to verify" and in-browser check; `docs/draw-algorithm.md`.
- Raffle audit export (JSON/CSV with formula-injection guard), API keys page (token shown once, read/write abilities, owner-checked revoke) and REST API `/api/v1` (bearer keys, problem+json, pagination, per-key rate limits, idempotent payment links scoped by key) with OpenAPI at `/api/openapi.json` and Scalar at `/api/docs`; generated TS types and zod schemas.
- Background: `serve` runs the queue worker and scheduler in-process (session/cache prune, reservation expiry, due draws).
- Security tests for numbers, payments and API (IDOR, CSRF, webhook forgery/replay, concurrent deliveries and payouts, rate limits, CSV injection).
- Foundation: Vukem v0.21.1 app scaffolded with Inertia v3 + Vue 3 + TypeScript + Tailwind CSS v4, switched to PostgreSQL (`drivers/postgres` v0.3.0).
- Home page with dark/light theme toggle and PT-BR/EN locale switcher; shared JSON catalogs in `resources/lang` for Go and Vue.
- Health endpoint `/up` and Go tests (unit + Postgres integration with one database per test).
- Docker multi-stage image and Compose stack (app, PostgreSQL 18.6, Mailpit).
- Tooling: Biome, golangci-lint v2, commitlint, markdownlint-cli2, lefthook hooks, `.editorconfig`.
- CI (lint, unit + race, integration, govulncheck, frontend checks, Playwright e2e, smoke, ZAP baseline, nuclei), release workflow and Dependabot.
- Auth from `vukem install:auth --social google`: registration, login with throttling, password reset, email verification and Google sign-in; the dashboard requires a verified email. Auth pages, flashes and errors are translated (pt-BR/en); users have an `is_admin` flag.
- E2E auth flow: register, verify through Mailpit, dashboard, logout and login.
- Mailtrap Email Sandbox test client (`tests/mailtrap`) and an opt-in real-delivery test (`-tags=mailtrap`, CI job when secrets exist).
- Localized validation messages and auth emails (pt-BR/en); auth abuse tests (cross-site POST, forgot-password rate limit, session rotation).
- Raffles: create (numbers 1..N pre-created), dashboard list, owner page with number grid, edit with lock after the first paid number, delete or archive, public page `/r/{slug}` without personal data; Postgres CHECK constraints; zod client validation.
- Project docs: PLAN.md, README, AGENTS.md, CLAUDE.md, CONTRIBUTING, SECURITY, MIT license; project-specific `.claude` skills and rules.

### Changed

- Bumped vukem to v0.23.0 and drivers/postgres to v0.4.0: rate limits go through `app.RateLimiters()` with `RATE_LIMIT_STORE=cache` (counters shared in the database cache), `tests/pgtest` delegates to the driver's `pgtest.Database`, `key:generate` runs from the app binary.
- The Mailtrap sandbox is never cleaned by tests (shared free sandbox); CI runs the Mailtrap and AbacatePay sandbox jobs when their secrets exist.
- Bumped vukem to v0.22.0: `auth.Verified` replaces the local redirect middleware, `inertia.I18n` shares the catalogs (the Vue composable no longer bundles the JSON), tests use `Inertia("")`.
- `serve` runs the database queue worker and the scheduler in-process (`BACKGROUND_IN_SERVE`).

### Fixed

- Raffle price no longer shows "R$" twice; closed raffles cannot be edited.
