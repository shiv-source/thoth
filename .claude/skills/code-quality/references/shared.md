# Shared checklist — both layers (yes/no; any "no" gets fixed before the PR)

Each item cites its rule source; the rules live in the code-rules skill, not here.
Walk this file plus the domain reference(s) your change touches
(go.md / react.md) and documentation.md.

## Correctness
- [ ] Happy path AND error paths handled — errors wrapped with `%w`, never swallowed (code-rules skill § Code Rules)
- [ ] Inputs validated at the boundary; fail fast, early returns, the happy path stays flat (code-rules skill § Code Rules)
- [ ] No panics in library code (code-rules skill § Invariants)

## Tests
- [ ] Every behavior change ships with a test — table-driven, asserting real outcomes (code-rules skill § Code Rules)
- [ ] Edge cases covered, not just the happy path (code-rules skill § Code Rules)
- [ ] Tests share no mutable fixtures — each test constructs its own state (code-rules skill § Code Rules)
- [ ] `go test -race ./...` green; coverage stays ≥ 90% (code-rules skill § Invariants)

## Structure
- [ ] Functions ≤ 40 lines; split at ~60 into named helpers (code-rules skill § Code Rules)
- [ ] ≤ 3 parameters; 4+ grouped into a typed struct/options; no function takes 7+ parameters — convert it to an object/struct (code-rules skill § Code Rules)
- [ ] One clear responsibility per type/function (code-rules skill § Code Rules)
- [ ] DRY: every rule, protocol, and convention lives in exactly one place; duplicated logic extracted — no copy-pasted blocks (code-rules skill § Code Rules)
- [ ] Established pattern reused over novelty; a new pattern needs a stated reason (code-rules skill § Code Rules: Patterns over novelty)

## Naming & style
- [ ] Clear names, no stutter (`wiki.New` not `wiki.NewWiki`); camelCase in TS (code-rules skill § Code Rules)
- [ ] Matches surrounding style — no reformatting of code you aren't changing (code-rules skill § Code Rules)
- [ ] No magic values — named constants for numbers/strings with meaning (code-rules skill § Code Rules)

Canonical: code-rules skill § Code Rules · § Invariants

Stale if: code-rules skill's rule sections change without this file following.
