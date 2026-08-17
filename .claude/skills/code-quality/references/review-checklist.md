# Review checklist (yes/no — any "no" gets fixed before the PR)

Each item cites its rule source; the rules live in CLAUDE.md, not here.
Walk **Shared** plus the section for the layer(s) your change touches;
skip the other layer's section.

## Shared (walk for every change)

### Correctness
- [ ] Happy path AND error paths handled — errors wrapped with `%w`, never swallowed (CLAUDE.md § Code Rules)
- [ ] Inputs validated at the boundary; fail fast, early returns, the happy path stays flat (CLAUDE.md § Code Rules)
- [ ] No panics in library code (CLAUDE.md § Invariants)

### Tests
- [ ] Every behavior change ships with a test — table-driven, asserting real outcomes (CLAUDE.md § Code Rules)
- [ ] Edge cases covered, not just the happy path (CLAUDE.md § Code Rules)
- [ ] Tests share no mutable fixtures — each test constructs its own state (CLAUDE.md § Code Rules)
- [ ] `go test -race ./...` green; coverage stays ≥ 80% (CLAUDE.md § Invariants)

### Structure
- [ ] Functions ≤ 40 lines; split at ~60 into named helpers (CLAUDE.md § Code Rules)
- [ ] ≤ 3 parameters; 4+ grouped into a typed struct/options; no function takes 7+ parameters — convert it to an object/struct (CLAUDE.md § Code Rules)
- [ ] One clear responsibility per type/function; no copy-pasted blocks (DRY — CLAUDE.md § Code Rules)

### Naming & style
- [ ] Clear names, no stutter (`wiki.New` not `wiki.NewWiki`); camelCase in TS (CLAUDE.md § Code Rules)
- [ ] Matches surrounding style — no reformatting of code you aren't changing (CLAUDE.md § Code Rules)
- [ ] No magic values — named constants for numbers/strings with meaning (CLAUDE.md § Code Rules)

### Docs & graph
- [ ] docs/ page updated in the same commit when behavior changes (CLAUDE.md § Repo rules)
- [ ] Affected skills updated in the same commit (skills suite spec § Maintenance — same-commit contract)
- [ ] `graphify update .` run after the change (CLAUDE.md § graphify)

## Go backend (internal/* + cmd/*)

- [ ] Wrapped errors compared with `errors.Is`/`errors.As`, never `==` (CLAUDE.md § Code Rules)
- [ ] context.Context is the first parameter, never stored in structs (CLAUDE.md § Code Rules)
- [ ] Every `go` statement has a ctx/done-channel; no goroutine outlives its owner (CLAUDE.md § Memory)
- [ ] Interfaces are small (1–3 methods) and defined at the consumer, not the producer (CLAUDE.md § Code Rules)
- [ ] `defer` immediately after every acquisition (rows, files, bodies, handles) (CLAUDE.md § Memory)
- [ ] Shared state behind mutex/atomic — CI runs -race and it must stay green (CLAUDE.md § Memory)
- [ ] No package-level mutable globals (CLAUDE.md § Invariants)

## React frontend (web/src)

- [ ] TS `strict`, zero `any` (eslint enforces) (CLAUDE.md § Invariants)
- [ ] zod validates every API boundary response (CLAUDE.md § Invariants)
- [ ] Every useEffect subscription/timer/socket has cleanup; no setInterval without clearInterval (CLAUDE.md § Memory)
- [ ] Components are reusable and composable — small, props-driven; shared pieces extracted to hooks/components, not duplicated (CLAUDE.md § Code Rules: Modular & composable)
- [ ] WS message types match `internal/api/chat.go` when the chat client changes (CLAUDE.md § Invariants)

Canonical: CLAUDE.md § Code Rules · § Invariants · § Memory

Stale if: CLAUDE.md's rule sections change without this checklist following,
or a gate in the code-quality SKILL.md workflow 1 changes.
