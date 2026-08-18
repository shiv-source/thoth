# Claude Test Fakes

> 10 nodes · cohesion 0.33

## Key Concepts

- **context.Context** (19 connections)
- **EventWriter** (9 connections) — `internal/claude/events.go`
- **.start()** (8 connections) — `internal/claude/persistent.go`
- **.Start()** (5 connections) — `internal/claude/persistent.go`
- **.Start()** (3 connections) — `internal/api/chat_test.go`
- **.Start()** (3 connections) — `internal/api/chat_test.go`
- **.Start()** (3 connections) — `internal/api/chat_test.go`
- **.Start()** (3 connections) — `internal/claude/fake.go`
- **ctxAwareFake** (2 connections) — `internal/api/chat_test.go`
- **hangClient** (2 connections) — `internal/api/chat_test.go`

## Relationships

- [Claude Process Pool](Claude_Process_Pool.md) (8 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (3 shared connections)
- [GitHub API Client](GitHub_API_Client.md) (3 shared connections)
- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (2 shared connections)
- [Stale Lock Client](Stale_Lock_Client.md) (2 shared connections)
- [Chat Hub](Chat_Hub.md) (2 shared connections)
- [Claude CLI Flags](Claude_CLI_Flags.md) (2 shared connections)
- [Doctor Diagnostics](Doctor_Diagnostics.md) (2 shared connections)
- [Serve Command](Serve_Command.md) (1 shared connections)
- [SQLite Index Engine](SQLite_Index_Engine.md) (1 shared connections)
- [Claude Event Types](Claude_Event_Types.md) (1 shared connections)

## Source Files

- `internal/api/chat_test.go`
- `internal/claude/events.go`
- `internal/claude/fake.go`
- `internal/claude/persistent.go`

## Audit Trail

- EXTRACTED: 42 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*