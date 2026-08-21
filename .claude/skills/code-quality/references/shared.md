# Shared checklist — both layers (yes/no; any "no" gets fixed before the PR)

Each item cites its rule source; the rules live in CLAUDE.md, not here.
Walk this file plus the domain reference(s) your change touches
(go.md / react.md) and documentation.md.

## Correctness
- [ ] Happy path AND error paths handled — errors wrapped with `%w`, never swallowed (CLAUDE.md § Code Rules)
- [ ] Inputs validated at the boundary; fail fast, early returns, the happy path stays flat (CLAUDE.md § Code Rules)
- [ ] No panics in library code (CLAUDE.md § Invariants)

## Tests
- [ ] Every behavior change ships with a test — table-driven, asserting real outcomes (CLAUDE.md § Code Rules)
- [ ] Edge cases covered, not just the happy path (CLAUDE.md § Code Rules)
- [ ] Tests share no mutable fixtures — each test constructs its own state (CLAUDE.md § Code Rules)
- [ ] `go test -race ./...` green; coverage stays ≥ 90% (CLAUDE.md § Invariants)

## Structure
- [ ] Functions ≤ 40 lines; split at ~60 into named helpers (CLAUDE.md § Code Rules)
- [ ] ≤ 3 parameters; 4+ grouped into a typed struct/options; no function takes 7+ parameters — convert it to an object/struct (CLAUDE.md § Code Rules)
- [ ] One clear responsibility per type/function (CLAUDE.md § Code Rules)
- [ ] DRY: every rule, protocol, and convention lives in exactly one place; duplicated logic extracted — no copy-pasted blocks (CLAUDE.md § Code Rules)
- [ ] Established pattern reused over novelty; a new pattern needs a stated reason (CLAUDE.md § Code Rules: Patterns over novelty)

## Naming & style
- [ ] Clear names, no stutter (`wiki.New` not `wiki.NewWiki`); camelCase in TS (CLAUDE.md § Code Rules)
- [ ] Matches surrounding style — no reformatting of code you aren't changing (CLAUDE.md § Code Rules)
- [ ] No magic values — named constants for numbers/strings with meaning (CLAUDE.md § Code Rules)

Canonical: CLAUDE.md § Code Rules · § Invariants

Stale if: CLAUDE.md's rule sections change without this file following.
