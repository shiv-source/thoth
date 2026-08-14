package wiki

import "testing"

func TestSafePath(t *testing.T) {
	root := t.TempDir()
	if got, err := SafePath(root, "meetings/a.md"); err != nil || got == "" {
		t.Fatalf("SafePath: %v %q", err, got)
	}
	for _, bad := range []string{"../x.md", "a/../../x.md", "/etc/passwd"} {
		if _, err := SafePath(root, bad); err == nil {
			t.Fatalf("expected rejection of %q", bad)
		}
	}
}
