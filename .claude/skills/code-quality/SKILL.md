---
name: code-quality
description: >-
  Thoth pre-PR quality gates — the six gate checks, coverage, lint,
  review checklist, failing-gate triage. Fires on: quality gates, before
  commit, pre-PR review, coverage, lint, checklist, make check, race, CI.
---

# Code quality — the pre-PR gate

## When to use
- Right before committing or opening a PR: run the gates, walk the checklist
- When a gate fails and you need to triage it
- Not domain procedures — those are the `go` / `react` skills
- Not PR mechanics (labels, template, merge) — that's the `git-workflow` skill

## Key files
- docs/development.md — § Gates: what `make check` enforces, in order (single source for the gate list)
- references/ — the review checklists: shared.md (both layers), go.md (backend), react.md (frontend), documentation.md (docs + delivery)
- .golangci.yml — the lint set (govet, staticcheck, errcheck, ineffassign, unused)
- go/references/quality.md — gate details and floors

## Workflows

### 1. Run the quality gates
Run `make check` — one command, everything CI enforces locally (fmt, lint, race, cover ≥ 90%, frontend typecheck/lint/test, build). Gate details and floors: docs/development.md § Gates and go/references/quality.md. The pre-commit hook already covers Go vet/lint/test when Go is staged.

### 2. Walk the review checklist
1. Read references/shared.md plus the domain file(s) your change touches — go.md for backend, react.md for frontend — and documentation.md; skip the untouched domain.
2. Each item is yes/no — any "no" gets fixed before the PR, not after.
3. Then fill the PR template's "How verified" checkboxes with only what actually ran.
4. If you fixed something, re-run the covering gates (workflow 1), not just the one you touched.

### 3. Triage a failing gate
- gofmt/vet/lint failing → fix the code; never suppress with lint directives
- race failing → a real data race: mutex/atomic/ownership — investigate, don't skip
- coverage < 90% → add tests for the new/changed code; never delete tests to raise the floor
- tsc errors → fix the types; no `any` escape hatches (code-rules skill invariant)
- vitest failing → reproduce the failure, then fix the test if it asserts wrong or the code if it behaves wrong
- After the fix: re-run the failed gate AND its neighbors (a fmt fix can shift a test)

## Gotchas
- The pre-commit hook autofixes staged files (eslint/prettier/golangci-lint --fix) — re-run the gates after any hook-triggered edit.
- ci-pr runs the quality gates but NOT the cross-compiles — those run only on push to main, so a PR can be green and a main push red (docs/development.md § CI).
- New code must be covered to keep the total ≥ 90% on agent/ + internal/ + cmd/ — a large feature with thin tests fails CI.

## Canonical docs
- docs/development.md — § Gates, § CI
- code-rules skill — § Code Rules, § Invariants (the rules behind the checklist)

## Maintenance
Derived view — update in the same commit as any change to docs/development.md § Gates, .golangci.yml, or quality.yml. Stale if the gate list differs from `make check`, the coverage floor changes, or the checklists diverge from code-rules skill § Code Rules.
