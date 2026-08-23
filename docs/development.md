# Development

## Toolchain

| Tool | Version |
|---|---|
| Go | 1.26.1 (authoritative: `go.mod`) |
| Node | 24 LTS |
| pnpm | 11.7+ |

Frameworks and libraries: Echo 4.15, Cobra 1.10, gorilla/websocket 1.5, modernc.org/sqlite 1.56, fsnotify 1.10, React 19.2, TypeScript 6.0, Vite 8.2, Tailwind 4.3 (authoritative: `go.mod` / `web/package.json`).

The frontend lives in a **pnpm workspace** at the repo root (`pnpm-workspace.yaml` registers `web/` and `docs-site/`): dependencies install and lock at the root (`pnpm-lock.yaml`), and the root scripts proxy into the web app — `pnpm dev` / `pnpm test` / `pnpm typecheck` / `pnpm lint` / `pnpm format` work from anywhere. `.npmrc` pins `save-exact`. The docs site (Docusaurus) runs through `pnpm --filter thoth-docs <cmd>` (`start`/`build`), or `make docs-dev` / `make docs-build`.

## Setup

```sh
git clone <repo>
pnpm install # workspace deps (root lockfile)
make web      # frontend build + embed sync — required before go build/test
make build    # bin/thoth
```

## Commands

Run `make help` for the full self-documenting list.

| Command | Purpose |
|---|---|
| `make dev` | Vite (HMR) + Go server with hot reload (air) on `:8334` via `serve --dev`; Ctrl+C stops both |
| `make dev-web` / `make dev-server` | Frontend only / backend only |
| `make web` | `pnpm install --frozen-lockfile` + build + sync into `internal/webui/dist` |
| `make web-sync` | Fast path: ensure the embed exists without reinstalling |
| `make build [VERSION=…]` | Release binary with version stamping |
| `make release VERSION=…` | Cross-compile all five targets into `dist/` (fails without a real `VERSION` — never ships `dev`-stamped binaries) |
| `make install-bin [PREFIX=…]` | Copy the binary system-wide (default `/usr/local/bin`) |
| `make test` / `make race` / `make cover` / `make lint` / `make fmt` | Quality gates |
| `make check` | Everything CI runs, locally |
| `make doctor` / `make init` / `make run` / `make clean` | Ops |
| `make run-fast` | Rebuild Go only and serve, reusing the existing embed (run `make web` after frontend edits) |
| `make docs-dev` / `make docs-build` | Docusaurus dev server (hot reload) / production build of the docs site |

## Gates (every commit must pass)

1. `gofmt -l` empty · `go vet ./...` clean
2. `go test -race ./...`
3. Coverage ≥ **90%** on `agent/` + `internal/` + `cmd/` (CI-enforced): `make cover` — the single gate, shared by the Makefile and CI

4. Cross-compiles: darwin amd64/arm64, linux amd64/arm64, windows amd64
5. Frontend: `pnpm typecheck` · `pnpm lint` · `pnpm test` · `pnpm run build`
6. Tooling JS: `node --test .github/actions/issue-labels/test/*.test.mjs` (the issue-labels action suite; also a CI job)

## Dev tools

- **Husky** pre-commit: `lint-staged` runs `eslint --fix` + prettier over staged `web/src` files and `golangci-lint --fix` over staged Go files; when Go files are staged it also gates on `go vet ./...`, `golangci-lint run`, and `go test ./...`
- **Prettier** (`.prettierrc` + `.prettierignore`): repo style — single quotes, no semicolons, no trailing commas, tab width 4, print width 120; only `web/src/` is formatted, everything else is ignored
- `eslint-config-prettier` disables the eslint rules that would fight prettier
- `pnpm format` rewrites `web/src` to the canonical style (one-shot)
- Hooks find node/pnpm even from IDE/GUI commits via `~/.config/husky/init.sh` (loads nvm)

## CI

