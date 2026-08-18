# Claude CLI Flags

> 9 nodes · cohesion 0.31

## Key Concepts

- **CLIClient** (9 connections) — `internal/claude/client.go`
- **.Start()** (7 connections) — `internal/claude/client.go`
- **stderrTail** (5 connections) — `internal/claude/client.go`
- **.args()** (4 connections) — `internal/claude/client.go`
- **.commonTail()** (3 connections) — `internal/claude/client.go`
- **.persistentArgs()** (3 connections) — `internal/claude/client.go`
- **.Write()** (2 connections) — `internal/claude/client.go`
- **.dir()** (1 connections) — `internal/claude/client.go`
- **.String()** (1 connections) — `internal/claude/client.go`

## Relationships

- [Claude Process Pool](Claude_Process_Pool.md) (5 shared connections)
- [Claude CLI Client](Claude_CLI_Client.md) (3 shared connections)
- [Claude Test Fakes](Claude_Test_Fakes.md) (2 shared connections)
- [Claude Event Types](Claude_Event_Types.md) (1 shared connections)
- [Stale Lock Client](Stale_Lock_Client.md) (1 shared connections)
- [Process Group Kill](Process_Group_Kill.md) (1 shared connections)

## Source Files

- `internal/claude/client.go`

## Audit Trail

- EXTRACTED: 23 (96%)
- INFERRED: 1 (4%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*