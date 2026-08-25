---
name: git-workflow
description: >-
  Thoth contribution workflow — branching, commits, PRs, labels, squash
  merge, CI gates. Fires on: commit, PR, pull request, branch, labels,
  squash merge, merge.
---

# Git workflow — contribution procedures & expectations

## When to use
- Starting any change to this repo: branching, committing, opening a PR
- Labeling issues or PRs, or interpreting ci-pr / final-gate results
- Not backend/frontend implementation — those are the `go` / `react` skills
- Not note-taking behavior — that's the wiki rulebook (~/.thoth/wiki/CLAUDE.md)

## Key files
- CONTRIBUTING.md — the workflow page (§ Workflow, § Before you push)
- .github/pull_request_template.md — the PR body shape
- .github/workflows/ — ci-pr.yml (PR gates) · quality.yml (area-gated quality) · final-gate.yml (single required check + report comment) · ci.yml (push-to-main cross-compiles)
- .husky/pre-commit — the commit gate: CodeGraph generate/sync (best-effort), main-guard, lint-staged autofixes, Go vet/lint/test when Go is staged
- references/labels.md — the three-tier label set
- docs/development.md — § Gates, § CI

## Workflows

### 1. Start a change (branch)
1. Assigned an issue/feature/bug? Read and analyze it before branching: (1) `gh issue view <n>`, (2) scope + areas, (3) confirm the target with the user (reuse the current branch or create a new one), (4) then explore.
2. Never commit to main (code-rules skill § Repo rules) — changes live on `<type>/<scope>/<slug>` branches:
   `git switch main && git pull --ff-only && git switch -c <type>/<scope>/<slug>`
3. `<type>` is a conventional-commit prefix: feat, fix, perf, ci, docs, refactor, test, chore. `<scope>` is the short area name (web, api, index, skills, …); `<slug>` is short kebab-case. The branch mirrors the commit message.
4. Large or cross-package change? Write the design doc first (workflow 5).

### 2. Commit
1. Conventional message: `<type>(<scope>): <summary>` — omit the scope only when no area fits.
2. The pre-commit hook runs automatically: CodeGraph generate/sync (best-effort), main-guard (no commits on main), lint-staged autofixes (eslint/prettier/golangci-lint --fix), and Go vet/lint/test when Go is staged.
3. Autofixes rewrite staged files — re-run the relevant tests after a hook-triggered edit.
4. No commit-msg hook validates the message — the convention is enforced by review.
5. Stage only what the change needs: no secrets, no generated dirs (bin/, web/dist/, internal/webui/dist/, node_modules/, *.db).

### 3. Open a PR
1. **Preferred — deliver with `./scripts/pr.sh`** from the feature branch: it runs the whole flow in one command — sync with main, branch-name validation, label derivation (validated against references/labels.md), `make check` (`--no-check` skips), CodeGraph sync, push, and `gh pr create` with the template. `--title` overrides the derived title; repeat `--area <label>` to add areas. pr.sh pre-fills the template's `## Summary` (from the branch's commit subjects) and `## Files changed` (from `main...HEAD`), so a non-interactive run still ships a real description.
2. Manual fallback — push, then `gh pr create --title "<type>(<scope>): <summary>" --label <type> --label <area>… --template .github/pull_request_template.md`.
3. Complete the body (pr.sh fills Summary + Files changed; you finish the rest): Related issue — set `Closes #<n>` to the issue number or delete the line when there's no issue; How verified — check only what actually ran; Notes — optional. Non-interactive runs skip `$EDITOR`, so edit afterward with `gh pr edit <n>`.
4. ci-pr quality gates run automatically; final-gate must pass before the human merges — don't hand off a red PR.

### 4. Label issues and PRs
1. Every issue/PR carries exactly one type label and one per area touched; issues also carry one priority (code-rules skill § Repo rules).
2. The label set lives in references/labels.md — types, areas (package-aligned), priorities (issues only). Branch prefixes (8) and type labels (9) overlap but are not identical.
3. `./scripts/pr.sh` enforces it — derived labels are validated against references/labels.md (branch scope → area label, falling back to `tooling`).
4. Issue-form labels apply automatically via .github/actions/issue-labels (add-only); blank issues are skipped.

### 5. Design doc first (large or cross-package changes)
Write the spec to docs/specs/ (untracked — never committed) before implementation; small, single-package changes skip it.

### 6. Merge is human-only — squash by default
1. A session never merges — it delivers: reviewed PR, green final-gate, labels applied. The human merges (squash unless the history is meaningful).
2. Every PR is reviewed — request a review before hand-off.
3. final-gate is the single required check: it renders a per-job report (a PR comment, updated in place) and fails unless every job succeeded — area-skipped jobs count as passing (docs/development.md § CI).
4. After the human merges, the next change starts with workflow 1's sync.

## Gotchas
- The pre-commit hook can rewrite staged files — re-run tests after any hook-triggered edit.
- ci-pr runs no cross-compiles — those run only after push to main, so a PR can be green and a main push red (docs/development.md § CI).
- A red final-gate means a job failed — read the PR report comment before re-pushing; it is updated in place, not stacked.

## Canonical docs
- CONTRIBUTING.md — § Workflow, § Before you push
- docs/development.md — § Gates, § CI
- .github/pull_request_template.md — PR body shape
- code-rules skill § Repo rules — the invariants behind these procedures

## Maintenance
Derived view — update in the same commit as: CONTRIBUTING.md, code-rules skill § Repo rules, .github/pull_request_template.md, .github/workflows/*, .husky/pre-commit behavior, scripts/pr.sh or scripts/lib-codegraph.sh, or references/labels.md. Stale if the branch/PR commands differ from CONTRIBUTING.md, the label set changes, a workflow gate name/order changes, or the label tables in references/labels.md change shape.
