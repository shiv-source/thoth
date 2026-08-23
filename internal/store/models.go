package store

import (
	"database/sql"
	"errors"
	"fmt"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// LLMModel is one row of the user-editable model registry. Provider is the
// owning providers row's name (via a join; empty for the Unassigned
// catch-all) and ProviderID is that row's id (0 when unassigned).
type LLMModel struct {
	ID         int64  `json:"id"`
	Value      string `json:"value"`
	Name       string `json:"name"`
	Tag        string `json:"tag"`
	Provider   string `json:"provider"`
	ProviderID int64  `json:"provider_id"`
}

// ErrModelExists is returned when a create/update collides with another
// row's UNIQUE value; the API layer maps it to 409.
var ErrModelExists = errors.New("a model with this value already exists")

// ErrModelNotFound is returned when a read/update/delete targets a missing
// row; the API layer maps it to 404.
var ErrModelNotFound = errors.New("model not found")

// modelSelect is the column list shared by the list and single-row reads; the
// LEFT JOIN keeps unassigned models (provider_id NULL) present as empty names.
const modelSelect = `
	SELECT m.id, m.value, m.name, m.tag, COALESCE(p.name, ''), m.provider_id
	FROM llm_models m
	LEFT JOIN providers p ON p.id = m.provider_id`

// ListModels returns every model in insertion (id) order, which keeps the
// seeded models.json order stable in the picker.
func (s *Store) ListModels() ([]LLMModel, error) {
	rows, err := s.db.Query(modelSelect + ` ORDER BY m.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []LLMModel
	for rows.Next() {
		var m LLMModel
		if err := scanModel(&m, rows.Scan); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Model returns one model by id.
func (s *Store) Model(id int64) (LLMModel, error) {
	var m LLMModel
	err := scanModel(&m, s.db.QueryRow(modelSelect+` WHERE m.id = ?`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return LLMModel{}, ErrModelNotFound
	}
	if err != nil {
		return LLMModel{}, fmt.Errorf("read model: %w", err)
	}
	return m, nil
}

// scanModel fills m from the shared modelSelect columns, mapping a NULL
// provider_id (the Unassigned catch-all) to 0. scan is the caller's Rows.Scan
// or Row.Scan.
func scanModel(m *LLMModel, scan func(...any) error) error {
	var providerID sql.NullInt64
	if err := scan(&m.ID, &m.Value, &m.Name, &m.Tag, &m.Provider, &providerID); err != nil {
		return err
	}
	if providerID.Valid {
		m.ProviderID = providerID.Int64
	}
	return nil
}

// CreateModel inserts a model and returns it with its id. providerID 0 means
// the Unassigned catch-all (provider_id NULL).
func (s *Store) CreateModel(value, name, tag string, providerID int64) (LLMModel, error) {
	pid := nullableID(providerID)
	res, err := s.db.Exec(
		`INSERT INTO llm_models(value, name, tag, provider_id) VALUES (?, ?, ?, ?)`,
		value, name, tag, pid)
	if err != nil {
		if isUniqueViolation(err) {
			return LLMModel{}, ErrModelExists
		}
		return LLMModel{}, fmt.Errorf("create model: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return LLMModel{}, fmt.Errorf("create model id: %w", err)
	}
	provider := ""
	if providerID != 0 {
		if p, err := s.Provider(providerID); err == nil {
			provider = p.Name
		}
	}
	return LLMModel{ID: id, Value: value, Name: name, Tag: tag, Provider: provider, ProviderID: providerID}, nil
}

// UpdateModel replaces every editable field of the model.
func (s *Store) UpdateModel(id int64, value, name, tag string, providerID int64) error {
	res, err := s.db.Exec(
		`UPDATE llm_models SET value = ?, name = ?, tag = ?, provider_id = ? WHERE id = ?`,
		value, name, tag, nullableID(providerID), id)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrModelExists
		}
		return fmt.Errorf("update model: %w", err)
	}
	return rowsAffectedOrNotFound(res, "update model")
}

// DeleteModel removes the model; deleting a missing row is ErrModelNotFound.
func (s *Store) DeleteModel(id int64) error {
	res, err := s.db.Exec(`DELETE FROM llm_models WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete model: %w", err)
	}
	return rowsAffectedOrNotFound(res, "delete model")
}

// nullableID maps a 0 id to NULL so unassigned rows store provider_id NULL.
func nullableID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint
// failure (the driver surfaces it as a typed *sqlite.Error, not a string).
func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	return errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

// rowsAffectedOrNotFound maps a zero-affected mutation to ErrModelNotFound.
func rowsAffectedOrNotFound(res interface{ RowsAffected() (int64, error) }, op string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows: %w", op, err)
	}
	if n == 0 {
		return ErrModelNotFound
	}
	return nil
}
