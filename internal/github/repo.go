package github

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shiv-source/thoth/internal/store"
)

// Auth is the stored GitHub connection. Token is never serialized: the API
// returns identity only. The sync repo URL lives in the settings table.
type Auth struct {
	Token            string `json:"-"`
	Username         string `json:"username"`
	DisplayName      string `json:"display_name"`
	Email            string `json:"email"`
	AvatarURL        string `json:"avatar_url"`
	ProfileURL       string `json:"profile_url"`
	Scopes           string `json:"scopes"`
	AccountCreatedAt string `json:"account_created_at"`
	AccountUpdatedAt string `json:"account_updated_at"`
}

// Repo owns the github_auth table on its own connection to thoth.db (the
// schema lives in the store's migrations).
type Repo struct {
	db *sql.DB
}

// OpenRepo opens a repository handle. The schema must exist — the store's
// migrations create it.
func OpenRepo(path string) (*Repo, error) {
	db, err := store.OpenDB(path)
	if err != nil {
		return nil, fmt.Errorf("open github repo: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	return &Repo{db: db}, nil
}

func (r *Repo) Close() error { return r.db.Close() }

// Save upserts the single row (id = 1). created_at is deliberately absent
// from the UPDATE set: a reconnect preserves when the connection was first
// made. expires_at ships reserved (no data source from /user).
func (r *Repo) Save(a Auth) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`
		INSERT INTO github_auth (id, token, username, display_name, email, avatar_url, scopes, expires_at, profile_url, account_created_at, account_updated_at, created_at, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			token = excluded.token,
			username = excluded.username,
			display_name = excluded.display_name,
			email = excluded.email,
			avatar_url = excluded.avatar_url,
			scopes = excluded.scopes,
			expires_at = excluded.expires_at,
			profile_url = excluded.profile_url,
			account_created_at = excluded.account_created_at,
			account_updated_at = excluded.account_updated_at,
			updated_at = excluded.updated_at`,
		a.Token, a.Username, a.DisplayName, a.Email, a.AvatarURL, a.Scopes, a.ProfileURL, a.AccountCreatedAt, a.AccountUpdatedAt, now, now)
	if err != nil {
		return fmt.Errorf("save github auth: %w", err)
	}
	return nil
}

// Get returns the stored connection; ok is false when none exists.
func (r *Repo) Get() (Auth, bool, error) {
	var a Auth
	var displayName, email, avatarURL, profileURL, scopes, accountCreated, accountUpdated sql.NullString
	err := r.db.QueryRow(`
		SELECT token, username, display_name, email, avatar_url, profile_url, scopes, account_created_at, account_updated_at
		FROM github_auth WHERE id = 1`).
		Scan(&a.Token, &a.Username, &displayName, &email, &avatarURL, &profileURL, &scopes, &accountCreated, &accountUpdated)
	if errors.Is(err, sql.ErrNoRows) {
		return Auth{}, false, nil
	}
	if err != nil {
		return Auth{}, false, fmt.Errorf("read github auth: %w", err)
	}
	a.DisplayName = displayName.String
	a.Email = email.String
	a.AvatarURL = avatarURL.String
	a.ProfileURL = profileURL.String
	a.Scopes = scopes.String
	a.AccountCreatedAt = accountCreated.String
	a.AccountUpdatedAt = accountUpdated.String
	return a, true, nil
}

// Clear removes the connection; clearing nothing is not an error.
func (r *Repo) Clear() error {
	if _, err := r.db.Exec(`DELETE FROM github_auth WHERE id = 1`); err != nil {
		return fmt.Errorf("clear github auth: %w", err)
	}
	return nil
}
