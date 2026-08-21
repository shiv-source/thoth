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
- docs/development.md — § Gates: what make check enforces, in order
- .golangci.yml — the lint set (govet, staticcheck, errcheck, ineffassign, unused)
- .claude/skills/go/references/quality.md — the gate details and floors
- .husky/pre-commit — what already runs at commit time
- .github/workflows/quality.yml + final-gate.yml — what CI re-runs on the PR
- references/ — the checklists: shared.md (both layers), go.md (backend), react.md (frontend), documentation.md (docs + delivery)

## Workflows

### 1. Run the quality gates
Run each gate; every one must pass before the PR:
1. `gofmt -l .` — must print nothing (formatting drift)
2. `go vet ./...` — must be clean (correctness suspicions)
3. `go test -race ./...` — must pass (data races are real bugs, never skip)
4. `make cover` — coverage ≥ 90% on agent/ + internal/ + cmd/ (the CI-enforced floor)
5. `golangci-lint run` — must be clean (staticcheck/errcheck)
6. `pnpm typecheck && pnpm lint && pnpm test` — TS strict, eslint, vitest
Details and floors: .claude/skills/go/references/quality.md

### 2. Walk the review checklist
1. Read references/shared.md plus the domain file(s) your change touches — go.md for backend, react.md for frontend — and documentation.md; skip the untouched domain
2. Each item is yes/no — any "no" gets fixed before the PR, not after
3. Then fill the PR template's "How verified" checkboxes with only what actually ran
4. If you fixed something, re-run the covering gates (workflow 1), not just the one you touched

### 3. Triage a failing gate
- gofmt/vet/lint failing → fix the code; never suppress with lint directives
- race failing → a real data race: mutex/atomic/ownership — investigate, don't skip
- coverage < 90% → add tests for the new/changed code; never delete tests to raise the floor
- tsc errors → fix the types; no `any` escape hatches (CLAUDE.md invariant)
- vitest failing → reproduce the failure, then fix the test if it asserts wrong or the code if it behaves wrong
- After the fix: re-run the failed gate AND its neighbors (a fmt fix can shift a test)

## Gotchas
- The pre-commit hook autofixes staged files (eslint/prettier/golangci-lint --fix) — re-run the gates after any hook-triggered edit
- ci-pr runs the quality gates but NOT the cross-compiles — those run only on push to main, so a PR can be green and a main push red (docs/development.md § CI)
- The gate checks here are the local commands; CI's quality.yml runs them as 6 jobs (vet+race+coverage, golangci-lint, vitest, eslint, tsc, and `issue-labels-test` running `node --test` over the `.github/actions/issue-labels` JS suite) — same gates, different packaging; `make check` includes the same JS suite via `tools-test` so local and CI stay in sync
- New code must be covered to keep the total ≥ 90% on agent/ + internal/ + cmd/ — a large feature with thin tests fails CI
- Pre-commit runs the full Go suite when Go files are staged — keep focused tests while iterating, full suite once before commit (CLAUDE.md § Token Efficiency)

## Canonical docs
- docs/development.md — § Gates, § CI
- CLAUDE.md — § Code Rules, § Invariants (the rules behind the checklist)
- .claude/skills/go/references/quality.md — gate details

## Maintenance
Derived view — update this skill in the same commit as any change to
docs/development.md § Gates, .golangci.yml, quality.yml, or the go quality
reference; then run `graphify update .`. Stale if: a gate command above no
longer matches `make check`, the coverage floor changes, or the checklist
diverges from CLAUDE.md § Code Rules.
