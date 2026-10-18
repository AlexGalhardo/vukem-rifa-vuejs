# QA checklist

Exploratory pass before launch. Automated coverage is listed so the manual pass focuses on what tests cannot see. Re-run it before each release tag and add a dated section.

## Run 1 — 2026-10-17 (main 86c7df5 + this change, local app on :3104, `ABACATEPAY_FAKE=true`, Postgres 18.6, Mailpit)

Legend: ✅ checked by hand (curl) in this run · 🤖 exercised by the named automated tests (green in CI for this commit) · ⏭ not done · ⚠️ finding (see Notes).

### Owner flows

| Check | Result |
| --- | --- |
| Register → verification mail → dashboard; login throttle; reset ends other sessions | 🤖 `e2e/auth.spec.ts`, `TestLoginLogoutAndThrottle`, `TestPostgres_PasswordResetEndsOtherSessions` |
| Create raffle at the limits (10/1000 numbers, R$ 1/100000, 7/60 days); client and server errors agree | 🤖 `TestRaffle_Validation`, `e2e/raffles.spec.ts` |
| Generate a link, copy/WhatsApp, countdown, link shown once, cancel | 🤖 `e2e/numbers.spec.ts`, `TestNumbers_*` |
| Price/count/duration locked after the first paid number; archive instead of delete | 🤖 `TestRaffle_LockedAfterFirstPaidNumber`, `TestRaffle_DeleteWithoutPaidNumbers` |
| Export JSON/CSV opens in a spreadsheet; formula cells are neutralized | 🤖 `TestExport_*Golden`, `TestSecurityAPI_CSVExportGuardsFormulas` |
| API keys: token shown once, read-only key gets 403 on writes, revoke → 401 | 🤖 `e2e/api.spec.ts`, `tests/security_api_test.go` |
| Admin payouts: non-admin 404, amount above balance 422, concurrent double submit | 🤖 `TestAdminPayouts_*`, `TestPostgres_ConcurrentPayoutsNeverExceedTheBalance` |

### Buyer flows

| Check | Result |
| --- | --- |
| `/pay/{token}` shows only masked email/phone, `Cache-Control: no-store`, `Referrer-Policy: no-referrer` | ✅ curl (404 page also no-store/no-referrer) + 🤖 `e2e/payments.spec.ts` |
| Confirm → PIX modal (QR + copy-paste), reopen keeps the charge, polling confirms | 🤖 `e2e/payments.spec.ts` |
| Expired/cancelled link text ("Link expirado. Peça ao dono…") with 410, unknown 404, EN translation | 🤖 `e2e/payments.spec.ts`, `e2e/a11y.spec.ts` |
| Public page `/r/{slug}` and draw result show no personal data; "how to verify" recomputes | 🤖 `TestRaffle_PublicPageHasNoPersonalData`, `e2e/draw.spec.ts` |

### Security (OWASP)

| Check | Result |
| --- | --- |
| Headers on `/`: CSP with nonce, `frame-ancestors 'none'`, `X-Frame-Options: DENY`, `nosniff`, Referrer-Policy, Permissions-Policy; no `Server`/`X-Powered-By` | ✅ curl; ZAP rules promoted to FAIL (`.zap/rules.tsv`) |
| Cross-site POST (`Origin: https://evil.example`, `Sec-Fetch-Site: cross-site`) to `/login` | ✅ 403 + 🤖 `TestSecurity_CrossSite*` |
| `/api/v1` without token → 401 `application/problem+json`; CORS preflight from another origin gets no `Access-Control-*` | ✅ curl (preflight 405, no CORS headers) |
| Webhook without `webhookSecret` → 401; forged/replayed deliveries | ✅ curl + 🤖 `TestSecurity_Webhook*`, `TestPostgres_ConcurrentWebhookDeliveries` |
| Oversized webhook body (2 MB) | ✅ 401 (rejected before parsing; fails closed) |
| IDOR on raffles, numbers, exports, API keys, API `{id}` routes | 🤖 `TestRaffle_OtherOwnersGet404`, `TestSecurity_Numbers*`, `TestExport_OtherOwnerGetsNotFound`, `tests/security_api_test.go` |
| Concurrent reservation of one number; concurrent draw | 🤖 `TestPostgres_ConcurrentReserveSameNumber`, `TestPostgres_ConcurrentDrawRunsOnce` |
| XSS payloads in title/description/buyer name render as text | 🤖 `e2e/raffles.spec.ts`, `e2e/numbers.spec.ts` |
| Rate limits shared across instances (database cache) | 🤖 `TestPostgres_LoginThrottleSharedInDatabase` |

### Accessibility and UI

| Check | Result |
| --- | --- |
| axe (serious/critical) on every page/state incl. PIX modal, expired link, admin payouts; light/dark, pt-BR/en, desktop/mobile | 🤖 `e2e/a11y.spec.ts` (24 passed locally) |
| Keyboard: tab through the link form, PIX modal close, theme and language switchers | ⏭ not done in run 1 (needs a person at a browser) |
| Dark mode has no flash on load; language persists via cookie | ⏭ not done in run 1; theme/locale covered by 🤖 `e2e/smoke.spec.ts` |

### Notes

- ✅ Per-IP rate limits are per client on Railway (`TRUSTED_PROXIES=100.64.0.0/24`, README "Deploy notes"); vukem v0.27.0 warns at boot if the trusted range is missing behind a proxy.
- ⚠️ `/api/docs` (bundled Scalar) has third-party axe findings; excluded from the a11y suite and reported in `framework-feedback/rifa.md`.
- ⚠️ Production emails go to the Mailtrap sandbox until the `vukem.dev` sender is verified (README "Production demo").
- The owner "number sold" email uses the app default locale (owners have no stored locale).
