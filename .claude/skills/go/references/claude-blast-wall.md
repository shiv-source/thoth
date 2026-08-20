# The claude blast wall (internal/claude)

Every version-sensitive fact about the Claude Code CLI lives in exactly two
files: client.go (flags) and persistent.go (the process pool). A CLI upgrade
can only ever break this package — everything else is stable.

## client.go — the flags
- Per-turn: -p --output-format stream-json --verbose --session-id <id>
- Persistent mode: -p --input-format stream-json … --autocompact auto
- Permissions: --dangerously-skip-permissions by default, or --permission-mode <mode> when configured
- Optional --model from the settings table
- Spawn, stream scanning, cancel all live here; stderr is captured and appended to exit errors
- canonical: internal/claude/client.go · docs/components.md §internal/claude
- VERIFY AGAINST `claude --help` WHENEVER THE CLI UPGRADES — no other file may hold a flag

## persistent.go — the process pool
- PersistentClient: lazy-spawned CLI processes keyed by session id
- One dispatcher goroutine per process turns stdout lines into events for the in-flight turn; the CLI's result line ends it
- Cancel kills the process; the next turn respawns (no per-turn interrupt in the plain CLI)
- Idle eviction after 10 min; Flush on wiki-path change or when the user leaves the chat page; Close on shutdown
- Cap: MaxProcs (default 4) evicts the least-recently-used idle process on overflow; busy processes are never killed to make room (the cap can be exceeded briefly)
- Warm(sessionID): eager spawn for one session — the exact getOrSpawn/spawnLocked path a turn uses, no prompt; serve pre-warms the most recently active conversation at boot (best-effort; idle eviction reaps it if unused)
- canonical: internal/claude/persistent.go

## events.go — stream parsing
- Tolerant parsing of stream-json lines into typed events
- Types: assistant_delta, thinking (thinking-only blocks → UI "Thinking…"), tool_activity, turn_done, error
- The raw stream is appended to ~/.thoth/stream-dump.json (rotated past 10 MB) for debugging
- canonical: internal/claude/events.go

## Process kill
- ctx cancel kills the process group (unix, proc_unix.go) or direct child (windows, proc_windows.go)
- Build-tagged — all five cross-compile targets must build
- canonical: internal/claude/proc_unix.go · internal/claude/proc_windows.go

## Client interface & FakeClient
- Client: Start(ctx, sessionID, prompt, w EventWriter) error — the only seam consumers see
- FakeClient replays scripted events and records calls — every consumer's tests use it, so no test ever touches the real CLI
- canonical: internal/claude/client.go · internal/claude/fake.go

Stale if: `claude --help` output changed and client.go wasn't updated, a
CLI flag appears outside this package, or events.go stops parsing a
stream-json line type.
