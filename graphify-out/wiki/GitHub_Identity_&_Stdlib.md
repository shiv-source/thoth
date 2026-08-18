# GitHub Identity & Stdlib

> 121 nodes · cohesion 0.06

## Key Concepts

- **testing.T** (259 connections)
- **testDeps()** (68 connections) — `internal/api/health_test.go`
- **New()** (63 connections) — `internal/api/server.go`
- **chat_test.go** (28 connections) — `internal/api/chat_test.go`
- **wsURL()** (25 connections) — `internal/api/chat_test.go`
- **readMsg()** (24 connections) — `internal/api/chat_test.go`
- **New()** (14 connections) — `internal/github/client.go`
- **doJSON()** (13 connections) — `internal/api/github_test.go`
- **putSettingsReq()** (13 connections) — `internal/api/settings_test.go`
- **github/client_test.go** (13 connections) — `internal/github/client_test.go`
- **newStubClient()** (13 connections) — `internal/github/client_test.go`
- **api/settings_test.go** (12 connections) — `internal/api/settings_test.go`
- **openTestRepo()** (12 connections) — `internal/settings/settings_test.go`
- **github_test.go** (10 connections) — `internal/api/github_test.go`
- **newLoggingServer()** (9 connections) — `internal/api/logging_test.go`
- **notes_test.go** (9 connections) — `internal/api/notes_test.go`
- **settings/settings_test.go** (8 connections) — `internal/settings/settings_test.go`
- **healthyThothDir()** (7 connections) — `internal/api/doctor_test.go`
- **TestConnectGitHub()** (7 connections) — `internal/api/github_test.go`
- **getSettingsReq()** (7 connections) — `internal/api/settings_test.go`
- **TestChatCancelBeforeSendIsNoop()** (6 connections) — `internal/api/chat_test.go`
- **TestChatCancelStopsInFlightTurn()** (6 connections) — `internal/api/chat_test.go`
- **TestChatForwardsThinkingFrames()** (6 connections) — `internal/api/chat_test.go`
- **TestChatHubCancellationEndsTurns()** (6 connections) — `internal/api/chat_test.go`
- **TestChatNewChatCancelsBusyTurn()** (6 connections) — `internal/api/chat_test.go`
- *... and 96 more nodes in this community*

## Relationships

- [Claude CLI Client](Claude_CLI_Client.md) (39 shared connections)
- [SQLite Index Engine](SQLite_Index_Engine.md) (33 shared connections)
- [Doctor Diagnostics](Doctor_Diagnostics.md) (23 shared connections)
- [GitHub Repo Storage](GitHub_Repo_Storage.md) (21 shared connections)
- [Doctor Tests](Doctor_Tests.md) (16 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (10 shared connections)
- [Claude Event Types](Claude_Event_Types.md) (10 shared connections)
- [CLI Entry & Init](CLI_Entry_&_Init.md) (10 shared connections)
- [Wiki Scaffolding](Wiki_Scaffolding.md) (8 shared connections)
- [Wiki File Contract](Wiki_File_Contract.md) (7 shared connections)
- [CLI Banner](CLI_Banner.md) (4 shared connections)
- [Serve Command](Serve_Command.md) (4 shared connections)

## Source Files

- `internal/api/chat_test.go`
- `internal/api/conversations_test.go`
- `internal/api/doctor_test.go`
- `internal/api/git_test.go`
- `internal/api/github_test.go`
- `internal/api/health_test.go`
- `internal/api/logging_test.go`
- `internal/api/models_test.go`
- `internal/api/notes_test.go`
- `internal/api/server.go`
- `internal/api/settings_test.go`
- `internal/github/client.go`
- `internal/github/client_test.go`
- `internal/settings/settings_test.go`

## Audit Trail

- EXTRACTED: 498 (80%)
- INFERRED: 125 (20%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*