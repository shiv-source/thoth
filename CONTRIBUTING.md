# Contributing to Thoth

- Conventional commits; branches off `main`; squash-merge PRs.
- All code is tested: `go test -race ./...` and `cd web && npx vitest run` must pass.
- Internal Go packages keep ≥80% coverage — CI enforces it.
- TypeScript is strict: no `any`, zod at the API boundary.
- The Claude CLI flag surface lives ONLY in `internal/claude/client.go`; when
  the CLI changes, that is the one file to update.
- Review happens on every PR; large changes go through the design docs in
  `docs/superpowers/specs/`.
