---
name: code-rules
description: >-
  Thoth code rules, memory/no-leak rules, token-efficiency notes, invariants,
  and repo rules — the rules that govern every change. Fires on: writing or
  changing code, implementing, refactoring, fixing, adding tests, committing.
---

# Code rules — read before changing anything

## When to use
- Before writing or changing any code in this repo
- When implementing, refactoring, fixing bugs, or adding tests
- Not navigation — that's `CLAUDE.md` / `AGENTS.md` (the repo map)
- Not domain procedures — those are the `go` / `react` / `git-workflow` / `code-quality` skills

## Key files
- CLAUDE.md — the repo map (toolchain, commands, routing)
- AGENTS.md — the token-lean repo map
- docs/ — design authority (components.md, frontend.md, security.md, development.md)

## Code Rules

Apply the standard principles — modular, composable, boring code that's easy to change:

- **DRY — don't repeat yourself.** Every rule, protocol, and convention lives in exactly one place and everything else imports it (rulebook == `Rulebook()`, CLI flags == `client.go`, WS types shared Go/TS). Copy-paste a block and you've created a divergence bug.
- **Modular & composable** — small units with one clear purpose behind a narrow interface; build features by composing units, not growing them. Split before a file outgrows its intent.
- **SOLID** — single responsibility per type/function; depend on interfaces (only at real seams); inject dependencies via constructors, never reach for globals.
- **Patterns over novelty** — reuse the codebase's established patterns (package idioms in `docs/components.md`: hub, process pool, repository, slice, hook); a new pattern or dependency needs a stated reason (YAGNI).
- **KISS** — the simplest thing that passes the tests; no speculative abstraction, no speculative error handling.
- **YAGNI** — build what's asked, nothing more; a new dependency needs a stated reason.
- **Small functions** — target ≤ 40 lines; at ~60, split into named helpers (what, not how).
- **Few parameters** — ≤ 3; at 4+, group into a typed struct/options; a function never takes 7+ parameters — convert it to an object/struct.
- **Fail fast, guard early** — validate at the boundary, return early, keep the happy path flat.
- **Don't break existing functionality** — the commit gates live in the go quality reference and the git-workflow skill; exported signatures change only with all call sites + tests updated.
- **Every behavior change ships with a test** — table-driven; assert real outcomes, not mocks of yourself.
- **Naming** — clear, no stutter (`wiki.New` not `wiki.NewWiki`), Go idiom; camelCase in TS.
- **Errors** — wrap with `%w`, never swallow silently; compare wrapped errors with `errors.Is`/`errors.As`, never `==`.
- **Interfaces at the consumer** — small (1–3 methods), defined where they're used, not where they're implemented.
- **context.Context first** — first parameter, never stored in structs; goroutines and long-lived loops select on `ctx.Done()`.
- **Logging** — structured `slog` with lowercase keys; warn paths always carry `path` and `err`.
- **Security** — security-sensitive changes consult `docs/security.md` (the threat model); wiki filesystem access routes through `SafePath`.
- **Agent tool placement** — a new tool's home is decided by what it knows: tools that operate on any filesystem with no wiki knowledge go in `agent/tools` (common, wiki-agnostic; work through the `FS` seam); tools that understand the wiki — its frontmatter/note contract, `type:` rule, or scaffolded layout (todos, inbox, memory) — go in `internal/agent/tools` and import the wiki contract from `internal/wiki` (`ParseNote`/`FormatNote`), never forking it. Everything else (shared arg/path/truncation helpers, the `FS` seam) lives in `agent/tools` and is imported, not duplicated. A host registers its own tools via `RegistryOptions.CustomTools` / `Client.WithTools(...)`.
- **No magic values** — numbers and strings with meaning get named constants; no unexplained literals in logic.
- **Go doc comments** — exported symbols carry doc comments (Go idiom; no linter enforces it here, so the rule must).
- **Match surrounding style** when editing; don't reformat code you're not changing.

## Memory & Resources (no leaks)

