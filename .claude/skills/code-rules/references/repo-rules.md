# Repo rules

Read this at commit/PR time or when changing repo structure. The core rules
live in the code-rules skill; this is the on-demand detail.

- **Branch workflow** — `main` is always deployable; never commit to it directly. Changes live on `<type>/<scope>/<slug>` branches with conventional-commit messages and land via reviewed PRs that a human squash-merges — a session never merges. The full procedure — sync-and-branch commands, commit conventions, PR template sections, label application, squash-merge specifics, and the `ci-pr`/`final-gate` expectations — is the `git-workflow` skill (`.claude/skills/git-workflow/SKILL.md`).
- **No secrets in the repo** — never commit real credentials, tokens, or keys in code, configs, tests, or docs; env vars or placeholders only.
- **Design authority** — design docs for large or cross-package changes live (untracked) in `docs/specs/` when needed; the committed `docs/` pages are the reference for current behavior.
- **Project docs** — committed documentation lives in `docs/` (`index.md` is the hub: architecture, API, CLI, indexing, frontend, security, development). Update the relevant page when behavior changes.
- **Issue/PR labels** — three tiers on GitHub: types, areas, priority. Every issue/PR carries exactly one type and one label per area it touches; issues also carry a priority. The label lists are `.claude/skills/git-workflow/references/labels.md`.
- **Generated/ignored** — `bin/`, `web/dist/`, `internal/webui/dist/`, `node_modules/`, `*.db`.
- **Repo-map freshness** — `CLAUDE.md` and `AGENTS.md` are identical copies of the routing map. Adding, renaming, or deleting any file or directory under `agent/`, `internal/`, `web/src/`, `cmd/`, `docs/`, `.claude/skills/`, or `scripts/` requires updating the routing tree in **both** files in the same commit (`cp CLAUDE.md AGENTS.md`). The pre-commit guard fails if they drift.
