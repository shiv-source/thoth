# Claude CLI Client

> 53 nodes · cohesion 0.12

## Key Concepts

- **WriterFunc** (30 connections) — `internal/claude/events.go`
- **persistent_test.go** (21 connections) — `internal/claude/persistent_test.go`
- **NewPersistent()** (19 connections) — `internal/claude/persistent.go`
- **claude/client_test.go** (18 connections) — `internal/claude/client_test.go`
- **New()** (16 connections) — `internal/claude/client.go`
- **startTurn()** (15 connections) — `internal/claude/persistent_test.go`
- **claude/client.go** (13 connections) — `internal/claude/client.go`
- **writePersistentFakeCLI()** (11 connections) — `internal/claude/persistent_test.go`
- **writeFakeCLIVariant()** (10 connections) — `internal/claude/persistent_test.go`
- **writeFakeCLI()** (8 connections) — `internal/claude/client_test.go`
- **TestPersistentCrashMidTurn()** (8 connections) — `internal/claude/persistent_test.go`
- **TestStartStreamsEventsAndPassesFlags()** (7 connections) — `internal/claude/client_test.go`
- **spawnCount()** (7 connections) — `internal/claude/persistent_test.go`
- **TestPersistentCancelKillsAndRespawns()** (7 connections) — `internal/claude/persistent_test.go`
- **TestPersistentSpawnsOneProcessPerSession()** (7 connections) — `internal/claude/persistent_test.go`
- **TestPersistentWithResume()** (7 connections) — `internal/claude/persistent_test.go`
- **TestStartWithResumeForksSession()** (6 connections) — `internal/claude/client_test.go`
- **TestStartWritesDebugStreamDump()** (6 connections) — `internal/claude/client_test.go`
- **WithResume()** (6 connections) — `internal/claude/client.go`
- **argvOf()** (6 connections) — `internal/claude/persistent_test.go`
- **TestPersistentArgs()** (6 connections) — `internal/claude/persistent_test.go`
- **TestPersistentCloseAndEmptySessionID()** (6 connections) — `internal/claude/persistent_test.go`
- **TestPersistentDebugDump()** (6 connections) — `internal/claude/persistent_test.go`
- **TestPersistentFlushOnDirChange()** (6 connections) — `internal/claude/persistent_test.go`
- **TestPersistentIdleEviction()** (6 connections) — `internal/claude/persistent_test.go`
- *... and 28 more nodes in this community*

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (39 shared connections)
- [Claude Process Pool](Claude_Process_Pool.md) (5 shared connections)
- [Serve Command](Serve_Command.md) (4 shared connections)
- [Claude Event Types](Claude_Event_Types.md) (4 shared connections)
- [Chat Hub](Chat_Hub.md) (3 shared connections)
- [Claude CLI Flags](Claude_CLI_Flags.md) (3 shared connections)

## Source Files

- `internal/claude/client.go`
- `internal/claude/client_test.go`
- `internal/claude/events.go`
- `internal/claude/persistent.go`
- `internal/claude/persistent_test.go`

## Audit Trail

- EXTRACTED: 153 (71%)
- INFERRED: 64 (29%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*