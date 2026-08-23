package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SyncProvider is one row of the sync_providers table: a sync destination
// type. Built-ins (github, gitlab, aws_s3, local) are seeded from
// assets/sync-providers.json when the table is empty; users add their own
// rows (custom base URLs / endpoints) and edit the editable fields.
// Protected rows (the local backup) are first-class and can be neither
// edited nor deleted.
type SyncProvider struct {
	ID        int64
	Slug      string // stable machine id, e.g. "github"
	Name      string // display label
	Driver    string // sync implementation: github | gitlab | s3 | local
	BaseURL   string // API endpoint override; "" = provider default
	Protected bool
	CreatedAt string
	// ConnectionCount is the number of connections under this provider;
	// populated by ListSyncProviders, zero elsewhere.
	ConnectionCount int
}

// ErrSyncProviderExists is returned when a create collides with another
// row's UNIQUE slug; the API layer maps it to 409.
var ErrSyncProviderExists = errors.New("a sync provider with this slug already exists")

// ErrSyncProviderNotFound is returned when a read/update/delete targets a
// missing row; the API layer maps it to 404.
var ErrSyncProviderNotFound = errors.New("sync provider not found")

// ErrSyncProviderProtected is returned when a protected row is edited or
// deleted; the API layer maps it to 403.
var ErrSyncProviderProtected = errors.New("sync provider is protected and cannot be modified or deleted")

// ErrSyncProviderInUse is returned when deleting a provider that still has
// connections; the API layer maps it to 409.
var ErrSyncProviderInUse = errors.New("sync provider has connections and cannot be deleted")

const syncProviderColumns = "id, slug, name, driver, base_url, protected, created_at"

// ListSyncProviders returns every sync provider in insertion (id) order,
// which keeps the seeded order stable in the Settings sync page.
func (s *Store) ListSyncProviders() ([]SyncProvider, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.slug, p.name, p.driver, p.base_url, p.protected, p.created_at,
		       (SELECT COUNT(*) FROM sync_connections c WHERE c.provider_id = p.id)
		FROM sync_providers p
		ORDER BY p.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list sync providers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []SyncProvider
	for rows.Next() {
		var p SyncProvider
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Driver, &p.BaseURL, &p.Protected, &p.CreatedAt, &p.ConnectionCount); err != nil {
			return nil, fmt.Errorf("scan sync provider: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SyncProvider returns one sync provider by id.
func (s *Store) SyncProvider(id int64) (SyncProvider, error) {
	var p SyncProvider
	err := s.db.QueryRow(
		`SELECT `+syncProviderColumns+` FROM sync_providers WHERE id = ?`, id).
		Scan(&p.ID, &p.Slug, &p.Name, &p.Driver, &p.BaseURL, &p.Protected, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return SyncProvider{}, ErrSyncProviderNotFound
	}
	if err != nil {
		return SyncProvider{}, fmt.Errorf("read sync provider: %w", err)
	}
	return p, nil
}

// SyncProviderBySlug returns one sync provider by its slug.
func (s *Store) SyncProviderBySlug(slug string) (SyncProvider, error) {
	var p SyncProvider
	err := s.db.QueryRow(
		`SELECT `+syncProviderColumns+` FROM sync_providers WHERE slug = ?`, slug).
		Scan(&p.ID, &p.Slug, &p.Name, &p.Driver, &p.BaseURL, &p.Protected, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return SyncProvider{}, ErrSyncProviderNotFound
	}
	if err != nil {
		return SyncProvider{}, fmt.Errorf("read sync provider %q: %w", slug, err)
	}
	return p, nil
}

// CreateSyncProvider inserts a provider and returns it with its id. The slug
// is the caller's job (the API derives a unique one); protected is never set
// by users — the only protected provider is the seeded local backup.
func (s *Store) CreateSyncProvider(slug, name, driver, baseURL string) (SyncProvider, error) {
	created := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO sync_providers(slug, name, driver, base_url, protected, created_at) VALUES (?, ?, ?, ?, 0, ?)`,
		slug, name, driver, baseURL, created)
	if err != nil {
		if isUniqueViolation(err) {
			return SyncProvider{}, ErrSyncProviderExists
		}
		return SyncProvider{}, fmt.Errorf("create sync provider: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return SyncProvider{}, fmt.Errorf("create sync provider id: %w", err)
	}
	return SyncProvider{ID: id, Slug: slug, Name: name, Driver: driver, BaseURL: baseURL, CreatedAt: created}, nil
}

// EnsureSyncProvider returns the provider with slug, creating it when absent.
// The seed uses it so every built-in row exists before connections attach.
func (s *Store) EnsureSyncProvider(slug, name, driver, baseURL string, protected bool) (SyncProvider, error) {
	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO sync_providers(slug, name, driver, base_url, protected, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		slug, name, driver, baseURL, protected, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return SyncProvider{}, fmt.Errorf("ensure sync provider %q: %w", slug, err)
	}
	return s.SyncProviderBySlug(slug)
}

// UpdateSyncProvider replaces every editable field of the provider (name and
// base URL). Protected rows are locked — the local backup is app-managed.
func (s *Store) UpdateSyncProvider(id int64, name, baseURL string) error {
	p, err := s.SyncProvider(id)
	if err != nil {
		return err
	}
	if p.Protected {
		return ErrSyncProviderProtected
	}
	res, err := s.db.Exec(
		`UPDATE sync_providers SET name = ?, base_url = ? WHERE id = ?`,
		name, baseURL, id)
	if err != nil {
		return fmt.Errorf("update sync provider: %w", err)
	}
	return providerRowsAffectedOrNotFound(res, "update sync provider")
}

// DeleteSyncProvider removes the provider; a provider with connections is
// ErrSyncProviderInUse (a dangling target would leave orphaned state), and a
// protected row is ErrSyncProviderProtected.
func (s *Store) DeleteSyncProvider(id int64) error {
	p, err := s.SyncProvider(id)
	if err != nil {
		return err
	}
	if p.Protected {
		return ErrSyncProviderProtected
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sync_connections WHERE provider_id = ?`, id).Scan(&n); err != nil {
		return fmt.Errorf("count sync provider connections: %w", err)
	}
	if n > 0 {
		return ErrSyncProviderInUse
	}
	res, err := s.db.Exec(`DELETE FROM sync_providers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete sync provider: %w", err)
	}
	return providerRowsAffectedOrNotFound(res, "delete sync provider")
}
