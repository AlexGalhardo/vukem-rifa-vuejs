# CLAUDE.md

Read [AGENTS.md](AGENTS.md) (commands, rules, layout) and [PLAN.md](PLAN.md) (design + current task) first.

- Language: code, docs and commits in English; the UI defaults to PT-BR (`resources/lang/pt-BR.json`).
- Skills in `.claude/skills/`: `tdd`, `golang-*`, `code-review-and-quality`, `karpathy-guidelines`, `grilling`, `frontend-design`, `web-design-guidelines`, `agent-browser`, `vukem-feedback`.
- `.claude/settings.local.json` disables all hooks for this repo by owner choice; it is git-ignored — do not commit or change it.
- Optional MCP: Mailtrap (`claude mcp add mailtrap -e MAILTRAP_API_TOKEN=… -e MAILTRAP_ACCOUNT_ID=… -- npx -y mcp-mailtrap`) to inspect sandbox emails.
- Never edit `../vukem`; log framework findings in `../framework-feedback/rifa.md`.
