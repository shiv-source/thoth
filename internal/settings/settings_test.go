package settings_test

import (
	"path/filepath"
	"testing"

	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
)

// openTestRepo builds a temp db via the store's migrations (which seed the
// settings defaults) and opens the settings repo on it.
func openTestRepo(t *testing.T) *settings.Repo {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := settings.OpenRepo(path)
	if err != nil {
		t.Fatalf("OpenRepo: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

// openTestRepos additionally keeps the store open so tests can seed provider
// rows (ProviderConfig now reads the providers table, not settings keys).
func openTestRepos(t *testing.T) (*settings.Repo, *store.Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	r, err := settings.OpenRepo(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, st
}

func TestOpenSeedsDefaults(t *testing.T) {
	r := openTestRepo(t)
	for key, want := range map[string]string{
		settings.KeyWikiPath: "~/.thoth/wiki",
	} {
		got, found, err := r.Setting(key)
		if err != nil || !found || got != want {
			t.Fatalf("Setting(%q) = %q/%v/%v, want %q/true/nil", key, got, found, err, want)
		}
	}
}

func TestProviderSlugAndKeys(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		apiKeyKey  string
		baseURLKey string
	}{
		{"plain name", "DeepSeek", "provider_deepseek_api_key", "provider_deepseek_base_url"},
		{"dotted name", "Z.AI", "provider_zai_api_key", "provider_zai_base_url"},
		{"mixed case", "xAI", "provider_xai_api_key", "provider_xai_base_url"},
		{"spaced name", "Anthropic", "provider_anthropic_api_key", "provider_anthropic_base_url"},
		{"empty name", "", "provider__api_key", "provider__base_url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := settings.ProviderAPIKeyKey(tt.provider); got != tt.apiKeyKey {
				t.Fatalf("ProviderAPIKeyKey(%q) = %q, want %q", tt.provider, got, tt.apiKeyKey)
			}
			if got := settings.ProviderBaseURLKey(tt.provider); got != tt.baseURLKey {
				t.Fatalf("ProviderBaseURLKey(%q) = %q, want %q", tt.provider, got, tt.baseURLKey)
			}
		})
	}
}

func TestProviderConfigResolution(t *testing.T) {
	t.Run("provider row wins", func(t *testing.T) {
		r, st := openTestRepos(t)
		if _, err := st.CreateProvider("DeepSeek", "https://api.deepseek.com", "deepseek-key"); err != nil {
			t.Fatal(err)
		}
		apiKey, baseURL, err := r.ProviderConfig("DeepSeek")
		if err != nil || apiKey != "deepseek-key" || baseURL != "https://api.deepseek.com" {
			t.Fatalf("ProviderConfig(DeepSeek) = %q/%q/%v", apiKey, baseURL, err)
		}
	})

	t.Run("absent provider resolves empty", func(t *testing.T) {
		r := openTestRepo(t)
		apiKey, baseURL, err := r.ProviderConfig("Anthropic")
		if err != nil || apiKey != "" || baseURL != "" {
			t.Fatalf("ProviderConfig(Anthropic) = %q/%q/%v", apiKey, baseURL, err)
		}
	})

	t.Run("empty provider name resolves empty", func(t *testing.T) {
		r, st := openTestRepos(t)
		if _, err := st.CreateProvider("Anthropic", "", "anthropic-key"); err != nil {
			t.Fatal(err)
		}
		apiKey, baseURL, err := r.ProviderConfig("")
		if err != nil || apiKey != "" || baseURL != "" {
			t.Fatalf("ProviderConfig(\"\") = %q/%q/%v", apiKey, baseURL, err)
		}
	})
}

func TestModelSettingRoundTrip(t *testing.T) {
	r := openTestRepo(t)
	if _, found, err := r.Setting(settings.KeyModel); err != nil || found {
		t.Fatalf("model must default to absent, found=%v err=%v", found, err)
	}
	if err := r.SetSetting(settings.KeyModel, "claude-haiku-4-5-20251001"); err != nil {
		t.Fatal(err)
	}
	got, found, err := r.Setting(settings.KeyModel)
	if err != nil || !found || got != "claude-haiku-4-5-20251001" {
		t.Fatalf("model round trip = %q/%v/%v", got, found, err)
	}
}

func TestSettingRoundTrip(t *testing.T) {
	r := openTestRepo(t)
	if err := r.SetSetting(settings.KeyActiveConnection, "7"); err != nil {
		t.Fatal(err)
	}
	got, found, err := r.Setting(settings.KeyActiveConnection)
	if err != nil || !found || got != "7" {
		t.Fatalf("after set: %q/%v/%v", got, found, err)
	}
	// Upsert semantics: a second write replaces the value.
	if err := r.SetSetting(settings.KeyActiveConnection, ""); err != nil {
		t.Fatal(err)
	}
	if got, found, err = r.Setting(settings.KeyActiveConnection); err != nil || !found || got != "" {
		t.Fatalf("after clear: %q/%v/%v", got, found, err)
	}
}

func TestSettingAbsent(t *testing.T) {
	r := openTestRepo(t)
	got, found, err := r.Setting("no-such-key")
	if err != nil || found || got != "" {
		t.Fatalf("absent key: %q/%v/%v, want empty/false/nil", got, found, err)
	}
}

func TestSyncEnabledParsing(t *testing.T) {
	r := openTestRepo(t)
	// The active-connection key is absent by default and round-trips.
	if on, found, err := r.Setting(settings.KeyActiveConnection); err != nil || found || on != "" {
		t.Fatalf("absent sync_active_connection = %q/%v/%v, want empty/false/nil", on, found, err)
	}
}

func TestContextInjectionParsing(t *testing.T) {
	r := openTestRepo(t)
	// Absent key defaults to off.
	if on, err := r.ContextInjection(); err != nil || on {
		t.Fatalf("absent context_injection = %v/%v, want false/nil", on, err)
	}
	for _, in := range []string{"true"} {
		if err := r.SetSetting(settings.KeyContextInjection, in); err != nil {
			t.Fatal(err)
		}
		if on, err := r.ContextInjection(); err != nil || !on {
			t.Fatalf("context_injection %q = %v/%v, want true/nil", in, on, err)
		}
	}
	for _, in := range []string{"false", "garbage", "TRUE"} {
		if err := r.SetSetting(settings.KeyContextInjection, in); err != nil {
			t.Fatal(err)
		}
		if on, err := r.ContextInjection(); err != nil || on {
			t.Fatalf("context_injection %q = %v/%v, want false/nil", in, on, err)
		}
	}
}

func TestFoldersParsing(t *testing.T) {
	r := openTestRepo(t)
	// Absent key falls back to nil.
	folders, err := r.Folders()
	if err != nil || folders != nil {
		t.Fatalf("absent wiki_folders = %v/%v, want nil/nil", folders, err)
	}
	// Comma-separated with whitespace is parsed and trimmed.
	if err := r.SetSetting(settings.KeyWikiFolders, "journal, recipes , notes"); err != nil {
		t.Fatal(err)
	}
	folders, err = r.Folders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 3 || folders[0] != "journal" || folders[1] != "recipes" || folders[2] != "notes" {
		t.Fatalf("parsed folders = %v, want [journal recipes notes]", folders)
	}
	// An empty value means the defaults again.
	if err := r.SetSetting(settings.KeyWikiFolders, ""); err != nil {
		t.Fatal(err)
	}
	if folders, err = r.Folders(); err != nil || folders != nil {
		t.Fatalf("empty wiki_folders = %v/%v, want nil/nil", folders, err)
	}
	// Stray commas produce no empty entries.
	if err := r.SetSetting(settings.KeyWikiFolders, ",,,"); err != nil {
		t.Fatal(err)
	}
	if folders, err = r.Folders(); err != nil || len(folders) != 0 {
		t.Fatalf("comma-only wiki_folders = %v/%v, want []/nil", folders, err)
	}
}

func TestRepoClosedErrors(t *testing.T) {
	r := openTestRepo(t)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Setting(settings.KeyWikiPath); err == nil {
		t.Fatal("Setting on closed repo must error")
	}
	if err := r.SetSetting(settings.KeyWikiPath, "x"); err == nil {
		t.Fatal("SetSetting on closed repo must error")
	}
}
