# Security rules (all files)

- Owner resources are always scoped by `user_id` in the query (IDOR); 404, not 403, for other owners' resources.
- Buyers have no account: payment links are unguessable tokens (`crypto/rand`, ≥128 bits) stored hashed, expiring with the reservation.
- CSRF on every state-changing browser route (vukem `csrf`); API routes use API keys (hashed, scoped read/write, revocable, owner-checked on revoke).
- Rate limit login, password reset, verification resend, payment-link generation and the webhook.
- No secrets in code, logs or the repo; `.env` and `.claude/settings.local.json` are git-ignored.
- `api.Idempotency`: always set an explicit `Scope` (user id / API key id).
- New security-relevant behavior needs a test in the OWASP abuse suite (`tests/security_*_test.go` or `e2e/security.*.spec.ts`).
