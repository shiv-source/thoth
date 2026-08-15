-- GitHub connection, single-row like app_metadata. Created on first connect;
-- the token is stored plaintext (same trust model as the gh CLI). expires_at
-- is reserved: GET /user does not return an expiry.

CREATE TABLE IF NOT EXISTS github_auth (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    token TEXT NOT NULL,
    username TEXT NOT NULL,
    display_name TEXT,
    email TEXT,
    avatar_url TEXT,
    scopes TEXT,
    expires_at TEXT,
    profile_url TEXT,         -- the GitHub account's html_url (from /user)
    account_created_at TEXT,  -- the GitHub account's own created_at (from /user)
    account_updated_at TEXT,  -- the GitHub account's own updated_at (from /user)
    repo_url TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
