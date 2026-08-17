# Chat Hub

> 25 nodes · cohesion 0.17

## Key Concepts

- **Hub** (22 connections) — `internal/api/chat.go`
- **serverMsg** (10 connections) — `internal/api/chat.go`
- **.runTurn()** (9 connections) — `internal/api/chat.go`
- **chat.go** (8 connections) — `internal/api/chat.go`
- **.chat()** (7 connections) — `internal/api/chat.go`
- **NewHub()** (7 connections) — `internal/api/chat.go`
- **.turnWriter()** (6 connections) — `internal/api/chat.go`
- **.handleSend()** (5 connections) — `internal/api/chat.go`
- **.record()** (5 connections) — `internal/api/chat.go`
- **.finishTurn()** (4 connections) — `internal/api/chat.go`
- **.handleResume()** (4 connections) — `internal/api/chat.go`
- **writeLoop()** (4 connections) — `internal/api/chat.go`
- **Client** (4 connections) — `internal/claude/client.go`
- **.replay()** (3 connections) — `internal/api/chat.go`
- **turn** (3 connections) — `internal/api/chat.go`
- **.cancelTurn()** (2 connections) — `internal/api/chat.go`
- **.conversationExists()** (2 connections) — `internal/api/chat.go`
- **.persistTurn()** (2 connections) — `internal/api/chat.go`
- **.sessionID()** (2 connections) — `internal/api/chat.go`
- **allowLocalOrigin()** (2 connections) — `internal/api/chat.go`
- **truncateTitle()** (2 connections) — `internal/api/chat.go`
- **clientMsg** (1 connections) — `internal/api/chat.go`
- **context.CancelFunc** (1 connections)
- **strings.Builder** (1 connections)
- **echo.Context** (1 connections)

## Relationships

- [API Server & DTOs](API_Server_&_DTOs.md) (3 shared connections)
- [Claude CLI Client](Claude_CLI_Client.md) (3 shared connections)
- [GitHub Repo Storage](GitHub_Repo_Storage.md) (2 shared connections)
- [Serve Command](Serve_Command.md) (2 shared connections)
- [Claude Test Fakes](Claude_Test_Fakes.md) (2 shared connections)
- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (2 shared connections)
- [Stale Lock Client](Stale_Lock_Client.md) (1 shared connections)

## Source Files

- `internal/api/chat.go`
- `internal/claude/client.go`

## Audit Trail

- EXTRACTED: 65 (98%)
- INFERRED: 1 (2%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*