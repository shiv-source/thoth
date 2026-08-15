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
	if opts[0].Value != "" {
		t.Fatalf("first option must be the empty default, got %q", opts[0].Value)
	}
	seen := map[string]bool{}
	for _, o := range opts {
		if o.Label == "" {
			t.Fatalf("option %+v has no label", o)
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
