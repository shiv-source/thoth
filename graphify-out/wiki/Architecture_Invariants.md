# Architecture Invariants

> 11 nodes · cohesion 0.25

## Key Concepts

- **Indexing and search - FTS5 and the file watcher** (8 connections) — `docs/indexing.md`
- **internal/index - search and sync** (6 connections) — `docs/components.md`
- **Incremental index Sync (startup and path change)** (5 connections) — `docs/indexing.md`
- **App layer - single Go binary** (4 connections) — `docs/architecture.md`
- **useSearch - debounced, supersede-guarded search** (4 connections) — `docs/frontend.md`
- **fsnotify watcher - 200 ms debounce** (4 connections) — `docs/indexing.md`
- **Data contract: files are the source of truth, thoth.db is derived** (3 connections) — `docs/architecture.md`
- **thoth serve command** (3 connections) — `docs/cli.md`
- **bm25 ranking with title weighted 8x** (3 connections) — `docs/indexing.md`
- **Project invariants (files as source of truth, percent-w errors, no globals)** (2 connections) — `CLAUDE.md`
- **internal/api - the Echo server** (2 connections) — `docs/components.md`

## Relationships

- [Package Docs](Package_Docs.md) (7 shared connections)
- [Knowledge Layer Docs](Knowledge_Layer_Docs.md) (5 shared connections)
- [Repo Governance Rules](Repo_Governance_Rules.md) (2 shared connections)
- [Toolchain Citations](Toolchain_Citations.md) (1 shared connections)
- [WS Chat Protocol](WS_Chat_Protocol.md) (1 shared connections)

## Source Files

- `CLAUDE.md`
- `docs/architecture.md`
- `docs/cli.md`
- `docs/components.md`
- `docs/frontend.md`
- `docs/indexing.md`

## Audit Trail

- EXTRACTED: 15 (50%)
- INFERRED: 15 (50%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*