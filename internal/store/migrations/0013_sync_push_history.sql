-- sync_push_history — a bounded per-connection record of every completed
-- push attempt (success or failure). The single last_synced_at / last_error
-- columns on sync_connections carry only the latest outcome; this table keeps
-- the recent history the Settings page can render, capped by the store
-- (AppendPushHistory prunes to a fixed window).
--
-- connection_id  the sync_connections row the attempt belongs to
-- at             UTC RFC3339 of the attempt
-- ok             1 = pushed, 0 = failed
-- error          the sanitized error for a failure; '' otherwise

CREATE TABLE IF NOT EXISTS sync_push_history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    connection_id INTEGER NOT NULL,
    at            TEXT NOT NULL,
    ok            INTEGER NOT NULL,
    error         TEXT,
    created_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sync_push_history_connection
    ON sync_push_history(connection_id, id DESC);
