# Process Group Kill

> 12 nodes · cohesion 0.18

## Key Concepts

- **os/exec.Cmd** (3 connections)
- **os.Process** (3 connections)
- **killProcess()** (3 connections) — `internal/claude/proc_unix.go`
- **killProcess()** (3 connections) — `internal/claude/proc_windows.go`
- **proc_unix.go** (2 connections) — `internal/claude/proc_unix.go`
- **.killProcessGroup()** (2 connections) — `internal/claude/proc_unix.go`
- **setProcessGroup()** (2 connections) — `internal/claude/proc_unix.go`
- **proc_windows.go** (2 connections) — `internal/claude/proc_windows.go`
- **.killProcessGroup()** (2 connections) — `internal/claude/proc_windows.go`
- **setProcessGroup()** (2 connections) — `internal/claude/proc_windows.go`
- **CLIClient** (1 connections) — `internal/claude/proc_unix.go`
- **CLIClient** (1 connections) — `internal/claude/proc_windows.go`

## Relationships

- [Claude Process Pool](Claude_Process_Pool.md) (1 shared connections)
- [Claude CLI Flags](Claude_CLI_Flags.md) (1 shared connections)

## Source Files

- `internal/claude/proc_unix.go`
- `internal/claude/proc_windows.go`

## Audit Trail

- EXTRACTED: 14 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*