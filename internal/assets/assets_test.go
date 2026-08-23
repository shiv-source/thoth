package assets

import "testing"

func TestModelOptionsParse(t *testing.T) {
	opts, err := ModelOptions()
	if err != nil {
		t.Fatalf("ModelOptions: %v", err)
	}
	if len(opts) < 2 {
		t.Fatalf("expected several models, got %+v", opts)
	}
	seen := map[string]bool{}
	for _, o := range opts {
		if o.Value == "" {
			t.Fatalf("option %+v has no value", o)
		}
		if o.Name == "" {
			t.Fatalf("option %+v has no name", o)
		}
		if o.Tag == "" {
			t.Fatalf("option %+v has no tag", o)
		}
		if o.Provider == "" {
			t.Fatalf("option %+v has no provider", o)
		}
		if seen[o.Value] {
			t.Fatalf("duplicate model value %q", o.Value)
		}
		seen[o.Value] = true
	}
}

// TestModelOptionsSplitShape pins the name/tag split: the label is two
// columns in the seeded table, so the embedded JSON must carry them as two
// fields (no "name — tag" parsing anywhere).
func TestModelOptionsSplitShape(t *testing.T) {
	opts, err := ModelOptions()
	if err != nil {
		t.Fatalf("ModelOptions: %v", err)
	}
	byValue := map[string]Option{}
	for _, o := range opts {
		byValue[o.Value] = o
	}
	opus, ok := byValue["claude-opus-4-8"]
	if !ok {
		t.Fatal("expected claude-opus-4-8 in models.json")
	}
	if opus.Name != "Claude Opus 4.8" {
		t.Fatalf("name = %q, want %q", opus.Name, "Claude Opus 4.8")
	}
	if opus.Tag != "strongest" {
		t.Fatalf("tag = %q, want %q", opus.Tag, "strongest")
	}
	if opus.Provider != "Anthropic" {
		t.Fatalf("provider = %q, want %q", opus.Provider, "Anthropic")
	}
}

func TestSyncProviderOptionsParse(t *testing.T) {
	opts, err := SyncProviderOptions()
	if err != nil {
		t.Fatalf("SyncProviderOptions: %v", err)
	}
	if len(opts) != 4 {
		t.Fatalf("expected 4 built-in sync providers, got %+v", opts)
	}
	seen := map[string]bool{}
	protected := 0
	for _, o := range opts {
		if o.Slug == "" || o.Name == "" || o.Driver == "" {
			t.Fatalf("option %+v is incomplete", o)
		}
		if seen[o.Slug] {
			t.Fatalf("duplicate sync provider slug %q", o.Slug)
		}
		seen[o.Slug] = true
		if o.Protected {
			protected++
		}
	}
	// Exactly one first-class provider: the local backup.
	if protected != 1 || !opts[3].Protected || opts[3].Slug != "local" {
		t.Fatalf("protected flag wrong: %+v", opts)
	}
}
