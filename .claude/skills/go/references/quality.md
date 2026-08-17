# Quality gates — how this repo verifies work

## make check — everything CI enforces, locally
`make check` runs, in order: fmt, lint, race, cover, build.
CI (.github/ workflows) runs: vet → race → 80% coverage gate → lint →
5 cross-compiles → frontend.

## Coverage
- Floor: 80% on internal/ + cmd/ — CI-enforced (`make cover`)
- Table-driven tests; assert real outcomes, not mocks of yourself (CLAUDE.md)
- internal/claude tests use FakeClient — no test ever touches the real CLI

## Concurrency
- CI runs `go test -race` and it must stay green
- Shared state behind mutex/atomic; every goroutine ends via ctx/done-channel (CLAUDE.md memory rules)

## Lint
- golangci-lint v2 (.golangci.yml)
- Husky pre-commit: golangci-lint --fix on Go, and gates commits on go vet + golangci-lint + go test

## Cross-compile
- All five targets must build: darwin/linux × amd64/arm64, windows/amd64
- Process-group code is build-tagged in internal/claude (proc_unix.go / proc_windows.go)

## Dependency bumps
- `go get <pkg>@latest`; lockfiles (go.sum) committed; CI verifies every bump
- A bump must not lower the coverage floor or break any cross-compile target

## Commit hygiene
- Conventional commits on a branch; squash-merge via PR; ci-pr gates every PR to main
- Never commit to main directly (CLAUDE.md repo rules)

## Style rules (source: CLAUDE.md — the single home)
- %w error wrapping; structured slog with lowercase keys; no panics in library code
- Small functions (≤40 lines), ≤3 params, fail fast at the boundary
- Every behavior change ships with a test

Canonical: docs/development.md · CLAUDE.md

Stale if: the Makefile gains or loses check targets, the coverage floor
changes, or the CI workflow order (.github/) changes.
