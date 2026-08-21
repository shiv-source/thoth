# What's new

Newest first. This page highlights user-facing and architectural changes since the last release. Current release milestone: **Thoth Agent** — the native Go assistant.

## August 2026 — Thoth Agent replaces the Claude Code CLI (epic #121)

The big one: the assistant stopped being a spawned `claude` CLI process and became **Thoth Agent**, a native, in-process assistant written in Go.

**What you get:**

- **No Claude Code install or login** — Thoth Agent calls model providers directly; the `claude` binary, its login, and `PATH` setup are gone. Just add an API key in Settings.
- **Any provider** — a `Provider` seam means the model picker isn't Anthropic-only: Anthropic *and* OpenAI-compatible endpoints (DeepSeek, Qwen, GLM, Grok, …) work through Settings → LLM Models.
- **Per-provider credentials** — each provider can have its own API key and base URL (`provider_<slug>_api_key` / `provider_<slug>_base_url`), falling back to the shared key and the provider's default endpoint. Settings → General exposes them.
- **Instant rulebook changes** — the system prompt is re-read from `wiki/CLAUDE.md` on every turn; edits apply without restarting anything.
- **A bounded write path** — the assistant reads/writes the wiki only through `SafePath`-bounded tools (`read_file`, `write_file`, `list`, `search`). No `--dangerously-skip-permissions`, no subprocess at all.

**Under the hood:**

- The reusable `agent/` module: the tool-use loop, the `Provider` wire seam, the tool registry, and normalized events/model — with Anthropic and OpenAI-compatible providers and a shared SSE transport.
- The `internal/agent` host glue wires the library to the wiki, store, and index; the chat server and `/api/health` now report the native backend (`thoth-agent`).
- `internal/claude` — flags, stream parsing, the process pool, and the process-kill files — was deleted.
- The UI gates setup on the native backend status instead of the CLI: the setup screen tells you when a provider API key or wiki is missing.
- `thoth doctor`'s old `claude` check became **provider** + **api key** checks: the provider check probes the selected model's provider endpoint with the resolved credential (401/429/timeout/5xx all named).

**Notes**

- The WebSocket chat protocol did not change — same frames, same cancel/supersede/resume semantics.
- Your wiki, conversations, and settings carry over; the unused `claude_session_id` column is retained for schema stability.

Full rationale and before/after comparison: [Migrating to Thoth Agent](thoth-agent.md).
