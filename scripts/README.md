# scripts/

Repo helper scripts — run manually by developers and agents, not part of CI.
(CI lives in `.github/workflows/`; commit hooks in `.husky/`.)

- `pr.sh` — one command for the whole contribute flow: preflight, sync with
  main, branch-name check, label derivation (parsed from
  `.claude/skills/git-workflow/references/labels.md`), `make check` (skip
  with `--no-check`), push, and `gh pr create` with the template.
- `main-guard.sh` — blocks commits made directly on `main` or a detached
  HEAD; wired into `.husky/pre-commit` (code-rules skill § Repo rules).
- `lib-codegraph.sh` — shared CodeGraph helpers (source-only): `codegraph_ensure`
  (init-if-missing else sync) runs from `.husky/pre-commit` and
  `.husky/post-checkout`; `codegraph_sync` (sync-only) runs from `pr.sh`.
  Both best-effort — a missing or failing codegraph never blocks a git op.
- `token-guard.sh` — read-guard hooks for Claude Code (`.claude/settings.json`).
- `setup.sh` — one-command developer setup (toolchain, deps, embed, doctor).
- `pr_test.sh` — smoke test for `pr.sh`'s main-sync step.

Changes here carry the `tooling` area label.
