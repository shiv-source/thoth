# scripts/

Repo helper scripts — run manually by developers and agents, not part of CI.
(CI lives in `.github/workflows/`; commit hooks in `.husky/`.)

- `graph-check.sh` — staleness guard for the committed graphify graph: exits 1
  when code (Go/TS/TSX) changed — uncommitted or since the last graph
  refresh; run `graphify update .` and commit the refresh.
- `pr.sh` — one command for the whole contribute flow: preflight, sync with
  main, branch-name check, graph-check, label derivation (parsed from
  `.claude/skills/git-workflow/references/labels.md`), `make check` (skip
  with `--no-check`), push, and `gh pr create` with the template.
- `main-guard.sh` — blocks commits made directly on `main`; wired into
  `.husky/pre-commit` (CLAUDE.md § Repo rules).
- `git-worktree.sh` — worktree helper for the bare-clone layout (a hidden
  `.bare` clone with one working directory per branch, e.g. `feat-api-x` for
  branch `feat/api/x`): `new <type>/<scope>/<slug>` creates the branch and
  its flat-hyphen worktree (inheriting `opencode.json`), `rm` removes a
  worktree and deletes its branch, `list` shows `git worktree list`. Run it
  from anywhere inside the container.
- `lib-worktree.sh` — shared bare-clone-layout helpers (sourced by
  `git-worktree.sh` and `pr.sh`); nothing runs on load.

Changes here carry the `tooling` area label.
