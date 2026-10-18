# Operations

Production runs at <https://rifa.vukem.dev> on Railway (one Docker service + Postgres).

## Email

Two transports. Production on Railway: `MAIL_MAILER=mailtrap` (Mailtrap Email API over HTTPS, vukem v0.26.0; `MAILTRAP_API_TOKEN`, `MAILTRAP_INBOX_ID` for the sandbox) because Railway blocks outgoing SMTP. Since vukem v0.28.0 a broken mail setting (for example `MAIL_MAILER=mailtrap` without `MAILTRAP_API_TOKEN`) fails the boot, `migrate` included. Local and CI: SMTP (`MAIL_MAILER=smtp`, `SMTP_*`). Local: Mailpit at <http://localhost:8025> (`SMTP_INSECURE=true`, local only). Dev/staging/e2e: [Mailtrap Email Sandbox](https://mailtrap.io) (`sandbox.smtp.mailtrap.io:2525`); tests read delivered emails through the Mailtrap Sandbox API when `MAILTRAP_API_TOKEN`, `MAILTRAP_ACCOUNT_ID` and `MAILTRAP_SANDBOX_ID` are set. Optional Mailtrap MCP for agents: `claude mcp add mailtrap -e MAILTRAP_API_TOKEN=… -e MAILTRAP_ACCOUNT_ID=… -- npx -y mcp-mailtrap`.

The free Mailtrap sandbox is shared with other projects: email tests use a unique random recipient, filter by it, run serially (`-p 1`, about one message per 10 s) and never clean the inbox.

## Production demo (no real delivery yet)

Until the `vukem.dev` sender domain is verified, production sends every email (verification, password reset, payment confirmations) to the Mailtrap **sandbox**, so nothing reaches the real inbox. Sign-up never logs in (it answers "check your email" for new and existing addresses alike) and the dashboard requires a verified email, so to try the demo either:

- open the verification link from the Mailtrap sandbox inbox (it is listed by recipient), then log in, or
- verify the account by hand: `UPDATE users SET email_verified_at = now() WHERE email = '…';` (Railway Postgres console) and log in, or sign in with Google (Google accounts are verified).

Buyers receive no email in this mode either; owners share the payment link manually (copy or WhatsApp), which is the main flow anyway.

## Railway

- One service runs HTTP, the database queue worker (`QUEUE_IN_SERVE`) and the scheduler (`SCHEDULE_IN_SERVE`), both vukem defaults. If a separate `schedule:work` or `queue:work` service is ever added, set the matching flag to `false` on the web service, never both.
- Client IP and HSTS (set on Railway 2026-10-17): the edge connects from `100.64.0.x` (vukem's request log `peer` field), so `TRUSTED_PROXIES=100.64.0.0/24`, `CLIENT_IP_HEADER=X-Real-IP` and `TRUST_FORWARDED_PROTO=true` give per-client `ip` for rate limits and logs; `HSTS_FORCE=true` sends HSTS. If Railway ever connects from outside that range, `c.IP()` falls back to the edge address (stricter limits, never spoofable): re-check `peer` in the logs. Never trust `0.0.0.0/0`.
- vukem v0.24.0 refuses clear-text queued mail jobs from older releases: deploy when the `jobs` table has no pending `vukem.mail` rows, or set `QUEUE_ALLOW_LEGACY_MAIL=true` for one deploy.
- vukem v0.28.0 namespaces webhook dedupe keys (the AbacatePay route uses `abacatepay`): a redelivery of an event handled just before that deploy runs once more and is stopped by the unique `payment_events.event_id`.
