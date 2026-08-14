package index

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestRebuild(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("meetings/ok.md", "---\ntitle: Good\ntype: meeting\n---\nvalid body\n")
	write("knowledge/bad.md", "no frontmatter at all\n")
	write("meetings/ignored.txt", "not markdown\n")

	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := ix.Rebuild(root, log); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	got, err := ix.Search("valid", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "meetings/ok.md" {
		t.Fatalf("expected exactly the valid note indexed, got %+v", got)
	}
}

func TestRebuildRemovesDeletedNotes(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("meetings/ok.md", "---\ntitle: Good\ntype: meeting\n---\nvalid body\n")

	ix, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := ix.Rebuild(root, log); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	got, err := ix.Search("valid", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != "meetings/ok.md" {
		t.Fatalf("expected the valid note indexed, got %+v", got)
	}

	if err := os.Remove(filepath.Join(root, "meetings/ok.md")); err != nil {
		t.Fatal(err)
	}
	if err := ix.Rebuild(root, log); err != nil {
		t.Fatalf("Rebuild after delete: %v", err)
	}
	got, err = ix.Search("valid", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected zero results after note deleted, got %+v", got)
	}
}