- **Every resource is released on every path** — `defer` immediately after acquisition: `rows.Close()`, files, bodies, DB handles, watchers.
- **Goroutines must end** — every `go` statement has a `ctx`/done-channel that stops it; long-lived loops select on `ctx.Done()`; no goroutine outlives its owner.
- **No unbounded growth** — capped buffers (like the 500-message replay), bounded maps with eviction, no slices/maps that only grow.
- **Frontend** — every `useEffect` subscription/timer/socket has a cleanup that runs on unmount; no setInterval without clearInterval.
- **Concurrency is guarded** — shared state behind mutex/atomic, never a bare data race; CI runs `-race` and it must stay green.
- **Process hygiene** — spawned children die with their context (process-group kill on unix, direct kill on windows).

## Token Efficiency

- **Read the repo map, then go straight to the target** — the layout tree replaces broad exploration; open only the package your task touches.
- **Check the code before answering** — never guess; cite `file:line`.
- **Reuse before adding** — existing helpers/packages first; new dependency only with a stated reason (YAGNI).
- **Focused tests while iterating** (`go test ./internal/<pkg>/ -run TestX -v`); full suite once, just before commit.
- **Don't re-read what you just wrote** — enforced by the read-guard hooks (`scripts/token-guard.sh`); don't reformat code you aren't changing; keep diffs minimal and scoped.
- **Chat output: code-first, prose minimal** — say what changed and why in one line; let the commit message carry detail.
- **Verify, don't assume** — a claim about behavior needs the test output or the `file:line` behind it.

## Invariants (do not break)

- Files are the source of truth; the index syncs with the tree (thoth.db is derived data). Notes require `---` frontmatter with `title`; the wiki never stores secrets.
- WS is chat + server-push transport (`wiki_changed` frames); REST for everything else. Server message types in `internal/api/chat.go` must match `web/src/ws/chat.tsx`.
- Go: `%w` errors, `context.Context` everywhere (cancel = the stop button), no panics in library code, no package-level mutable globals.
- TS: `strict`, no `any` (eslint), zod at the API boundary.
- Cross-compile: all five targets (darwin/linux × amd64/arm64, windows/amd64) must build.
- Coverage floor 90% on `agent/` + `internal/` + `cmd/`, CI-enforced.

## Repo rules

- **Branch workflow** — `main` is always deployable; never commit to it directly. Changes live on `<type>/<scope>/<slug>` branches with conventional-commit messages and land via reviewed PRs that a human squash-merges — a session never merges. The full procedure — sync-and-branch commands, commit conventions, PR template sections, label application, squash-merge specifics, and the `ci-pr`/`final-gate` expectations — is the `git-workflow` skill (`.claude/skills/git-workflow/SKILL.md`).
- **No secrets in the repo** — never commit real credentials, tokens, or keys in code, configs, tests, or docs; env vars or placeholders only.
- **Design authority** — design docs for large or cross-package changes live (untracked) in `docs/specs/` when needed; the committed `docs/` pages are the reference for current behavior.
- **Project docs** — committed documentation lives in `docs/` (`index.md` is the hub: architecture, API, CLI, indexing, frontend, security, development). Update the relevant page when behavior changes.
- **Issue/PR labels** — three tiers on GitHub: types, areas, priority. Every issue/PR carries exactly one type and one label per area it touches; issues also carry a priority. The label lists are `.claude/skills/git-workflow/references/labels.md`.
- **Generated/ignored** — `bin/`, `web/dist/`, `internal/webui/dist/`, `node_modules/`, `*.db`.
- **Repo-map freshness** — `CLAUDE.md` and `AGENTS.md` are identical copies of the routing map. Adding, renaming, or deleting any file or directory under `agent/`, `internal/`, `web/src/`, `cmd/`, `docs/`, `.claude/skills/`, or `scripts/` requires updating the routing tree in **both** files in the same commit (`cp CLAUDE.md AGENTS.md`). The pre-commit guard fails if they drift.

## Maintenance
Single source of truth for the rules. Stale if: any rule changes without this file following.
