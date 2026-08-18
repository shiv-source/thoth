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

Changes here carry the `tooling` area label.
