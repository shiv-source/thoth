# Claude Event Types

> 20 nodes · cohesion 0.16

## Key Concepts

- **ParseLine()** (13 connections) — `internal/claude/events.go`
- **events_test.go** (10 connections) — `internal/claude/events_test.go`
- **events.go** (8 connections) — `internal/claude/events.go`
- **Event** (6 connections) — `internal/claude/events.go`
- **rawBlock** (3 connections) — `internal/claude/events.go`
- **TestParseLineAssistantText()** (3 connections) — `internal/claude/events_test.go`
- **TestParseLineAssistantWithEmptyText()** (3 connections) — `internal/claude/events_test.go`
- **TestParseLineAssistantWithoutMessage()** (3 connections) — `internal/claude/events_test.go`
- **TestParseLineIgnoresStringShapedMessage()** (3 connections) — `internal/claude/events_test.go`
- **TestParseLineIgnoresUnknown()** (3 connections) — `internal/claude/events_test.go`
- **TestParseLineRejectsGarbage()** (3 connections) — `internal/claude/events_test.go`
- **TestParseLineResult()** (3 connections) — `internal/claude/events_test.go`
- **TestParseLineThinking()** (3 connections) — `internal/claude/events_test.go`
- **TestParseLineToolUse()** (3 connections) — `internal/claude/events_test.go`
- **TestWriterFuncAdapter()** (3 connections) — `internal/claude/events_test.go`
- **EventType** (2 connections) — `internal/claude/events.go`
- **rawLine** (2 connections) — `internal/claude/events.go`
- **rawMsg** (2 connections) — `internal/claude/events.go`
- **.Write()** (2 connections) — `internal/claude/events.go`
- **encoding/json.RawMessage** (2 connections)

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (10 shared connections)
- [Claude CLI Client](Claude_CLI_Client.md) (4 shared connections)
- [Stale Lock Client](Stale_Lock_Client.md) (1 shared connections)
- [Claude Test Fakes](Claude_Test_Fakes.md) (1 shared connections)
- [Claude CLI Flags](Claude_CLI_Flags.md) (1 shared connections)
- [Claude Process Pool](Claude_Process_Pool.md) (1 shared connections)

## Source Files

- `internal/claude/events.go`
- `internal/claude/events_test.go`

## Audit Trail

- EXTRACTED: 37 (76%)
- INFERRED: 12 (24%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*