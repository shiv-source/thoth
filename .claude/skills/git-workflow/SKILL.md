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
- Reviewing a PR or preparing a change for review
- Not backend/frontend implementation — those are the `go` / `react` skills
- Not note-taking behavior — that's the wiki rulebook (~/.thoth/wiki/CLAUDE.md)

## Key files
- CONTRIBUTING.md — the workflow page: § Workflow, § Before you push
- .github/pull_request_template.md — the PR body shape (Summary / Related issue / Files changed / How verified / Notes)
- .github/workflows/ — ci-pr.yml (PR gates on PRs targeting `main`; quality.yml is path-aware — its internal `changes` job diffs the PR and skips the quality gates for areas the PR didn't touch — docs-only PRs skip them all) · quality.yml (the quality gates, each gated on its own `changes` job's area outputs) · final-gate.yml (single required check + PR report comment; skipped gates count as passing) · ci.yml (push to main adds 5 cross-compiles + frontend build) · pr-assignee.yml (auto-assigns PR committers; runs on every PR, never gated) · issue-labels.yml (applies the form-selected labels to new/edited issues)
- .github/actions/issue-labels/ — reusable composite action: applies the three-tier labels (type, priority, areas) from issue-form answers; JS, add-only
- .github/actions/ci-report/ — reusable composite action: renders the final-gate per-job report (step summary + marker-tagged PR comment) and gates the run; JS (final-gate.yml is just the caller)
- .husky/pre-commit — the commit gate: lint-staged autofixes, plus Go vet/lint/test when Go is staged
- docs/development.md — § Gates (what make check enforces), § CI (workflow mechanics)
- docs/specs/ — untracked design docs for large or cross-package changes (convention, may be empty)
- references/labels.md — the full three-tier label set

## Workflows

### 1. Start a change (branch)
1. **Assigned an issue/feature/bug? Read and analyze it before branching or exploring.** When told to work on a specific issue, feature, or bug, the order is fixed: (1) fetch and read the issue (`gh issue view <n>`), (2) analyze it — understand the problem, scope, and which areas it touches — (3) only then decide where to work (reuse an existing worktree/branch, or create a new one, confirming the target with the user), and (4) start code exploration inside that worktree. Never create a branch or start digging into the codebase before the issue is read and the target is confirmed.
   - A branch/worktree for it already exists? Reuse it — `./scripts/git-worktree.sh list` (bare-clone layout) names the worktree dir to switch into; a standard clone just `git switch <branch>`.
   - Creating new? Use step 2's per-layout commands: `git fetch origin` then `./scripts/git-worktree.sh new <type>/<scope>/<slug>` for the bare-clone layout, or `git switch main && git pull --ff-only && git switch -c <type>/<scope>/<slug>` for a standard clone.
