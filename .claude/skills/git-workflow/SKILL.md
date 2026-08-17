---
name: git-workflow
description: >-
  Thoth contribution workflow — branching, commits, PRs, labels, squash
  merge, CI gates. Fires on: commit, PR, pull request, branch, labels,
  squash merge.
---

# Git workflow — contribution procedures & expectations

## When to use
- Starting any change to this repo: branching, committing, opening a PR
- Labeling issues or PRs, or interpreting ci-pr / final-gate results
- Reviewing a PR or preparing a change for review
- Not backend/frontend implementation — those are the `go` / `react` skills
- Not note-taking behavior — that's the wiki rulebook (~/.thoth/wiki/CLAUDE.md)

## Key files
- CONTRIBUTING.md — the workflow page: § Workflow, § Before you push
- .github/pull_request_template.md — the PR body shape (Summary / Files changed / How verified / Notes)
- .github/workflows/ — ci-pr.yml (PR gates) · quality.yml (the five quality gates) · final-gate.yml (single required check + PR report comment) · ci.yml (push to main adds 5 cross-compiles + frontend build)
- .husky/pre-commit — the commit gate: lint-staged autofixes, plus Go vet/lint/test when Go is staged
- docs/development.md — § Gates (what make check enforces), § CI (workflow mechanics)
- docs/specs/ — untracked design docs for large or cross-package changes (convention, may be empty)
- references/labels.md — the full three-tier label set

## Workflows

### 1. Start a change (branch)
1. Never commit to main (CLAUDE.md § Repo rules) — sync and branch first:
   `git switch main && git pull --ff-only && git switch -c <type>/<scope>/<slug>`
2. `<type>` is a conventional-commit prefix: feat, fix, perf, ci, docs, refactor, test, chore — `perf` maps to the `performance` type label (CONTRIBUTING.md § Workflow)
3. `<scope>` is the short area name (web, api, index, skills, …); `<slug>` is short kebab-case (lowercase letters, digits, hyphens) — the branch mirrors the commit message `<type>(<scope>): <summary>`, e.g. fix/web/reject-empty-titles
4. Large or cross-package change? Write the design doc first — see workflow 5

### 2. Commit
1. Conventional message: `<type>(<scope>): <summary>`, e.g. `fix: reject-empty-titles`, `feat(skills): add git-workflow` — `<scope>` is the short area name (web, api, index, skills, …); omit it only when no area fits
2. The pre-commit hook runs automatically: lint-staged applies eslint --fix + prettier to staged web/src files and golangci-lint --fix to staged Go files; a Go-staged commit additionally gates on `go vet ./...`, `golangci-lint run`, `go test ./...` (CONTRIBUTING.md § Before you push)
3. Autofixes rewrite your staged files — re-run the relevant tests after a hook-triggered edit
4. No commit-msg hook validates the message — the convention is enforced by review, so get it right on the commit
5. Stage only what the change needs: no secrets, no generated dirs (bin/, web/dist/, internal/webui/dist/, node_modules/, *.db)

### 3. Open a PR
1. Push the branch, then create the PR with the `gh` CLI (not the web UI):
   `gh pr create --title "<type>(<scope>): <summary>" --label <type> --label <area>… --template .github/pull_request_template.md`
   — one type label plus every area label the change touches (workflow 4); the web UI auto-fills the PR template, the CLI does not — pass it explicitly with `--template` (verify flags against `gh pr create --help`)
2. Write the body per .github/pull_request_template.md:
   - ## Summary — what changed and why; bullets when they help
   - ## Files changed — key files/packages and the role of each
   - ## How verified — check the boxes you ran: gofmt/vet clean, go test -race ./..., coverage >= 80% (make cover), golangci-lint run, frontend tsc --noEmit / lint / vitest run, docs updated
   - ## Notes — optional; design decisions, follow-ups
3. Run `make check` before opening — it is everything CI enforces, locally (CONTRIBUTING.md § Before you push)
4. ci-pr quality gates run automatically; final-gate posts its report as a PR comment and must pass before the human merges — don't hand off a red PR

### 4. Label issues and PRs
1. Every issue/PR carries exactly one type label and one label per area it touches; issues also carry one priority label (CLAUDE.md § Repo rules)
2. Types: bug, feature, enhancement, documentation, chore, refactor, test, performance, ci
3. Areas (package-aligned): api, chat, cli, github, index, search, settings, store, sync, ui, webui, wiki
4. Priorities (issues only): p-critical, p-high, p-medium, p-low
5. Branch prefixes (7) and type labels (9) overlap but are not identical — choose labels from the label set, not the prefix list
6. Full data: references/labels.md

### 5. Design doc first (large or cross-package changes)
1. Before implementation, write the design doc to docs/specs/ (untracked — never committed; CLAUDE.md § Repo rules)
2. The spec in docs/specs/ is the working authority for that change until it lands
3. Small, single-package changes skip it

### 6. Merge is human-only — squash by default
1. A session never merges — it delivers: reviewed PR, green final-gate, labels applied. The human merges (human-in-the-loop delivery; squash by default, "unless the commit history is meaningful" — CONTRIBUTING.md § Workflow)
2. Every PR is reviewed — request a review before handing off (CONTRIBUTING.md § Workflow)
3. final-gate is the single required check: it always renders a per-job report (step summary; on PRs a `<!-- thoth-ci-report -->` tagged comment, updated in place) and fails unless every other job succeeded (docs/development.md § CI)
4. After the human merges, the next change starts with workflow 1's sync: `git switch main && git pull --ff-only`

## Gotchas
- The pre-commit hook can rewrite your staged files (eslint/prettier/golangci-lint --fix) — re-run tests after any hook-triggered edit
- No secrets in the repo — env vars or placeholders only (CLAUDE.md § Repo rules)
- Lockfiles (go.sum, pnpm-lock.yaml) are committed; generated dirs (bin/, web/dist/, internal/webui/dist/, node_modules/, *.db) are never committed (CLAUDE.md § Repo rules)
- ci-pr is the fast feedback loop — it runs no cross-compiles; the extra builds run only after push to main (docs/development.md § CI)
- A red final-gate means a job failed — read the PR report comment before re-pushing; it is updated in place, not stacked

## Canonical docs
- CONTRIBUTING.md — § Workflow, § Before you push
- docs/development.md — § Gates, § CI, § Rules that keep the codebase healthy
- .github/pull_request_template.md — PR body shape
- CLAUDE.md § Repo rules — the invariants behind these procedures

## Maintenance
Derived view — update this skill in the same commit as any of: CONTRIBUTING.md,
CLAUDE.md § Repo rules, .github/pull_request_template.md, .github/workflows/*,
or .husky/pre-commit behavior changes; then run `graphify update .`. Stale if:
the branch or PR commands in CONTRIBUTING.md differ from these steps, the label
set changes, a workflow file changes gate names, order, or the report marker,
or `gh pr create --help` no longer shows the `--template` flag.
