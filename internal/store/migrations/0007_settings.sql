-- settings — the app's user-facing settings, key/value (config.toml is
-- deprecated; this table is the single source). New keys need no schema
-- change — that is the point of the shape.
--
-- wiki_path             where the wiki lives; the seed mirrors
--                       settings.DefaultWikiPath (internal/settings) so the
--                       row always exists and readers never need a fallback
-- github_sync_repo      the wiki's sync repo URL (the git remote); written
--                       by the settings Git tab, read by git setup/sync
-- github_sync_enabled   the auto-sync switch for the future background
--                       sync; stored as 'true'/'false'
-- github_last_synced_at UTC RFC3339 of the last successful git sync, or ''
--                       when never
-- github_sync_error     the sanitized error of the last failed sync, or ''

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT OR IGNORE INTO settings(key, value) VALUES
    ('wiki_path', '~/.thoth/wiki'),
    ('github_sync_repo', ''),
    ('github_sync_enabled', 'false'),
    ('github_last_synced_at', ''),
    ('github_sync_error', '');
