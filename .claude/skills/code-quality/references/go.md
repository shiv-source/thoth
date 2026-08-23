# Go backend checklist (internal/* + cmd/*)

Walk shared.md first; this file adds the backend-specific items.
Each item cites its rule source in the code-rules skill.

- [ ] Wrapped errors compared with `errors.Is`/`errors.As`, never `==` (code-rules skill § Code Rules)
- [ ] context.Context is the first parameter, never stored in structs (code-rules skill § Code Rules)
- [ ] Every `go` statement has a ctx/done-channel; no goroutine outlives its owner (code-rules skill § Memory)
- [ ] Interfaces are small (1–3 methods) and defined at the consumer, not the producer (code-rules skill § Code Rules)
- [ ] `defer` immediately after every acquisition (rows, files, bodies, handles) (code-rules skill § Memory)
- [ ] Shared state behind mutex/atomic — CI runs -race and it must stay green (code-rules skill § Memory)
- [ ] No package-level mutable globals (code-rules skill § Invariants)
- [ ] Exported symbols carry doc comments (code-rules skill § Code Rules)
- [ ] Follows the package's established pattern — hub, process pool, repository, facade at the blast wall (docs/components.md); no parallel ad-hoc structure

Canonical: code-rules skill § Code Rules · § Invariants · § Memory

Stale if: code-rules skill's Go-side rules change without this file following.
