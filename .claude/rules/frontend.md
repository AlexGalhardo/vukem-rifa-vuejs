---
paths:
  - "resources/js/**"
  - "resources/css/**"
  - "resources/lang/**"
  - "e2e/**"
---

# Frontend rules (Inertia v3 + Vue 3 + TS + Tailwind v4)

- Vue SFCs with `<script setup lang="ts">`; strict TS, no `any` (use `unknown` + narrowing).
- Pages live in `resources/js/pages/<Area>/<Name>.vue` and render via `inertia.Render(c, "<Area>/<Name>", props)`; prop shapes in `resources/js/types.ts` mirror the Go structs.
- Every user-facing string goes through `t("key")` (`composables/useI18n.ts`); add the key to **both** `resources/lang/pt-BR.json` and `resources/lang/en.json` (same files feed the Go i18n bundle). PT-BR is the default.
- Dark mode is class-based (`html.dark`, `@custom-variant dark` in `app.css`); always pair light and `dark:` classes.
- Validate forms client-side with zod mirroring the Go `z` schema; the server is still the authority.
- Never render buyer personal data on public pages (`/r/{slug}`).
- Lint/format with Biome (`bun run lint:fix`), typecheck with `bun run typecheck`. Bun only, never npm/yarn/pnpm.
- Add `data-testid` to elements e2e tests use.
