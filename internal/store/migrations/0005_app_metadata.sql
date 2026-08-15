-- app_metadata — one-time install facts, single row.
--
-- The CHECK (id = 1) constraint enforces exactly one row; inserts omit id
-- (DEFAULT 1) and use INSERT OR IGNORE for idempotent seeding on every boot
-- (Store.EnsureMetadata). installation_id identifies this machine's
-- installation (v4 UUID) and never leaves the DB; created_at is the first
-- boot. Git sync state lives in the settings table (0007_settings.sql),
-- not here.

CREATE TABLE IF NOT EXISTS app_metadata (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    installation_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);
