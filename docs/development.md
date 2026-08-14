# Development

## Toolchain

| Tool | Version |
|---|---|
| Go | 1.26.1 (authoritative: `go.mod`) |
| Node | 24 LTS |
| pnpm | 11.7+ |

Frameworks and libraries: Echo 4.15, Cobra 1.10, gorilla/websocket 1.5, modernc.org/sqlite 1.56, fsnotify 1.10, React 19.2, TypeScript 6.0, Vite 8.2, Tailwind 4.3 (authoritative: `go.mod` / `web/package.json`).

## Setup

```sh
git clone <repo>
make web      # frontend build + embed sync — required before go build/test
make build    # bin/thoth
```

## Commands

Run `make help` for the full self-documenting list.

| Command | Purpose |
|---|---|
| `make dev` | Vite (HMR) + Go server together; Ctrl+C stops both |
| `make dev-web` / `make dev-server` | Frontend only / backend only |
| `make web` | `pnpm install --frozen-lockfile` + build + sync into `internal/webui/dist` |
| `make web-sync` | Fast path: ensure the embed exists without reinstalling |
| `make build [VERSION=…]` | Release binary with version stamping |
| `make release [VERSION=…]` | Cross-compile all five targets into `dist/` |
| `make install` | Everything: frontend deps + build + binary into `$(GOBIN)` |
| `make install-bin [PREFIX=…]` | Copy the binary system-wide (default `/usr/local/bin`) |
| `make test` / `make race` / `make cover` / `make lint` / `make fmt` | Quality gates |
| `make check` | Everything CI runs, locally |
| `make doctor` / `make init` / `make run` / `make clean` | Ops |

## Gates (every commit must pass)

1. `gofmt -l` empty · `go vet ./...` clean
2. `go test -race ./...`
3. Coverage ≥ **80%** on `internal/` + `cmd/` (CI-enforced):

```sh
go test -coverprofile=coverage.out ./internal/... ./cmd/...
go tool cover -func=coverage.out | tail -1
```

4. Cross-compiles: darwin amd64/arm64, linux amd64/arm64, windows amd64
5. Frontend: `pnpm exec tsc --noEmit` · `pnpm run lint` · `pnpm exec vitest run` · `pnpm run build`

## CI

`.github/workflows/quality.yml` — the five quality gates: `backend-test` (make web → vet → race → coverage gate) and `backend-lint` (golangci-lint), plus `frontend-test` / `frontend-lint` / `frontend-typecheck`, with a 10-minute timeout each; the Go jobs share the toolchain + embed preamble via the `.github/actions/setup-go-web` composite action and the frontend jobs share the install preamble via `.github/actions/setup-web` (after checkout, since same-repo actions resolve only once checked out). `.github/workflows/ci.yml` runs on pushes to `main`: the shared quality gates, then `build-linux` / `build-darwin` / `build-windows` (the 5 targets, each on its native OS) and `frontend-build`; `.github/workflows/ci-pr.yml` runs the same quality gates on PRs targeting `main` (no builds). Both finish with `final-gate.yml`, the single required check for branch protection — it always renders a per-job report into the Actions step summary (success or failure) and fails itself unless every job succeeded.

## Rules that keep the codebase healthy

- Develop directly on `main`; conventional commits
- Code rules, memory-safety rules, and token-efficiency rules live in the root `CLAUDE.md` — read it before coding
- Design authority: `docs/superpowers/specs/` (untracked working docs — never commit)
- `docs/` (this documentation set) is committed and maintained alongside code
- Generated output (`bin/`, `web/dist/`, `internal/webui/dist/`, `node_modules/`, `*.db`) is never committed
- No secrets anywhere — env vars or placeholders only
