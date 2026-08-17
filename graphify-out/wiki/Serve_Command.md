# Serve Command

> 18 nodes · cohesion 0.22

## Key Concepts

- **runServe()** (22 connections) — `internal/cli/serve.go`
- **log/slog.Logger** (16 connections)
- **onSettingsSaved()** (12 connections) — `internal/cli/serve.go`
- **serve.go** (10 connections) — `internal/cli/serve.go`
- **TestOnSettingsSavedFailureLeavesRootUntouched()** (8 connections) — `internal/cli/serve_test.go`
- **TestOnSettingsSavedSwitchesRootAndRestartsWatcher()** (8 connections) — `internal/cli/serve_test.go`
- **ensureWiki()** (7 connections) — `internal/cli/serve.go`
- **rootHolder** (6 connections) — `internal/cli/serve.go`
- **newRootHolder()** (5 connections) — `internal/cli/serve.go`
- **openIndex()** (5 connections) — `internal/cli/serve.go`
- **serveUntilShutdown()** (4 connections) — `internal/cli/serve.go`
- **resolveClaudeBin()** (3 connections) — `internal/cli/serve.go`
- **serve_test.go** (3 connections) — `internal/cli/serve_test.go`
- **.get()** (2 connections) — `internal/cli/serve.go`
- **thothDir()** (2 connections) — `internal/cli/serve.go`
- **.set()** (1 connections) — `internal/cli/serve.go`
- **sync.RWMutex** (1 connections)
- **echo.Echo** (1 connections)

## Relationships

- [SQLite Index Engine](SQLite_Index_Engine.md) (10 shared connections)
- [Wiki File Contract](Wiki_File_Contract.md) (6 shared connections)
- [CLI Entry & Init](CLI_Entry_&_Init.md) (4 shared connections)
- [Wiki Scaffolding](Wiki_Scaffolding.md) (4 shared connections)
- [GitHub Repo Storage](GitHub_Repo_Storage.md) (4 shared connections)
- [Claude CLI Client](Claude_CLI_Client.md) (4 shared connections)
- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (4 shared connections)
- [Doctor Diagnostics](Doctor_Diagnostics.md) (3 shared connections)
- [Chat Hub](Chat_Hub.md) (2 shared connections)
- [Config Expansion](Config_Expansion.md) (2 shared connections)
- [CLI Banner](CLI_Banner.md) (2 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (1 shared connections)

## Source Files

- `internal/cli/serve.go`
- `internal/cli/serve_test.go`

## Audit Trail

- EXTRACTED: 76 (92%)
- INFERRED: 7 (8%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*