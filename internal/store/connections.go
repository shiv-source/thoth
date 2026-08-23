package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Connection is one configured sync destination: the credentials + target +
// sync state for a sync_providers row. The single-row github_auth table is
// superseded by this — a user may connect any number of destinations across
// providers.
type Connection struct {
	ID         int64
	ProviderID int64
	Name       string
	// Config is the raw JSON credential + target blob ("token", "repo_url",
	// "access_key_id", "path", …), stored local-plaintext under the
	// documented trust model. Write-only over the wire: the API returns
	// token-free views and treats secret fields as leave-unchanged on PUT.
	Config   string
	Identity string // token-free identity JSON for display; "" when none
	Enabled  bool
	// Protected marks first-class connections (the auto-created local
	// backup) that cannot be deleted — only their config edited.
	Protected    bool
	LastSyncedAt string // UTC RFC3339 of the last successful sync; "" when never
	LastError    string // sanitized error of the last failed sync; "" when none
	CreatedAt    string
	UpdatedAt    string
	// ProviderSlug / ProviderName / ProviderDriver are the joined provider
	// fields, populated by reads; zero elsewhere.
	ProviderSlug   string
	ProviderName   string
	ProviderDriver string
}

// ErrConnectionNotFound is returned when a read/update/delete targets a
// missing row; the API layer maps it to 404.
var ErrConnectionNotFound = errors.New("sync connection not found")

// ErrConnectionProtected is returned when deleting a protected connection;
// the API layer maps it to 403.
var ErrConnectionProtected = errors.New("sync connection is protected and cannot be deleted")

// connectionJoinColumns are the wire-shape columns with the provider join —
// the single read shape for every Connection scan.
const connectionJoinColumns = `
	c.id, c.provider_id, c.name, c.config, c.identity, c.enabled, c.protected,
	c.last_synced_at, c.last_error, c.created_at, c.updated_at,
	p.slug, p.name, p.driver`

// scanConnection reads the connectionJoinColumns order into c. Nullable
// columns and integer flags are normalized for the wire shape.
func scanConnection(s scanner, c *Connection) error {
	var identity, lastSyncedAt, lastError sql.NullString
	var enabled, protected int
	if err := s.Scan(&c.ID, &c.ProviderID, &c.Name, &c.Config, &identity, &enabled, &protected,
		&lastSyncedAt, &lastError, &c.CreatedAt, &c.UpdatedAt,
		&c.ProviderSlug, &c.ProviderName, &c.ProviderDriver); err != nil {
		return err
	}
	c.Identity = identity.String
	c.LastSyncedAt = lastSyncedAt.String
	c.LastError = lastError.String
	c.Enabled = enabled != 0
	c.Protected = protected != 0
	return nil
}

// scanner is the row-scan surface shared by QueryRow and Rows.
type scanner interface {
	Scan(dest ...any) error
}

