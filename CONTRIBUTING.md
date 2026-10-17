# Contributing

1. Read [PLAN.md](PLAN.md) and [AGENTS.md](AGENTS.md); pick the first unchecked task of the current phase.
2. `bun install` (installs lefthook git hooks), `docker compose up -d postgres mailpit`.
3. Work test-first. A task is done when its `→ verify:` line passes.
4. Before pushing: `go test -race ./...`, `go test -tags=integration ./...`, `golangci-lint run ./...`, `bun run lint`, `bun run typecheck`, `bun run lint:md`.
5. Commit with [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) (`type(scope): subject`, scopes in `commitlint.config.js`), in English.
6. Add a line to `CHANGELOG.md` under `## [Unreleased]` ([Keep a Changelog](https://keepachangelog.com/en/1.1.0/)).
7. Open a pull request; CI must be green.

## Style

- `.editorconfig`: tabs, width 4 (YAML: 2 spaces).
- Go: `golangci-lint fmt` (gofumpt + gci). Frontend: Biome.
- JavaScript tooling: Bun only. Dependencies pinned to exact stable versions.

## Releases

SemVer. Tag `vX.Y.Z` on `main` after updating `CHANGELOG.md`; the release workflow builds the Docker image (GHCR) and publishes the GitHub release.
