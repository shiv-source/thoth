# Doctor Tests

> 23 nodes · cohesion 0.20

## Key Concepts

- **cli/doctor_test.go** (13 connections) — `internal/cli/doctor_test.go`
- **executeDoctor()** (13 connections) — `internal/cli/doctor_test.go`
- **OpenRepo()** (13 connections) — `internal/settings/settings.go`
- **healthyEnv()** (12 connections) — `internal/cli/doctor_test.go`
- **Repo** (10 connections) — `internal/settings/settings.go`
- **TestDoctorDetectsMissingIndexTables()** (7 connections) — `internal/cli/doctor_test.go`
- **TestDoctorMissingWikiAndFix()** (7 connections) — `internal/cli/doctor_test.go`
- **writeFakeClaude()** (7 connections) — `internal/cli/doctor_test.go`
- **serveThothOnFixedPort()** (5 connections) — `internal/cli/doctor_test.go`
- **TestDoctorDetectsNonWALDatabase()** (5 connections) — `internal/cli/doctor_test.go`
- **TestDoctorFixesOutOfSyncIndex()** (5 connections) — `internal/cli/doctor_test.go`
- **TestDoctorHealthy()** (5 connections) — `internal/cli/doctor_test.go`
- **TestDoctorDetectsBusyPort()** (4 connections) — `internal/cli/doctor_test.go`
- **TestDoctorDetectsMissingClaude()** (4 connections) — `internal/cli/doctor_test.go`
- **TestDoctorFixesMissingDefaultWiki()** (4 connections) — `internal/cli/doctor_test.go`
- **TestDoctorReportsUnknownLogin()** (4 connections) — `internal/cli/doctor_test.go`
- **.Setting()** (3 connections) — `internal/settings/settings.go`
- **settings/settings.go** (2 connections) — `internal/settings/settings.go`
- **.SetSetting()** (2 connections) — `internal/settings/settings.go`
- **.SetSyncResult()** (2 connections) — `internal/settings/settings.go`
- **.SyncEnabled()** (2 connections) — `internal/settings/settings.go`
- **.SyncState()** (2 connections) — `internal/settings/settings.go`
- **.Close()** (1 connections) — `internal/settings/settings.go`

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (16 shared connections)
- [GitHub Repo Storage](GitHub_Repo_Storage.md) (4 shared connections)
- [CLI Entry & Init](CLI_Entry_&_Init.md) (3 shared connections)
- [Wiki Scaffolding](Wiki_Scaffolding.md) (3 shared connections)
- [Doctor Diagnostics](Doctor_Diagnostics.md) (3 shared connections)
- [SQLite Index Engine](SQLite_Index_Engine.md) (1 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (1 shared connections)
- [Serve Command](Serve_Command.md) (1 shared connections)

## Source Files

- `internal/cli/doctor_test.go`
- `internal/settings/settings.go`

## Audit Trail

- EXTRACTED: 79 (96%)
- INFERRED: 3 (4%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*