// ListConnections returns every connection in insertion (id) order with its
// provider joined — the Settings sync page list.
func (s *Store) ListConnections() ([]Connection, error) {
	rows, err := s.db.Query(`
		SELECT ` + connectionJoinColumns + `
		FROM sync_connections c
		JOIN sync_providers p ON p.id = c.provider_id
		ORDER BY c.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Connection
	for rows.Next() {
		var c Connection
		if err := scanConnection(rows, &c); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Connection returns one connection by id with its provider joined.
func (s *Store) Connection(id int64) (Connection, error) {
	var c Connection
	err := scanConnection(s.db.QueryRow(`
		SELECT `+connectionJoinColumns+`
		FROM sync_connections c
		JOIN sync_providers p ON p.id = c.provider_id
		WHERE c.id = ?`, id), &c)
	if errors.Is(err, sql.ErrNoRows) {
		return Connection{}, ErrConnectionNotFound
	}
	if err != nil {
		return Connection{}, fmt.Errorf("read connection: %w", err)
	}
	return c, nil
}

// CreateConnection inserts a connection and returns it with its provider
// joined. The provider must exist (ErrSyncProviderNotFound otherwise).
func (s *Store) CreateConnection(providerID int64, name, config, identity string, enabled bool) (Connection, error) {
	if _, err := s.SyncProvider(providerID); err != nil {
		return Connection{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO sync_connections(provider_id, name, config, identity, enabled, protected, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 0, ?, ?)`,
		providerID, name, config, nullString(identity), enabled, now, now)
	if err != nil {
		return Connection{}, fmt.Errorf("create connection: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Connection{}, fmt.Errorf("create connection id: %w", err)
	}
	return s.Connection(id)
}

// UpdateConnection replaces the editable fields of the connection (name,
// config, enabled). Protected connections are editable (the local backup's
// folder changes) — only deletion is blocked.
func (s *Store) UpdateConnection(id int64, name, config string, enabled bool) error {
	res, err := s.db.Exec(
		`UPDATE sync_connections SET name = ?, config = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		name, config, enabled, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("update connection: %w", err)
	}
	return providerRowsAffectedOrNotFound(res, "update connection")
}

// SetConnectionSyncResult records the outcome of a sync: success stamps
// last_synced_at and clears last_error; failure records the error and keeps
// last_synced_at at the last successful sync.
func (s *Store) SetConnectionSyncResult(id int64, ok bool, detail string) error {
	lastSyncedAt := ""
	if ok {
		lastSyncedAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		// Preserve the last successful sync: a read error degrades to "" (no
		// previous success) rather than blocking a sync result on state.
		var v sql.NullString
		if err := s.db.QueryRow(`SELECT last_synced_at FROM sync_connections WHERE id = ?`, id).Scan(&v); err == nil {
			lastSyncedAt = v.String
		}
	}
	lastError := ""
	if !ok {
		lastError = detail
	}
	_, err := s.db.Exec(
		`UPDATE sync_connections SET last_synced_at = ?, last_error = ?, updated_at = ? WHERE id = ?`,
		lastSyncedAt, nullString(lastError), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("set connection sync result: %w", err)
	}
	return nil
}

// DeleteConnection removes a connection; a protected row is
// ErrConnectionProtected.
func (s *Store) DeleteConnection(id int64) error {
	var protected int
	if err := s.db.QueryRow(`SELECT protected FROM sync_connections WHERE id = ?`, id).Scan(&protected); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConnectionNotFound
		}
		return fmt.Errorf("read connection protection: %w", err)
	}
	if protected != 0 {
		return ErrConnectionProtected
	}
	res, err := s.db.Exec(`DELETE FROM sync_connections WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	return providerRowsAffectedOrNotFound(res, "delete connection")
}

// EnsureLocalBackup guarantees the first-class local backup connection
// exists: one protected connection under the local provider, created with no
// folder configured (the user picks it in Settings). Idempotent — every boot
// may call it, and the connection can never be deleted by the API.
func (s *Store) EnsureLocalBackup() (Connection, error) {
	local, err := s.SyncProviderBySlug("local")
	if errors.Is(err, ErrSyncProviderNotFound) {
		return Connection{}, fmt.Errorf("local backup provider missing — seed sync providers first: %w", err)
	}
	if err != nil {
		return Connection{}, err
	}
	var id int64
	err = s.db.QueryRow(
		`SELECT id FROM sync_connections WHERE provider_id = ? AND protected = 1 LIMIT 1`, local.ID).Scan(&id)
	if err == nil {
		return s.Connection(id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Connection{}, fmt.Errorf("read local backup: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO sync_connections(provider_id, name, config, identity, enabled, protected, created_at, updated_at)
		 VALUES (?, 'Local backup', '{}', NULL, 1, 1, ?, ?)`,
		local.ID, now, now)
	if err != nil {
		return Connection{}, fmt.Errorf("create local backup: %w", err)
	}
	id, err = res.LastInsertId()
	if err != nil {
		return Connection{}, fmt.Errorf("create local backup id: %w", err)
	}
	return s.Connection(id)
}

// nullString returns a driver-usable NULL for an empty string, so absent
// identity/error values stay SQL NULL (the wire treats them as "").
func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
