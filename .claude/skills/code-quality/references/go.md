# Go backend checklist (internal/* + cmd/*)

Walk shared.md first; this file adds the backend-specific items.
Each item cites its rule source in CLAUDE.md.

- [ ] Wrapped errors compared with `errors.Is`/`errors.As`, never `==` (CLAUDE.md § Code Rules)
- [ ] context.Context is the first parameter, never stored in structs (CLAUDE.md § Code Rules)
- [ ] Every `go` statement has a ctx/done-channel; no goroutine outlives its owner (CLAUDE.md § Memory)
- [ ] Interfaces are small (1–3 methods) and defined at the consumer, not the producer (CLAUDE.md § Code Rules)
- [ ] `defer` immediately after every acquisition (rows, files, bodies, handles) (CLAUDE.md § Memory)
- [ ] Shared state behind mutex/atomic — CI runs -race and it must stay green (CLAUDE.md § Memory)
- [ ] No package-level mutable globals (CLAUDE.md § Invariants)
- [ ] Exported symbols carry doc comments (CLAUDE.md § Code Rules)
- [ ] Follows the package's established pattern — hub, process pool, repository, facade at the blast wall (docs/components.md); no parallel ad-hoc structure

Canonical: CLAUDE.md § Code Rules · § Invariants · § Memory

Stale if: CLAUDE.md's Go-side rules change without this file following.
