# Thoth — Personal Knowledge Base with Claude

- **Date:** 2026-08-14
- **Status:** Design approved (this document is the source of truth for the implementation plan)
- **Repo:** `/Users/shivkumar/projects/wiki` (monorepo)
- **License:** MIT

## 1. Overview

Thoth is an open-source, local-first personal knowledge base. The user interacts with a web UI (React, Tailwind) served by a Go/Echo backend, which drives the Claude Code CLI in headless mode to answer questions from, and save notes to, a plain-markdown wiki directory. All knowledge is plain markdown on disk; the app adds a SQLite full-text index on top.

### Goals

- One organized home for all day-to-day knowledge: meeting notes, project notes, system/server setup docs, tooling knowledge, important links (GitHub repos, articles, tools), TODOs, and quick capture.
- Ask questions in the UI; Claude answers from the wiki and saves new content in the organized manner, following a rulebook (`CLAUDE.md`) that ships as an installable template.
- Claude Code in the terminal remains a fully supported second interface to the same wiki directory.
- Open source and installable by anyone: one static binary, one app directory (`~/.thoth`), no auth, localhost-only by default.

### Non-goals (YAGNI, decided explicitly)

- No multi-user support, no authentication, no cloud sync, no deployment story. Local single-user only.
- No mobile app. No React SSR — the Go server serves the built SPA (single-user local app; SSR buys nothing).
- No multi-wiki management: one wiki path at a time; `thoth.db` indexes the currently configured wiki. Switching paths triggers a rebuild.
- No per-folder `INDEX.md` files in the wiki. SQLite FTS serves the app's search; terminal Claude Code relies on descriptive filenames and grep. One indexing system, no dual maintenance.
- No secrets in the wiki, ever. Setup notes use placeholders (`<db-password>`).

## 2. System architecture

Two layers, cleanly separated:

- **Knowledge layer** — plain markdown in a wiki directory (default `~/.thoth/wiki`, configurable via settings). Owns the folder structure, naming conventions, frontmatter, and the `CLAUDE.md` rulebook. Exists independently of the app; it is its own git repository with a private remote.
- **App layer** — the Thoth monorepo (this repo): Go/Echo server + React SPA + SQLite index + Claude CLI integration. Everything the app creates lives under `~/.thoth/`:

```
~/.thoth/
├── config.toml      ← settings (wiki path, host/port, claude binary, permission mode, model)
├── thoth.db         ← SQLite index (metadata + FTS5) for the currently configured wiki
└── wiki/            ← default knowledge base (configurable; its own git repo)
```

### Data contract

**Files are the source of truth. `thoth.db` is derived data** — deleting it costs only a reindex. The wiki is never dependent on the app: both interfaces (app-spawned Claude CLI sessions, and Claude Code run directly in the terminal) read and write the same markdown tree under the same rules.

## 3. Knowledge layer

### 3.1 Directory layout (default wiki)

```
wiki/
├── CLAUDE.md          ← the rulebook; loaded automatically by Claude Code when
│                        cwd is the wiki dir (app-spawned sessions set this too)
├── inbox/             ← quick capture, filed later per the rulebook
├── meetings/          ← one file per meeting: 2026-08-14-standup.md
├── projects/<name>/   ← one folder per project; project.md is the anchor
├── links/             ← bookmarks.md master list + per-link notes when deserved
├── setup/             ← one file per machine; setup/servers/<name>.md per server
├── knowledge/         ← one topic per file: software & tooling knowledge
├── todos/             ← TODO.md — the single running task list
└── daily/             ← one file per day: 2026-08-14.md quick capture
```

New top-level domains are added by convention, not enumeration: add a folder, and if it needs rules, extend `CLAUDE.md`. The structure grows with its user.

### 3.2 Conventions

