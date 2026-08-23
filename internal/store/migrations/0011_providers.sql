-- providers — one row per LLM provider the user configures. Providers are
-- first-class entities: they are created before any model (a provider owns
-- its models via llm_models.provider_id) and hold their own base URL override
-- and API key. The api_key is write-only in the UI (reads report has_api_key);
-- an empty base_url means the provider's default endpoint.
--
-- name       display label, e.g. "Anthropic"; UNIQUE — the provider is the
--            grouping key of the model registry and the settings resolver
--            matches on it
-- base_url   API base URL override; empty = the provider's default endpoint
-- api_key    the provider's own API key (local plaintext, write-only)
-- created_at UTC timestamp of creation

CREATE TABLE IF NOT EXISTS providers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    base_url   TEXT NOT NULL DEFAULT '',
    api_key    TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

-- llm_models: the free-text provider label becomes a real foreign key. The
-- legacy provider column is kept until its data is migrated below, then
-- dropped — applied migrations are immutable, so the whole move lives here.
-- ON DELETE CASCADE keeps the model registry consistent when a provider is
-- deleted (the app also cascades explicitly; the pragma is off today, so this
-- documents intent rather than enforcing it).

ALTER TABLE llm_models ADD COLUMN provider_id INTEGER REFERENCES providers(id) ON DELETE CASCADE;

-- Backfill: every distinct provider label already in the registry becomes a
-- providers row. The per-provider credentials still live in the legacy
-- provider_<slug>_* settings keys at this point; a Go data-migration in
-- store.Open copies them into these rows once, then deletes the keys.

INSERT INTO providers(name, base_url, api_key, created_at)
SELECT DISTINCT provider, '', '', datetime('now')
FROM llm_models
WHERE provider <> '';

-- Point each model at its provider row, then drop the label column. Models
-- without a provider label (the Unassigned catch-all) keep provider_id NULL.

UPDATE llm_models
SET provider_id = (SELECT id FROM providers WHERE providers.name = llm_models.provider)
WHERE provider <> '';

ALTER TABLE llm_models DROP COLUMN provider;
