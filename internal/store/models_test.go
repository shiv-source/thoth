package store

import (
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"
)

// openModels opens a temp store for the llm_models tests.
func openModels(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestModelsCRUD(t *testing.T) {
	s := openModels(t)

	created, err := s.CreateModel("my-model", "My Model", "test", "Vendor")
	if err != nil {
		t.Fatalf("CreateModel: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected a non-zero id")
	}

	models, err := s.ListModels()
	if err != nil || len(models) != 1 {
		t.Fatalf("ListModels: %v %+v", err, models)
	}
	got := models[0]
	if got.Value != "my-model" || got.Name != "My Model" || got.Tag != "test" || got.Provider != "Vendor" {
		t.Fatalf("round trip mismatch: %+v", got)
	}

	if err := s.UpdateModel(created.ID, "my-model-2", "My Model 2", "", ""); err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
	models, err = s.ListModels()
	if err != nil || len(models) != 1 {
		t.Fatalf("ListModels after update: %v %+v", err, models)
	}
	if models[0].Value != "my-model-2" || models[0].Name != "My Model 2" {
		t.Fatalf("update mismatch: %+v", models[0])
	}

	if err := s.DeleteModel(created.ID); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	models, err = s.ListModels()
	if err != nil || len(models) != 0 {
		t.Fatalf("ListModels after delete: %v %+v", err, models)
	}
}

func TestModelByID(t *testing.T) {
	s := openModels(t)
	created, err := s.CreateModel("m", "M", "d", "p")
	if err != nil {
		t.Fatalf("CreateModel: %v", err)
	}
	got, err := s.Model(created.ID)
	if err != nil {
		t.Fatalf("Model: %v", err)
	}
	if got.Value != "m" || got.Name != "M" || got.Tag != "d" || got.Provider != "p" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	if _, err := s.Model(created.ID + 1); !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestCreateModelDuplicateValue(t *testing.T) {
	s := openModels(t)
	if _, err := s.CreateModel("dup", "First", "", ""); err != nil {
		t.Fatalf("CreateModel: %v", err)
	}
	_, err := s.CreateModel("dup", "Second", "", "")
	if !errors.Is(err, ErrModelExists) {
		t.Fatalf("expected ErrModelExists, got %v", err)
	}
}

func TestUpdateModelNotFound(t *testing.T) {
	s := openModels(t)
	if err := s.UpdateModel(404, "x", "X", "", ""); !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestDeleteModelNotFound(t *testing.T) {
	s := openModels(t)
	if err := s.DeleteModel(404); !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestListModelsSeedOrder(t *testing.T) {
	s := openModels(t)
	for _, v := range []string{"b", "a", "c"} {
		if _, err := s.CreateModel(v, v, "", ""); err != nil {
			t.Fatalf("CreateModel %s: %v", v, err)
		}
	}
	models, err := s.ListModels()
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 3 || models[0].Value != "b" || models[1].Value != "a" || models[2].Value != "c" {
		t.Fatalf("expected insertion order, got %+v", models)
	}
}

// TestUpgradeRenamesDescriptionToTag reproduces a database that applied 0008
// while the column was still named description (the in-development rename):
// opening it must apply 0009, keep the data, and expose the column as tag.
func TestUpgradeRenamesDescriptionToTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	// Apply everything through 0008 and stop — the state the user's database
	// was in before the rename landed as its own migration.
	for i, name := range names[:8] {
		raw, err := fs.ReadFile(migrationFS, name)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyMigration(db, i+1, splitStatements(string(raw))); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	if _, err := db.Exec(
		`INSERT INTO llm_models(value, name, description, provider) VALUES ('v', 'V', 'd', 'P')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()
	models, err := s.ListModels()
	if err != nil {
		t.Fatalf("ListModels after upgrade: %v", err)
	}
	if len(models) != 1 || models[0].Tag != "d" || models[0].Value != "v" {
		t.Fatalf("data lost in the rename: %+v", models)
	}
}

// TestMigrationSeedsAPIKeyRow pins the 0008 seed: api_key exists (empty =
// not configured, readers validate against ”) and models_seeded is gone —
// seeding is table-empty-driven now, not marker-driven.
func TestMigrationSeedsAPIKeyRow(t *testing.T) {
	s := openModels(t)
	var value string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = 'api_key'`).Scan(&value); err != nil {
		t.Fatalf("settings key api_key: %v", err)
	}
	if value != "" {
		t.Fatalf("api_key = %q, want empty", value)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM settings WHERE key = 'models_seeded'`).Scan(&count); err != nil {
		t.Fatalf("models_seeded probe: %v", err)
	}
	if count != 0 {
		t.Fatalf("models_seeded must not exist, found %d row(s)", count)
	}
}
