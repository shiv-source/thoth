# Stale Lock Client

> 5 nodes · cohesion 0.50

## Key Concepts

- **sync.Mutex** (6 connections)
- **FakeClient** (6 connections) — `internal/claude/fake.go`
- **staleLockClient** (3 connections) — `internal/api/chat_test.go`
- **Call** (2 connections) — `internal/claude/fake.go`
- **fake.go** (2 connections) — `internal/claude/fake.go`

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (2 shared connections)
- [Claude Test Fakes](Claude_Test_Fakes.md) (2 shared connections)
- [Claude Process Pool](Claude_Process_Pool.md) (2 shared connections)
- [Chat Hub](Chat_Hub.md) (1 shared connections)
- [Claude CLI Flags](Claude_CLI_Flags.md) (1 shared connections)
- [Claude Event Types](Claude_Event_Types.md) (1 shared connections)

## Source Files

- `internal/api/chat_test.go`
- `internal/claude/fake.go`

## Audit Trail

- EXTRACTED: 14 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*