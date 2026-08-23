package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func openProviders(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestProvidersCRUD(t *testing.T) {
	s := openProviders(t)

	created, err := s.CreateProvider("DeepSeek", "https://api.deepseek.com", "ds-key")
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	if created.ID == 0 || created.Name != "DeepSeek" || created.BaseURL != "https://api.deepseek.com" ||
		created.APIKey != "ds-key" || created.CreatedAt == "" {
		t.Fatalf("created mismatch: %+v", created)
	}

	providers, err := s.ListProviders()
	if err != nil || len(providers) != 1 {
		t.Fatalf("ListProviders: %v %+v", err, providers)
	}
	if providers[0].Name != "DeepSeek" || providers[0].ModelCount != 0 {
		t.Fatalf("list mismatch: %+v", providers[0])
	}

	if err := s.UpdateProvider(created.ID, "Deepseek AI", "https://api.deepseek.ai", "ds-key-2"); err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}
	got, err := s.Provider(created.ID)
	if err != nil {
		t.Fatalf("Provider: %v", err)
	}
	if got.Name != "Deepseek AI" || got.BaseURL != "https://api.deepseek.ai" || got.APIKey != "ds-key-2" {
		t.Fatalf("update mismatch: %+v", got)
	}

	if err := s.DeleteProvider(created.ID); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}
	providers, err = s.ListProviders()
	if err != nil || len(providers) != 0 {
		t.Fatalf("ListProviders after delete: %v %+v", err, providers)
	}
}

func TestCreateProviderDuplicate(t *testing.T) {
	s := openProviders(t)
	if _, err := s.CreateProvider("DeepSeek", "", ""); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateProvider("DeepSeek", "", "")
	if !errors.Is(err, ErrProviderExists) {
		t.Fatalf("expected ErrProviderExists, got %v", err)
	}
}

func TestUpdateProviderDuplicate(t *testing.T) {
	s := openProviders(t)
	if _, err := s.CreateProvider("A", "", ""); err != nil {
		t.Fatal(err)
	}
	b, err := s.CreateProvider("B", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateProvider(b.ID, "A", "", ""); !errors.Is(err, ErrProviderExists) {
		t.Fatalf("expected ErrProviderExists, got %v", err)
	}
}

func TestProviderNotFound(t *testing.T) {
	s := openProviders(t)
	if _, err := s.Provider(404); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
	if _, err := s.ProviderByName("nope"); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
	if err := s.UpdateProvider(404, "x", "", ""); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
	if err := s.DeleteProvider(404); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestEnsureProviderCreatesOnce(t *testing.T) {
	s := openProviders(t)
	first, err := s.EnsureProvider("Anthropic")
	if err != nil {
		t.Fatalf("EnsureProvider: %v", err)
	}
	second, err := s.EnsureProvider("Anthropic")
	if err != nil {
		t.Fatalf("EnsureProvider second: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("EnsureProvider created a duplicate: %d vs %d", first.ID, second.ID)
	}
	providers, err := s.ListProviders()
	if err != nil || len(providers) != 1 {
		t.Fatalf("expected one provider: %v %+v", err, providers)
	}
	if _, err := s.EnsureProvider(""); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("EnsureProvider(\"\") = %v, want ErrProviderNotFound", err)
	}
}

func TestListProvidersSortedWithModelCounts(t *testing.T) {
	s := openProviders(t)
	z := mustProvider(t, s, "Zeta")
	_ = mustProvider(t, s, "alpha")
	m := mustProvider(t, s, "Mido")
	if _, err := s.CreateModel("z1", "Z1", "", z.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateModel("m1", "M1", "", m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateModel("m2", "M2", "", m.ID); err != nil {
		t.Fatal(err)
	}
	providers, err := s.ListProviders()
	if err != nil || len(providers) != 3 {
		t.Fatalf("ListProviders: %v %+v", err, providers)
	}
	// Case-insensitive A→Z: alpha, Mido, Zeta.
	if providers[0].Name != "alpha" || providers[1].Name != "Mido" || providers[2].Name != "Zeta" {
		t.Fatalf("not sorted A→Z: %+v", providers)
	}
	if providers[1].ModelCount != 2 || providers[2].ModelCount != 1 || providers[0].ModelCount != 0 {
		t.Fatalf("model counts wrong: %+v", providers)
	}
}

func TestDeleteProviderCascadesModels(t *testing.T) {
	s := openProviders(t)
	p := mustProvider(t, s, "Doomed")
	other := mustProvider(t, s, "Survivor")
	if _, err := s.CreateModel("m1", "M1", "", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateModel("m2", "M2", "", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateModel("m3", "M3", "", other.ID); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteProvider(p.ID); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}
	models, err := s.ListModels()
	if err != nil || len(models) != 1 || models[0].ProviderID != other.ID {
		t.Fatalf("cascade left wrong models: %v %+v", err, models)
	}
}

func mustProvider(t *testing.T, s *Store, name string) Provider {
	t.Helper()
	p, err := s.CreateProvider(name, "", "")
	if err != nil {
		t.Fatalf("CreateProvider(%q): %v", name, err)
	}
	return p
}
