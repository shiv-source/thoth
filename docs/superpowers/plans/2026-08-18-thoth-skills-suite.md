# Thoth Project Skills Suite Implementation Plan

> **Historical record** — this plan covers the original go/react implementation
> only. The suite later gained the `git-workflow` and `code-quality` skills, and
> the workflow counts grew to go: 11 / react: 7 (see the spec's decisions log,
> `docs/superpowers/specs/2026-08-17-thoth-skills-design.md`). The counts and
> verification expectations below reflect the state at implementation time.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the committed project skill suite — `.claude/skills/go` and `.claude/skills/react` — that gives future sessions procedure workflows plus distilled reference indexes, per the approved design.

**Architecture:** Markdown-only change. Each skill is a `SKILL.md` (procedures) plus `references/` (distilled indexes with canonical pointers). Rules stay in CLAUDE.md; `docs/` owns detail; skills own procedure + pointers. One pointer line is added to CLAUDE.md.

**Tech Stack:** Claude Code project skills (SKILL.md frontmatter + references), Markdown. All facts sourced from the source tree and `docs/` as of 2026-08-18.

**Spec:** `docs/superpowers/specs/2026-08-17-thoth-skills-design.md`

## Global Constraints

- **Branch:** work on `feat/project-skills` (exists; design doc committed at `ca85dcc`). Conventional messages + `Co-Authored-By: Claude <noreply@anthropic.com>` line.
- **NO AUTO-COMMITS:** the user approves every commit before it happens. Each task's Commit step stages the files, shows the diff summary, then pauses for approval — do not commit until the user says yes. The user may batch-approve several tasks' commits at once. The same gate applies to committing this plan file and to creating the PR (outward-facing).
- **Approach C hybrid:** every reference entry = `path` + `purpose` + `props/api` (or `api`) + `canonical` — never a re-telling of docs content.
- **Coverage rule:** every file in an indexed dir gets an entry, or a named "intentionally skipped" line. A missing entry reads as stale, not as never-indexed.
- **Every reference file ends with a `Stale if: …` line.**
- **Never duplicate a rule from CLAUDE.md — reference it.** Skills copy no rules; they point at them.
- **Workflow counts pinned by the spec:** go has 8 workflows, react has 6, names exactly as in the spec's §Workflow lists.
- **CLAUDE.md gets exactly one pointer line** — exact text in the spec's §Wiring. No other CLAUDE.md change. NOTE: `CLAUDE.md` had pre-existing uncommitted edits when this branch was created — read it before editing and place the line without disturbing them.
- **File trees for coverage checks** (from `ls` on 2026-08-18; if a file moved since, update the entry — pointer discipline, don't delete the entry):

```
web/src/components (source files; *.test.tsx are skipped per coverage rule):
  ActivityChart, Card, chartSetup, ChatActivityChart, ChatPanel, CodeBlock,
  Composer, CopyButton, dashboardMock, DashboardView, EmptyState, IconButton,
  Markdown, MessageItem, NavRail, NotesByFolderChart, NotesByKindChart,
  NotesView, NoteViewer, NotificationPanel, notifications, NotificationToasts,
  SearchPanel, SearchView, SettingsView, SetupScreen, Sidebar, Toast, Tooltip,
  TopBar, Tree, useThemeColors, WikiTree
web/src/hooks: useChat, useConversationRoute, useSearch, useView, useViewShortcuts
web/src/store/slices: chat, connection, conversations, health, notifications,
  searchHistory, settings
web/src/test: fakeWS, mockAxios, renderWithStore, setup
web/src/api: client.ts   web/src/ws: chat.ts
internal/: api, assets, claude, cli, config, doctor, github, index, settings,
  store, webui, wiki   plus cmd/thoth
```

---

### Task 1: go/SKILL.md — the backend procedure skill

**Files:**
- Create: `.claude/skills/go/SKILL.md`

**Interfaces:**
- Consumes: spec §SKILL.md anatomy + §Workflow lists (go 8); facts from `docs/components.md`, `docs/api.md`, `docs/schema.md`, `docs/indexing.md`
- Produces: the go skill; later go reference tasks must not contradict its workflow steps

- [ ] **Step 1: Write the skill**

````markdown
---
name: go
description: >-
  Thoth Go backend — REST endpoints, WS chat protocol, store migrations,
  claude CLI flags, settings keys, wiki contract, doctor checks, deps.
---

# Go backend (internal/* + cmd/*) — procedures & expertise

## When to use
- Adding or changing REST endpoints, handlers, or the Echo server (internal/api)
- Extending the WebSocket chat protocol (internal/api/chat.go — types are mirrored in web/src/ws/chat.ts)
- Schema work: store migrations, settings keys, index/FTS5 changes
- Touching the Claude CLI integration (internal/claude — the blast wall)
- Extending the wiki file contract (internal/wiki) or doctor checks (internal/doctor)
- Not note-taking behavior — that's the wiki rulebook (~/.thoth/wiki/CLAUDE.md), not this repo
- Not frontend work — use the `react` skill

## Key files
- internal/api/ — Echo server: server.go = Deps + New() wiring all routes; chat.go = Hub + WS protocol; one handler file per domain (notes.go, git.go, github.go, doctor.go, health.go, conversations.go, …)
- internal/claude/ — the blast wall: client.go = every CLI flag; persistent.go = process pool; events.go = stream parsing; proc_unix.go / proc_windows.go = process-group kill
- internal/wiki/ — the file contract: note.go (ParseNote), path.go (SafePath), scaffold.go, template.go (Rulebook), wiki.go (Wiki), templates/CLAUDE.md
- internal/index/ — SQLite + FTS5 + fsnotify watcher: index.go, sync.go, watcher.go
- internal/store/ — conversations/messages; migrations/ holds ALL DDL (0001–0007), applied in order
- internal/settings/ — the settings KV table (single source for user-facing settings)
- internal/github/ — GitHub identity (PAT) + git sync (client.go, repo.go, service.go)
- internal/doctor/ — the six shared install checks (CLI + Settings → Doctor tab)
- internal/config/ — bind constants (127.0.0.1:8333) + ExpandHome
- internal/assets/ — embedded models.json (the Settings model picker list)
- internal/webui/ — embedded frontend (generated by make web)
- cmd/thoth/ — thin main → internal/cli Execute()

## Workflows

### 1. Add a REST endpoint
1. Add the handler to its domain file (notes.go, git.go, …) or create one; handlers close over Deps
2. Register the route in internal/api/server.go (New)
3. Errors: return JSON {"error":"<msg>"} — 400 client, 404 not found, 500 always {"error":"internal error"} with details to the server log only
4. Add the test in the matching _test.go (httptest against New(deps)); table-driven
5. Update docs/api.md's endpoint table in the same commit
6. Request logging is automatic (internal/api/logging.go) — nothing to add

### 2. Extend the WS protocol
1. CHANGE BOTH SIDES: internal/api/chat.go (server frames) AND web/src/ws/chat.ts (client types) — server message types must match (CLAUDE.md invariant)
2. Read the semantics in docs/api.md first: supersede, cancel, resume (≤500-message ring), open, new_chat
3. Update docs/api.md's WS table in the same commit as the code
4. Test server side against Hub with a scripted client; client side with the fakeWS double
5. Grep both files for the old frame name — no stale frames left on either side

### 3. Add a store migration
1. List internal/store/migrations/ — the new file is 000N_<name>.sql with the next number
2. Go code issues no DDL of its own; the schema lives entirely in these files, gated on PRAGMA user_version
3. Add the table section to docs/schema.md in the same commit
4. Test: store_test.go pattern — open a temp db, migrate, exercise the new table
5. Reminder: thoth.db is derived data — deleting it is a supported upgrade path

### 4. Change claude CLI flags (BLAST WALL)
1. ALL CLI flags live only in internal/claude/client.go — verify against `claude --help` before changing anything
2. Never inline CLI flags outside this package; consumers see Client.Start only
3. Update docs/components.md's blast-wall section in the same commit
4. Run: go test ./internal/claude/ -race
5. If the CLI upgraded: check events.go's stream-json parsing still matches the new output

### 5. Add a settings key
1. Add the key constant + accessor in internal/settings/settings.go — the KV table needs no migration
2. Mirror the seed pattern (settings.DefaultWikiPath-style) if the key has a default
3. REST surface: extend GET/PUT /api/settings DTOs in internal/api (route wired in server.go) AND the zod schema in web/src/api/client.ts (both sides)
4. Update docs/schema.md's settings table and docs/api.md in the same commit
5. Test: settings repo test + the api settings test

### 6. Extend the wiki contract
1. Contract functions live in internal/wiki (note.go, path.go, scaffold.go, template.go)
2. Rulebook text == Rulebook() == internal/wiki/templates/CLAUDE.md — one source, templated
3. Every filesystem access routes through SafePath (absolute paths and .. rejected)
4. Notes require --- frontmatter with a title (ParseNote enforces it)
5. Update docs/knowledge-base.md and docs/components.md in the same commit

### 7. Bump a dependency
1. go get <pkg>@latest — go.mod is authoritative; lockfiles (go.sum) are committed
2. Run make check (fmt, lint, race, cover, build) — CI enforces every bump
3. The coverage floor is 80% on internal/ + cmd/ (make cover) — a bump can't lower it
4. If a framework version changed, update the version in CLAUDE.md's Toolchain section

### 8. Add a doctor install check
1. Add the check to internal/doctor/doctor.go's Run() suite — []Check{Name, OK, Message}
2. The same suite serves `thoth doctor` and GET /api/doctor — one implementation, no duplication
3. Doctor reads only: no migrations, no WAL pragma (internal/settings doc comment)
4. Test: table-driven doctor_test.go
5. Update docs/cli.md's doctor section in the same commit

## Gotchas
- Fresh clone: run make web before go build/test (embed)
- WS is chat-only transport; REST for everything else (CLAUDE.md invariant)
- ctx cancel = the stop button; every goroutine must end (CLAUDE.md memory rules)
- thoth.db is derived data; wiki files are the source of truth
- All five cross-compile targets must build — process-group code is build-tagged (proc_unix.go / proc_windows.go)
- Errors wrap with %w; slog keys are lowercase; no panics in library code

## Canonical docs
- docs/components.md — every Go package deep-dived
- docs/api.md — REST endpoints + WS protocol
- docs/schema.md — every table, column, settings key
- docs/indexing.md — FTS5 + watcher mechanics
- docs/architecture.md — the two layers, data contract

## Maintenance
Derived view — after a behavior change, update this skill + docs/ in the
same commit, then run `graphify update .`. When the Claude CLI upgrades,
verify the flags against `claude --help` (blast wall). Stale if: a
workflow's file paths stop resolving, or docs/ gains a workflow this
file lacks.
````

- [ ] **Step 2: Verify structure**

Run:
```bash
grep -c '^### ' .claude/skills/go/SKILL.md        # expect 8 (the workflows)
test -f .claude/skills/go/SKILL.md && grep -q 'Stale if:' .claude/skills/go/SKILL.md
```
Expected: `8`, and the Stale-if grep exits 0. Every file named in Key files must exist: `test -f internal/api/server.go && test -f internal/claude/client.go && test -f internal/wiki/note.go && test -d internal/store/migrations`.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/go/SKILL.md
git commit -m "feat(skills): add go backend procedure skill

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: go/references/packages.md — the package index

**Files:**
- Create: `.claude/skills/go/references/packages.md`

**Interfaces:**
- Consumes: spec §Reference file anatomy (4-line entries + coverage rule + Stale-if); `docs/components.md` package table
- Produces: the package index; Task 12's coverage check greps this file for every `internal/<pkg>` name

- [ ] **Step 1: Write the index**

````markdown
# Go packages (internal/* + cmd/thoth)

Each entry: path, purpose, api, canonical. Coverage: every package in
internal/ plus cmd/thoth — a missing package means this index is stale.

## cmd/thoth
- path: cmd/thoth/main.go
- purpose: Thin binary entrypoint — calls the CLI and exits
- api: main
- canonical: docs/components.md §cmd/thoth · cmd/thoth/main.go

## internal/cli
- path: internal/cli/
- purpose: Cobra commands — serve, init, version, doctor
- api: Execute(); serve helpers: loadConfig, ensureWiki, openStores, resolveClaudeBin, onSettingsSaved, serveUntilShutdown
- canonical: docs/components.md §internal/cli · internal/cli/root.go

## internal/claude — the blast wall
- path: internal/claude/
- purpose: The only package that knows CLI flags, stream parsing, and process kill
- api: Client (Start(ctx, sessionID, prompt, w EventWriter) error), CLIClient, PersistentClient, Event, ParseLine, FakeClient
- canonical: docs/components.md §internal/claude · internal/claude/client.go
- see: claude-blast-wall.md

## internal/wiki
- path: internal/wiki/
- purpose: The file contract — scaffolding, parsing, path safety, tree
- api: Scaffold, ParseNote, SafePath, Wiki (New/Exists/Read/Tree), Rulebook
- canonical: docs/components.md §internal/wiki · internal/wiki/wiki.go

## internal/index
- path: internal/index/
- purpose: SQLite + FTS5 search + fsnotify watcher
- api: Index, Sync, Watch, Search — Open, Upsert, Delete, DeletePrefix
- canonical: docs/components.md §internal/index · internal/index/index.go

## internal/assets
- path: internal/assets/
- purpose: Embedded static data — models.json → ModelOptions (Settings model picker)
- api: ModelOptions
- canonical: docs/components.md §internal/assets · internal/assets/assets.go

## internal/store
- path: internal/store/
- purpose: Conversations + messages (same db file); migrations/ = all DDL
- api: Store (Open, ListConversations, Messages, Close), EnsureMetadata, SetSyncResult
- canonical: docs/components.md §internal/store · internal/store/store.go

## internal/api
- path: internal/api/
- purpose: Echo server — routes, WS hub, handlers
- api: Deps, New(d Deps) *echo.Echo, Hub
- canonical: docs/components.md §internal/api · internal/api/server.go

## internal/config
- path: internal/config/
- purpose: Localhost bind constants + path helper
- api: DefaultHost, DefaultPort (127.0.0.1:8333), ExpandHome
- canonical: docs/components.md §internal/config · internal/config/config.go

## internal/doctor
- path: internal/doctor/
- purpose: The six shared install checks (CLI + Settings → Doctor tab)
- api: Run(ctx, dir, addr, log) []Check; Check{Name, OK, Message}
- canonical: docs/components.md §internal/doctor · internal/doctor/doctor.go

## internal/github
- path: internal/github/
- purpose: GitHub identity (PAT storage) + git sync
- api: Client, Auth, Repo, Service
- canonical: docs/components.md §internal/github · internal/github/service.go

## internal/settings
- path: internal/settings/
- purpose: The settings KV table — single source for user-facing settings
- api: Repo, OpenRepo(path); SyncEnabled/SyncState/SetSyncResult conveniences
- canonical: docs/components.md §internal/settings · internal/settings/settings.go

## internal/webui
- path: internal/webui/
- purpose: Embedded frontend (generated dist; //go:embed all:dist)
- api: Register
- canonical: docs/components.md §internal/webui · internal/webui/embed.go

Stale if: a new package appears in internal/, an export listed above is
renamed, or docs/components.md's package table gains a row this index
lacks.
````

- [ ] **Step 2: Verify coverage**

Run:
```bash
for pkg in api assets claude cli config doctor github index settings store webui wiki; do
  grep -q "internal/$pkg" .claude/skills/go/references/packages.md || echo "missing $pkg"
done
grep -q 'cmd/thoth' .claude/skills/go/references/packages.md && grep -q 'Stale if:' .claude/skills/go/references/packages.md
```
Expected: no output from the loop; the last grep exits 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/go/references/packages.md
git commit -m "feat(skills): add go package reference index

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: go/references/claude-blast-wall.md — the version-sensitive zone

**Files:**
- Create: `.claude/skills/go/references/claude-blast-wall.md`

**Interfaces:**
- Consumes: spec §Reference anatomy; `docs/components.md` §internal/claude; CLAUDE.md blast-wall rules (referenced, not copied)
- Produces: the blast-wall reference; packages.md's "see: claude-blast-wall.md" points here

- [ ] **Step 1: Write the reference**

````markdown
# The claude blast wall (internal/claude)

Every version-sensitive fact about the Claude Code CLI lives in exactly two
files: client.go (flags) and persistent.go (the process pool). A CLI upgrade
can only ever break this package — everything else is stable.

## client.go — the flags
- Per-turn: -p --output-format stream-json --verbose --session-id <id>
- Persistent mode: -p --input-format stream-json … --autocompact auto
- Permissions: --dangerously-skip-permissions by default, or --permission-mode <mode> when configured
- Optional --model from the settings table
- Spawn, stream scanning, cancel all live here; stderr is captured and appended to exit errors
- canonical: internal/claude/client.go · docs/components.md §internal/claude
- VERIFY AGAINST `claude --help` WHENEVER THE CLI UPGRADES — no other file may hold a flag

## persistent.go — the process pool
- PersistentClient: lazy-spawned CLI processes keyed by session id
- One dispatcher goroutine per process turns stdout lines into events for the in-flight turn; the CLI's result line ends it
- Cancel kills the process; the next turn respawns (no per-turn interrupt in the plain CLI)
- Idle eviction after 10 min; Flush on wiki-path change; Close on shutdown
- canonical: internal/claude/persistent.go

## events.go — stream parsing
- Tolerant parsing of stream-json lines into typed events
- Types: assistant_delta, thinking (thinking-only blocks → UI "Thinking…"), tool_activity, turn_done, error
- The raw stream is appended to ~/.thoth/stream-dump.json (rotated past 10 MB) for debugging
- canonical: internal/claude/events.go

## Process kill
- ctx cancel kills the process group (unix, proc_unix.go) or direct child (windows, proc_windows.go)
- Build-tagged — all five cross-compile targets must build
- canonical: internal/claude/proc_unix.go · internal/claude/proc_windows.go

## Client interface & FakeClient
- Client: Start(ctx, sessionID, prompt, w EventWriter) error — the only seam consumers see
- FakeClient replays scripted events and records calls — every consumer's tests use it, so no test ever touches the real CLI
- canonical: internal/claude/client.go · internal/claude/fake.go

Stale if: `claude --help` output changed and client.go wasn't updated, a
CLI flag appears outside this package, or events.go stops parsing a
stream-json line type.
````

- [ ] **Step 2: Verify pointers**

Run:
```bash
for f in client.go persistent.go events.go proc_unix.go proc_windows.go fake.go; do
  test -f "internal/claude/$f" || echo "missing internal/claude/$f"
done
grep -q 'Stale if:' .claude/skills/go/references/claude-blast-wall.md
```
Expected: no output from the loop; grep exits 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/go/references/claude-blast-wall.md
git commit -m "feat(skills): add claude blast wall reference

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: go/references/persistence.md — thoth.db and migrations

**Files:**
- Create: `.claude/skills/go/references/persistence.md`

**Interfaces:**
- Consumes: spec §Reference anatomy; `docs/schema.md` (every table) + `docs/indexing.md`
- Produces: the persistence reference; migration list must match `ls internal/store/migrations` (0001–0007)

- [ ] **Step 1: Write the reference**

````markdown
# Persistence — thoth.db, migrations, index

Everything lives in one SQLite file, ~/.thoth/thoth.db (WAL mode). The
schema is defined entirely by SQL migrations in internal/store/migrations/,
applied in filename order and gated on PRAGMA user_version (currently 7).
Go code issues no DDL of its own.

## Migrations (internal/store/migrations/)
- 0001_conversations.sql — conversations: id (v4 UUID, also the /chat/<id> URL id), title, created_at (UTC RFC3339), claude_session_id (seeded as id; rotated via --resume/--fork-session on stale locks)
- 0002_messages.sql — messages: id, conversation_id, role (user|assistant), content, created_at; replayed in id order
- 0003_notes.sql — notes: path PK, title (frontmatter, required), kind (from frontmatter type), tags, body, updated_at — derived from the wiki tree
- 0004_notes_fts.sql — notes_fts: FTS5 external-content index over notes, kept in sync by triggers; path UNINDEXED
- 0005_app_metadata.sql — app_metadata: single row (CHECK id = 1) — installation_id, created_at; seeded by EnsureMetadata on boot
- 0006_github_auth.sql — github_auth: PAT (plaintext, gh-CLI trust model, never serialized by the API) + identity + scopes
- 0007_settings.sql — settings: key/value table; new keys need no schema change

## Settings keys (0007)
- wiki_path — seeded to ~/.thoth/wiki (mirrors settings.DefaultWikiPath)
- model — the --model value on every CLI spawn; absent/empty keeps the CLI default
- github_sync_repo / github_sync_enabled / github_last_synced_at / github_sync_error — the sync switch + state

## Ownership (who owns which table)
- internal/settings → settings (runs no migrations, no WAL pragma — the doctor must never mutate a database it only reads)
- internal/github → github_auth
- internal/store → conversations, messages, app_metadata
- internal/index → notes, notes_fts

## The index (internal/index)
- WAL + schema migration on Open; Upsert/Delete/DeletePrefix keep FTS5 in sync via triggers
- Search(q, limit): bm25 ranking (title 8×), HTML-escaped snippets with safe <mark> highlights
- Sync(root, log): reconciles the index with the tree in one transaction; malformed notes skipped
- Watch(ctx, root, ix, log): fsnotify with 200 ms debounce, new-directory rescan

## Rules that matter here (source: CLAUDE.md)
- thoth.db is derived data — files are the source of truth; deleting thoth.db is a supported upgrade path
- IDs are RFC 4122 v4 UUIDs (google/uuid) because the Claude CLI requires UUIDs for --session-id
- Timestamps are stored UTC so ordering is chronological

Canonical: docs/schema.md · docs/indexing.md

Stale if: migration count ≠ 7, a new settings key is missing above, or a
table gains a column without a docs/schema.md update.
````

- [ ] **Step 2: Verify against the migrations dir**

Run:
```bash
ls internal/store/migrations | wc -l       # expect 7
grep -c '^- 000[1-7]_' .claude/skills/go/references/persistence.md   # expect 7
grep -q 'Stale if:' .claude/skills/go/references/persistence.md
```
Expected: `7`, `7`, grep exits 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/go/references/persistence.md
git commit -m "feat(skills): add persistence reference (migrations, settings, index)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 5: go/references/quality.md — the verification gates

**Files:**
- Create: `.claude/skills/go/references/quality.md`

**Interfaces:**
- Consumes: spec §Reference anatomy; CLAUDE.md §Commands + Code Rules (referenced, not copied); `.github/` CI order
- Produces: the quality reference; Task 12 checks it names every `make` gate

- [ ] **Step 1: Write the reference**

````markdown
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
````

- [ ] **Step 2: Verify sections**

Run:
```bash
grep -c '^## ' .claude/skills/go/references/quality.md   # expect 8
grep -q '80%' .claude/skills/go/references/quality.md && grep -q 'Stale if:' .claude/skills/go/references/quality.md
```
Expected: `8`, grep exits 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/go/references/quality.md
git commit -m "feat(skills): add quality gates reference

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 6: react/SKILL.md — the frontend procedure skill

**Files:**
- Create: `.claude/skills/react/SKILL.md`

**Interfaces:**
- Consumes: spec §SKILL.md anatomy + §Workflow lists (react 6); facts from `docs/frontend.md`, `docs/api.md`, and the file inventory in Global Constraints
- Produces: the react skill; later react reference tasks must not contradict its workflow steps

- [ ] **Step 1: Write the skill**

````markdown
---
name: react
description: >-
  Thoth React frontend — components, hooks, Redux slices, REST client,
  WS chat client, Vitest tests, Tailwind v4 design tokens.
---

# React frontend (web/src) — procedures & expertise

## When to use
- Adding or changing components, hooks, or Redux slices under web/src/
- Wiring new REST calls (web/src/api/client.ts — zod at the boundary)
- Touching the WS chat client (web/src/ws/chat.ts — types mirrored in internal/api/chat.go)
- Frontend tests (Vitest + the web/src/test doubles)
- Not backend work — use the `go` skill
- Not note-taking behavior — that's the wiki rulebook (~/.thoth/wiki/CLAUDE.md)

## Key files
- web/src/components/ — one component per file, co-located .test.tsx; App.tsx/main.tsx compose them
- web/src/hooks/ — useChat, useSearch, useConversationRoute, useView, useViewShortcuts
- web/src/store/ — Redux Toolkit: index.ts (makeStore), hooks.ts (typed hooks), slices/ (one per feature)
- web/src/api/client.ts — typed REST client (axios + zod)
- web/src/ws/chat.ts — ChatSocket: protocol frames, reconnect/resume
- web/src/test/ — mockAxios, fakeWS, renderWithStore, setup
- web/src/index.css — Tailwind v4 @theme tokens

## Workflows

### 1. Add a component
1. One component per file in web/src/components/<Name>.tsx; icons from lucide-react
2. Style with semantic tokens (bg-surface, text-ink, border-line) — no raw hex (see references/patterns.md)
3. Co-locate the test <Name>.test.tsx using the renderWithStore/mockAxios doubles
4. Hover hints use the Tooltip wrapper; icon-only buttons use IconButton
5. Update docs/frontend.md's component table AND references/components.md in the same commit

### 2. Add a Redux slice
1. Create web/src/store/slices/<name>Slice.ts — actions, selectors, thunks co-located
2. Wire it in web/src/store/index.ts (makeStore)
3. Use the typed hooks (useAppDispatch/useAppSelector from store/hooks.ts) — never bare useDispatch/useSelector
4. Only shared or screen-spanning state lives in the store; component-local state stays in hooks/components (docs/frontend.md)
5. Co-locate the slice test; update references/store.md in the same commit

### 3. Add a hook
1. New file web/src/hooks/useX.ts, exported as a named function
2. Every useEffect subscription/timer/socket gets a cleanup that runs on unmount (CLAUDE.md memory rule)
3. Co-locate the test; update references/hooks.md + docs/frontend.md in the same commit

### 4. Wire an API call
1. Add or extend the endpoint in web/src/api/client.ts with a zod schema — validation at the boundary, zero any (CLAUDE.md invariant)
2. Server side must match: use the `go` skill for internal/api; DTOs on both sides
3. Test with mockAxios — assert the parsed payload, not the transport
4. Update docs/api.md in the same commit

### 5. Test a component/slice
1. Use the doubles in web/src/test/ (mockAxios, fakeWS, renderWithStore, setup) — never hand-rolled mocks of the app itself
2. Assert real outcomes: what renders, what's dispatched, what the user sees
3. Run: pnpm test (Vitest) — pnpm only, never npm
4. Every behavior change ships with a test (CLAUDE.md)

### 6. Touch the WS client
1. CHANGE BOTH SIDES: web/src/ws/chat.ts (client types) AND internal/api/chat.go (server frames) — they must match
2. Frames: send/cancel/resume/open/new_chat out; assistant_*/tool_activity/turn_done/error in (docs/api.md)
3. Reconnect behavior: exactly once after 1 s, resume from onopen — changing it changes chat recovery semantics
4. Test with fakeWS; update docs/api.md in the same commit

## Gotchas
- pnpm only — never npm; the workspace lockfile (root pnpm-lock.yaml) is committed
- TS strict, zero any — eslint enforces; zod at the API boundary
- make web is REQUIRED before go build/test — frontend changes don't reach the binary without it
- WS is chat-only transport; REST for everything else
- Design tokens flip under prefers-color-scheme; dark mode follows the OS — no toggle
- Every useEffect has cleanup; no setInterval without clearInterval

## Canonical docs
- docs/frontend.md — structure, components, hooks, state, design system
- docs/api.md — REST endpoints + WS protocol (both sides)
- docs/architecture.md — the two layers

## Maintenance
Derived view — after a behavior change, update this skill + docs/ in the
same commit, then run `graphify update .`. Stale if: a workflow's file
paths stop resolving, or docs/frontend.md gains a component this skill's
workflow list doesn't mention.
````

- [ ] **Step 2: Verify structure**

Run:
```bash
grep -c '^### ' .claude/skills/react/SKILL.md        # expect 6 (the workflows)
grep -q 'Stale if:' .claude/skills/react/SKILL.md
test -f web/src/api/client.ts && test -f web/src/ws/chat.ts && test -d web/src/test
```
Expected: `6`, greps exit 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/react/SKILL.md
git commit -m "feat(skills): add react frontend procedure skill

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 7: react/references/components.md — the component index

**Files:**
- Create: `.claude/skills/react/references/components.md`

**Interfaces:**
- Consumes: spec §Reference anatomy; the 33-file coverage list in Global Constraints; docs/frontend.md §Components
- Produces: the component index; Task 12's coverage check greps this file for every name in the tree

- [ ] **Step 1: Write the index**

````markdown
# Components (web/src/components)

Scope: this directory only — App.tsx / main.tsx / index.css live in web/src/
and are covered by docs/frontend.md. Each entry: path, purpose, props/api,
canonical. Skipped: *.test.tsx — one co-located test per component, same
base name, skip by convention.

## App shell & navigation
## NavRail
- path: web/src/components/NavRail.tsx
- purpose: Persistent left rail switching dashboard/chat/notes/search views + settings button; routes through useView
- props/api: `NavRail()` — no props
- canonical: NavRail.tsx:14

## Sidebar
- path: web/src/components/Sidebar.tsx
- purpose: Left rail — branding, New chat button, day-grouped conversation history (ChatsList), health/version footer
- props/api: `Sidebar({ health, loading })`
- canonical: Sidebar.tsx:184

## TopBar
- path: web/src/components/TopBar.tsx
- purpose: Header bar — title, unread-count notification bell opening NotificationPanel, optional settings button
- props/api: `TopBar({ title, onOpenSettings? })`
- canonical: TopBar.tsx:9

## Chat
## ChatPanel
- path: web/src/components/ChatPanel.tsx
- purpose: Main chat view — owns the WebSocket, connection-status banner, thinking/tool status lines, message list, Composer
- props/api: `ChatPanel({ onOpenSettings })`
- canonical: ChatPanel.tsx:18

## Composer
- path: web/src/components/Composer.tsx
- purpose: Chat input form — Enter-to-send textarea, Send/Stop toggle
- props/api: `Composer({ onSend, onCancel, streaming })`
- canonical: Composer.tsx:3

## MessageItem
- path: web/src/components/MessageItem.tsx
- purpose: One chat message — user bubble plain-text, assistant bubble Markdown + copy button + streaming caret
- props/api: `MessageItem({ message: ChatMessage, streaming? })`
- canonical: MessageItem.tsx:6

## Markdown
- path: web/src/components/Markdown.tsx
- purpose: GFM renderer with Shiki code blocks (via CodeBlock) in a shared prose wrapper
- props/api: `Markdown({ children, trailing?, className = '' })`
- canonical: Markdown.tsx:28

## CodeBlock
- path: web/src/components/CodeBlock.tsx
- purpose: Fenced code block via Shiki (github-dark, module-level 200-entry cache) + copy button, plain <pre> fallback
- props/api: `CodeBlock({ code, lang })`
- canonical: CodeBlock.tsx:24

## CopyButton
- path: web/src/components/CopyButton.tsx
- purpose: Copies text to clipboard, flips to a check for 2 s, optional success toast
- props/api: `CopyButton({ text, label, toast?, className? })`
- canonical: CopyButton.tsx:9

## Wiki & notes
## NotesView
- path: web/src/components/NotesView.tsx
- purpose: Browse-and-read wiki surface — WikiTree left, NoteViewer or EmptyState right
- props/api: `NotesView({ openPath, onOpenNote, onOpenSettings })`
- canonical: NotesView.tsx:12

## WikiTree
- path: web/src/components/WikiTree.tsx
- purpose: Wiki directory as a folder tree via Tree; fetches on mount, focus, and chat-turn end
- props/api: `WikiTree({ openPath, onOpenNote, expandedKeys, onExpandedChange, onDirsChange? })`
- canonical: WikiTree.tsx:13

## Tree
- path: web/src/components/Tree.tsx
- purpose: Reusable accessible folder tree — roving tabIndex keyboard nav, memoized rows, controlled or internal expansion
- props/api: `Tree<T>({ nodes, getKey, getLabel, isDir, getChildren, renderIcon, renderTrailing?, renderTooltip?, onSelect, selectedKey, expandedKeys?, onExpandedChange? })` + `TreeProps<T>`
- canonical: Tree.tsx:109

## NoteViewer
- path: web/src/components/NoteViewer.tsx
- purpose: Side-panel viewer — fetches a note by path, renders Markdown, copy-raw + close
- props/api: `NoteViewer({ path, onClose })`
- canonical: NoteViewer.tsx:7

## Search
## SearchPanel
- path: web/src/components/SearchPanel.tsx
- purpose: Wiki search synced to the URL ?q= — debounced results, keyboard nav, recent-search history
- props/api: `SearchPanel({ onOpen: (path: string) => void })`
- canonical: SearchPanel.tsx:9

## SearchView
- path: web/src/components/SearchView.tsx
- purpose: Full-page search surface — TopBar plus the shared SearchPanel
- props/api: `SearchView({ onOpenNote, onOpenSettings })`
- canonical: SearchView.tsx:6

## Settings & dashboard
## SettingsView
- path: web/src/components/SettingsView.tsx
- purpose: Tabbed settings screen (General, Doctor, Git) — wiki path/model form, doctor checks, GitHub connect + repo sync
- props/api: `SettingsView()` — no props; the tab rides the URL segment
- canonical: SettingsView.tsx:49

## DashboardView
- path: web/src/components/DashboardView.tsx
- purpose: Dashboard landing — greeting, stat tiles, action buttons, mock overview/insights cards + charts
- props/api: `DashboardView({ onOpenSettings })`
- canonical: DashboardView.tsx:64

## SetupScreen
- path: web/src/components/SetupScreen.tsx
- purpose: Full-screen blocker listing install problems with fix commands + Re-check button
- props/api: `SetupScreen({ health, loading, onRecheck })`
- canonical: SetupScreen.tsx:31

## Card
- path: web/src/components/Card.tsx
- purpose: App-wide section panel with uppercase kicker title
- props/api: `Card({ title, children })`
- canonical: Card.tsx:5

## Notifications
## NotificationPanel
- path: web/src/components/NotificationPanel.tsx
- purpose: Dropdown bell panel — notification list, mark-all-read, per-item dismiss, empty state
- props/api: `NotificationPanel({ onClose })` — reads selectNotifications, dispatches markAllRead/dismissNotification
- canonical: NotificationPanel.tsx:8

## NotificationToasts
- path: web/src/components/NotificationToasts.tsx
- purpose: NEW notifications (not seen at mount) as transient toast cards top-left, auto-dismissed after 5 s
- props/api: `NotificationToasts()` — internal ToastCard({kind, title, body, onDismiss})
- canonical: NotificationToasts.tsx:14

## notifications.tsx
- path: web/src/components/notifications.tsx
- purpose: Single source for per-kind notification emoji icons, shared by panel + toasts
- props/api: `NotificationIcon({kind})` + `NOTIFICATION_ICONS: Record<NotificationKind, string>`
- canonical: notifications.tsx:13

## Shared UI
## EmptyState
- path: web/src/components/EmptyState.tsx
- purpose: Centered placeholder — emoji icon, title, optional hint
- props/api: `EmptyState({ icon, title, hint, className = '' })`
- canonical: EmptyState.tsx:4

## IconButton
- path: web/src/components/IconButton.tsx
- purpose: Subtle hover button used across the chrome; aria-label required
- props/api: `IconButton({ label, onClick, className = '', children })`
- canonical: IconButton.tsx:4

## Tooltip
- path: web/src/components/Tooltip.tsx
- purpose: Radix UI tooltip wrapper — accessible, collision-flipping, the app's dark bubble styling
- props/api: `Tooltip({ label, children, side = 'top', align = 'center' })`
- canonical: Tooltip.tsx:7

## Toast
- path: web/src/components/Toast.tsx
- purpose: Toast system — provider renders a bottom-center stack; auto-dismiss after 3 s or on click
- props/api: `ToastProvider({ children })` + `useToast(): { toast(message, kind?) }` + type `ToastKind`
- canonical: Toast.tsx:19

## Charts (Chart.js)
## chartSetup.ts
- path: web/src/components/chartSetup.ts
- purpose: Side-effect module registering all Chart.js pieces (bar/line/doughnut/arc/category/linear/tooltip) exactly once
- props/api: no exports — Chart.register(...) on import
- canonical: chartSetup.ts:17

## ActivityChart
- path: web/src/components/ActivityChart.tsx
- purpose: Single-series emerald bar chart of notes per day, theme-colored, tooltip with date counts
- props/api: `ActivityChart({ counts: number[] })`
- canonical: ActivityChart.tsx:11

## ChatActivityChart
- path: web/src/components/ChatActivityChart.tsx
- purpose: Single-series line chart of chat messages per day, hidden value axis, hover-only points
- props/api: `ChatActivityChart({ counts })` — counts: number[]
- canonical: ChatActivityChart.tsx:10

## NotesByKindChart
- path: web/src/components/NotesByKindChart.tsx
- purpose: Doughnut of note counts by kind, series-palette hues, legend list below
- props/api: `NotesByKindChart({ slices })` — slices: {kind, count}[]
- canonical: NotesByKindChart.tsx:11

## NotesByFolderChart
- path: web/src/components/NotesByFolderChart.tsx
- purpose: Horizontal bar chart of notes per top-level wiki folder, single emerald series
- props/api: `NotesByFolderChart({ rows })` — rows: {folder, count}[]
- canonical: NotesByFolderChart.tsx:9

## useThemeColors.ts
- path: web/src/components/useThemeColors.ts
- purpose: Chart colors read from --thoth-* CSS variables, re-renders on OS theme flips
- props/api: `useThemeColors(): ChartColors` + `chartColors()` + `interface ChartColors {accent, accentHover, subtle, ink, surface, series: string[]}`
- canonical: useThemeColors.ts:36

## Data
## dashboardMock.ts
- path: web/src/components/dashboardMock.ts
- purpose: Mock data for dashboard tiles until the real index endpoints land
- props/api: data only — mockInbox, mockMeetings, mockTodos, mockRecent, mockTags, mockActivity, mockChatActivity, mockNotesByKind, mockNotesByFolder, mockStats
- canonical: dashboardMock.ts:7

## Intentional skips
- *.test.tsx — co-located Vitest tests; skipped by convention
- App.tsx / main.tsx / index.css — live in web/src/, outside this dir; see docs/frontend.md structure section

Stale if: a file appears in web/src/components without an entry or a named
skip, a component's props change, or docs/frontend.md's component table
gains a row this index lacks.
````

- [ ] **Step 2: Verify coverage**

Run:
```bash
for f in ActivityChart Card chartSetup ChatActivityChart ChatPanel CodeBlock \
         Composer CopyButton dashboardMock DashboardView EmptyState IconButton \
         Markdown MessageItem NavRail NotesByFolderChart NotesByKindChart \
         NotesView NoteViewer NotificationPanel notifications NotificationToasts \
         SearchPanel SearchView SettingsView SetupScreen Sidebar Toast Tooltip \
         TopBar Tree useThemeColors WikiTree; do
  grep -q "$f" .claude/skills/react/references/components.md || echo "missing $f"
done
grep -q 'Intentional skips' .claude/skills/react/references/components.md && grep -q 'Stale if:' .claude/skills/react/references/components.md
```
Expected: no output from the loop; greps exit 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/react/references/components.md
git commit -m "feat(skills): add react component reference index

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 8: react/references/hooks.md — the hook index

**Files:**
- Create: `.claude/skills/react/references/hooks.md`

**Interfaces:**
- Consumes: spec §Reference anatomy; docs/frontend.md §Hooks; the 5-file hook coverage list in Global Constraints
- Produces: the hook index; Task 12 checks all five hook names appear

- [ ] **Step 1: Write the index**

````markdown
# Hooks (web/src/hooks)

Each entry: path, purpose, props/api, canonical. Coverage: all five hooks;
a missing hook means this index is stale.

## useChat
- path: web/src/hooks/useChat.ts
- purpose: Adapts the chat slice to the WebSocket — WS frames become store dispatches, send/cancel/load/reset touch the socket; conversation state survives component remounts in the chat slice
- props/api: `useChat(socket: ChatSocket | null) => { messages, streaming, conversationId, lastTool, thinking, thinkingText, send, cancel, load, reset }`; re-exports `ChatMessage`
- canonical: useChat.ts:29 · docs/frontend.md §Hooks

## useSearch
- path: web/src/hooks/useSearch.ts
- purpose: Debounced (300 ms) search with abort + sequence guards so slow responses can't overwrite newer ones
- props/api: `useSearch(query: string) => { results: SearchResult[], loading: boolean }`
- canonical: useSearch.ts:4 · docs/frontend.md §Hooks

## useConversationRoute
- path: web/src/hooks/useConversationRoute.ts
- purpose: Keeps the URL /chat/<uuid> and the active conversation in sync (URL → state on popstate, state → URL via pushState)
- props/api: `useConversationRoute({ socket, conversationId, load, reset, onError })`; also exports `navigate(path: string)`
- canonical: useConversationRoute.ts:37

## useView
- path: web/src/hooks/useView.ts
- purpose: Pathname-based view routing — maps the URL to a view plus segment/query, push URLs with popstate dispatch
- props/api: `useView(): View` · `useViewRoute(): ViewRoute` · `navigateView(v)` · `navigateSegment(v, segment)` · `navigateNote(path)`; `View = 'chat' | 'notes' | 'dashboard' | 'search' | 'settings'`
- canonical: useView.ts:73

## useViewShortcuts
- path: web/src/hooks/useViewShortcuts.ts
- purpose: Binds Cmd/Ctrl+1..4 to dashboard/chat/notes/search and Cmd/Ctrl+K to search
- props/api: `useViewShortcuts(): void` — window listener removed on unmount
- canonical: useViewShortcuts.ts:9

Stale if: a new hook appears in web/src/hooks, a signature above changes,
or docs/frontend.md's hooks section gains an entry this index lacks.
````

- [ ] **Step 2: Verify coverage**

Run:
```bash
for h in useChat useSearch useConversationRoute useView useViewShortcuts; do
  grep -q "$h" .claude/skills/react/references/hooks.md || echo "missing $h"
done
grep -q 'Stale if:' .claude/skills/react/references/hooks.md
```
Expected: no output from the loop; grep exits 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/react/references/hooks.md
git commit -m "feat(skills): add react hooks reference index

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 9: react/references/store.md — the Redux slice index

**Files:**
- Create: `.claude/skills/react/references/store.md`

**Interfaces:**
- Consumes: spec §Reference anatomy; docs/frontend.md §State; the 7-slice coverage list in Global Constraints
- Produces: the store index; Task 12 checks all seven slice names appear

- [ ] **Step 1: Write the index**

````markdown
# Redux store (web/src/store)

Redux Toolkit owns the server-backed and shared state. Each slice lives in
slices/ with its actions, selectors, and thunks co-located; makeStore()
wires them; hooks.ts exports the typed hooks — always use
useAppDispatch/useAppSelector, never the bare versions.

## index.ts
- path: web/src/store/index.ts
- purpose: makeStore() wires all slices; exports RootState and AppDispatch
- canonical: web/src/store/index.ts

## hooks.ts
- path: web/src/store/hooks.ts
- purpose: The typed hooks — useAppDispatch / useAppSelector
- canonical: web/src/store/hooks.ts

## Slices (web/src/store/slices/)

## health
- path: healthSlice.ts
- purpose: Server health, fetched at boot (main.tsx), re-checked by the setup screen
- canonical: healthSlice.ts · docs/frontend.md §State

## settings
- path: settingsSlice.ts
- purpose: Settings loaded when the settings view mounts, saved through the slice (submit button reflects saving)
- canonical: settingsSlice.ts · docs/frontend.md §State

## conversations
- path: conversationsSlice.ts
- purpose: Conversation list — refetched on URL changes and new-chat; deletes filter the list in the slice
- canonical: conversationsSlice.ts · docs/frontend.md §State

## chat
- path: chatSlice.ts
- purpose: The live conversation — messages, streaming, thinking, lastTool, conversationId — fed by WS frames via useChat
- canonical: chatSlice.ts · docs/frontend.md §State

## connection
- path: connectionSlice.ts
- purpose: The WebSocket status, reported by ChatSocket and read by ChatPanel
- canonical: connectionSlice.ts · docs/frontend.md §State

## notifications
- path: notificationsSlice.ts
- purpose: Notification items (id/kind/title/body/read), capped ring of 50 — ephemeral UI state, never persisted
- props/api: actions notify({kind,title,body?}), markNotificationRead(id), markAllRead(), dismissNotification(id); selectors selectNotifications, selectUnreadCount
- canonical: notificationsSlice.ts:24

## searchHistory
- path: searchHistorySlice.ts
- purpose: The last committed searches (strings) — cap 10, deduped, most-recent-first; lazy localStorage load + persistSearchHistory middleware writes back
- props/api: actions commitSearch(q), clearSearchHistory(); selector selectSearchHistory
- canonical: searchHistorySlice.ts:24 · docs/frontend.md §Components

Stale if: a slice appears or disappears in web/src/store/slices/, makeStore
wiring changes, or docs/frontend.md's state list gains an entry this
index lacks.
````

- [ ] **Step 2: Verify coverage**

Run:
```bash
for s in chat connection conversations health notifications searchHistory settings; do
  grep -q "$s" .claude/skills/react/references/store.md || echo "missing $s"
done
grep -q 'Stale if:' .claude/skills/react/references/store.md
```
Expected: no output from the loop; grep exits 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/react/references/store.md
git commit -m "feat(skills): add react store reference index

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 10: react/references/patterns.md — the cross-cutting conventions

**Files:**
- Create: `.claude/skills/react/references/patterns.md`

**Interfaces:**
- Consumes: spec §Reference anatomy; docs/frontend.md §Design system + §WebSocket client; CLAUDE.md TS rules (referenced, not copied)
- Produces: the patterns reference; Task 12 checks it names the zod boundary, the WS frames, and the test doubles

- [ ] **Step 1: Write the reference**

````markdown
# Frontend patterns — the cross-cutting conventions

## The API boundary (zod)
- web/src/api/client.ts is the typed REST client: axios + zod, every
  response parsed by a zod schema — validation at the boundary
- TS strict, zero any (CLAUDE.md invariant); DTOs must match the Go side
- Tests use mockAxios and assert the parsed payload, not the transport
- canonical: web/src/api/client.ts · docs/api.md

## The WS protocol (ChatSocket)
- Out: send {text}, cancel, resume {conversation_id}, open {conversation_id}, new_chat
- In: assistant_start, assistant_thinking {text}, assistant_delta {text}, tool_activity {tool, detail}, turn_done {conversation_id}, error {message}
- Reconnects exactly once after 1 s, sending resume from onopen so the turn re-syncs
- open pins the server-side conversation and never becomes the reconnect-resume id
- Server message types in internal/api/chat.go must match web/src/ws/chat.ts — CHANGE BOTH SIDES
- canonical: web/src/ws/chat.ts · docs/api.md §WebSocket chat

## Test doubles (web/src/test)
- mockAxios — the axios mock for client.ts consumers
- fakeWS — the scripted WebSocket double for ChatSocket consumers
- renderWithStore — renders with the real store provider
- setup — Vitest setup file
- Rule: use these; never hand-roll mocks of the app itself (CLAUDE.md)

## State placement
- Shared or screen-spanning data → Redux slices
- Component-local state (form fields, tree expansion, debounce, openNote) → hooks/components
- canonical: docs/frontend.md §State

## Design tokens
- Tokens in web/src/index.css (@theme) resolve to CSS custom properties that flip under prefers-color-scheme — one semantic class works in both themes
- The five semantic groups: app/surface/raised (bg-surface…), line (border-line…), subtle/ink/heading (text-ink…); emerald accent (#059669 → #34d399)
- Use semantic classes; no raw hex in components; dark mode follows the OS — no toggle
- Display type: Fraunces (self-hosted via @fontsource-variable — no runtime network); body: system stack
- canonical: web/src/index.css · docs/frontend.md §Design system

## Routing
- Hand-rolled: useView maps the pathname to a view; useConversationRoute keeps /chat/<uuid> synced; SearchPanel rides ?q=
- canonical: web/src/hooks/useView.ts · web/src/hooks/useConversationRoute.ts

## Package discipline
- pnpm only — never npm; workspace root; lockfile committed; save-exact
- make web syncs web/dist into internal/webui/dist — required before go build/test
- canonical: CLAUDE.md §Toolchain

Stale if: a zod schema changes shape without a client.ts update, the WS
frame set changes, a new test double appears in web/src/test, or new
semantic tokens land in index.css without an entry above.
````

- [ ] **Step 2: Verify pointers**

Run:
```bash
for f in web/src/api/client.ts web/src/ws/chat.ts web/src/test/mockAxios.ts web/src/test/fakeWS.ts web/src/test/renderWithStore.tsx web/src/test/setup.ts web/src/index.css; do
  test -f "$f" || echo "missing $f"
done
grep -q 'assistant_start' .claude/skills/react/references/patterns.md && grep -q 'Stale if:' .claude/skills/react/references/patterns.md
```
Expected: no output from the loop; greps exit 0.

- [ ] **Step 3: Commit — pause for user approval first**

```bash
git add .claude/skills/react/references/patterns.md
git commit -m "feat(skills): add react patterns reference

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 11: CLAUDE.md — the one pointer line

**Files:**
- Modify: `CLAUDE.md` (append the pointer line after the graphify section's staleness bullet)

**Interfaces:**
- Consumes: spec §Wiring — the exact line text is pinned there
- Produces: the single CLAUDE.md change; nothing else in CLAUDE.md may be touched

- [ ] **Step 1: Read CLAUDE.md and find the anchor**

Run:
```bash
git diff CLAUDE.md | head -40   # inspect the pre-existing uncommitted edits
tail -5 CLAUDE.md
```
The file ends with the graphify staleness bullet ("…files are the source of truth, the graph is derived."). The edit appends after that bullet and must not disturb the existing working-tree edits.

- [ ] **Step 2: Add the line (exact text from the spec)**

Append after the staleness bullet:

```markdown

- **Skills** — `.claude/skills/` holds the go (backend) and react
  (frontend) procedure skills. Rules stay in this file; procedures
  live there; `docs/` owns detail.
```

- [ ] **Step 3: Verify**

Run:
```bash
grep -A3 '**Skills**' CLAUDE.md
git diff --stat CLAUDE.md
```
Expected: the pointer line renders with the exact text above; the diff touches only this addition on top of the pre-existing edits.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: point CLAUDE.md at the project skills suite

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 12: Full-suite verification + PR

**Files:**
- None created; verifies Tasks 1–11 and opens the PR

**Interfaces:**
- Consumes: everything produced by Tasks 1–11; the coverage lists in Global Constraints

- [ ] **Step 1: Verify workflow counts**

Run:
```bash
grep -c '^### ' .claude/skills/go/SKILL.md     # expect 8
grep -c '^### ' .claude/skills/react/SKILL.md  # expect 6
```

- [ ] **Step 2: Verify canonical pointers resolve**

Run:
```bash
grep -h '^\- canonical' .claude/skills/go/references/*.md .claude/skills/react/references/*.md \
  | grep -o 'internal/[a-z_/]*\.go\|web/src/[a-z/]*\.\(ts\|tsx\)\|docs/[a-z.]*\.md\|cmd/thoth/main\.go' \
  | sort -u | while read f; do test -f "$f" || echo "missing $f"; done
```
Expected: no output — every prefixed canonical pointer (internal/, web/src/, docs/, cmd/) names an existing file. Bare filenames like ChatPanel.tsx are covered by the per-task coverage loops instead.

- [ ] **Step 3: Verify no CLAUDE.md rule is duplicated**

Run:
```bash
grep -rn 'zero any\|80%\|blast wall' .claude/skills/ | wc -l
grep -q 'Rules stay in this file' CLAUDE.md
```
Expected: the count is ≤ 30 — pointer-style mentions only ("CLAUDE.md invariant", one-line gotchas), never full rule re-statements; the second grep exits 0.

- [ ] **Step 4: Review the whole diff**

Run:
```bash
git status --short
git log --oneline main..HEAD
git diff main..HEAD --stat
```
Expected: 12 commits — `ca85dcc` (design doc) + one per task; files confined to `.claude/skills/`, `CLAUDE.md` (one line), `docs/superpowers/`. Read through `git diff main..HEAD` once — anything that contradicts the spec is a defect.

- [ ] **Step 5: Open the PR — pause for user approval first**

```bash
gh pr create --title "feat: project skills suite (go + react)" \
  --label feature \
  --body "$(cat <<'EOF'
## What
Committed project skills at .claude/skills/ — go (backend) and react (frontend) — so future Claude Code sessions get procedure workflows plus distilled reference indexes instead of re-deriving conventions.

## Why
Procedure + expertise on demand: 8 backend workflows (REST, WS protocol, migrations, claude flags blast wall, settings keys, wiki contract, deps, doctor) and 6 frontend workflows (component, slice, hook, API wiring, tests, WS client). References index every component/hook/slice/package with canonical pointers.

## Design
See the committed design doc: docs/superpowers/specs/2026-08-17-thoth-skills-design.md

Division of authority: rules stay in CLAUDE.md; docs/ owns detail; skills own procedure + pointers. No rule is duplicated.

## Changes
- .claude/skills/go/{SKILL.md, references/{packages,claude-blast-wall,persistence,quality}.md}
- .claude/skills/react/{SKILL.md, references/{components,hooks,store,patterns}.md}
- CLAUDE.md — one pointer line under graphify
- docs/superpowers/specs/2026-08-17-thoth-skills-design.md (approved design)
- docs/superpowers/plans/2026-08-18-thoth-skills-suite.md (implementation plan)

## Verification
- Coverage: every file in web/src/{components,hooks,store/slices} and every internal/ package has an index entry or a named skip
- All canonical pointers resolve to existing files
- Workflow counts match the spec (8 go / 6 react)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
Expected: PR created against main. If the `feature` label is rejected by the maintainer's taste, `documentation` is the fallback type label (no area label matches — areas are package-aligned).

