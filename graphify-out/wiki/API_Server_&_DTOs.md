# API Server & DTOs

> 65 nodes · cohesion 0.06

## Key Concepts

- **Deps** (32 connections) — `internal/api/server.go`
- **newServer()** (28 connections) — `internal/api/server.go`
- **internalError()** (16 connections) — `internal/api/notes.go`
- **gitSetup()** (9 connections) — `internal/api/git.go`
- **git.go** (7 connections) — `internal/api/git.go`
- **gitCmd()** (6 connections) — `internal/api/git.go`
- **github.go** (6 connections) — `internal/api/github.go`
- **connectGitHub()** (6 connections) — `internal/api/github.go`
- **getGitHubAuth()** (6 connections) — `internal/api/github.go`
- **models()** (6 connections) — `internal/api/models.go`
- **createConversation()** (5 connections) — `internal/api/conversations.go`
- **deleteConversation()** (5 connections) — `internal/api/conversations.go`
- **getConversation()** (5 connections) — `internal/api/conversations.go`
- **listConversations()** (5 connections) — `internal/api/conversations.go`
- **doctorHandler()** (5 connections) — `internal/api/doctor.go`
- **disconnectGitHub()** (5 connections) — `internal/api/github.go`
- **identityFromAuth()** (5 connections) — `internal/api/github.go`
- **listGitHubRepos()** (5 connections) — `internal/api/github.go`
- **note()** (5 connections) — `internal/api/notes.go`
- **search()** (5 connections) — `internal/api/notes.go`
- **tree()** (5 connections) — `internal/api/notes.go`
- **getSettings()** (5 connections) — `internal/api/settings.go`
- **putSettings()** (5 connections) — `internal/api/settings.go`
- **conversations.go** (4 connections) — `internal/api/conversations.go`
- **echo.Context** (4 connections)
- *... and 40 more nodes in this community*

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (10 shared connections)
- [Chat Hub](Chat_Hub.md) (3 shared connections)
- [Claude Test Fakes](Claude_Test_Fakes.md) (3 shared connections)
- [GitHub Repo Storage](GitHub_Repo_Storage.md) (2 shared connections)
- [Wiki File Contract](Wiki_File_Contract.md) (2 shared connections)
- [Serve Command](Serve_Command.md) (1 shared connections)
- [Doctor Tests](Doctor_Tests.md) (1 shared connections)
- [SQLite Index Engine](SQLite_Index_Engine.md) (1 shared connections)
- [Doctor Diagnostics](Doctor_Diagnostics.md) (1 shared connections)
- [Embedded WebUI](Embedded_WebUI.md) (1 shared connections)
- [Logging Middleware](Logging_Middleware.md) (1 shared connections)

## Source Files

- `internal/api/conversations.go`
- `internal/api/doctor.go`
- `internal/api/git.go`
- `internal/api/github.go`
- `internal/api/health.go`
- `internal/api/models.go`
- `internal/api/notes.go`
- `internal/api/server.go`
- `internal/api/settings.go`
- `internal/assets/assets.go`
- `internal/assets/assets_test.go`
- `internal/github/service.go`
- `internal/wiki/path.go`
- `internal/wiki/path_test.go`
- `internal/wiki/wiki.go`

## Audit Trail

- EXTRACTED: 124 (78%)
- INFERRED: 35 (22%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*