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
- **SOLID** — S: single responsibility per type/function; O: open/closed — extend by composing, never modify working code to add behavior; L: Liskov — implementations honor the interface's contract (no surprises, no `if`-on-type); I: interface segregation — depend on small interfaces at real seams (Interfaces at the consumer); D: dependency inversion — inject dependencies via constructors, never reach for globals.
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
- **context.Context first** — first parameter, never stored in structs.
- **Logging** — structured `slog` with lowercase keys; warn paths always carry `path` and `err`.
- **Security** — security-sensitive changes consult `docs/security.md` (the threat model); wiki filesystem access routes through `SafePath`.
- **Agent tool placement** — a new tool's home is decided by what it knows: tools that operate on any filesystem with no wiki knowledge go in `agent/tools` (common, wiki-agnostic; work through the `FS` seam); tools that understand the wiki — its frontmatter/note contract, `type:` rule, or scaffolded layout (todos, inbox, memory) — go in `internal/agent/tools` and import the wiki contract from `internal/wiki` (`ParseNote`/`FormatNote`), never forking it. Everything else (shared arg/path/truncation helpers, the `FS` seam) lives in `agent/tools` and is imported, not duplicated. A host registers its own tools via `RegistryOptions.CustomTools` / `Client.WithTools(...)`.
- **No magic values** — numbers and strings with meaning get named constants; no unexplained literals in logic.
- **Go doc comments** — exported symbols carry doc comments (Go idiom; no linter enforces it here, so the rule must).
- **Match surrounding style** when editing.

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

## On-demand references

- **Memory & Resources (no leaks)** — `references/memory.md` — read when writing concurrent/resource-managing code (goroutines, DB handles, files, watchers, frontend effects); the go/react skills also restate the critical rules.
- **Repo rules** — `references/repo-rules.md` — read at commit/PR time or when changing repo structure: branch workflow, secrets, design authority, project docs, labels, generated/ignored, repo-map freshness.

## Maintenance
Single source of truth for the rules. Stale if: any rule changes without this file following.
