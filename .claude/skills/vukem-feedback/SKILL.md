---
name: vukem-feedback
description: Record vukem framework bugs, gaps, awkward APIs or docs problems found while building this app, and bump the framework when a new vukem release lands. Use whenever a vukem API is missing, wrong, insecure or confusing, or when told a new vukem tag is out.
---

# Vukem feedback loop

This app battle-tests the vukem framework (`github.com/AlexGalhardo/vukem`) for 1.0.

## When you hit a framework problem

1. Confirm it against the pinned source: `git -C ../vukem show <tag>:<path>` (never edit `../vukem`).
2. Check `../framework-feedback/audit.md` and `../framework-feedback/rifa.md`; do not duplicate.
3. Append to `../framework-feedback/rifa.md` using its template (Severity, Package, Found while, Docs page with the <https://vukem.vercel.app/docs/...> URL for docs findings, Repro, Workaround, Suggested fix, `Status: open`).
4. Implement the smallest local workaround and mark it in code: `// workaround(vukem v0.21.1): <entry title>`.

## When a new vukem release is announced

1. `go get github.com/AlexGalhardo/vukem@vX.Y.Z && go mod tidy` (also `drivers/postgres` if released).
2. `rg "workaround\(vukem"` and remove the workarounds the release fixed.
3. `go test -race ./...`, `go test -tags=integration ./...`, `bun run typecheck`, push, check `gh run list`.
4. Commit `build(deps): bump vukem to vX.Y.Z`; set the fixed entries in `rifa.md` to `Status: fixed in vX.Y.Z`.
