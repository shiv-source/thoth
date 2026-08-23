package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shiv-source/thoth/internal/settings"
)

// Provider is one row of the providers table: a model provider the user
// configured, owning its own base URL override, API key, and models.
type Provider struct {
	ID        int64
	Name      string
	BaseURL   string
	APIKey    string
	CreatedAt string
	// ModelCount is the number of models registered under this provider;
	// populated by ListProviders, zero elsewhere.
	ModelCount int
}

// ErrProviderExists is returned when a create/update collides with another
// row's UNIQUE name; the API layer maps it to 409.
var ErrProviderExists = errors.New("a provider with this name already exists")

// ErrProviderNotFound is returned when a read/update/delete targets a missing
// row; the API layer maps it to 404.
var ErrProviderNotFound = errors.New("provider not found")

const providerColumns = "id, name, base_url, api_key, created_at"

// ListProviders returns every provider, case-insensitive A→Z, each carrying
// the number of models registered under it.
func (s *Store) ListProviders() ([]Provider, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.base_url, p.api_key, p.created_at,
		       (SELECT COUNT(*) FROM llm_models m WHERE m.provider_id = p.id)
		FROM providers p
		ORDER BY lower(p.name) ASC, p.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.CreatedAt, &p.ModelCount); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Provider returns one provider by id.
func (s *Store) Provider(id int64) (Provider, error) {
	var p Provider
	err := s.db.QueryRow(
		`SELECT `+providerColumns+` FROM providers WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Provider{}, ErrProviderNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("read provider: %w", err)
	}
	return p, nil
}

// ProviderByName returns one provider by its exact name.
func (s *Store) ProviderByName(name string) (Provider, error) {
	var p Provider
	err := s.db.QueryRow(
		`SELECT `+providerColumns+` FROM providers WHERE name = ?`, name).
		Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Provider{}, ErrProviderNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("read provider %q: %w", name, err)
	}
	return p, nil
}

// CreateProvider inserts a provider and returns it with its id.
func (s *Store) CreateProvider(name, baseURL, apiKey string) (Provider, error) {
	created := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO providers(name, base_url, api_key, created_at) VALUES (?, ?, ?, ?)`,
		name, baseURL, apiKey, created)
	if err != nil {
		if isUniqueViolation(err) {
			return Provider{}, ErrProviderExists
		}
		return Provider{}, fmt.Errorf("create provider: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Provider{}, fmt.Errorf("create provider id: %w", err)
	}
	return Provider{ID: id, Name: name, BaseURL: baseURL, APIKey: apiKey, CreatedAt: created}, nil
}

// EnsureProvider returns the provider with name, creating it (empty
// credentials) when absent. The seed uses it so every model option's provider
// row exists before its models are inserted.
func (s *Store) EnsureProvider(name string) (Provider, error) {
	if name == "" {
		return Provider{}, ErrProviderNotFound
	}
	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO providers(name, base_url, api_key, created_at) VALUES (?, '', '', ?)`,
		name, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return Provider{}, fmt.Errorf("ensure provider %q: %w", name, err)
	}
	return s.ProviderByName(name)
}

// UpdateProvider replaces every editable field of the provider.
func (s *Store) UpdateProvider(id int64, name, baseURL, apiKey string) error {
	res, err := s.db.Exec(
		`UPDATE providers SET name = ?, base_url = ?, api_key = ? WHERE id = ?`,
		name, baseURL, apiKey, id)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrProviderExists
		}
		return fmt.Errorf("update provider: %w", err)
	}
	return providerRowsAffectedOrNotFound(res, "update provider")
}

// DeleteProvider removes the provider and its models in one transaction;
// deleting a missing row is ErrProviderNotFound.
func (s *Store) DeleteProvider(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin delete provider: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit
	if _, err := tx.Exec(`DELETE FROM llm_models WHERE provider_id = ?`, id); err != nil {
		return fmt.Errorf("delete provider models: %w", err)
	}
	res, err := tx.Exec(`DELETE FROM providers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	if err := providerRowsAffectedOrNotFound(res, "delete provider"); err != nil {
		return err
	}
	return tx.Commit()
}

// providerRowsAffectedOrNotFound maps a zero-affected mutation to
// ErrProviderNotFound.
func providerRowsAffectedOrNotFound(res interface{ RowsAffected() (int64, error) }, op string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows: %w", op, err)
	}
	if n == 0 {
		return ErrProviderNotFound
	}
	return nil
}

// backfillProviderCredentials is the one-time data migration of the 0011
// cutover: the providers table (created by the migration) starts with empty
// credentials, while the user's real keys still live in the legacy
// per-provider settings keys (provider_<slug>_api_key / _base_url). It copies
// them into the providers rows and deletes the keys, so it runs once in
// effect — later opens find no keys and no-op. Called from store.Open after
// the SQL migrations, which guarantees the providers table exists.
func backfillProviderCredentials(db *sql.DB) error {
	// Collect the providers first and close the cursor before the per-provider
	// lookups: the pool holds a single connection (OpenDB), so querying while
	// rows is open would deadlock on itself.
	rows, err := db.Query(`SELECT id, name FROM providers`)
	if err != nil {
		return fmt.Errorf("read providers for backfill: %w", err)
	}
	var providers []struct {
		id   int64
		name string
	}
	for rows.Next() {
		var p struct {
			id   int64
			name string
		}
		if err := rows.Scan(&p.id, &p.name); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan provider for backfill: %w", err)
		}
		providers = append(providers, p)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close backfill providers: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate backfill providers: %w", err)
	}

	type credUpdate struct {
		id              int64
		apiKey, baseURL string
	}
	var updates []credUpdate
	for _, p := range providers {
		// Absent keys scan as empty (QueryRow on a missing row leaves dest
		// untouched); only a non-empty value is worth migrating.
		var apiKey, baseURL string
		_ = db.QueryRow(`SELECT value FROM settings WHERE key = ?`, settings.ProviderAPIKeyKey(p.name)).Scan(&apiKey)
		_ = db.QueryRow(`SELECT value FROM settings WHERE key = ?`, settings.ProviderBaseURLKey(p.name)).Scan(&baseURL)
		if apiKey == "" && baseURL == "" {
			continue
		}
		updates = append(updates, credUpdate{id: p.id, apiKey: apiKey, baseURL: baseURL})
	}
	for _, u := range updates {
		if _, err := db.Exec(
			`UPDATE providers SET api_key = ?, base_url = ? WHERE id = ?`,
			u.apiKey, u.baseURL, u.id); err != nil {
			return fmt.Errorf("backfill provider credentials: %w", err)
		}
	}
	// The legacy keys are dead now — nothing reads them after the cutover.
	// Removing them makes this a one-time copy instead of a two-source truth.
	if _, err := db.Exec(`DELETE FROM settings WHERE key LIKE 'provider\_%\_api\_key' ESCAPE '\'`); err != nil {
		return fmt.Errorf("remove legacy provider api keys: %w", err)
	}
	if _, err := db.Exec(`DELETE FROM settings WHERE key LIKE 'provider\_%\_base\_url' ESCAPE '\'`); err != nil {
		return fmt.Errorf("remove legacy provider base urls: %w", err)
	}
	return nil
}