2. Never commit to main (code-rules skill § Repo rules) — main is always deployable; changes live on `<type>/<scope>/<slug>` branches and land via reviewed PRs. Create the branch per your clone layout:
   - **Bare-clone layout** (this repo's container: a dir holding a hidden `.bare` clone + a `.git` gitfile, one working directory per branch) — `git switch main` is impossible (main is checked out in its sibling worktree), so use `./scripts/git-worktree.sh`:
     - `git fetch origin` first — `./scripts/git-worktree.sh new` bases on `origin/main` and does not fetch for you
     - `./scripts/git-worktree.sh new <type>/<scope>/<slug>` creates the branch and its flat-hyphen worktree dir (`feat-api-x` for `feat/api/x`), basing on `origin/main` (override with `--base <ref>`), copies `opencode.json` in, and runs `codegraph init` in the new worktree (best-effort — no-op when codegraph is missing, so branching is never blocked)
     - `./scripts/git-worktree.sh rm <dir-or-branch> [--force]` removes the worktree and deletes its branch
     - `./scripts/git-worktree.sh list` shows all worktrees
     Each worktree is its own checkout, so parallel branches (or agent runs) never collide; `git fetch` in any worktree updates the shared refs for all of them (working trees only change on pull/checkout inside each).
   - **Standard clone**: `git switch main && git pull --ff-only && git switch -c <type>/<scope>/<slug>`
   Fast path when the branch already exists: `./scripts/pr.sh` runs this sync plus the whole PR flow (workflow 3) in one command (in the bare-clone layout it syncs via `git fetch origin` — workflow 3 step 1).
   The pre-commit hook enforces this — `./scripts/main-guard.sh` blocks commits made directly on main.
3. `<type>` is a conventional-commit prefix: feat, fix, perf, ci, docs, refactor, test, chore — `perf` maps to the `performance` type label (CONTRIBUTING.md § Workflow)
4. `<scope>` is the short area name (web, api, index, skills, …); `<slug>` is short kebab-case (lowercase letters, digits, hyphens) — the branch mirrors the commit message `<type>(<scope>): <summary>`, e.g. fix/web/reject-empty-titles
5. Large or cross-package change? Write the design doc first — see workflow 5

### 2. Commit
1. Conventional message: `<type>(<scope>): <summary>`, e.g. `fix: reject-empty-titles`, `feat(skills): add git-workflow` — `<scope>` is the short area name (web, api, index, skills, …); omit it only when no area fits
2. The pre-commit hook runs automatically: lint-staged applies eslint --fix + prettier to staged web/src files and golangci-lint --fix to staged Go files; a Go-staged commit additionally gates on `go vet ./...`, `golangci-lint run`, `go test ./...` (CONTRIBUTING.md § Before you push)
3. Autofixes rewrite your staged files — re-run the relevant tests after a hook-triggered edit
4. No commit-msg hook validates the message — the convention is enforced by review, so get it right on the commit
5. Stage only what the change needs: no secrets, no generated dirs (bin/, web/dist/, internal/webui/dist/, node_modules/, *.db)

### 3. Open a PR
1. **Preferred — deliver with `./scripts/pr.sh`** from the feature branch: it is the single guarded command for the whole flow — syncs main, validates the branch name, derives labels from the branch (validated against references/labels.md), runs `make check` (`--no-check` skips), refreshes the CodeGraph index (`scripts/lib-codegraph.sh` `codegraph_sync`, only when `.codegraph/codegraph.db` exists — best-effort, never blocks), pushes, and creates the PR with the template. `--title` overrides the derived title; repeat `--area <label>` to add areas. Run it on every PR delivery so the guarded flow runs end-to-end. In the bare-clone layout `./scripts/pr.sh` detects the container and syncs via `git fetch origin` instead of `git switch main` (workflow 1 step 2).
2. Manual fallback — push the branch, then create the PR with the `gh` CLI (not the web UI):
   `gh pr create --title "<type>(<scope>): <summary>" --label <type> --label <area>… --template .github/pull_request_template.md`
   — one type label plus every area label the change touches (workflow 4); the web UI auto-fills the PR template, the CLI does not — pass it explicitly with `--template` (verify flags against `gh pr create --help`)
3. Write the body per .github/pull_request_template.md:
   - ## Summary — what changed and why; bullets when they help
   - ## Related issue — `Closes #<n>` (auto-closes the issue on merge); omit when there is no issue
   - ## Files changed — key files/packages and the role of each
   - ## How verified — check the boxes you ran: gofmt/vet clean, go test -race ./..., coverage >= 90% (make cover), golangci-lint run, frontend tsc --noEmit / lint / vitest run, docs updated
   - ## Notes — optional; design decisions, follow-ups
4. Run `make check` before opening — it is everything CI enforces, locally (CONTRIBUTING.md § Before you push)
5. ci-pr quality gates run automatically; final-gate posts its report as a PR comment and must pass before the human merges — don't hand off a red PR

### 4. Label issues and PRs
1. Every issue/PR carries exactly one type label and one label per area it touches; issues also carry one priority label (code-rules skill § Repo rules)
2. Types: bug, feature, enhancement, documentation, chore, refactor, test, performance, ci
3. Areas (package-aligned): api, chat, cli, github, index, search, settings, store, sync, ui, webui, wiki, tooling
4. Priorities (issues only): p-critical, p-high, p-medium, p-low
5. Branch prefixes (8) and type labels (9) overlap but are not identical — choose labels from the label set, not the prefix list
6. `./scripts/pr.sh` enforces this — derived labels are validated against references/labels.md (branch scope → area label, falling back to `tooling`)
7. Issues filed through `.github/ISSUE_TEMPLATE/` get their bare-minimum labels (type, priority, areas) applied automatically from the form answers — `.github/workflows/issue-labels.yml` + the reusable `.github/actions/issue-labels` action. Add-only: labels a human adds afterwards are never removed; blank issues (no form) are skipped.
8. Full data: references/labels.md

### 5. Design doc first (large or cross-package changes)
1. Before implementation, write the design doc to docs/specs/ (untracked — never committed; code-rules skill § Repo rules)
2. The spec in docs/specs/ is the working authority for that change until it lands
3. Small, single-package changes skip it

### 6. Merge is human-only — squash by default
1. A session never merges — it delivers: reviewed PR, green final-gate, labels applied. The human merges (human-in-the-loop delivery; squash by default, "unless the commit history is meaningful" — CONTRIBUTING.md § Workflow)
2. Every PR is reviewed — request a review before handing off (CONTRIBUTING.md § Workflow)
3. final-gate is the single required check: it always renders a per-job report (step summary; on PRs a `<!-- thoth-ci-report -->` tagged comment, updated in place) and fails unless every other job succeeded — jobs skipped because a PR didn't touch their area (quality.yml's `changes` gating) count as passing (docs/development.md § CI)
4. After the human merges, the next change starts with workflow 1's sync — bare-clone layout: `git fetch origin`; standard clone: `git switch main && git pull --ff-only`

## Gotchas
- The pre-commit hook can rewrite your staged files (eslint/prettier/golangci-lint --fix) — re-run tests after any hook-triggered edit
- No secrets in the repo — env vars or placeholders only (code-rules skill § Repo rules)
- Lockfiles (go.sum, pnpm-lock.yaml) are committed; generated dirs (bin/, web/dist/, internal/webui/dist/, node_modules/, *.db) are never committed (code-rules skill § Repo rules)
- ci-pr is the fast feedback loop — it runs no cross-compiles (those run only after push to main) and only the quality gates for the areas a PR touches (docs/development.md § CI)
- A red final-gate means a job failed — read the PR report comment before re-pushing; it is updated in place, not stacked

## Canonical docs
- CONTRIBUTING.md — § Workflow, § Before you push
- docs/development.md — § Gates, § CI, § Rules that keep the codebase healthy
- .github/pull_request_template.md — PR body shape
- code-rules skill § Repo rules — the invariants behind these procedures

## Maintenance
Derived view — update this skill in the same commit as any of: CONTRIBUTING.md,
code-rules skill § Repo rules, .github/pull_request_template.md, .github/workflows/*,
or .husky/pre-commit behavior changes. Stale if:
the branch or PR commands in CONTRIBUTING.md differ from these steps, the label
set changes, a workflow file changes gate names, order, or the report marker,
`gh pr create --help` no longer shows the `--template` flag, scripts/pr.sh,
scripts/git-worktree.sh, scripts/lib-worktree.sh, or scripts/lib-codegraph.sh
change the steps they automate, or the label tables in references/labels.md
change shape
(scripts/pr.sh parses `| label |` rows under `## Types` / `## Areas`).
