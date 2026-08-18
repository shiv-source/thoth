# SQLite Index Engine

> 60 nodes · cohesion 0.07

## Key Concepts

- **Open()** (30 connections) — `internal/index/index.go`
- **discardLog()** (14 connections) — `internal/index/index_test.go`
- **openTest()** (13 connections) — `internal/index/index_test.go`
- **index_test.go** (12 connections) — `internal/index/index_test.go`
- **Index** (11 connections) — `internal/index/index.go`
- **Watch()** (11 connections) — `internal/index/watcher.go`
- **ParseNote()** (11 connections) — `internal/wiki/note.go`
- **apply()** (9 connections) — `internal/index/watcher.go`
- **index.go** (7 connections) — `internal/index/index.go`
- **apply_test.go** (6 connections) — `internal/index/apply_test.go`
- **note_test.go** (6 connections) — `internal/wiki/note_test.go`
- **.Sync()** (5 connections) — `internal/index/sync.go`
- **TestApply()** (5 connections) — `internal/index/apply_test.go`
- **TestApplyClosedIndexLogsAndContinues()** (5 connections) — `internal/index/apply_test.go`
- **TestApplyPathOutsideRoot()** (5 connections) — `internal/index/apply_test.go`
- **TestApplyUnreadablePath()** (5 connections) — `internal/index/apply_test.go`
- **TestWatchErrorOnMissingRoot()** (5 connections) — `internal/index/apply_test.go`
- **TestWatchReturnsOnCancel()** (5 connections) — `internal/index/apply_test.go`
- **upsert()** (5 connections) — `internal/index/index.go`
- **sync_test.go** (5 connections) — `internal/index/sync_test.go`
- **Note** (4 connections) — `internal/index/index.go`
- **del()** (4 connections) — `internal/index/index.go`
- **TestClosedIndexErrors()** (4 connections) — `internal/index/index_test.go`
- **TestSyncErrorOnMissingRoot()** (4 connections) — `internal/index/sync_test.go`
- **TestSyncIndexesTree()** (4 connections) — `internal/index/sync_test.go`
- *... and 35 more nodes in this community*

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (33 shared connections)
- [Serve Command](Serve_Command.md) (10 shared connections)
- [GitHub Repo Storage](GitHub_Repo_Storage.md) (5 shared connections)
- [Doctor Diagnostics](Doctor_Diagnostics.md) (3 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (1 shared connections)
- [Doctor Tests](Doctor_Tests.md) (1 shared connections)
- [Claude Test Fakes](Claude_Test_Fakes.md) (1 shared connections)

## Source Files

- `internal/index/apply_test.go`
- `internal/index/index.go`
- `internal/index/index_test.go`
- `internal/index/sync.go`
- `internal/index/sync_test.go`
- `internal/index/watcher.go`
- `internal/index/watcher_test.go`
- `internal/wiki/note.go`
- `internal/wiki/note_test.go`

## Audit Trail

- EXTRACTED: 129 (74%)
- INFERRED: 45 (26%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*