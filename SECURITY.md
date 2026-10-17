# Security Policy

## Supported versions

Only the latest release and `main` receive security fixes.

## Reporting a vulnerability

Please do not open public issues for vulnerabilities. Report privately through [GitHub Security Advisories](https://github.com/AlexGalhardo/vukem-rifa-vuejs/security/advisories/new) or by email to the maintainer (Alex Galhardo, see the GitHub profile). Include steps to reproduce and impact. You will get an answer within 7 days.

## Scope highlights

- Payment links (buyer tokens), the AbacatePay webhook, owner data isolation (IDOR), API keys and the draw are the most sensitive areas.
- Never test against production payments; use the AbacatePay dev mode.

## Practices

Secure defaults from Vukem (CSP with nonces, secure cookies in production, CSRF), argon2id passwords, hashed tokens, verified and idempotent webhooks, rate limits, `govulncheck` + `bun audit`, ZAP baseline and nuclei scans in CI. See the OWASP checklist in [PLAN.md](PLAN.md#7-owasp-security-checklist).
