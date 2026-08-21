<!--
Conventions:
- Title: conventional commit (`feat:`, `fix:`, `ci:`, `docs:`, `refactor:`, `test:`, `chore:`)
- `ci-pr` runs the quality gates; `final-gate` posts its report as a comment and must pass before merging
- Never commit secrets or generated files (`web/dist/`, `internal/webui/dist/`, `*.db`)
-->

## Summary

<!--
What changed and why? Give the full picture — and use bullet points when it helps:

- change 1 — why it was needed
- change 2 — why it was needed
-->

## Related issue

<!--
The GitHub issue(s) this PR addresses. `Closes #<n>` auto-closes the issue
when the PR merges, e.g. `Closes #43`. Omit this section when there is no issue.
-->

Closes #<issue>

## Files changed

<!-- Key files/packages touched and the role of each, e.g.
- internal/api/chat.go — cancel frames now stop the turn
-->

## How verified

- [ ] `gofmt -l` empty and `go vet ./...` clean
- [ ] `go test -race ./...` passes
- [ ] Coverage ≥ 90% on `agent/` + `internal/` + `cmd/` (`make cover`)
- [ ] `golangci-lint run` clean
- [ ] Frontend `pnpm exec tsc --noEmit` · `pnpm run lint` · `pnpm exec vitest run`
- [ ] Docs updated in `docs/` if behavior changed

## Notes

<!-- Anything reviewers should know: design decisions, follow-ups, screenshots. Optional — delete if empty. -->
