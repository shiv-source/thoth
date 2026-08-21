package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agentlib "github.com/shiv-source/thoth/agent"
	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/wiki"
)

func TestWikiToolsBoundToSafePath(t *testing.T) {
	dir := t.TempDir()
	reg, err := registry(RegistryOptions{Wiki: wiki.New(dir)})
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
	reg, err := registry(RegistryOptions{Wiki: wiki.New(dir)})
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

func TestWikiToolsRejectSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "secret.txt")); err != nil {
		t.Fatal(err)
	}
	reg, err := registry(RegistryOptions{Wiki: wiki.New(root)})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()

	read, err := reg.Get("read_file")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"escape/secret.txt", "secret.txt"} {
		if _, err := read.Run(ctx, map[string]any{"path": p}); err == nil {
			t.Fatalf("read_file(%q) must reject a symlink escaping the root", p)
		}
	}

	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write.Run(ctx, map[string]any{"path": "escape/new.txt", "content": "x"}); err == nil {
		t.Fatal("write_file through an escaping symlinked dir succeeded")
	}

	list, err := reg.Get("list")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := list.Run(ctx, map[string]any{"path": "escape"}); err == nil {
		t.Fatal("list of an escaping symlink succeeded")
	}
}

func TestWikiWriteFileAtomicLeavesNoPartialFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg, err := registry(RegistryOptions{Wiki: wiki.New(root)})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	// Renaming onto an existing directory fails after the temp file is fully
	// written; the temp file must be cleaned up and the target untouched.
	if _, err := write.Run(context.Background(), map[string]any{"path": "dir", "content": "x"}); err == nil {
		t.Fatal("write over a directory succeeded, want error")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".thoth-write-") {
			t.Fatalf("leftover temp file %q", e.Name())
		}
	}
}

