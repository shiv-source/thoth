# scripts/

Repo helper scripts — run manually by developers and agents, not part of CI.
(CI lives in `.github/workflows/`; commit hooks in `.husky/`.)

- `pr.sh` — one command for the whole contribute flow: preflight, sync with
  main, branch-name check, label derivation (parsed from
  `.claude/skills/git-workflow/references/labels.md`), `make check` (skip
  with `--no-check`), push, and `gh pr create` with the template.
- `main-guard.sh` — blocks commits made directly on `main`; wired into
  `.husky/pre-commit` (code-rules skill § Repo rules).
- `git-worktree.sh` — worktree helper for the bare-clone layout (a hidden
  `.bare` clone with one working directory per branch, e.g. `feat-api-x` for
  branch `feat/api/x`): `new <type>/<scope>/<slug>` creates the branch and
  its flat-hyphen worktree (inheriting `opencode.json`), `rm` removes a
  worktree and deletes its branch, `list` shows `git worktree list`. Run it
  from anywhere inside the container. New worktrees get the CodeGraph MCP
  config via `opencode.json` but no index — run `codegraph init` inside each
  new worktree to build its `.codegraph/` graph (the helper prints this hint).
- `lib-worktree.sh` — shared bare-clone-layout helpers (sourced by
  `git-worktree.sh` and `pr.sh`); nothing runs on load.

Changes here carry the `tooling` area label.
