-- messages.usage — the turn's token breakdown, JSON text, on the assistant
-- message that ended the turn (per-turn token-usage telemetry).
--
-- id        primary key
-- usage     NULL or JSON: {"input_tokens":N,"output_tokens":N,
--            "cache_read_tokens":N,"cache_write_tokens":N}; NULL on user
--            messages and on rows written before usage was tracked

ALTER TABLE messages ADD COLUMN usage TEXT;
