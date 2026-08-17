# Claude Process Pool

> 22 nodes · cohesion 0.18

## Key Concepts

- **PersistentClient** (18 connections) — `internal/claude/persistent.go`
- **proc** (15 connections) — `internal/claude/persistent.go`
- **.spawnLocked()** (8 connections) — `internal/claude/persistent.go`
- **.dispatch()** (7 connections) — `internal/claude/persistent.go`
- **.evict()** (6 connections) — `internal/claude/persistent.go`
- **startConfig** (6 connections) — `internal/claude/client.go`
- **.getOrSpawn()** (5 connections) — `internal/claude/persistent.go`
- **openDebugDump()** (5 connections) — `internal/claude/client.go`
- **persistent.go** (5 connections) — `internal/claude/persistent.go`
- **.Close()** (4 connections) — `internal/claude/persistent.go`
- **.dumpLine()** (4 connections) — `internal/claude/persistent.go`
- **.unregister()** (4 connections) — `internal/claude/persistent.go`
- **.Flush()** (3 connections) — `internal/claude/persistent.go`
- **os.File** (3 connections)
- **turnFailure()** (3 connections) — `internal/claude/persistent.go`
- **poolEntry** (2 connections) — `internal/claude/persistent.go`
- **time.Duration** (2 connections)
- **.poolSize()** (1 connections) — `internal/claude/persistent.go`
- **CLIClient** (1 connections)
- **bufio.Writer** (1 connections)
- **io.ReadCloser** (1 connections)
- **time.Timer** (1 connections)

## Relationships

- [Claude Test Fakes](Claude_Test_Fakes.md) (8 shared connections)
- [Claude CLI Client](Claude_CLI_Client.md) (5 shared connections)
- [Claude CLI Flags](Claude_CLI_Flags.md) (5 shared connections)
- [Stale Lock Client](Stale_Lock_Client.md) (2 shared connections)
- [Claude Event Types](Claude_Event_Types.md) (1 shared connections)
- [Process Group Kill](Process_Group_Kill.md) (1 shared connections)
- [CLI Banner](CLI_Banner.md) (1 shared connections)

## Source Files

- `internal/claude/client.go`
- `internal/claude/persistent.go`

## Audit Trail

- EXTRACTED: 61 (95%)
- INFERRED: 3 (5%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*