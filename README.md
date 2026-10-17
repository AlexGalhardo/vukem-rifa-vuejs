# Vukem Rifa

Online raffles (rifas) with PIX payments for a Brazilian audience, running at <https://rifa.vukem.com>.

Built with [Vukem](https://vukem.vercel.app) (Go) + Inertia v3 + Vue 3 + TypeScript + Tailwind CSS v4, on PostgreSQL. It also serves as a real-world test bed for the Vukem framework.

- Owners sign up (email/password or Google), create raffles (10–1000 numbers, whole-real prices, 7–60 days) and generate a payment link per number that reserves it for 10 minutes.
- Buyers (no account) pay by PIX through AbacatePay; a verified, idempotent webhook marks the number as paid.
- An auditable `crypto/rand` draw runs at the deadline, or as soon as every number is paid.
- Exports (JSON/CSV), a REST API with personal API keys and OpenAPI/Scalar docs.

See [PLAN.md](PLAN.md) for the full design, data model, roadmap and owner to-do list.

## Requirements

- Go 1.27.2, Bun 1.4.3, Docker (Compose v2)
- Optional: [vukem CLI](https://vukem.vercel.app/docs/installation) v0.21.1 (`go install github.com/AlexGalhardo/vukem/cmd/vukem@v0.21.1`), golangci-lint v2.15

## Quick start (Docker)

```sh
cp .env.example .env
vukem key:generate                # or: echo "APP_KEY=base64:$(openssl rand -base64 32)" >> .env
docker compose up -d --build      # app :3000, postgres :5432, mailpit :8025
curl http://localhost:3000/up     # {"status":"ok"}
```

## Local development (hot reload)

```sh
bun install                       # also installs the git hooks (lefthook)
docker compose up -d postgres mailpit
go run ./cmd/app migrate
bun run dev                       # Vite dev server (writes resources/public/hot)
go run ./cmd/app                  # Go app on http://localhost:3000
```

## Commands

| Task | Command |
| --- | --- |
| Go unit tests (race) | `go test -race ./...` |
| Go integration tests (Postgres) | `TEST_DATABASE_URL=postgres://rifa:rifa@localhost:5432/rifa?sslmode=disable go test -tags=integration -race ./...` |
| Go lint / format | `golangci-lint run ./...` / `golangci-lint fmt` |
| Vulnerabilities | `go tool govulncheck ./...`, `bun audit` |
| Frontend lint / format | `bun run lint` / `bun run lint:fix` |
| Typecheck | `bun run typecheck` |
| Build assets | `bun run build` |
| E2E (stack running) | `bun run test:e2e` |
| Markdown lint | `bun run lint:md` |
| Migrations | `go run ./cmd/app migrate`, `migrate:status`, `migrate:rollback` |

## Email

SMTP only (`MAIL_MAILER=smtp`, `SMTP_*`). Local: Mailpit at <http://localhost:8025> (`SMTP_INSECURE=true`, local only). Dev/staging/e2e: [Mailtrap Email Sandbox](https://mailtrap.io) (`sandbox.smtp.mailtrap.io:2525`); tests read delivered emails through the Mailtrap Sandbox API when `MAILTRAP_API_TOKEN`, `MAILTRAP_ACCOUNT_ID` and `MAILTRAP_SANDBOX_ID` are set. Optional Mailtrap MCP for agents: `claude mcp add mailtrap -e MAILTRAP_API_TOKEN=… -e MAILTRAP_ACCOUNT_ID=… -- npx -y mcp-mailtrap`.

The free Mailtrap sandbox is shared with other projects: email tests use a unique random recipient, filter by it, run serially (`-p 1`, about one message per 10 s) and never clean the inbox.

### Production demo (no real delivery yet)

Until the `vukem.com` sender domain is verified, production sends every email (verification, password reset, payment confirmations) to the Mailtrap **sandbox**, so nothing reaches the real inbox. The dashboard requires a verified email, so to try the demo either:

- open the verification link from the Mailtrap sandbox inbox (it is listed by recipient), or
- verify the account by hand: `UPDATE users SET email_verified_at = now() WHERE email = '…';` (Railway Postgres console), or sign in with Google (Google accounts are verified).

Buyers receive no email in this mode either; owners share the payment link manually (copy or WhatsApp), which is the main flow anyway.

## Contributing and security

[CONTRIBUTING.md](CONTRIBUTING.md) · [SECURITY.md](SECURITY.md) · [CHANGELOG.md](CHANGELOG.md) · [LICENSE](LICENSE) (MIT)
