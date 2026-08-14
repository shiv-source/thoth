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

| Command | Purpose |
|---|---|
| `make web` | `pnpm install --frozen-lockfile` + build + sync into `internal/webui/dist` |
| `make build` | web + `go build -o bin/thoth` |
| `make run` | build + `thoth serve` |
| `make test` | `go test ./...` |
| `make race` | `go test -race ./...` |
| `make lint` | golangci-lint + `pnpm lint` + `tsc --noEmit` |
| `make clean` | remove `bin/` + embedded dist |

Frontend day-to-day: `cd web && pnpm dev` (Vite proxies `/api` and `/ws` to the running server).

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

`.github/workflows/ci.yml` — backend job (make web → vet → race → coverage gate → golangci-lint → cross-compile matrix) and frontend job (install → tsc → lint → vitest → build), both on every push and PR.

## Rules that keep the codebase healthy

- Develop directly on `main`; conventional commits
- Code rules, memory-safety rules, and token-efficiency rules live in the root `CLAUDE.md` — read it before coding
- Design authority: `docs/superpowers/specs/` (untracked working docs — never commit)
- `docs/` (this documentation set) is committed and maintained alongside code
- Generated output (`bin/`, `web/dist/`, `internal/webui/dist/`, `node_modules/`, `*.db`) is never committed
- No secrets anywhere — env vars or placeholders only
