# Documentation checklist (docs, skills, PR)

Walk shared.md first; this file adds the docs-and-delivery items.
Each item cites its source.

- [ ] docs/ page updated in the same commit when behavior changes (code-rules skill § Repo rules)
- [ ] docs/index.md map updated when a page is added or removed
- [ ] Affected skills updated in the same commit (suite spec § Maintenance — same-commit contract)
- [ ] PR body follows .github/pull_request_template.md; "How verified" lists only what actually ran (git-workflow skill § 3)
- [ ] PR carries one type label + every touched area label (git-workflow skill § 4)
- [ ] No secrets or generated files committed (code-rules skill § Repo rules)

Canonical: code-rules skill § Repo rules · docs/index.md · .github/pull_request_template.md

Stale if: the PR template, label workflow, or code-rules skill § Repo rules change
without this file following.