`.github/workflows/quality.yml` — reusable workflow with the quality gates: `backend-test` (make web → vet → race → `make cover`) and `backend-lint` (golangci-lint), plus `frontend-test` / `frontend-lint` / `frontend-typecheck` and `issue-labels-test` (`node --test` on the `.github/actions/issue-labels` JS suite), and `docs-build` (`pnpm --filter thoth-docs build` — the Docusaurus site must compile), with a 10-minute timeout each; the Go jobs share the toolchain + embed preamble via the `.github/actions/setup-go-web` composite action and the frontend jobs share the install preamble via `.github/actions/setup-web` (installs at the workspace root against the root `pnpm-lock.yaml`; after checkout, since same-repo actions resolve only once checked out). Caching: `setup-go` and `setup-node` keep the Go build/module cache and the pnpm store warm by default, and the golangci-lint analysis cache is persisted via `actions/cache` keyed on `.golangci.yml`. `.github/workflows/ci.yml` runs on pushes to `main`: the shared quality gates, then `build-linux` / `build-darwin` / `build-windows` (the 5 targets, each on its native OS) and `frontend-build`; `.github/workflows/ci-pr.yml` runs the quality gates on PRs targeting `main` (no builds) plus `pr-assignee.yml`, which auto-assigns the PR's committers (adds, never replaces). CI is path-aware: `quality.yml`'s own `changes` job diffs the PR against its base and sets the areas to run — `backend` (Go files, `go.mod`/`go.sum`, `.golangci.yml`, plus `Makefile`/`scripts/`/`.husky/`/`.github/workflows|actions/` changes), `frontend` (`web/`, root pnpm/package files, plus the same tooling set), `docs` (`docs-site/` or `docs/` changes), and `issue_labels` (`.github/actions/issue-labels/` changes); on a push to `main` (not a PR) every area runs. A gate only runs when its area was touched — a docs-only PR runs only the `docs` gate; a Go-only PR skips the frontend and docs jobs; a CI/tooling PR runs everything. `assignee` and `final-gate` are never gated. Both workflows finish with `.github/workflows/final-gate.yml` (reusable, takes the caller's job id as `wrapper` so the gate excludes itself and its caller from the job list), the single required check for branch protection — it always renders a per-job report into the Actions step summary (success or failure), fails itself unless every job succeeded or was skipped (skipped = gated off for an area the PR didn't touch, counts as passing), and on PR-triggered runs mirrors the report onto the PR as a marker-tagged comment (updated in place on each run). `.github/workflows/issue-labels.yml` runs on every issue open/edit and applies the three-tier bare-minimum labels (type, priority, areas) straight from the form answers via the reusable `.github/actions/issue-labels` composite action — a dependency-free JS parser (whitelist-driven `config.json`, so a body can't inject arbitrary labels) that POSTs only the missing labels, never removes any (human-added labels survive), and skips blank issues. Every workflow declares top-level `permissions`: the callers `ci.yml` / `ci-pr.yml` cover the union (`contents: read` for checkout, plus `actions: read` + `pull-requests: write` — a caller must grant at least everything its reusable workflows request), `quality.yml` needs `contents: read`, and `final-gate.yml` needs `actions: read` + `pull-requests: write` (jobs API + PR comment).

## Rules that keep the codebase healthy

- Branch workflow: never commit to `main` directly — sync and branch first (`git switch main && git pull --ff-only && git switch -c <type>/<scope>/<slug>`), conventional commits on the branch, squash-merge back via PR
- PRs follow `.github/pull_request_template.md` — conventional-commit title, full summary (bullet points when it helps), files changed, and the verification checklist
- Code rules, memory-safety rules, and token-efficiency rules live in the `code-rules` skill (`.claude/skills/code-rules/SKILL.md`) — load it before coding
- `docs/` (this documentation set) is committed and maintained alongside code; `docs-site/` renders it (docs is the single source — never fork content into the site)
- Generated output (`bin/`, `web/dist/`, `internal/webui/dist/`, `docs-site/build/`, `.docusaurus/`, `node_modules/`, `*.db`) is never committed
- No secrets anywhere — env vars or placeholders only

## Debugging chat turns

Chat turns run entirely in-process: `internal/agent` builds an `agent.Agent` per turn, and the only network hop is the provider stream. There is no subprocess or pool to inspect — cancel/supersede/shutdown abort a turn by cancelling its context, and the loop is bounded by `MaxIterations` (default 25), so a model stuck requesting tools ends with an explicit error event instead of hanging. To debug a failing or hanging turn:

- `thoth doctor` — the **provider** check probes the selected model's provider endpoint with the resolved credential and names the failure (401 bad key, 429 rate limited, timeout, 5xx).
- `thoth serve --dev` — isolates data under `~/.thoth/dev/`; watch the structured server log, which `serve` wires to the agent via `WithLogger`, so loop diagnostics and turn errors land there with their context (`err`).
- The doctor's hidden `--provider-base-url` flag (test-only) points the provider probe at a local or alternate endpoint for offline investigation.
