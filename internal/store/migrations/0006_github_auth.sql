-- github_auth — the connected GitHub account, single row like app_metadata.
-- Created on first connect (POST /api/github/auth) and removed on
-- disconnect; the row also gates the connected state of the settings Git
-- tab.
--
-- token               the personal access token, stored plaintext — the same
--                     trust model as the gh CLI's own credentials file
--                     (localhost-only app). Never serialized: the API
--                     returns identity only (Auth.Token is json:"-"), and
--                     errors never echo it.
-- username/display_name/email/avatar_url/profile_url
--                     identity from GET /user (+ GET /user/emails for the
--                     primary verified email) — displayed view-only
-- scopes              X-OAuth-Scopes header, kept to warn about a missing
--                     user:email scope
-- expires_at          reserved: GET /user does not return an expiry
-- account_created_at / account_updated_at
--                     the GitHub account's own timestamps from /user
-- created_at/updated_at
--                     when the connection was first made / last refreshed
--                     (a reconnect preserves created_at)
--
-- The sync repo URL lives in the settings table (0007_settings.sql), not
-- here — identity and settings stay separate.

CREATE TABLE IF NOT EXISTS github_auth (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    token TEXT NOT NULL,
    username TEXT NOT NULL,
    display_name TEXT,
    email TEXT,
    avatar_url TEXT,
    scopes TEXT,
    expires_at TEXT,
    profile_url TEXT,
    account_created_at TEXT,
    account_updated_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
