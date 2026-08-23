-- provider_headers — the per-provider custom request headers (e.g. for
-- gateways like Portkey that identify a route via headers alongside the API
-- key). One row per header, so a provider can carry many and the UI edits
-- them individually. A provider with no rows sends no extra headers.
--
-- provider_id  owning providers row; deleted with the provider (the app also
--              deletes explicitly — the FK pragma is off today)
-- name         header name, e.g. x-portkey-provider
-- value        header value, e.g. anthropic
--              UNIQUE(provider_id, name) — a duplicate name replaces

CREATE TABLE IF NOT EXISTS provider_headers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id INTEGER NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    value       TEXT NOT NULL,
    UNIQUE(provider_id, name)
);

CREATE INDEX IF NOT EXISTS idx_provider_headers_provider ON provider_headers(provider_id);
