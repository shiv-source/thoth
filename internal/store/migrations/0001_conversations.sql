-- conversations — one row per chat shown in the UI history.
--
-- id                the conversation id (v4 UUID, created by the store); it
--                   is also the wire id the frontend uses in /chat/<id> URLs
-- title             the first user message, truncated (display only)
-- created_at        UTC RFC3339; the list orders by it lexically DESC, so a
--                   fixed offset is required (local offsets would misorder)
-- claude_session_id the Claude CLI session backing the chat. Seeded with the
--                   conversation id by CreateConversation and rotated to a
--                   fresh id (via --resume/--fork-session) when the CLI
--                   reports the stored session as locked ("already in use").
--                   The CLI keeps that session's history in
--                   ~/.claude/projects/, so revisiting a chat resumes it.

CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    created_at TEXT NOT NULL,
    claude_session_id TEXT NOT NULL DEFAULT ''
);
