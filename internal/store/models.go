package store

import (
	"database/sql"
	"errors"
	"fmt"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// LLMModel is one row of the user-editable model registry.
type LLMModel struct {
	ID       int64  `json:"id"`
	Value    string `json:"value"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Provider string `json:"provider"`
}

// ErrModelExists is returned when a create/update collides with another
// row's UNIQUE value; the API layer maps it to 409.
var ErrModelExists = errors.New("a model with this value already exists")

// ErrModelNotFound is returned when a read/update/delete targets a missing
// row; the API layer maps it to 404.
var ErrModelNotFound = errors.New("model not found")

// ListModels returns every model in insertion (id) order, which keeps the
// seeded models.json order stable in the picker.
func (s *Store) ListModels() ([]LLMModel, error) {
	rows, err := s.db.Query(
		`SELECT id, value, name, tag, provider FROM llm_models ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []LLMModel
	for rows.Next() {
		var m LLMModel
		if err := rows.Scan(&m.ID, &m.Value, &m.Name, &m.Tag, &m.Provider); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Model returns one model by id.
func (s *Store) Model(id int64) (LLMModel, error) {
	var m LLMModel
	err := s.db.QueryRow(
		`SELECT id, value, name, tag, provider FROM llm_models WHERE id = ?`, id).
		Scan(&m.ID, &m.Value, &m.Name, &m.Tag, &m.Provider)
	if errors.Is(err, sql.ErrNoRows) {
		return LLMModel{}, ErrModelNotFound
	}
	if err != nil {
		return LLMModel{}, fmt.Errorf("read model: %w", err)
	}
	return m, nil
}

// CreateModel inserts a model and returns it with its id.
func (s *Store) CreateModel(value, name, tag, provider string) (LLMModel, error) {
	res, err := s.db.Exec(
		`INSERT INTO llm_models(value, name, tag, provider) VALUES (?, ?, ?, ?)`,
		value, name, tag, provider)
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
	return LLMModel{ID: id, Value: value, Name: name, Tag: tag, Provider: provider}, nil
}

// UpdateModel replaces every editable field of the model.
func (s *Store) UpdateModel(id int64, value, name, tag, provider string) error {
	res, err := s.db.Exec(
		`UPDATE llm_models SET value = ?, name = ?, tag = ?, provider = ? WHERE id = ?`,
		value, name, tag, provider, id)
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
