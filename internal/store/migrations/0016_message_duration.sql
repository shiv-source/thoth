-- messages.duration_secs — the wall-clock seconds the assistant turn took to
-- generate its reply, captured server-side and stored on the assistant message
-- that ended the turn (per-turn latency telemetry). Full precision is kept in
-- the database; the UI formats it to two decimal places.
--
-- id             primary key
-- duration_secs  NULL or REAL seconds; NULL on user messages and on rows
--                written before duration was tracked

ALTER TABLE messages ADD COLUMN duration_secs REAL;
