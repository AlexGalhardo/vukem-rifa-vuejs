# PLAN — Vukem Rifa

The onboarding file for every session and agent. Read it top to bottom once, then pick the first unchecked task of the current phase.

- Repository: <https://github.com/AlexGalhardo/vukem-rifa-vuejs> (public). Production: <https://rifa.vukem.com> (Railway).
- Current phase: **Phase 8 — Security hardening and QA** (Phases 0–7 done; Phase 1 has owner-blocked items).

## 1. Briefing

Online raffles (rifas) for a Brazilian audience. A raffle owner signs up, creates a raffle with numbers `1..N`, and for each buyer generates a unique payment link that reserves a number for 10 minutes. The buyer opens the link, confirms their data and pays by PIX (AbacatePay). A webhook confirms payment and the number becomes paid. At the end date (or when every number is paid) the app draws a winner among paid numbers with `crypto/rand` and publishes an auditable result.

The project also exists to **battle-test the vukem framework** before its 1.0: every framework bug, missing feature, awkward API, security issue or docs problem goes to `../framework-feedback/rifa.md` (skill `.claude/skills/vukem-feedback`), with a local workaround so work continues.

## 2. Decision log

| # | Decision | Why / notes |
| --- | --- | --- |
| D1 | Backend: vukem pinned to a released tag in `go.mod` (no `replace`). Started at **v0.21.1**; bumped after each vukem release the coordinator announces (flow in §9). | Battle-test real releases. |
| D2 | Frontend: Inertia **v3 (3.9.1, stable since 2026)** + Vue 3.5 + TypeScript + Tailwind v4, built by Vite 8, scaffolded with `vukem new --inertia vue`. | Inertia v3 has stable releases (3.0.0 → 3.9.1), so v3 is used. |
| D3 | TypeScript **6.0.3**, not 7.0.2. | TS 7 is the native (Go) compiler without the JS language-service API `vue-tsc` 3.3 needs; 6.0.3 is the latest stable 6.x. Revisit when vue-tsc supports TS 7. |
| D4 | PostgreSQL only (18.6). Sessions (`SESSION_DRIVER=database`), queue (`QUEUE_CONNECTION=database`, `FOR UPDATE SKIP LOCKED`), cache + scheduler locks (`CACHE_STORE=database`) all in Postgres. | Owner requirement; one managed service on Railway. |
| D5 | Rate limiting through `app.RateLimiters()` with `RATE_LIMIT_STORE=cache` over `CACHE_STORE=database` (vukem v0.23.0): counters shared by every Railway instance; `memory` stays the default when unset. | Replaces the per-process GCRA workaround (audit F1). |
| D6 | Email: SMTP transport only (`MAIL_MAILER=smtp`, `SMTP_*`). Local offline: **Mailpit** (compose). Dev/staging/e2e: **Mailtrap Email Sandbox** (`sandbox.smtp.mailtrap.io:2525`, STARTTLS). Tests assert delivered email through the Mailtrap Sandbox API. Production sender switchable by env (Mailtrap Sending `live.smtp.mailtrap.io` or any SMTP). Resend dropped. | Owner: Resend needs DNS first; Mailtrap Sandbox works today and is automatable. |
| D7 | Payments: AbacatePay API v2, PIX via `POST /v2/transparents/create`; dev key (`abc_dev_…`) = sandbox, same base URL. No cancel endpoint: the PIX `expiresIn` equals the remaining reservation time, and expiry is enforced locally. | From docs.abacatepay.com (2026-10-17). |
| D8 | Platform receives 100%; ledger records 90% owner share + 10% platform fee per paid number; payouts recorded manually by an admin (`users.is_admin`). | Owner decision. |
| D9 | Buyers have no account. Payment links are random tokens (≥128 bits), stored as SHA-256, valid only while the reservation lives. | Owner decision + IDOR safety. |
| D10 | i18n: one JSON catalog per locale in `resources/lang/{pt-BR,en}.json`, loaded by vukem `i18n` (Go: emails, validation) and shared with Vue through `inertia.I18n(prefixes…)` (once prop per locale; `mail.*`/`validation.*` stay server-side), read by the `useI18n` composable. Locale from `?lang` → `lang` cookie → `Accept-Language` → `APP_LOCALE=pt-BR`. | No extra dependency (no vue-i18n). |
| D11 | Theme: class-based dark mode (`html.dark`), pre-paint inline script with the CSP nonce, `localStorage.theme`, OS preference as default. | No flash of wrong theme; CSP-safe. |
| D12 | Draw: `crypto/rand` 32-byte seed at draw time, winner by rejection sampling over the sorted paid numbers; store seed, timestamp, participant count, SHA-256 of the participant list, algorithm version. Public result page shows all of it so anyone can recompute. | Auditable. Commit-reveal (publish `sha256(seed)` at creation) is a possible later enhancement. |
| D13 | Locks: price and number count become immutable after the first paid number; title/description stay editable. Raffles with paid numbers are archived, never deleted. | Owner rules. |
| D14 | Number rows are pre-created (`1..N`, N ≤ 1000) when a raffle is created. | Row-level locking per number, simple grid queries. |
| D15 | Background work runs in the web process: vukem v0.24.0 runs the scheduler in `serve` (`SCHEDULE_IN_SERVE`, default true; `session:prune`, `cache:prune`, `reservations:expire`, `draws:due`) and `bootstrap.Background()` adds the database queue worker (`BACKGROUND_IN_SERVE=false` when `queue:work` runs separately). | One Railway instance, low cost. |
| D16 | JS tooling via Bun 1.4.3; Biome 2.5 for lint/format; Playwright 1.64 e2e; lefthook git hooks; commitlint; markdownlint-cli2. | Owner standards. |
| D18 | Duration is locked together with price and number count after the first paid number; before that, shrinking the count is refused if a removed number is reserved or paid. | Changing the deadline after sales would be unfair to buyers. |
| D19 | Tests: unit/HTTP tests run on SQLite via `vukemtest`, so domain SQL stays dialect-neutral (recursive CTE for numbers); Postgres-only parts (CHECKs, `FOR UPDATE`, partial indexes) are covered by `-tags=integration`. | vukem has no Postgres test helper yet. |
| D20 | AbacatePay webhook authenticity = `webhookSecret` query parameter (constant-time); `X-Webhook-Signature` (HMAC with AbacatePay's published public key) only proves body integrity. Registered URL: `https://vukem-rifa.up.railway.app/webhooks/abacatepay?webhookSecret=…`. | docs.abacatepay.com/pages/webhooks (security). |
| D21 | Archived raffles are never drawn; a draw expires every live link in its transaction; payments on non-active raffles become `refund_due`. | Raffles/payments stream decisions. |
| D17 | Local `.claude/settings.local.json` (`disableAllHooks: true`) is owner-managed and git-ignored. | Owner. |

## 3. Versions (verified 2026-10-17, exact pins)

| Tool | Version | Where |
| --- | --- | --- |
| Go | 1.27.2 | `go.mod`, Docker `golang:1.27.2-alpine3.24` |
| vukem | v0.24.0 (started on v0.21.1) | `go.mod` |
| vukem drivers/postgres | v0.4.0 (pgx v5.11.0, `pgtest.Database`) | `go.mod` |
| golangci-lint | v2.15.0 (CI), action v9.3.0 | `.golangci.yml` |
| govulncheck | v1.9.0 (`go tool`) | `go.mod` tool directive |
| Bun | 1.4.3 | CI, Docker `oven/bun:1.4.3-alpine` |
| @inertiajs/vue3, @inertiajs/vite | 3.9.1 | `package.json` |
| Vue | 3.5.44 | |
| Vite / @vitejs/plugin-vue | 8.3.4 / 6.0.10 | |
| Tailwind CSS / @tailwindcss/vite | 4.3.3 | |
| TypeScript / vue-tsc | 6.0.3 / 3.3.13 | see D3 |
| Biome | 2.5.15 | `biome.json` |
| Playwright | 1.64.1 | |
| zod | 4.6.7 | e2e helpers now; forms next |
| commitlint | 21.2.3 | |
| markdownlint-cli2 | 0.23.3 | |
| lefthook | 2.2.2 | |
| PostgreSQL image | `postgres:18.6-alpine3.24` | compose, CI service |
| Mailpit image | `axllent/mailpit:v1.31.6` | compose |
| Runtime image | `alpine:3.24.2` | Dockerfile |
| GitHub Actions | checkout v7.0.1, setup-go v7.0.0, setup-bun v2.2.0, upload-artifact v7.0.2, zaproxy/action-baseline v0.15.0, nuclei-action v3.1.0, buildx v4.4.1, login v4.6.0, build-push v7.4.0, gh-release v3.0.3 | `.github/workflows` |

## 4. Architecture

### 4.1 Folder layout

```text
cmd/app/main.go            entrypoint (serve, migrate, queue:work, schedule:work, …)
bootstrap/                 explicit wiring: providers, commands, options (Postgres driver, i18n, Inertia)
routes/                    web.go (browser), api.go (REST v1), webhooks.go, schedule.go
app/controllers/           Inertia + JSON handlers (thin)
app/domain/                pure logic: raffle rules, number state machine, money split, draw algorithm
app/repo/                  SQL per aggregate (raffles, numbers, reservations, ledger, draws, api keys)
app/payments/abacatepay/   HTTP client (net/http, z-validated responses), webhook verifier
app/jobs/                  queue jobs: ExpireReservation, RunDraw, SendPaymentEmails
app/mail/                  mailables + views in resources/views/mail
app/requests/              z schemas for every input
database/migrations/       schema (raw SQL where CHECK/partial indexes are needed)
resources/js/              Vue pages, layouts, components, composables (useI18n, useTheme)
resources/lang/            pt-BR.json, en.json (shared Go + Vue)
resources/views/           app.html root view, mail templates
tests/                     Go HTTP/unit tests; *_integration via build tag; pgtest helper; security abuse suite
e2e/                       Playwright specs (smoke, flows, security)
docs/                      QA checklist, ADRs when needed
```

### 4.2 Data model (PostgreSQL)

All timestamps `timestamptz`; money in integer **centavos** except `raffles.price_reais` (whole reais, owner input).

| Table | Columns and constraints |
| --- | --- |
| `users` (install:auth) | `id`, `name`, `email` unique (lowercased), `password` (argon2id, nullable for Google-only), `email_verified_at`, `is_admin bool default false`, timestamps |
| auth tables (install:auth) | password reset tokens, email verification, `social_accounts` (`provider`, `provider_id` unique together, `user_id`) |
| `raffles` | `id`, `user_id` FK → users, `slug` unique (random, URL-safe), `title` 3..120, `description` ≤ 5000, `number_count int CHECK 10..1000`, `price_reais int CHECK 1..100000`, `duration_days int CHECK 7..60`, `starts_at`, `ends_at` (= starts_at + duration), `status text CHECK IN ('active','drawn','closed','archived')`, `locked_at` (first paid number), `archived_at`, timestamps |
| `raffle_numbers` | `id`, `raffle_id` FK ON DELETE CASCADE, `number int CHECK number >= 1`, `status text CHECK IN ('available','reserved','paid')`, `reservation_id` FK nullable, `paid_at`, `UNIQUE (raffle_id, number)`, index `(raffle_id, status)` |
| `reservations` (payment links) | `id`, `raffle_id`, `number_id`, `buyer_name` 2..100, `buyer_email`, `buyer_phone` (E.164 `+55…`), `token_hash bytea unique`, `status text CHECK IN ('pending','awaiting_payment','paid','expired','cancelled')`, `expires_at`, `amount_cents int CHECK > 0`, `charge_id` unique nullable, `br_code`, `br_code_base64`, `paid_at`, timestamps; partial unique index `ON (number_id) WHERE status IN ('pending','awaiting_payment')` (one live link per number) |
| `payment_events` | `id`, `provider`, `event_id` unique, `event_type`, `payload jsonb`, `received_at`, `processed_at` — idempotency for webhooks |
| `ledger_entries` (append-only) | `id`, `user_id` (owner), `raffle_id`, `reservation_id` nullable, `kind CHECK IN ('sale_owner_share','platform_fee','payout','refund_due','refund')`, `amount_cents bigint` (signed), `note`, `created_by` nullable, `created_at`; trigger rejects UPDATE/DELETE |
| `payouts` | `id`, `user_id`, `amount_cents CHECK > 0`, `reference` (PIX end-to-end id or note), `recorded_by` admin FK, `created_at` (writes a negative `payout` ledger entry in the same tx) |
| `draws` | `id`, `raffle_id` unique, `drawn_at`, `seed` (hex, 32 bytes), `participants_count`, `participants_sha256`, `winning_number` nullable (null = no paid numbers), `algorithm` (`v1`), `trigger CHECK IN ('deadline','sold_out')` |
| `api_keys` (vukem `auth.Tokens` table) | hashed token, `user_id`, name, abilities (`raffles:read`, `raffles:write`), `last_used_at`, `revoked_at` |
| framework tables | `sessions`, `jobs`, `failed_jobs`, `cache`, `schema_migrations` |

### 4.3 Number state machine

```text
             generate link (owner)                 webhook PAID (verified, idempotent)
 available ─────────────────────────▶ reserved ───────────────────────────────────────▶ paid
     ▲                                   │
     └────── 10 min expiry / owner cancel┘   (reservation → expired | cancelled)
```

- `available → reserved`: one tx — `UPDATE raffle_numbers SET status='reserved' … WHERE id=$1 AND status='available'` (0 rows ⇒ 409), insert reservation (`expires_at = now + 10m`), dispatch `ExpireReservation` with `Delay(10m)` via `DispatchTx`.
- `reserved → paid`: webhook handler, one tx — lock reservation `FOR UPDATE`, require `status IN ('pending','awaiting_payment')` and `now ≤ expires_at + grace(60s)`; mark paid, number paid, set `raffles.locked_at` if null, write ledger rows (owner 90% / fee 10%), dispatch emails via `DispatchTx`. If every number is paid, dispatch `RunDraw{trigger: sold_out}`.
- `reserved → available`: expiry job / scheduler sweeper (every minute, `expires_at < now`) / owner cancel — guarded `WHERE status IN ('pending','awaiting_payment')`.
- Late payment (webhook after expiry): record event + `refund_due` ledger entry and show it in the export; never assign the number.

### 4.4 Money flow

1. Buyer pays `price_reais × 100` centavos by PIX to the **platform** AbacatePay account.
2. On confirmed payment: `platform_fee = amount × 10 / 100` (integer), `sale_owner_share = amount − fee`.
3. Owner balance = Σ ledger (`sale_owner_share` + `payout` (negative) + `refund`…). Shown on the dashboard.
4. An admin pays the owner outside the app and records a payout (amount ≤ balance).

### 4.5 Request flows

- **Owner generates a link:** dashboard → `POST /raffles/{id}/numbers/{n}/link` (CSRF, owner scope, rate limit) → reservation + token → page shows the link `https://rifa.vukem.com/pay/{token}` (copy/share) and a 10-minute countdown.
- **Buyer pays:** `GET /pay/{token}` → shows raffle, number, buyer data (masked email/phone) → `POST /pay/{token}/confirm` → create AbacatePay PIX (`expiresIn` = remaining seconds, `externalId` = reservation id) → modal with QR (`brCodeBase64`) + copy-paste `brCode` → page polls `GET /pay/{token}/status` (JSON) every 3 s → "Pagamento confirmado".
- **Expired link:** any `/pay/{token}` on an expired/cancelled reservation renders "Link expirado. Peça ao dono da rifa para gerar um novo link e te enviar novamente para pagar".
- **Webhook:** `POST /webhooks/abacatepay?webhookSecret=…` → verify secret + `X-Webhook-Signature` HMAC-SHA256 (base64, raw body) → insert `payment_events` (unique `event_id`; duplicate ⇒ 200 no-op) → handle `transparent.completed` → 200 only after commit.
- **Draw:** `RunDraw` job dispatched with `Delay(until ends_at)` at creation plus a scheduler sweep every minute for overdue active raffles; job locks the raffle row, skips if already drawn.

## 5. Pages and routes

| Route | Page / purpose | Access |
| --- | --- | --- |
| `GET /` | Landing (Home) | public |
| `GET /up` | Health (`?checks=1` adds DB) | public |
| `GET /r/{slug}` | Public raffle: title, description, price, deadline, number grid (available/reserved/paid), draw result | public, no personal data |
| `GET /pay/{token}` | Buyer checkout: confirm data | token |
| `POST /pay/{token}/confirm` | Create PIX, return QR + code | token, CSRF, rate limit |
| `GET /pay/{token}/status` | Poll payment status (JSON) | token |
| `POST /webhooks/abacatepay` | Payment webhook | secret + HMAC |
| `/register`, `/login`, `/logout`, `/forgot-password`, `/reset-password/{token}`, `/verify-email…` | Auth (install:auth) | guest / auth |
| `/auth/google/redirect`, `/auth/google/callback` | Google login | guest |
| `GET /dashboard` | Owner raffles + balance | auth + verified |
| `GET /raffles/create`, `POST /raffles` | Create raffle | owner |
| `GET /raffles/{id}` | Manage: grid, per-number buyer form, links, payments | owner |
| `GET /raffles/{id}/edit`, `PUT /raffles/{id}` | Edit (locked fields after first paid) | owner |
| `DELETE /raffles/{id}` | Delete (only with no paid numbers) | owner |
| `POST /raffles/{id}/archive` | Archive | owner |
| `POST /raffles/{id}/numbers/{n}/link` | Generate payment link (reserve) | owner |
| `POST /raffles/{id}/numbers/{n}/cancel` | Cancel live reservation | owner |
| `GET /raffles/{id}/export.json`, `.csv` | Audit export (numbers, buyers, payments, ledger) | owner |
| `GET /settings/api-keys`, `POST`, `DELETE /settings/api-keys/{id}` | API keys | owner |
| `GET /admin/payouts`, `POST /admin/payouts` | Balances + record payouts | admin |
| `GET /api/docs`, `GET /api/openapi.json` | Scalar + OpenAPI | public |

### REST API v1 (`Authorization: Bearer <api key>`)

| Method + path | Ability |
| --- | --- |
| `GET /api/v1/raffles` (paginated) | `raffles:read` |
| `POST /api/v1/raffles` | `raffles:write` |
| `GET /api/v1/raffles/{id}` | `raffles:read` |
| `PATCH /api/v1/raffles/{id}` | `raffles:write` |
| `DELETE /api/v1/raffles/{id}` | `raffles:write` |
| `POST /api/v1/raffles/{id}/archive` | `raffles:write` |
| `GET /api/v1/raffles/{id}/numbers` | `raffles:read` |
| `POST /api/v1/raffles/{id}/numbers/{n}/payment-links` | `raffles:write` (Idempotency-Key, Scope = API key id) |
| `GET /api/v1/raffles/{id}/export?format=json\|csv` | `raffles:read` |
| `GET /api/v1/me/balance` | `raffles:read` |

Errors are `application/problem+json`; every body validated by `z` and documented by `openapi`.

## 6. Engineering rules

- Read `AGENTS.md` and `.claude/rules/*` (Go, frontend, money, security, testing, commits).
- TDD: a task is done when its `→ verify:` passes, CI is green and `CHANGELOG.md` `[Unreleased]` is updated.
- Validate all inputs and external responses with vukem `z` (Go) and zod (forms).
- Conventional Commits, SemVer (`v0.N.0` per phase, `v1.0.0` at launch), Keep a Changelog.
- `.editorconfig`: tabs, size 4 (YAML: 2 spaces).
- Never edit `../vukem`. Framework findings → `../framework-feedback/rifa.md`; workarounds tagged `// workaround(vukem vX.Y.Z): …`.
- Bun only for JS; exact versions, no pre-releases; Docker images pinned, never `latest`.

## 7. OWASP security checklist

- [ ] A01 Broken access control: every owner query scoped by `user_id`; IDOR tests for raffles, numbers, reservations, exports, API keys (revoke owner-checked).
- [x] A02 Security misconfiguration: vukem secure headers + CSP nonce on; production refuses insecure cookies; `/_vukem` devtools off in production.
- [ ] A03 Supply chain: pinned versions, Dependabot, govulncheck + `bun audit` in CI.
- [ ] A04 Cryptographic failures: argon2id passwords; tokens from `crypto/rand`, stored hashed; HTTPS only in production; `APP_KEY` rotation via `APP_PREVIOUS_KEYS`.
- [ ] A05 Injection/XSS: parameterized SQL; Vue escapes by default (no `v-html`); CSV export guards formula injection (`=`,`+`,`-`,`@` prefixed with `'`).
- [x] A06 Insecure design: number state machine guarded in SQL; race test reserving the same number concurrently; expired link reuse test.
- [ ] A07 Auth failures: login/reset/verification rate limits (brute-force test), email verification required for dashboard, session rotation on login.
- [x] A08 Integrity: webhook secret + HMAC verification, replay/duplicate event test, forged signature test.
- [ ] A09 Logging: security events (login failures, webhook rejections, payout records) logged with request id, no PII/secrets.
- [ ] A10 Exceptional conditions: AbacatePay timeouts/5xx handled (retry job, user-facing error), fail closed on verification errors.
- [ ] CSRF on all browser mutations (test), CORS denied by default for `/api`.
- [x] DAST: ZAP baseline + nuclei in CI against the running container.

## 8. Phases

### Phase 0 — Foundation ✅

#### 0.1 Research and plan

- [x] Verify current stable versions (npm registry, go.dev, Docker Hub, GitHub releases) → verify: §3 table filled with exact versions.
- [x] Read AbacatePay `llms.txt` + v2 docs (transparents, webhooks, dev mode) and Mailtrap Sandbox docs → verify: D6/D7 recorded.
- [x] Write PLAN.md → verify: `bun run lint:md` passes.

#### 0.2 App skeleton

- [x] Scaffold with `vukem new rifa --inertia vue` (v0.21.1), drop the posts sample → verify: `go build ./...`.
- [x] Switch to Postgres (`drivers/postgres` v0.3.0, `WithDatabaseDrivers`) → verify: integration test asserts dialect `postgres`.
- [x] Health route `/up` → verify: `TestHealth`, `curl localhost:3000/up` = `{"status":"ok"}`.
- [x] Home page via Inertia with theme toggle + locale switcher + i18n skeleton (`resources/lang`, `useI18n`, `useTheme`) → verify: `TestHome*`, e2e `smoke.spec.ts`.
- [x] Migrations setup (registry, `migrate` on container start) → verify: compose log `Nothing to migrate.`

#### 0.3 Tooling and docs

- [x] `.editorconfig`, Biome, golangci-lint v2, commitlint, markdownlint-cli2, lefthook (pre-commit fmt/lint, commit-msg, pre-push tests) → verify: `bun run lint`, `golangci-lint run`.
- [x] README, AGENTS.md, CLAUDE.md, CONTRIBUTING.md, SECURITY.md, LICENSE (MIT), CHANGELOG.md → verify: `bun run lint:md`.
- [x] `.claude/skills` (Go, TDD, review, frontend, vukem-feedback) and `.claude/rules` (project-specific) → verify: files present.

#### 0.4 Docker and CI/CD

- [x] Multi-stage Dockerfile (bun → go → alpine, non-root, healthcheck) + compose (app, postgres, mailpit) → verify: `docker compose up -d --build` healthy, `curl /up` 200.
- [x] CI: golangci-lint, unit+race, integration (Postgres service), govulncheck, Biome/typecheck/build/markdownlint, commitlint (PRs), e2e Playwright + smoke + ZAP baseline + nuclei; release on tag (GHCR image + GitHub release); Dependabot → verify: `gh run list` green.
- [x] First push to `main` → verify: CI green (run 39034343255: all jobs incl. e2e, ZAP, nuclei).

### Phase 1 — Auth

#### 1.1 Scaffold and Postgres

- [x] Run `vukem install:auth --social google` in a scratch copy, port the generated files, adapt migrations to Postgres → verify: `go test ./...` and integration green.
- [x] Add `users.is_admin` (default false) → verify: `TestRegister_IsNotAdmin` (SQLite + Postgres).
- [x] Translate auth pages, flashes and auth errors (pt-BR/en) + Google button + nav → verify: `TestLangCatalogs_SameKeys`, `TestForgotPassword_FlashIsLocalized`.
- [x] Translate validation messages (`validation.*` + `validation.attributes.*` catalogs) → verify: `TestRegister_ValidationIsLocalized`.
- [x] Translate auth emails (subject + views via `t`) → verify: `TestVerificationEmail_IsLocalized`.
- [x] Localize the custom password-confirmation message (vukem v0.24.0 `z.FieldErrorKey`) → verify: `TestRegister_PasswordMismatchIsLocalized`.

#### 1.2 Email (Mailtrap / Mailpit)

- [x] SMTP config documented and wired (`MAIL_MAILER=smtp`) → verify: e2e `auth.spec.ts` registers, reads the verification mail from Mailpit (`/api/v1/search`), verifies, logs out and in.
- [x] `tests/mailtrap` helper (net/http): list sandbox messages by recipient, read `body.txt`, extract links, clean sandbox; skip without `MAILTRAP_API_TOKEN` → verify: `go test ./tests/mailtrap` (fake API server).
- [x] Real delivery test `go test -tags=mailtrap -run Mailtrap ./tests/` (CI job `mailtrap` runs when secrets exist) → verify: green with owner creds (CI run 39045216068: `--- PASS: TestMailtrap_VerificationEmailDelivered`).
- [x] Email verification required for the dashboard → verify: unverified user redirected (test `TestDashboard_RequiresVerifiedEmail`).
- [ ] Password reset end-to-end → verify: integration test with array mailer + Mailtrap e2e.

#### 1.3 Google login

- [x] Google provider via `social` with `SOCIAL_FAKE_*` in tests → verify: `TestSocialSignUpAndLogin`.
- [x] Account linking by verified email only → verify: `TestSocialDoesNotTakeOverUnverifiedEmails` (409).

#### 1.4 Abuse protections

- [x] Rate limits on login (per email+IP), forgot-password, verification resend → verify: `TestLoginLogoutAndThrottle`, `TestSecurity_ForgotPasswordRateLimited` (429).
- [x] Session rotation on login, cross-site POST blocked (CSRF) → verify: `TestSecurity_SessionRotatesOnLogin`, `TestSecurity_CrossSiteLoginBlocked`.

### Phase 2 — Raffles

- [x] Migration `raffles` + `raffle_numbers` with CHECKs/uniques (D14) → verify: `TestPostgres_RaffleNumbersAndConstraints`.
- [x] `z` schema for create/update (title, description, number count 10..1000, price 1..100000, duration 7..60) + zod mirror → verify: `TestRaffle_Validation`, e2e client validation.
- [x] Create raffle (pre-create numbers, slug, ends_at) → verify: `TestRaffle_CreateWithNumbers` (+1000 numbers on Postgres).
- [x] Dashboard list + manage page (grid with status) → verify: Inertia prop tests + e2e `raffles.spec.ts`.
- [x] Edit with lock rule (price/count/duration immutable after first paid, D18) + resize before lock → verify: `TestRaffle_LockedAfterFirstPaidNumber`, `TestRaffle_UpdateAndResize`.
- [x] Delete only without paid numbers, else archive → verify: `TestRaffle_DeleteWithoutPaidNumbers`, 409 + archive in lock test.
- [x] Public page `/r/{slug}` without personal data → verify: `TestRaffle_PublicPageHasNoPersonalData`, e2e (XSS payload rendered as text).
- [x] IDOR tests (another owner gets 404 on every raffle route) → verify: `TestRaffle_OtherOwnersGet404`, `TestRaffle_GuestsAndUnverifiedCannotManage`.

### Phase 3 — Numbers and payment links

- [x] Migration `reservations` (partial unique index) → verify: integration test.
- [x] Buyer form (name, email, BR phone → E.164) with `z` → verify: validation tests (phone formats).
- [x] Generate link: guarded reserve tx + token + `ExpireReservation` (`DispatchTx`, `Delay(10m)`) → verify: test with fake clock.
- [x] Concurrency: 20 goroutines reserve the same number → exactly 1 succeeds → verify: integration `-race`.
- [x] Expiry job + every-minute sweeper → verify: fake clock test, number back to available.
- [x] Owner cancel → verify: test.
- [x] Buyer page `/pay/{token}` + expired message (PT-BR text in D-brief) → verify: tests + e2e.
- [x] Background runners (queue worker + scheduler) inside `serve` (D15) → verify: integration test processes a delayed job.

### Phase 4 — Payments (AbacatePay)

- [x] `abacatepay` client: create PIX (`/v2/transparents/create`), check, simulate-payment (dev), `z`-validated responses, timeouts → verify: unit tests with `httptest.Server`.
- [x] Confirm endpoint creates PIX with `expiresIn` = remaining seconds, stores `charge_id`, `brCode`, `brCodeBase64` → verify: test.
- [x] Modal with QR + copy-paste + polling → verify: e2e (fake AbacatePay in CI).
- [x] Webhook: secret + HMAC verification, `payment_events` dedupe, `transparent.completed` → paid → verify: tests for valid, forged, replayed, out-of-order events.
- [x] Late payment → `refund_due` → verify: test.
- [x] Confirmation emails to buyer and owner (queued) → verify: array mailer test + Mailtrap e2e.
- [x] Sandbox end-to-end with real dev key + `simulate-payment` (manual/opt-in tag `abacatepay`) → verify: script output.

### Phase 5 — Ledger and payouts

- [x] Migration `ledger_entries` (append-only trigger) + `payouts` → verify: UPDATE/DELETE rejected (integration).
- [x] Split 90/10 on payment → verify: table test (amount edge cases).
- [x] Owner balance on dashboard → verify: test.
- [x] Admin payouts page (admin only, amount ≤ balance) → verify: tests incl. non-admin 404.

### Phase 6 — Draw

- [x] Draw algorithm v1 (seed, rejection sampling, participants hash) as pure function → verify: deterministic test vectors + distribution sanity test.
- [x] `RunDraw` job (deadline + sold out) with row lock, idempotent → verify: concurrency test runs once.
- [x] Zero paid numbers → `closed` without winner → verify: test.
- [x] Public result + "how to verify" section → verify: page test, e2e.

### Phase 7 — Export and REST API

- [x] Export JSON + CSV (encoding/csv, formula-injection guard) → verify: golden tests.
- [x] API keys page (create shows token once, revoke owner-checked, abilities read/write) → verify: tests incl. IDOR revoke.
- [x] `/api/v1` endpoints (§5) with `z`, problem+json, pagination, Idempotency (explicit Scope) → verify: HTTP tests + `AssertMatchesOpenAPI`.
- [x] OpenAPI + Scalar at `/api/docs` → verify: e2e loads docs.
- [x] Generate TS types + zod schemas from OpenAPI (`go run ./cmd/app codegen:types -zod`) and use them in forms → verify: `bun run typecheck`.

### Phase 8 — Security hardening and QA

- [ ] OWASP abuse suite (Go + e2e): IDOR, CSRF, XSS payloads in title/description/buyer name, brute force rate limit, webhook forgery/replay, concurrent reserve race, expired link reuse → verify: `go test ./tests -run Security`, `bun run test:e2e -- security`.
- [x] Postgres rate-limit store (D5, vukem v0.23.0 `RATE_LIMIT_STORE=cache`) → verify: `TestPostgres_LoginThrottleSharedInDatabase`.
- [x] `docs/qa-checklist.md` exploratory checklist executed once with notes → verify: checklist filled (run 1, 2026-10-17).
- [x] ZAP rules tuned (promote relevant rules to FAIL) → verify: CI green (`.zap/rules.tsv`: 12 FAIL, 3 IGNORE with reasons).

### Phase 9 — Deploy and launch

- [ ] Railway project: app service from Dockerfile + Postgres plugin; env vars from §10 → verify: `/up?checks=1` on Railway URL.
- [ ] Custom domain `rifa.vukem.com` (CNAME) + HTTPS → verify: `curl -I https://rifa.vukem.com/up`.
- [ ] AbacatePay webhook registered to production URL; production SMTP sender → verify: sandbox payment end-to-end on staging.
- [ ] Tag `v1.0.0` → verify: release workflow publishes image and notes.

## 9. Framework bump flow

When the coordinator announces a vukem release (CI green): `go get github.com/AlexGalhardo/vukem@vX.Y.Z && go mod tidy` → remove `workaround(vukem …)` code the release fixed → `go test -race ./...`, integration, `bun run typecheck`, push, `gh run list` → commit `build(deps): bump vukem to vX.Y.Z` → mark entries in `../framework-feedback/rifa.md` as `Status: fixed in vX.Y.Z`. Known upcoming change: `api.Idempotency` refuses keyed requests with an empty Scope — always set Scope.

## 10. Developer actions (owner to-do)

| What | Env / where | Needed by |
| --- | --- | --- |
| Mailtrap account: API token, account id, sandbox id, sandbox SMTP user/pass (`sandbox.smtp.mailtrap.io:2525`) | `MAILTRAP_API_TOKEN`, `MAILTRAP_ACCOUNT_ID`, `MAILTRAP_SANDBOX_ID`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD` (local `.env`; GitHub Actions secrets `MAILTRAP_API_TOKEN`, `MAILTRAP_SANDBOX_ID`, `MAILTRAP_SMTP_USERNAME`, `MAILTRAP_SMTP_PASSWORD`) | Phase 1 |
| Google OAuth client (Web), redirect URIs `http://localhost:3000/auth/google/callback` and `https://rifa.vukem.com/auth/google/callback` | `SOCIAL_GOOGLE_CLIENT_ID`, `SOCIAL_GOOGLE_CLIENT_SECRET` | Phase 1 |
| AbacatePay account in Dev mode: API key `abc_dev_…` with `TRANSPARENT:CREATE`, `TRANSPARENT:READ`, `WEBHOOK:CREATE`; webhook secret | `ABACATEPAY_API_KEY`, `ABACATEPAY_WEBHOOK_SECRET` (+ the HMAC key AbacatePay shows for `X-Webhook-Signature`, if different) | Phase 4 |
| Production sender: Mailtrap Sending (verified domain `vukem.com`: SPF/DKIM/DMARC CNAMEs) or another SMTP | `SMTP_*`, `MAIL_FROM` | Phase 9 |
| Railway project + Postgres plugin; set `APP_KEY` (`vukem key:generate`), `APP_ENV=production`, `APP_URL=https://rifa.vukem.com`, all of the above | Railway variables | Phase 9 |
| DNS: CNAME `rifa.vukem.com` → Railway domain | DNS provider | Phase 9 |
| AbacatePay production key + webhook to `https://rifa.vukem.com/webhooks/abacatepay` | Railway variables | Launch |
| First admin user: `UPDATE users SET is_admin = true WHERE email = …` (or `app make:admin` command, Phase 5) | DB | Phase 5 |

Optional dev tool: Mailtrap MCP server — `claude mcp add mailtrap -e MAILTRAP_API_TOKEN=… -e MAILTRAP_ACCOUNT_ID=… -- npx -y mcp-mailtrap`.
