# Doctor Diagnostics

> 53 nodes · cohesion 0.11

## Key Concepts

- **Run()** (31 connections) — `internal/doctor/doctor.go`
- **doctor/doctor_test.go** (24 connections) — `internal/doctor/doctor_test.go`
- **testLog()** (19 connections) — `internal/doctor/doctor_test.go`
- **byName()** (18 connections) — `internal/doctor/doctor_test.go`
- **healthyThothDir()** (17 connections) — `internal/doctor/doctor_test.go`
- **doctor/doctor.go** (15 connections) — `internal/doctor/doctor.go`
- **Check** (13 connections) — `internal/doctor/doctor.go`
- **.runChecks()** (8 connections) — `internal/cli/doctor.go`
- **TestRunAPIHealthy()** (7 connections) — `internal/doctor/doctor_test.go`
- **TestRunAPIPartialFailures()** (7 connections) — `internal/doctor/doctor_test.go`
- **TestRunClaudeLoginUnknown()** (7 connections) — `internal/doctor/doctor_test.go`
- **TestRunClaudeVersionFailure()** (7 connections) — `internal/doctor/doctor_test.go`
- **doctorRunner** (6 connections) — `internal/cli/doctor.go`
- **.repair()** (6 connections) — `internal/cli/doctor.go`
- **cli/doctor.go** (6 connections) — `internal/cli/doctor.go`
- **checkWiki()** (6 connections) — `internal/doctor/doctor.go`
- **seedWikiPath()** (6 connections) — `internal/doctor/doctor_test.go`
- **TestRunAPIWebsocketFails()** (6 connections) — `internal/doctor/doctor_test.go`
- **TestRunDatabaseMissingTables()** (6 connections) — `internal/doctor/doctor_test.go`
- **TestRunHealthy()** (6 connections) — `internal/doctor/doctor_test.go`
- **TestRunIndexOutOfSync()** (6 connections) — `internal/doctor/doctor_test.go`
- **TestRunNonThothProcessOnPort()** (6 connections) — `internal/doctor/doctor_test.go`
- **TestRunNonWALDatabase()** (6 connections) — `internal/doctor/doctor_test.go`
- **TestRunWikiMissingFolders()** (6 connections) — `internal/doctor/doctor_test.go`
- **TestRunWikiMissingNoteIsSkipped()** (6 connections) — `internal/doctor/doctor_test.go`
- *... and 28 more nodes in this community*

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (23 shared connections)
- [Serve Command](Serve_Command.md) (3 shared connections)
- [Wiki Scaffolding](Wiki_Scaffolding.md) (3 shared connections)
- [Doctor Tests](Doctor_Tests.md) (3 shared connections)
- [SQLite Index Engine](SQLite_Index_Engine.md) (3 shared connections)
- [Config Expansion](Config_Expansion.md) (2 shared connections)
- [CLI Entry & Init](CLI_Entry_&_Init.md) (2 shared connections)
- [Claude Test Fakes](Claude_Test_Fakes.md) (2 shared connections)
- [GitHub Repo Storage](GitHub_Repo_Storage.md) (2 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (1 shared connections)

## Source Files

- `internal/cli/doctor.go`
- `internal/doctor/doctor.go`
- `internal/doctor/doctor_test.go`

## Audit Trail

- EXTRACTED: 181 (91%)
- INFERRED: 17 (9%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*