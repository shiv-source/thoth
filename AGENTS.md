# Thoth

Local-first personal knowledge base. Go/Echo backend drives a native agent library (direct-to-provider, no CLI spawn) over a plain-markdown wiki; React/TS/Tailwind dashboard embedded in one binary.

Rules live in the `code-rules` skill — load it before writing or changing code. Wiki rulebook: `~/.thoth/wiki/CLAUDE.md` (templated at `internal/wiki/templates/CLAUDE.md`). Toolchain: `docs/development.md` (authoritative in `go.mod` / `web/package.json`).

## Routing (go straight to the target, don't explore)

Open only the package your task touches; cite `file:line`. Full detail: `docs/components.md` (Go) · `docs/frontend.md` (web/src). If files/dirs change, update this map in the same commit.

```
agent/                    native agent library; no internal/* imports
  tools/                  common wiki-agnostic tools + FS seam
  provider/               anthropic + openai (OpenAI-compatible: DeepSeek, Qwen, GLM, Grok)
  transport/ model/ events/ git/
cmd/thoth/main.go         thin entrypoint
internal/
  agent/                  host: client, context, host, system, history, tools.go
    tools/                wiki-specific (note, notes, ops, tree, discover)
  api/                    Echo; chat.go = WS (/ws), server.go = wiring, one handler file per domain
  cli/                    Cobra: root, serve, init, version, doctor
  wiki/                   file contract: ParseNote, SafePath, Tree, Rulebook, Save/Bookmark (capture + links lists)
  index/                  SQLite WAL + FTS5 + watcher (derived); CountByPrefix backs the capture inbox badge
  store/                  conversations/messages + providers + llm_models + sync_providers + connections + sync_push_history; migrations 0001–0013
  settings/ config/ doctor/ webui/          (see docs/components.md)
  sync/                   multi-provider sync engine: git/s3/local drivers + restore capability + auto-sync scheduler over sync_providers + sync_connections
  retention/              chat-history retention scheduler: purges conversations older than the configured window (default 7d, 0 disables)
  github/                 GitHub REST client (identity + repos) — consumed by the sync git driver
web/src/                  React 19 · TS strict · antd 6 · pnpm
  app/ App.tsx · api/ client.tsx (axios+zod) · ws/ chat/protocol/events
  hooks/ store/slices/ pages/ components/ shared/ test/   (one slice per feature)
browser-ext/              MV3 Chrome+Firefox capture extension: src/core = shared logic (api, server discovery, menus, badge, draft), src/{chrome,firefox} = thin manifests+entries, src/popup = React+antd draft form; dist/ is gitignored (make ext-build)
docs/                     index.md hub; docs-site/ renders (never forked)
.claude/skills/           go, react, git-workflow, code-quality, code-rules
scripts/                  pr, lib-codegraph, main-guard, token-guard
```

## Commands

```sh
make help         # authoritative target list
make dev          # Vite HMR + Go server (air, serve --dev, :8334)
make build        # bin/thoth · make release VERSION=vX.Y.Z # 5 cross-compile targets
make check        # everything CI enforces, locally
make doctor       # diagnose the local setup
make docs-dev     # Docusaurus dev server · make docs-build # production build
make ext-build    # browser extension → browser-ext/dist/{chrome,firefox}
pnpm dev|test|typecheck|lint|format   # pnpm only, never npm
pnpm --filter thoth-ext <cmd>         # extension: lint|typecheck|test|build
go test ./internal/<pkg>/ -run TestX -v   # focused while iterating; full suite before commit
```

Frontend: `pnpm <cmd>` from repo root (workspace proxy) or `cd web` — **pnpm only**, never npm. Docs site: `pnpm --filter thoth-docs <cmd>`. Root `pnpm-lock.yaml` committed.

## Runtime data

`~/.thoth/`: `thoth.db` (settings KV) + `wiki/` (default). `serve --dev` isolates under `~/.thoth/dev/`. Localhost-only, no auth.

## Skills

`.claude/skills/` = go (backend), react (frontend), git-workflow (contribution workflow), code-quality (pre-PR gates), code-rules (rules — loaded before writing code). Procedures here; `docs/` owns detail.

## Rules

Code rules, memory/no-leak, token-efficiency, invariants, repo rules: [`code-rules` skill](.claude/skills/code-rules/SKILL.md) — load before writing code, not for read-only work. Token-efficiency: open only the target package, cite `file:line`, don't re-read what you wrote.

<!-- CODEGRAPH_START -->
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tool** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them, including dynamic-dispatch hops grep can't follow. Name a file or symbol in the query to read its current line-numbered source. If it's listed but deferred, load it by name via tool search.
- **Shell** (always works): `codegraph explore "<symbol names or question>"` prints the same output.

If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
<!-- CODEGRAPH_END -->

