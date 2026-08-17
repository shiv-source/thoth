# GitHub Repo Storage

> 51 nodes · cohesion 0.07

## Key Concepts

- **Open()** (29 connections) — `internal/store/store.go`
- **Store** (15 connections) — `internal/store/store.go`
- **store_test.go** (11 connections) — `internal/store/store_test.go`
- **openTestRepo()** (9 connections) — `internal/github/repo_test.go`
- **Repo** (7 connections) — `internal/github/repo.go`
- **database/sql.DB** (7 connections)
- **OpenRepo()** (7 connections) — `internal/github/repo.go`
- **repo_test.go** (6 connections) — `internal/github/repo_test.go`
- **saved()** (6 connections) — `internal/github/repo_test.go`
- **OpenDB()** (6 connections) — `internal/store/sqlite.go`
- **Auth** (5 connections) — `internal/github/repo.go`
- **migrate()** (5 connections) — `internal/store/migrations.go`
- **store.go** (5 connections) — `internal/store/store.go`
- **TestRepoClear()** (4 connections) — `internal/github/repo_test.go`
- **TestRepoRoundTrip()** (4 connections) — `internal/github/repo_test.go`
- **newID()** (4 connections) — `internal/store/store.go`
- **.Close()** (4 connections) — `internal/store/store.go`
- **time.Time** (3 connections)
- **repo.go** (3 connections) — `internal/github/repo.go`
- **TestRepoClosedErrors()** (3 connections) — `internal/github/repo_test.go`
- **TestRepoSingleRowConstraint()** (3 connections) — `internal/github/repo_test.go`
- **migrations.go** (3 connections) — `internal/store/migrations.go`
- **applyMigration()** (3 connections) — `internal/store/migrations.go`
- **TestClosedStoreErrors()** (3 connections) — `internal/store/store_test.go`
- **TestConversationRoundTrip()** (3 connections) — `internal/store/store_test.go`
- *... and 26 more nodes in this community*

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (21 shared connections)
- [SQLite Index Engine](SQLite_Index_Engine.md) (5 shared connections)
- [Doctor Tests](Doctor_Tests.md) (4 shared connections)
- [Serve Command](Serve_Command.md) (4 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (2 shared connections)
- [Chat Hub](Chat_Hub.md) (2 shared connections)
- [Doctor Diagnostics](Doctor_Diagnostics.md) (2 shared connections)
- [CLI Entry & Init](CLI_Entry_&_Init.md) (1 shared connections)

## Source Files

- `internal/github/repo.go`
- `internal/github/repo_test.go`
- `internal/store/migrations.go`
- `internal/store/sqlite.go`
- `internal/store/store.go`
- `internal/store/store_test.go`

## Audit Trail

- EXTRACTED: 115 (89%)
- INFERRED: 14 (11%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*