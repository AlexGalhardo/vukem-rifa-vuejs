# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Bumped vukem to v0.22.0: `auth.Verified` replaces the local redirect middleware, `inertia.I18n` shares the catalogs (the Vue composable no longer bundles the JSON), tests use `Inertia("")`.
- `serve` runs the database queue worker and the scheduler in-process (`BACKGROUND_IN_SERVE`).

### Added

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
