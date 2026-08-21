# Contributing to Thoth

Thoth is a local-first personal knowledge base: one Go binary (Echo + SQLite + a headless Claude Code CLI) serving an embedded React dashboard, with your plain-markdown wiki as the source of truth. It is intentionally small — the whole stack fits in your head after a read of the docs.

This page is the workflow; the ground rules for writing code in this repository live in [CLAUDE.md](CLAUDE.md) — read it before you touch anything. The documentation set in [docs/](docs/index.md) is the design authority.

## Setup

Prerequisites: Go 1.26+ (see `go.mod`), Node 24+ via nvm, pnpm 11.7+, and a logged-in Claude Code CLI (the app drives it headless; `thoth doctor` verifies this).

```sh
git clone git@github.com:shiv-source/thoth.git
cd thoth
pnpm install   # pnpm workspace — web/ is the only member, one root lockfile
make web       # frontend build + embed sync — required before go build/test
make dev       # Vite HMR + Go server together (http://127.0.0.1:8333)
```

## Workflow

1. **Never commit to `main`.** Sync and branch first:
   ```sh
   git switch main && git pull --ff-only && git switch -c <type>/<scope>/<slug>
   ```
2. **Conventional commits** — `<type>(<scope>): <summary>`; prefixes: `feat:`, `fix:`, `perf:`, `chore:`, `docs:`, `refactor:`, `test:`, `ci:` (`perf` maps to the `performance` type label).
3. **Open a PR** using the template — conventional title, a summary that gives the full picture (bullets when it helps), files changed, and the verification checklist. The `ci-pr` quality gates run automatically; `final-gate` posts its report as a comment and must pass before merging.
4. **Squash-merge** PRs unless the commit history is meaningful.
5. **Every PR is reviewed.** Large or cross-package changes go through a design doc in `docs/specs/` (untracked working docs — never committed) *before* implementation.

## Before you push

The pre-commit hook runs automatically: it refuses commits made directly on `main` (changes land via branches and reviewed PRs), `lint-staged` applies `eslint --fix` + prettier to staged `web/src` files and `golangci-lint --fix` to staged Go files; Go commits additionally gate on `go vet ./...`, `golangci-lint run`, and `go test ./...`. Formatting and autofixes happen for you — but the gate fails on anything a linter cannot fix, by design.

Before opening a PR, run what CI enforces, locally:

```sh
make check
```

That is: `gofmt`/`golangci-lint` clean, `go test -race ./...`, coverage ≥ **90%** on `agent/` + `internal/` + `cmd/`, all five cross-compiles, and the frontend `pnpm typecheck` · `pnpm lint` · `pnpm test` · `pnpm run build`.

## Code standards

### Go

- Every behavior change ships with a test — no exception. Write it first.
- Coverage floor is CI-enforced; a red cover gate means tests, not luck.
- Migrations are **additive**: one new file per change in `internal/store/migrations/`, gated on `PRAGMA user_version`. Never edit an applied migration.
- Small packages with one purpose; everything lives in `internal/` except the binary entrypoint.

### TypeScript

- Strict mode throughout — `noUncheckedIndexedAccess`, `verbatimModuleSyntax`, zero `any`.
- Data crossing the API boundary is validated with zod (`web/src/api/client.ts`).
- Small functions (≤ 3 params); no dead code (`noUnusedLocals`).
- State that is shared across screens or server-backed lives in the Redux store (`web/src/store/`, one slice file per feature); genuinely local UI state stays in components — when in doubt, keep it local.

## Guardrails you must not break

- **The blast wall.** The Claude CLI flag surface lives *only* in `internal/claude/client.go`. When the CLI changes, that is the one file to update — verify flags against `claude --help`.
- **Files are the source of truth.** `thoth.db` is derived data: deleting it costs a reindex, never knowledge. No code may make the wiki depend on the app.
- **No memory leaks.** Every subscription, timer, socket, and goroutine has a defined end: effects return cleanups, timers are cleared, goroutines terminate with their context.
- **No secrets.** Env vars or placeholders only; never echo credentials in errors (the GitHub token is never returned by the API).

## Documentation

`docs/` is committed and maintained alongside code — update the relevant page whenever behavior changes (the hub is [docs/index.md](docs/index.md)). CLAUDE.md documents the repository itself and must stay in sync with the layout and toolchain as they evolve.

## Need help?

- `thoth doctor` (or `make doctor`) diagnoses your setup — run it first when something misbehaves.
- Open an issue for bugs and feature requests; questions are welcome in the issue tracker too.
