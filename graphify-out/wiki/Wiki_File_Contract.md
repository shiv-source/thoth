# Wiki File Contract

> 13 nodes · cohesion 0.26

## Key Concepts

- **New()** (13 connections) — `internal/wiki/wiki.go`
- **Wiki** (8 connections) — `internal/wiki/wiki.go`
- **wiki_test.go** (5 connections) — `internal/wiki/wiki_test.go`
- **wiki.go** (4 connections) — `internal/wiki/wiki.go`
- **TestWikiReadAndTree()** (4 connections) — `internal/wiki/wiki_test.go`
- **TestWikiNotExists()** (3 connections) — `internal/wiki/wiki_test.go`
- **TestWikiReadMissingNote()** (3 connections) — `internal/wiki/wiki_test.go`
- **TestWikiTreeErrorOnMissingRoot()** (3 connections) — `internal/wiki/wiki_test.go`
- **TestWikiTreeErrorOnUnreadableSubdir()** (3 connections) — `internal/wiki/wiki_test.go`
- **tree()** (3 connections) — `internal/wiki/wiki.go`
- **Node** (3 connections) — `internal/wiki/wiki.go`
- **.Tree()** (3 connections) — `internal/wiki/wiki.go`
- **.Exists()** (1 connections) — `internal/wiki/wiki.go`

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (7 shared connections)
- [Serve Command](Serve_Command.md) (6 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (2 shared connections)
- [Wiki Scaffolding](Wiki_Scaffolding.md) (1 shared connections)

## Source Files

- `internal/wiki/wiki.go`
- `internal/wiki/wiki_test.go`

## Audit Trail

- EXTRACTED: 30 (83%)
- INFERRED: 6 (17%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*