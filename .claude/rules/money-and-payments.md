---
paths:
  - "app/**"
  - "database/**"
  - "routes/**"
  - "tests/**"
---

# Money, numbers and payments

- Prices are integer BRL whole reais in the raffle (`price_reais`); AbacatePay amounts are integer **centavos** (`price_reais * 100`). Never floats.
- Platform receives 100%; each paid number writes append-only `ledger_entries`: owner share 90% and platform fee 10% (integer centavos, fee = amount*10/100, owner = amount - fee). Never update or delete ledger rows.
- Number state machine: `available → reserved → paid`, `reserved → available` (expiry/cancel). Any other transition is a bug. Transitions happen in one DB transaction guarded by `WHERE status = '<from>'`.
- A reservation lasts exactly 10 minutes (`reservations.expires_at`); expiry runs from a delayed queue job plus a sweeper scheduled every minute.
- Webhooks: verify `webhookSecret` query param **and** `X-Webhook-Signature` (HMAC-SHA256, base64, constant-time compare) before parsing; dedupe by event id in `payment_events` (unique); only mark processed after the handler commits.
- A payment confirmed after the reservation expired is recorded but never silently assigns a number already re-reserved: flag it for refund in the ledger/export.
- Draws use `crypto/rand` and store seed, timestamp and SHA-256 of the ordered paid-number list.