func TestWikiToolsFollowRootSwap(t *testing.T) {
	rootA, rootB := t.TempDir(), t.TempDir()
	w := wiki.New(rootA)
	reg, err := registry(RegistryOptions{Wiki: w})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()
	write, err := reg.Get("write_file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write.Run(ctx, map[string]any{"path": "note.md", "content": "a"}); err != nil {
		t.Fatalf("write before swap: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootA, "note.md")); err != nil {
		t.Fatalf("old-root note missing: %v", err)
	}

	// Swap the live root (onSettingsSaved): the same registry must now serve
	// the new root, so system prompt and tools agree.
	w.SetRoot(rootB)
	if _, err := write.Run(ctx, map[string]any{"path": "note.md", "content": "b"}); err != nil {
		t.Fatalf("write after swap: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootB, "note.md")); err != nil {
		t.Fatalf("new-root note missing: %v", err)
	}

	read, err := reg.Get("read_file")
	if err != nil {
		t.Fatal(err)
	}
	got, err := read.Run(ctx, map[string]any{"path": "note.md"})
	if err != nil {
		t.Fatalf("read after swap: %v", err)
	}
	if got != "b" {
		t.Fatalf("read after swap = %q, want %q (tools must follow the new root)", got, "b")
	}
}

func TestWikiFSErrorPaths(t *testing.T) {
	root := t.TempDir()
	fsys := wikiFS{wiki: wiki.New(root)}

	// SafePath rejection (defense in depth: the tools already cleanRel, but
	// the FS seam must reject escapes on its own too).
	if err := fsys.WriteFile("../escape.md", []byte("x"), 0o644); err == nil {
		t.Fatal("WriteFile with an escaping path succeeded")
	}
	if err := fsys.MkdirAll("../escape", 0o755); err == nil {
		t.Fatal("MkdirAll with an escaping path succeeded")
	}
	if err := fsys.Remove("../escape.md"); err == nil {
		t.Fatal("Remove with an escaping path succeeded")
	}
	if err := fsys.Rename("a.md", "../escape.md"); err == nil {
		t.Fatal("Rename to an escaping path succeeded")
	}
	if err := fsys.Rename("../escape.md", "a.md"); err == nil {
		t.Fatal("Rename from an escaping path succeeded")
	}
	if _, err := fsys.Stat("../escape.md"); err == nil {
		t.Fatal("Stat with an escaping path succeeded")
	}

	// A file squatting on a directory name makes MkdirAll fail.
	if err := os.WriteFile(filepath.Join(root, "block"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsys.MkdirAll("block/sub", 0o755); err == nil {
		t.Fatal("MkdirAll under a file succeeded")
	}

	// Listing a missing directory surfaces the error.
	if _, err := fsys.ReadDir("nope"); err == nil {
		t.Fatal("ReadDir of a missing directory succeeded")
	}

	// Stat/Remove/Rename on the wikiFS boundary resolve through SafePath.
	if err := fsys.WriteFile("real.md", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if fi, err := fsys.Stat("real.md"); err != nil || fi.Name() != "real.md" {
		t.Fatalf("Stat = %+v, %v", fi, err)
	}
	if err := fsys.Rename("real.md", "moved.md"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if err := fsys.Remove("moved.md"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

// TestFSBackedToolsBoundAndFollowRoot checks that the FS-backed catalog
// (write_note, edit_file, rename_file, delete_file, list_tree, get_todos)
// stays bounded to the wiki root and follows a root swap, like read/write/list.
func TestFSBackedToolsBoundAndFollowRoot(t *testing.T) {
	rootA, rootB := t.TempDir(), t.TempDir()
	w := wiki.New(rootA)
	reg, err := registry(RegistryOptions{Wiki: w})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	ctx := context.Background()

	get := func(name string) agenttools.Tool {
		t.Helper()
		tl, err := reg.Get(name)
		if err != nil {
			t.Fatalf("Get(%q): %v", name, err)
		}
		return tl
	}

	// write_note scaffolds a note in the live root.
	writeNote := get("write_note")
	if _, err := writeNote.Run(ctx, map[string]any{"path": "meetings/x.md", "title": "X", "body": "b"}); err != nil {
		t.Fatalf("write_note: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootA, "meetings", "x.md")); err != nil {
		t.Fatalf("note missing in old root: %v", err)
	}

	// Escaping paths are rejected across the catalog.
	for _, tl := range []agenttools.Tool{
		writeNote,
		get("read_note"),
		get("edit_file"),
		get("append_file"),
		get("rename_file"),
		get("delete_file"),
	} {
		for _, bad := range []string{"../x.md", "/etc/passwd"} {
			if _, err := tl.Run(ctx, map[string]any{"path": bad}); err == nil {
				t.Fatalf("%s(%q) should be rejected", tl.Name(), bad)
			}
		}
	}

	// rename_file cannot move outside the root.
	rename := get("rename_file")
	if _, err := rename.Run(ctx, map[string]any{"path": "meetings/x.md", "new_path": "../esc.md"}); err == nil {
		t.Fatal("rename_file to an escaping destination succeeded")
	}

	// todos stay under the root and follow the swap.
	todos := get("update_todos")
	if _, err := todos.Run(ctx, map[string]any{"content": "## Now\n- ship"}); err != nil {
		t.Fatalf("update_todos: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootA, "todos", "TODO.md")); err != nil {
		t.Fatalf("TODO.md missing in old root: %v", err)
	}

	// Swap the live root: the same registry must now serve the new root.
	w.SetRoot(rootB)
	if _, err := writeNote.Run(ctx, map[string]any{"path": "meetings/y.md", "title": "Y", "body": "c"}); err != nil {
		t.Fatalf("write_note after swap: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootB, "meetings", "y.md")); err != nil {
		t.Fatalf("note missing in new root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootA, "meetings", "y.md")); err == nil {
		t.Fatal("note written to old root after swap")
	}
	getTodos := get("get_todos")
	out, err := getTodos.Run(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("get_todos after swap: %v", err)
	}
	// The new root has no TODO.md yet — the tool must follow the new root and
	// report its state, not the old root's todos.
	if strings.Contains(out, "Now") {
		t.Fatalf("get_todos after swap returned the old root's todos: %q", out)
	}
}

// TestWikiRegistryShipsCatalog asserts the full FS-backed tool catalog is
// registered by name, so the model can call every tool the issue ships.
func TestWikiRegistryShipsCatalog(t *testing.T) {
	reg, err := registry(RegistryOptions{Wiki: wiki.New(t.TempDir())})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	want := []string{
		"append_file", "delete_file", "edit_file", "file_inbox", "get_inbox",
		"get_time", "get_todos", "grep", "list", "list_recent", "list_tree",
		"read_file", "read_note", "remember", "rename_file", "search_by_tag",
		"update_todos", "write_file", "write_note",
	}
	got := map[string]bool{}
	for _, tl := range reg.List() {
		got[tl.Name()] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("registry missing tool %q", name)
		}
	}
}

func TestWriteNoteProducesWikiParseableNote(t *testing.T) {
	dir := t.TempDir()
	reg, err := registry(RegistryOptions{Wiki: wiki.New(dir)})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	writeNote, err := reg.Get("write_note")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeNote.Run(context.Background(), map[string]any{
		"path":  "meetings/2026-08-21-standup.md",
		"title": "Standup",
		"body":  "body text",
		"tags":  []any{"work"},
	}); err != nil {
		t.Fatalf("write_note: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "meetings", "2026-08-21-standup.md"))
	if err != nil {
		t.Fatalf("read back note: %v", err)
	}
	meta, body, err := wiki.ParseNote(data)
	if err != nil {
		t.Fatalf("wiki.ParseNote: %v", err)
	}
	if meta.Title != "Standup" || meta.Kind != "meeting" {
		t.Fatalf("meta = %+v", meta)
	}
	// FormatNote guarantees a trailing newline, so the round-tripped body ends
	// with one.
	if string(body) != "body text\n" {
		t.Fatalf("body = %q", body)
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

	reg, err := registry(RegistryOptions{Wiki: wiki.New(t.TempDir()), Index: ix})
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

// stubTool is a minimal host-registered custom tool.
type stubTool struct{}

func (stubTool) Name() string { return "custom_tool" }
func (stubTool) Description() string {
	return "A host-registered custom tool."
}
func (stubTool) Schema() map[string]any { return map[string]any{"type": "object"} }
func (stubTool) Run(ctx context.Context, args map[string]any) (string, error) {
	return "custom result", nil
}

// TestRegistryRegistersCustomTools asserts a host can extend the catalog with
// its own tools via RegistryOptions.CustomTools and the Client WithTools
// option.
func TestRegistryRegistersCustomTools(t *testing.T) {
	reg, err := registry(RegistryOptions{
		Wiki:        wiki.New(t.TempDir()),
		CustomTools: []agenttools.Tool{stubTool{}},
	})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if _, err := reg.Get("custom_tool"); err != nil {
		t.Fatalf("custom tool not registered: %v", err)
	}
	out, err := reg.Get("custom_tool")
	if err != nil {
		t.Fatal(err)
	}
	got, err := out.Run(context.Background(), map[string]any{})
	if err != nil || got != "custom result" {
		t.Fatalf("custom tool run = %q, %v", got, err)
	}
}

// TestRegistryRejectsDuplicateCustomTool asserts a custom tool colliding with
// a built-in name fails registration rather than silently overwriting it.
func TestRegistryRejectsDuplicateCustomTool(t *testing.T) {
	_, err := registry(RegistryOptions{
		Wiki:        wiki.New(t.TempDir()),
		CustomTools: []agenttools.Tool{stubTool{}, stubTool{}},
	})
	if err == nil {
		t.Fatal("duplicate custom tool registration succeeded, want error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("duplicate error = %q", err)
	}
}

// TestGitToolsRegisteredWhenConfigured asserts the git tools are registered
// only when Git options carry a RepoPath, so a host that does not wire git
// does not expose them to the model.
func TestGitToolsRegisteredWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	reg, err := registry(RegistryOptions{
		Wiki: wiki.New(dir),
		Git:  agenttools.GitOptions{RepoPath: func() string { return dir }},
	})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	for _, name := range []string{"git_init", "git_status", "git_log", "git_diff", "git_commit", "git_push"} {
		if _, err := reg.Get(name); err != nil {
			t.Fatalf("missing git tool %q: %v", name, err)
		}
	}
	reg, err = registry(RegistryOptions{Wiki: wiki.New(dir)})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	for _, name := range []string{"git_init", "git_status", "git_log", "git_diff", "git_commit", "git_push"} {
		if _, err := reg.Get(name); err == nil {
			t.Fatalf("git tool %q should not be registered without RepoPath", name)
		}
	}
}

// TestWithToolsOption asserts Client.WithTools surfaces custom tools on the
// turn registry.
func TestWithToolsOption(t *testing.T) {
	prov := &fakeProvider{stream: &fakeStream{deltas: []agentlib.Delta{agentlib.StopDelta("end_turn")}}}
	c, err := New("claude-test", "key", wiki.New(t.TempDir()), openStore(t), nil,
		WithProvider(prov),
		WithTools(stubTool{}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.tools.Get("custom_tool"); err != nil {
		t.Fatalf("custom tool missing from client registry: %v", err)
	}
}
