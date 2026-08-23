-- sync_providers — the catalog of sync destination types. Built-ins (github,
-- gitlab, aws_s3, local) are seeded here on a fresh database and again at
-- serve time from assets/sync-providers.json when the table is empty (the
-- ensureModels self-heal pattern). Users add their own rows (custom GitHub/
-- GitLab base URLs, S3-compatible endpoints) and edit name/base_url.
--
-- slug       stable machine id ("github", "local"); UNIQUE
-- name       display label ("GitHub", "Local backup")
-- driver     the sync implementation: github | gitlab | s3 | local
-- base_url   API endpoint override for git/s3 drivers; '' = provider default
--            (meaningless for local)
-- protected  first-class providers the user can neither delete nor edit —
--            currently only local, the always-available backup

CREATE TABLE IF NOT EXISTS sync_providers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    slug       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    driver     TEXT NOT NULL,
    base_url   TEXT NOT NULL DEFAULT '',
    protected  INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

-- sync_connections — one row per configured sync destination (credentials +
-- target + sync state). Supersedes the single-row github_auth table: a user
-- may connect any number of destinations across providers.
--
-- provider_id  the sync_providers row this connection belongs to (FK off
--              today, so this documents intent; the app also blocks deleting
--              a provider with connections)
-- name         user label ("home wiki", "work", …)
-- config       JSON of credentials + target ("token", "repo_url",
--              "access_key_id"/"secret_access_key"/"region"/"bucket",
--              "path", …). Local plaintext — the documented localhost trust
--              model; write-only over the wire.
-- identity     token-free identity JSON for display (git: username/email/…;
--              s3: account from sts; local: none)
-- enabled      per-connection sync switch (replaces github_sync_enabled)
-- protected    first-class connection (the auto-created local backup) that
--              cannot be deleted — only its config edited
-- last_synced_at / last_error   replaces github_last_synced_at /
--              github_sync_error; on the connection row now

CREATE TABLE IF NOT EXISTS sync_connections (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id   INTEGER NOT NULL REFERENCES sync_providers(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    config        TEXT NOT NULL DEFAULT '{}',
    identity      TEXT,
    enabled       INTEGER NOT NULL DEFAULT 1,
    protected     INTEGER NOT NULL DEFAULT 0,
    last_synced_at TEXT,
    last_error    TEXT,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

-- Seed the built-in catalog so the cutover below has a github row to attach
-- to and fresh databases boot with every provider present. INSERT OR IGNORE
-- keeps the seed inert on a table that already has rows. The JSON seed
-- (assets/sync-providers.json) mirrors this; a test pins the two in sync.
INSERT OR IGNORE INTO sync_providers(slug, name, driver, base_url, protected, created_at) VALUES
    ('github', 'GitHub',     'github', '', 0, datetime('now')),
    ('gitlab', 'GitLab',     'gitlab', '', 0, datetime('now')),
    ('aws_s3', 'AWS S3',     's3',     '', 0, datetime('now')),
    ('local',  'Local backup', 'local', '', 1, datetime('now'));

-- Carry the connected account forward: the single github_auth row becomes a
-- connection under the github provider, and the four github_sync_* settings
-- keys become its target/enabled/state. The token and identity live in the
-- config/identity JSON; the wire never echoes them (the API returns
-- token-free views). COALESCE keeps the identity free of JSON nulls.
INSERT INTO sync_connections
    (provider_id, name, config, identity, enabled, protected, last_synced_at, last_error, created_at, updated_at)
SELECT
    (SELECT id FROM sync_providers WHERE slug = 'github'),
    'GitHub',
    json_object(
        'token', token,
        'repo_url', COALESCE((SELECT value FROM settings WHERE key = 'github_sync_repo'), '')),
    json_object(
        'username', username,
        'display_name', COALESCE(display_name, ''),
        'email', COALESCE(email, ''),
        'avatar_url', COALESCE(avatar_url, ''),
        'profile_url', COALESCE(profile_url, ''),
        'scopes', COALESCE(scopes, ''),
        'account_created_at', COALESCE(account_created_at, ''),
        'account_updated_at', COALESCE(account_updated_at, '')),
    CASE WHEN (SELECT value FROM settings WHERE key = 'github_sync_enabled') = 'true' THEN 1 ELSE 0 END,
    0,
    (SELECT value FROM settings WHERE key = 'github_last_synced_at'),
    (SELECT value FROM settings WHERE key = 'github_sync_error'),
    created_at, updated_at
FROM github_auth WHERE id = 1;

-- The sync state moved onto the connection row; the legacy keys are dead.
DELETE FROM settings WHERE key IN
    ('github_sync_repo', 'github_sync_enabled', 'github_last_synced_at', 'github_sync_error');

-- The single-row github_auth table is fully replaced by sync_connections.
DROP TABLE github_auth;
