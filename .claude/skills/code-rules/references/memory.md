# Memory & Resources (no leaks)

Read this when writing concurrent or resource-managing code — goroutines, DB
handles, files, watchers, and frontend effects. The core rules live in the
code-rules skill; this is the on-demand detail.

- **Every resource is released on every path** — `defer` immediately after acquisition: `rows.Close()`, files, bodies, DB handles, watchers.
- **Goroutines must end** — every `go` statement has a `ctx`/done-channel that stops it; long-lived loops select on `ctx.Done()`; no goroutine outlives its owner.
- **No unbounded growth** — capped buffers (like the 500-message replay), bounded maps with eviction, no slices/maps that only grow.
- **Frontend** — every `useEffect` subscription/timer/socket has a cleanup that runs on unmount; no setInterval without clearInterval.
- **Concurrency is guarded** — shared state behind mutex/atomic, never a bare data race; CI runs `-race` and it must stay green.
- **Process hygiene** — spawned children die with their context (process-group kill on unix, direct kill on windows).
