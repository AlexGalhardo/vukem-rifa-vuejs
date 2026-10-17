# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Public checkout (owner request): visitors pick a free number on `/r/{slug}`, give name, email, phone and LGPD consent, and pay on the same PIX page; rate limits, honeypot, a cap of 5 live links per email; the owner sees the origin; signed-in (or verified-email) buyers get `/account/purchases` with paid and unpaid numbers. The navbar collapses into a menu on phones.
- Owner requests (2026-10-17): the navbar uses the vukem docs container (sticky, 90rem, same gutters); the theme toggle is a light/dark/system radiogroup with sun, moon and monitor icons; the buyer page shows the PIX QR code inline (no modal) with a "Copiar código" button (clipboard.js 2.0.11) and a toast; every live payment link stays listed for the owner for its 10 minutes (token sealed with APP_KEY in `reservations.token_sealed`).
- Security tests: argon2id password hashes, production HSTS and http `APP_URL` refusal, failed-login and payout security events without secrets, enumeration-safe sign-up (`docs/owasp-matrix.md`).
- Consolidated OWASP Top 10 suite (`docs/owasp-matrix.md`): new tests for mass assignment, session cookie flags, injection payloads, security headers, security-event logging without secrets, gateway URL hardening, logout killing copied session cookies and reset-mail timing; `e2e/security.spec.ts` (no CSP violations, cancelled link reuse).
- QA checklist (`docs/qa-checklist.md`, run 1) and tuned ZAP baseline: security headers, cookie flags, version leaks and error disclosure now fail CI.
- Admin payouts page under the axe suite, with an env-gated e2e admin seeder (`E2E_ADMIN_EMAIL`/`E2E_ADMIN_PASSWORD`, CI only).
- Accessibility e2e (`e2e/a11y.spec.ts`, `@axe-core/playwright` 4.14.0): every page and state (guest, owner, link form/result, buyer card, checkout, PIX modal, confirmation, expired and cancelled links, public and drawn raffles) in light/dark, pt-BR/en, desktop/mobile; fails on serious or critical violations.
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

- Bumped vukem to v0.26.0: API keys are soft revoked (`access_tokens.revoked_at`, revoked keys hidden from the settings page), payouts and edit-cancelled links are audit events (`web.AuditEvent`), the sign-up notice path never answers differently when the cache fails, `MAIL_MAILER=mailtrap` (Mailtrap Email API) is documented for production; new tests for APP_KEY rotation and Mailtrap HTTP delivery.
- Editing a raffle cancels its live payment links in the same transaction (numbers released, audit event `raffle.links_cancelled`, flash with the count); paid numbers still lock price, size and duration.
- Production moved to <https://rifa.vukem.dev> (docs: <https://vukem.dev>); every repository reference to the Railway and `rifa.vukem.com` URLs now points there.
- Production on Railway sends HSTS (`HSTS_FORCE`) and sees real client IPs (`TRUSTED_PROXIES=100.64.0.0/24`, `CLIENT_IP_HEADER=X-Real-IP`), so per-IP rate limits no longer share one bucket.
- Bumped vukem to v0.25.0: the database queue worker runs in `serve` by vukem (`QUEUE_IN_SERVE`; `bootstrap.Background` removed), the queue follows `app.Now`, buyer error pages use `inertia.RenderStatus`, the REST API registers with `api.Register` + `ValidateResponse` and authenticates with `auth.Required` (APIGuard, 401 problem+json), API tokens use the app clock, generated types are zod 4, CSV downloads stream through vukem `export` (stronger formula guard; plain numbers such as E.164 phones and negative amounts are no longer prefixed).
- Sign-up no longer reveals existing accounts: every attempt answers "check your email" (`/register/check-email`) without logging in; a new address gets the verification link, an existing one an "account exists" notice (at most one per hour) and nothing is created. The verification link now works signed out and leads to the login page. Verification and reset mails are queued.
- Production refuses to boot with a non-HTTPS `APP_URL`.
- Production refuses a non-HTTPS `ABACATEPAY_BASE_URL` (the API key never travels in clear text).
- Bumped vukem to v0.24.0: the scheduler runs in `serve` by vukem itself (`SCHEDULE_IN_SERVE`; `bootstrap.Background` keeps only the database queue worker), job handlers are registered with `WithJobsFunc`, `DispatchTx` replaces the after-commit fallbacks (delayed jobs on the sync driver fall back to the sweepers), the buyer phone uses `z.Preprocess`, payment locks use `orm.LockForUpdate`.
- Bumped vukem to v0.23.0 and drivers/postgres to v0.4.0: rate limits go through `app.RateLimiters()` with `RATE_LIMIT_STORE=cache` (counters shared in the database cache), `tests/pgtest` delegates to the driver's `pgtest.Database`, `key:generate` runs from the app binary.
- The Mailtrap sandbox is never cleaned by tests (shared free sandbox); CI runs the Mailtrap and AbacatePay sandbox jobs when their secrets exist.
- Bumped vukem to v0.22.0: `auth.Verified` replaces the local redirect middleware, `inertia.I18n` shares the catalogs (the Vue composable no longer bundles the JSON), tests use `Inertia("")`.
- `serve` runs the database queue worker and the scheduler in-process (`BACKGROUND_IN_SERVE`).

### Fixed

- Password reset mail is queued instead of sent during the request, so known and unknown emails answer in the same time (no enumeration by timing).

- A password reset now signs the account out of every other browser session (database sessions).
- The password-confirmation mismatch message is translated (pt-BR/en).
- Accessibility: brand buttons and secondary text meet WCAG AA contrast in light and dark, the theme toggle's accessible name includes its visible text, the generated payment link field has a label.
- Raffle price no longer shows "R$" twice; closed raffles cannot be edited.