- **Filenames:** kebab-case. Time-based notes get date prefixes: `meetings/2026-08-14-standup.md`, `daily/2026-08-14.md`.
- **Frontmatter:** every note carries minimal YAML — `title`, `date`, `tags`, `type` — so the indexer extracts metadata deterministically and the rulebook makes Claude write it on every save.
- **links/**: `bookmarks.md` is the master list, grouped by category, one line per link with a one-word "why". A link gets its own note file only when it deserves a full note.
- **projects/<name>/**: `project.md` holds overview + current status; extra files are free-form but descriptively named.
- **setup/**: one file per machine or server; secrets are always placeholders.
- **knowledge/**: one topic per file.
- **todos/TODO.md**: the single source of truth for tasks, with Now/Next/Someday sections; items link back to the meeting/project/note they came from.
- **inbox/**: anything unfiled; cleanup moves items to their proper homes.
- **daily/**: one file per day for loose thoughts and capture.

### 3.3 The rulebook (`CLAUDE.md`, wiki root)

Contents: the folder map; naming + frontmatter rules; the **save protocol** (choose folder → filename → write with frontmatter → never store secrets); the **retrieval protocol** (how to answer efficiently from the tree); the **health rules** (one TODO list, inbox cleanup, placeholders not credentials). Target: ~40 lines, crisp.

The rulebook ships as an **installable template** scaffolded by `thoth init` (and by first-run of the app). The template in the app repo and the validation in `internal/wiki` are generated from the same source — see §8 (DRY).

## 4. App architecture

### 4.1 Monorepo layout

```
wiki/                        ← this repo
├── go.mod                   ← Go module at root (latest Go)
├── cmd/thoth/               ← Cobra CLI: serve | init [path] | version (+ shell completion)
├── internal/
│   ├── claude/              ← the blast wall: the only package that knows CLI flags and parsing
│   ├── index/               ← SQLite (modernc.org/sqlite, WAL) + FTS5 + fsnotify watcher
│   ├── wiki/                ← scaffold from template, path validation, note read, frontmatter validation
│   ├── api/                 ← Echo routes: WS chat + REST (conversations, search, notes, settings, health)
│   └── config/              ← ~/.thoth/config.toml load/save
├── web/                     ← React 19 + TypeScript (strict) + Vite + Tailwind; embedded via embed.FS
├── docs/                    ← specs, README material
└── LICENSE                  ← MIT
```

### 4.2 Components

- **`cmd/thoth` (Cobra):** `thoth serve` runs the app (single static binary — the built `web/` is `embed.FS`'d in at compile time; no Node runtime needed to run it). `thoth init [path]` scaffolds a wiki directory with the folder skeleton + rulebook template (defaults to `~/.thoth/wiki`). `thoth version` prints version info.
- **`internal/claude`:** the isolated seam (see §4.4).
- **`internal/index`:** full scan on startup → FTS5 index (title-weighted, ranked, snippets). fsnotify keeps it current when any interface writes files. Wiki path change in settings → drop + rebuild. Only well-formed notes (valid frontmatter) are indexed; partial or malformed files are skipped, logged, and retried on the next scan.
- **`internal/wiki`:** owns the file contract. Scaffolding, path validation (no writes outside the wiki root), note reads, and frontmatter validation — enforced in code, not only in the rulebook, so the app cannot write a malformed note.
- **`internal/api`:** Echo routes. See §4.3.
- **`internal/config`:** TOML load/save with defaults and validation.
- **`web/`:** the dashboard — chat, search, notes viewer, settings.

### 4.3 Transport

**WebSocket is the live chat channel only.** Chosen over SSE because the UI must *send while receiving* (cancel mid-turn, queue a message mid-stream) and the underlying pipe to the CLI is bidirectional.

```
client → server:  send {text} | cancel | resume {conversationID}
server → client:  assistant_start | assistant_delta {text} | tool_activity {tool, detail}
                  | turn_done | error {message}
```

**REST for everything else** (no streaming data over the socket):

| Route | Purpose |
|---|---|
| `GET/POST /api/conversations` | list / create conversations |
| `GET /api/conversations/{id}` | conversation history |
| `GET /api/search?q=` | FTS search: ranked results with paths + snippets |
| `GET /api/notes/{path}` | read a note |
| `GET /api/wiki/tree` | browse the wiki tree |
| `GET/PUT /api/settings` | read / update settings |
| `GET /api/health` | app + CLI availability |

### 4.4 Claude CLI integration (the blast wall)

Everything the rest of the app needs is one interface:

```go
package claude

type Client interface {
    // Start spawns a claude process for sessionID, streams parsed events to w.
    // Cancelling ctx kills the process group — this is the cancel path.
    Start(ctx context.Context, sessionID, prompt string, w EventWriter) error
}
```

The implementation spawns `claude` in headless print mode with JSON streamed output and a per-conversation session ID, with **cwd set to the wiki path** so `CLAUDE.md` auto-loads and every read/write lands in the wiki. Stdout lines are parsed into typed events and forwarded to the WebSocket.

**Exact CLI flags are deliberately not pinned in this spec.** The CLI's flag surface changes between versions. Step one of implementing `internal/claude` is reading `claude --help` on the current CLI and encoding the verified flags in `internal/claude/client.go` — the only file a CLI upgrade can break. Permission mode (how the spawned CLI may edit wiki files unattended) is a setting with a safe default, verified against the real CLI first.

**Process hygiene:** cancel = context cancellation → kill the process group; server shutdown reaps all child processes; a watchdog reports a zombie process rather than leaving one silently.

### 4.5 Indexing

- Startup: full scan of the wiki path → SQLite (WAL mode) with FTS5.
- fsnotify: incremental updates when files change — regardless of which interface wrote them (terminal Claude Code edits stay in sync).
- Settings wiki-path change → rebuild for the new path.
- Only the Go process writes `thoth.db`.

## 5. Data flow

```
ASK:    browser ──WS──▶ Go api ──spawn──▶ claude CLI (cwd = wiki path)
        ◀────── stream-json parsed → typed WS events ◀──────
        Claude reads CLAUDE.md + files to answer.

SAVE:   "save this link" is just a prompt. Claude follows the rulebook
        (folder → filename → frontmatter → write). No special app code —
        fsnotify sees the new file and indexes it within seconds.

INDEX:  startup full scan → FTS5. fsnotify keeps it current.
        Path change → rebuild.

SEARCH: GET /api/search?q= → FTS5 → UI shows ranked results with paths
        and snippets → GET /api/notes/{path} to view.
```

## 6. Settings

`~/.thoth/config.toml`, editable in the UI:

| Key | Default | Purpose |
|---|---|---|
| `wiki_path` | `~/.thoth/wiki` | the knowledge base directory |
| `host` / `port` | `127.0.0.1` / `8333` | bind address (localhost by default, even with no auth) |
| `claude_bin` | resolved from `PATH` | path to the Claude Code CLI |
| `permission_mode` | safe default, verified at implementation | how spawned sessions may edit wiki files unattended |
| `model` | (unset = CLI default) | optional model pass-through |

Nothing else. YAGNI.

## 7. Error handling

- **CLI missing** → `/api/health` reports it; the UI shows a setup screen instead of a dead chat box.
- **CLI crash / nonzero exit** → typed `error` WS event; the turn is marked failed in conversation history.
- **Cancel mid-write** → process killed; the indexer skips malformed partials, logs, and retries on the next scan.
- **SQLite** → WAL mode; single writer (the Go process).
- **Reconnect** → client sends `resume`; server replays in-flight turn state so the UI re-syncs.
- **First run** → no wiki dir? Auto-scaffold via the `thoth init` flow — never a silent empty app.
- **Path safety** → `internal/wiki` resolves and validates every path against the wiki root before any read/write.

## 8. Engineering standards

Written to the standard of a senior engineer; these are requirements, not aspirations.

### Go (the core)

- One package = one purpose; small files; interfaces only at real seams (`claude.Client` is the big one).
- `context.Context` propagated everywhere — cancellation is what makes the stop button work.
- Errors wrapped (`%w`) with sentinel errors; no panics in library code.
- Dependency injection via constructors; no package-level mutable globals.
- **Unit tests for every package:** table-driven; a fake Claude client (script-based, no network, no real binary); SQLite `:memory:` for indexer tests; `httptest` for REST handlers; a WS test client for the chat handler. Coverage floor: 80% on `internal/`, enforced in CI.
- Static analysis in CI: `go vet` + `staticcheck` + `golangci-lint` + `go test -race`.

### TypeScript / React

- `strict: true`; `any` banned by lint; zod validation at the API boundary.
- Tailwind CSS for the dashboard; shared components and hooks (`useChat`, `useSearch`) instead of duplicated logic.
- Vitest + React Testing Library for components and hooks.

### DRY — each rule lives in exactly one place

- Wiki conventions: the rulebook template and `internal/wiki` validation are generated from one shared source (they cannot drift).
- CLI flags: one file, `internal/claude/client.go`.
- WS protocol message types: shared, typed, single definition.

### CI & tooling

- GitHub Actions: lint → unit tests (+race) → build with embedded web.
- Conventional commits; squash-merge PRs; superpowers code review before milestones.
- Latest versions of all dependencies; Dependabot (or equivalent) opens bump PRs, CI verifies each bump.

## 9. Open-source requirements

- **Name:** Thoth. MIT license, README (install, usage, config), CONTRIBUTING, design specs in `docs/`.
- **First-run experience:** install binary → `thoth init` (scaffolds `~/.thoth`) → `thoth serve` → open browser. Requires the Claude Code CLI installed and logged in; stated plainly in the README.
- **Single static binary** with the web build embedded; cross-compile targets: darwin/amd64+arm64, linux/amd64+arm64, windows/amd64.
- **Trademark hygiene:** the name avoids Anthropic trademarks; "uses the Claude Code CLI" appears only as an accurate description.

## 10. Success criteria

1. Ask a question in the UI chat → streamed answer grounded in the wiki.
2. Say "save this" in the UI chat → content lands in the right folder with valid frontmatter and is searchable within seconds.
3. Edit the wiki with Claude Code in the terminal → the app's search reflects it without a restart.
4. Fresh clone: build, `thoth init`, `thoth serve` → working app.
5. `go test ./...` green with ≥80% coverage on `internal/`; CI green on every push.

## 11. Implementation phases (for the plan)

1. Wiki template + rulebook + `thoth init`
2. Go skeleton: config, Cobra CLI, Echo server, embed, health
3. `internal/wiki`: scaffold, validation, reads
4. `internal/index`: SQLite + FTS5 + fsnotify + search API
5. `internal/claude`: verified flags, spawn/stream/cancel, WS chat end-to-end
6. React UI: chat, search, notes viewer, settings
7. Hardening: full test coverage, CI, README, cross-platform build
