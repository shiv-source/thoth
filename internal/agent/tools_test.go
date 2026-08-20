package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

func TestWikiToolsBoundToSafePath(t *testing.T) {
	dir := t.TempDir()
	reg, err := registry(dir, nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()

	read, err := reg.Get("read_file")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"../x.md", "a/../../x.md", "/etc/passwd"} {
		if _, err := read.Run(ctx, map[string]any{"path": bad}); err == nil {
			t.Fatalf("read_file(%q) should be rejected", bad)
		}
	}

	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"../x.md", "/tmp/evil.md"} {
		if _, err := write.Run(ctx, map[string]any{"path": bad, "content": "x"}); err == nil {
			t.Fatalf("write_file(%q) should be rejected", bad)
		}
	}

	list, err := reg.Get("list")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := list.Run(ctx, map[string]any{"path": ".."}); err == nil {
		t.Fatal("list(..) should be rejected")
	}
}

func TestWriteFileProducesParseableNote(t *testing.T) {
	dir := t.TempDir()
	reg, err := registry(dir, nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()

	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: A Note\ntype: meeting\n---\nbody text"
	if _, err := write.Run(ctx, map[string]any{"path": "meetings/2026-08-21-x.md", "content": content}); err != nil {
		t.Fatalf("write_file: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "meetings", "2026-08-21-x.md"))
	if err != nil {
		t.Fatalf("read back note: %v", err)
	}
	meta, body, err := wiki.ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if meta.Title != "A Note" || meta.Kind != "meeting" {
		t.Fatalf("meta = %+v", meta)
	}
	if string(body) != "body text" {
		t.Fatalf("body = %q", body)
	}

	list, err := reg.Get("list")
	if err != nil {
		t.Fatal(err)
	}
	out, err := list.Run(ctx, map[string]any{"path": "meetings"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "2026-08-21-x.md\tfile") {
		t.Fatalf("list output = %q", out)
	}

	// read_file reads the note back through the tool; a missing file and a
	// missing directory surface as errors.
	read, err := reg.Get("read_file")
	if err != nil {
		t.Fatal(err)
	}
	got, err := read.Run(ctx, map[string]any{"path": "meetings/2026-08-21-x.md"})
	if err != nil {
		t.Fatalf("read_file: %v", err)
	}
	if !strings.Contains(got, "body text") {
		t.Fatalf("read_file output = %q", got)
	}
	if _, err := read.Run(ctx, map[string]any{"path": "nope.md"}); err == nil {
		t.Fatal("read_file of a missing file should error")
	}
	if _, err := list.Run(ctx, map[string]any{"path": "nope"}); err == nil {
		t.Fatal("list of a missing directory should error")
	}
}

func TestIndexSearchTool(t *testing.T) {
	ix := openIndex(t)
	n := index.Note{
		Path: "meetings/standup.md", Title: "Standup", Kind: "meeting",
		Body: "Decided to ship the native agent.", UpdatedAt: time.Now(),
	}
	if err := ix.Upsert(n); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	reg, err := registry(t.TempDir(), ix)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	search, err := reg.Get("search")
	if err != nil {
		t.Fatal(err)
	}
	out, err := search.Run(context.Background(), map[string]any{"query": "native agent"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(out, "meetings/standup.md") {
		t.Fatalf("search output = %q, want the note path", out)
	}
}
