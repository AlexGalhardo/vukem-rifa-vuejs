# OWASP Top 10 (2021) abuse suite

Every category maps to concrete tests. Run them all with:

```sh
go test ./tests -run 'Security|OWASP|IDOR|Webhook|OtherOwner|Concurrent' -race
TEST_DATABASE_URL=postgres://rifa:rifa@localhost:5432/rifa?sslmode=disable go test -tags=integration ./tests -run 'Postgres_' -race
bunx playwright test e2e/security.spec.ts e2e/a11y.spec.ts
```

New tests written for the consolidated suite live in `tests/security_owasp_test.go`, `tests/integration_test.go` (`TestPostgres_OWASP_*`) and `e2e/security.spec.ts`; the rest already existed and are referenced by name.

## A01 Broken access control

| Abuse case | Tests |
| --- | --- |
| IDOR on raffles (another owner gets 404 on every route) | `TestRaffle_OtherOwnersGet404`, `TestRaffle_GuestsAndUnverifiedCannotManage` |
| IDOR on numbers / payment links / cancel | `TestSecurity_NumbersOtherOwnersGet404`, `TestSecurity_NumbersGuestsAndUnverifiedBlocked` |
| IDOR on exports | `TestExport_OtherOwnerGetsNotFound` |
| IDOR on API keys (revoke another user's key) and API `{id}` routes | `TestAPIKeys_RevokeIsOwnerChecked`, `TestSecurityAPI_OtherOwnersRafflesAreNotFound` |
| Organizer contact is opt-in (defaults off), validated, never leaks across owners; email split against scrapers | `TestOrganizer_PublicPageShowsTheOrganizerButNoContactByDefault`, `TestOrganizer_ContactTogglesShowExactlyWhatWasChosen`, `TestOrganizer_InvalidContactsAreRejected`, `TestOrganizer_OtherOwnersNeverLeak` |
| Buyer sees only their own masked data; public page has no personal data | `TestPay_ShowsOnlyTheBuyersOwnMaskedData`, `TestRaffle_PublicPageHasNoPersonalData`, `TestNumbers_LiveLinksStayVisibleToTheOwnerForTenMinutes` |
| Vertical escalation: non-admin on payouts (GET and POST) | `TestAdminPayouts_HiddenFromNonAdmins`, `TestRegister_IsNotAdmin` |
| Mass assignment (`user_id`, `status`, `slug`, `locked_at`, `is_admin` in bodies) | `TestOWASP_A01_MassAssignmentIsIgnored` |
| Public checkout: only free numbers, closed raffles refuse, purchase history shows only my own reservations, no buyer data on the public page | `TestPublicCheckout_OnlyFreeNumbers`, `TestPublicCheckout_ClosedRafflesRefuse`, `TestPublicCheckout_PurchaseHistoryShowsOnlyMyPurchases`, `TestPublicCheckout_FreeNumberGoesToThePayPage` |
| Read-only API key on write endpoints | `TestSecurityAPI_ReadOnlyKeyCannotWrite` |

## A02 Cryptographic failures

| Abuse case | Tests |
| --- | --- |
| Payment-link tokens are 256-bit, looked up by SHA-256 and sealed with APP_KEY for the owner view (never stored in clear, never on public pages) | `TestReservation_ReserveCreatesHashedLink`, `TestNumbers_LiveLinksStayVisibleToTheOwnerForTenMinutes` |
| Session cookie HttpOnly + SameSite | `TestOWASP_A02_SessionCookieFlags` |
| Passwords hashed with salted argon2id (m=64 MiB, t=1, p=4) outside tests | `TestOWASP_A02_PasswordsAreArgon2id` |
| Production refuses an http `APP_URL`; HSTS over HTTPS | `TestOWASP_A05_ProductionRefusesHTTPAppURL`, `TestOWASP_A05_ProductionSendsHSTS` |
| API key revoked/invalid tokens refused | `TestSecurityAPI_RejectsMissingInvalidAndRevokedTokens` |
| `APP_KEY` rotation: sealed data opens with the old key in `APP_PREVIOUS_KEYS`, fails closed without it | `TestPostgres_OWASP_A04_AppKeyRotationKeepsSealedData` |

On Railway HSTS is sent by `HSTS_FORCE` (README "Deploy notes").

## A03 Injection

| Abuse case | Tests |
| --- | --- |
| SQL payloads in slugs, tokens, ids and login credentials are inert | `TestOWASP_A03_InjectionPayloadsAreInert` |
| XSS payloads in title/description/buyer name render as text | `e2e/raffles.spec.ts`, `e2e/numbers.spec.ts` |
| CSV formula injection in exports | `TestExport_TextGuardsFormulas`, `TestSecurityAPI_CSVExportGuardsFormulas`, `TestExport_CSVGolden` |
| Database constraints refuse invalid states even if code is bypassed | `TestPostgres_RaffleNumbersAndConstraints`, `TestPostgres_PaymentSchemaConstraints` |

## A04 Insecure design

| Abuse case | Tests |
| --- | --- |
| Concurrent reservation of the same number → exactly one link (owner and public) | `TestPostgres_ConcurrentReserveSameNumber`, `TestPostgres_PublicCheckoutRacesAndCap` |
| Public checkout abuse: honeypot, LGPD consent, per-IP rate limit, cap of 5 live links per email (race-safe) | `TestPublicCheckout_RefusesBotsAndMissingConsent`, `TestPublicCheckout_RateLimitedPerIP`, `TestPublicCheckout_CapsLiveLinksPerEmail`, `TestPostgres_PublicCheckoutRacesAndCap` |
| Purchase history linking: signed-in buyer or verified email only; unverified emails link on verification | `TestPublicCheckout_LinksTheBuyerAccount` |
| Expired / cancelled link reuse (page 410, confirm refused) | `TestPay_ExpiredCancelledAndUnknownLinks`, `e2e/security.spec.ts` |
| Link expiry and sweeper; expiry never releases a paid number | `TestNumbers_LinkExpiresAfterTenMinutes`, `TestReservation_ExpiryNeverReleasesPaidNumbers`, `TestPostgres_SweepersExpireLinksAndDraw` |
| PIX never outlives the reservation or the raffle | `TestPay_ConfirmChargesTheRemainingTime`, `TestPay_ConfirmNeverOutlivesTheRaffle` |
| Late / mismatched payments never assign a number (`refund_due`) | `TestWebhook_LatePaymentIsFlaggedForRefund`, `TestWebhook_PaymentAfterTheNumberWasReReserved`, `TestWebhook_PaymentAfterTheRaffleClosedIsRefundDue`, `TestWebhook_AmountMismatchIsNotAPayment` |
| Draw runs exactly once; payouts never exceed the balance under races | `TestPostgres_ConcurrentDrawRunsOnce`, `TestPostgres_ConcurrentPayoutsNeverExceedTheBalance` |
| Price/count/duration locked after the first payment | `TestRaffle_LockedAfterFirstPaidNumber` |

## A05 Security misconfiguration

| Abuse case | Tests |
| --- | --- |
| CSP with nonce (no `unsafe-inline`), `frame-ancestors 'none'`, XFO, nosniff, Referrer/Permissions-Policy, no `X-Powered-By` | `TestOWASP_A05_SecurityHeaders`, `e2e/security.spec.ts` (no CSP violations), ZAP rules promoted to FAIL (`.zap/rules.tsv`) |
| No CORS on the API; cross-site browser requests blocked | `TestSecurityAPI_NoCORSAndCrossSiteBrowsersBlocked` |
| Dev payment helpers off unless the fake gateway is on; fake refused in production | `TestSecurity_DevPaymentHelpersOffByDefault`, `TestOWASP_A10_GatewayURLIsHTTPSOnlyInProduction` |
| API docs keep the CSP | `TestAPI_DocsPageKeepsCSP` |

## A06 Vulnerable and outdated components

CI: `govulncheck`, `bun audit --audit-level=high`, Dependabot (Go, npm, Actions, Docker); exact version pins (PLAN.md §3).

## A07 Identification and authentication failures

| Abuse case | Tests |
| --- | --- |
| Login brute force (per email+IP and per account), forgot-password rate limit, shared across instances | `TestLoginLogoutAndThrottle`, `TestSecurity_ForgotPasswordRateLimited`, `TestPostgres_LoginThrottleSharedInDatabase` |
| Session fixation (id rotates on login) | `TestSecurity_SessionRotatesOnLogin` |
| Copied session cookie is dead after logout | `TestPostgres_OWASP_A07_LogoutKillsCopiedSessionCookie` |
| Password reset: single use, ends tokens and other sessions | `TestPasswordReset`, `TestPostgres_PasswordResetEndsOtherSessions` |
| Reset enumeration: same answer for unknown emails; mail queued so timing does not differ | `TestPasswordReset`, `TestPostgres_OWASP_A07_ResetMailIsQueuedNotSentInline` |
| Sign-up enumeration: same status, redirect and cookies for new and existing emails; existing owner gets a throttled notice, nothing is created; one queued mail either way | `TestRegister_ExistingEmailIsNotRevealed`, `TestPostgres_OWASP_A07_SignupIsEnumerationSafe` |
| Unverified email cannot use the dashboard; Google cannot take over unverified accounts | `TestDashboard_RequiresVerifiedEmail`, `TestSocialDoesNotTakeOverUnverifiedEmails` |
| CSRF on login and every browser mutation | `TestSecurity_CrossSiteLoginBlocked`, `TestSecurity_NumbersCrossSitePostBlocked`, `TestSecurity_CrossSiteConfirmBlocked` |
| Cookie session is not ambient authority on the API | `TestSecurityAPI_CookieSessionIsNotAmbientAuthority` |
| Spoofed `X-Forwarded-For` / `X-Real-IP` from untrusted peers ignored; trusted proxy header honoured; IPv6 limited by /64 | `TestOWASP_A07_SpoofedClientIPHeadersAreIgnored`, `TestOWASP_A07_TrustedProxyHeaderSeparatesClients`, `TestLimitKeys_IPv6GroupsBySlash64` |

## A08 Software and data integrity failures

| Abuse case | Tests |
| --- | --- |
| Webhook forgery (wrong/missing secret, forged or public-key signature, tampered body), fail closed without config | `TestSecurity_WebhookRejectsForgedDeliveries`, `TestSecurity_WebhookWithoutConfiguredSecretFailsClosed`, `TestSecurity_WebhookInvalidPayload` |
| Webhook replay (same event, after cache loss, concurrent deliveries) | `TestSecurity_WebhookReplayIsANoOp`, `TestSecurity_WebhookReplayAfterCacheLossIsANoOp`, `TestPostgres_ConcurrentWebhookDeliveries`, `TestWebhook_SecondEventForAPaidChargeIsANoOp` |
| Idempotent API payment links scoped per key | `TestSecurityAPI_IdempotentPaymentLinks`, `TestPostgres_APIIdempotencyAndPagination` |
| Append-only ledger (UPDATE/DELETE refused by trigger) | `TestPostgres_PaymentSchemaConstraints` |
| Draw result recomputable from the published data | `TestDraw_ResultPages`, `e2e/draw.spec.ts` |

## A09 Security logging and monitoring failures

| Abuse case | Tests |
| --- | --- |
| Rejected webhooks logged as security events with the request id, never with the secret or signature | `TestOWASP_A09_RejectedWebhookIsLoggedWithoutSecrets` |
| Failed logins logged (`auth.login_failed`) with the request id, without the password or the email tried | `TestOWASP_A09_FailedLoginIsLoggedWithoutCredentials` |
| Payouts logged (`payout.recorded`: admin, owner, amount) without the free-text reference | `TestOWASP_A09_PayoutIsLogged` |

## A10 Server-side request forgery

The only outbound HTTP is the AbacatePay client at the operator-set `ABACATEPAY_BASE_URL`; no request URL comes from users (the WhatsApp share link is built and opened in the browser).

| Abuse case | Tests |
| --- | --- |
| Production refuses a non-HTTPS gateway URL (no API key over clear text, no `http://169.254.169.254`-style targets) | `TestOWASP_A10_GatewayURLIsHTTPSOnlyInProduction` |
| Gateway responses are validated before use | `app/payments/abacatepay` unit tests (`go test ./app/payments/...`) |
