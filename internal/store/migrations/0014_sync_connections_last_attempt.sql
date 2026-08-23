-- sync_connections.last_attempt_at — UTC RFC3339 of the last sync attempt,
-- success or failure. last_synced_at still records the last success (for
-- display); the auto-sync scheduler schedules off last_attempt_at so a failing
-- connection cools down between retries instead of re-firing every tick while
-- last_synced_at stays stale.
--
-- Existing rows backfill last_attempt_at from last_synced_at so a connection
-- that synced recently is not re-fired immediately after the migration.

ALTER TABLE sync_connections ADD COLUMN last_attempt_at TEXT;

UPDATE sync_connections SET last_attempt_at = last_synced_at
WHERE last_attempt_at IS NULL AND last_synced_at IS NOT NULL;
