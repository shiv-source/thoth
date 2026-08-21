-- conversations — one row per chat shown in the UI history.
--
-- id                the conversation id (v4 UUID, created by the store); it
--                   is also the wire id the frontend uses in /chat/<id> URLs
-- title             the first user message, truncated (display only)
-- created_at        UTC RFC3339; the list orders by it lexically DESC, so a
--                   fixed offset is required (local offsets would misorder)
-- claude_session_id the Claude CLI session that used to back the chat. The
--                   native agent dropped the CLI (and the column's writes);
--                   the column is retained for schema stability — nothing
--                   reads or writes it, and no migration drops it (T12).

CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    created_at TEXT NOT NULL,
    claude_session_id TEXT NOT NULL DEFAULT ''
);
