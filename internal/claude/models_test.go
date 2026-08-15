package claude

import "testing"

func TestModelOptions(t *testing.T) {
	opts := ModelOptions()
	if len(opts) < 2 {
		t.Fatalf("expected at least the default and one model, got %+v", opts)
	}
	if opts[0].Value != "" {
		t.Fatalf("first option must be the empty default value, got %q", opts[0].Value)
	}
	seen := map[string]bool{}
	for _, o := range opts {
		if o.Label == "" {
			t.Fatalf("option %+v has no label", o)
		}
		if seen[o.Value] {
			t.Fatalf("duplicate model value %q", o.Value)
		}
		seen[o.Value] = true
	}
}
