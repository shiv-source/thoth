-- llm_models — the user-editable model registry. Every startup seeds it from
-- internal/assets/models.json when the table is empty (models.json stays the
-- single source for the built-in list); afterwards the table is authoritative
-- and users add/edit/delete rows from the Settings → LLM Models tab. The
-- settings model key stores the selected value; a deleted or renamed selected
-- model is kept consistent by the API layer.
--
-- value       the --model argument; UNIQUE because the settings key points
--             at it and two rows must never claim the same value
-- name        display name (e.g. "Claude Opus 4.8")
-- description preset-friendly label rendered as secondary text (renamed to
--             tag by 0009)
-- provider    grouping label for the picker (e.g. "Anthropic")

CREATE TABLE IF NOT EXISTS llm_models (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    value       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    provider    TEXT NOT NULL DEFAULT ''
);

-- One new settings key accompanies the table, following the 0007 pattern:
-- the row always exists, so readers never need an absent-key fallback.
--
-- api_key  set from the web Settings (General tab); passed to spawned claude
--          CLI processes as ANTHROPIC_API_KEY. Empty ('') = inherit the
--          server's environment.

INSERT OR IGNORE INTO settings(key, value) VALUES
    ('api_key', '');
