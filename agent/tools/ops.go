package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DefaultTodosPath is the scaffolded location of the wiki's TODO list.
const DefaultTodosPath = "todos/TODO.md"

// DefaultInboxDir is the scaffolded inbox folder.
const DefaultInboxDir = "inbox"

// DefaultMemoryPath is the default memory note for the remember tool.
const DefaultMemoryPath = "knowledge/memory.md"

// GetTodos is the "get_todos" tool: it returns the wiki's TODO list
// (todos/TODO.md by default), or a clear "none yet" message when absent.
type GetTodos struct {
	fs   FS
	path string
}

// NewGetTodos returns the get_todos tool backed by fs, reading the TODO list
// at path (defaults to todos/TODO.md).
func NewGetTodos(fs FS, path string) *GetTodos {
	if path == "" {
		path = DefaultTodosPath
	}
	return &GetTodos{fs: fs, path: path}
}

// Name implements Tool.
func (t *GetTodos) Name() string { return "get_todos" }

// Description implements Tool.
func (t *GetTodos) Description() string {
	return "Read the wiki's TODO list (todos/TODO.md by default)."
}

// Schema implements Tool.
func (t *GetTodos) Schema() map[string]any {
	return map[string]any{"type": "object"}
}

// Run implements Tool.
func (t *GetTodos) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	data, err := t.fs.ReadFile(t.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Sprintf("no TODO list at %s yet", t.path), nil
		}
		return "", fmt.Errorf("get_todos: %w", err)
	}
	return string(data), nil
}

// UpdateTodos is the "update_todos" tool: it writes the full TODO list
// atomically, creating parent directories as needed.
type UpdateTodos struct {
	fs   FS
	path string
}

// NewUpdateTodos returns the update_todos tool backed by fs, writing the TODO
// list at path (defaults to todos/TODO.md).
func NewUpdateTodos(fs FS, path string) *UpdateTodos {
	if path == "" {
		path = DefaultTodosPath
	}
	return &UpdateTodos{fs: fs, path: path}
}

// Name implements Tool.
func (t *UpdateTodos) Name() string { return "update_todos" }

// Description implements Tool.
func (t *UpdateTodos) Description() string {
	return "Replace the entire TODO list (todos/TODO.md by default) with content."
}

// Schema implements Tool.
func (t *UpdateTodos) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The full content of the TODO list.",
			},
		},
		"required": []string{"content"},
	}
}

// Run implements Tool.
func (t *UpdateTodos) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	content, err := stringArg(args, "content")
	if err != nil {
		return "", err
	}
	rel, err := cleanRel(t.path)
	if err != nil {
		return "", err
	}
	if dir := filepath.Dir(rel); dir != "." {
		if err := t.fs.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("update_todos: %w", err)
		}
	}
	if err := t.fs.WriteFile(rel, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("update_todos: %w", err)
	}
	return fmt.Sprintf("updated %s", rel), nil
}

// GetInbox is the "get_inbox" tool: it lists the entries of the inbox folder.
type GetInbox struct {
	fs  FS
	dir string
}

// NewGetInbox returns the get_inbox tool backed by fs, listing dir (defaults
// to inbox).
func NewGetInbox(fs FS, dir string) *GetInbox {
	if dir == "" {
		dir = DefaultInboxDir
	}
	return &GetInbox{fs: fs, dir: dir}
}

// Name implements Tool.
func (t *GetInbox) Name() string { return "get_inbox" }

// Description implements Tool.
func (t *GetInbox) Description() string {
	return "List the entries in the inbox folder (inbox/ by default)."
}

// Schema implements Tool.
func (t *GetInbox) Schema() map[string]any {
	return map[string]any{"type": "object"}
}

// Run implements Tool.
func (t *GetInbox) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	entries, err := t.fs.ReadDir(t.dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Sprintf("no inbox at %s yet", t.dir), nil
		}
		return "", fmt.Errorf("get_inbox: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return strings.Join(names, "\n"), nil
}

// FileInbox is the "file_inbox" tool: it moves an entry from the inbox into a
// folder, keeping the filename. Both paths are bounded to the wiki root.
type FileInbox struct {
	fs  FS
	dir string
}

// NewFileInbox returns the file_inbox tool backed by fs, moving out of dir
// (defaults to inbox).
func NewFileInbox(fs FS, dir string) *FileInbox {
	if dir == "" {
		dir = DefaultInboxDir
	}
	return &FileInbox{fs: fs, dir: dir}
}

// Name implements Tool.
func (t *FileInbox) Name() string { return "file_inbox" }

// Description implements Tool.
func (t *FileInbox) Description() string {
	return "Move an inbox entry into a folder, keeping its filename. Both the entry name and the destination folder are relative to the wiki root."
}

// Schema implements Tool.
func (t *FileInbox) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The inbox entry to move (relative to the inbox folder).",
			},
			"dest": map[string]any{
				"type":        "string",
				"description": "The destination folder (relative to the wiki root).",
			},
		},
		"required": []string{"path", "dest"},
	}
}

// Run implements Tool.
func (t *FileInbox) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := stringArg(args, "path")
	if err != nil {
		return "", err
	}
	dest, err := stringArg(args, "dest")
	if err != nil {
		return "", err
	}
	if path == "" || filepath.Base(path) != path {
		return "", errors.New("file_inbox: path must be a single inbox entry name")
	}
	src, err := cleanRel(filepath.Join(t.dir, path))
	if err != nil {
		return "", err
	}
	dst, err := cleanRel(filepath.Join(dest, path))
	if err != nil {
		return "", err
	}
	if dir := filepath.Dir(dst); dir != "." {
		if err := t.fs.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("file_inbox: %w", err)
		}
	}
	if err := t.fs.Rename(src, dst); err != nil {
		return "", fmt.Errorf("file_inbox: %w", err)
	}
	return fmt.Sprintf("filed %s to %s", src, dst), nil
}

// Remember is the "remember" tool: it appends a timestamped fact to a memory
// note (knowledge/memory.md by default), creating it when absent.
type Remember struct {
	fs   FS
	path string
	now  func() time.Time
}

// NewRemember returns the remember tool backed by fs, appending to the memory
// note at path (defaults to knowledge/memory.md). now supplies the timestamp;
// nil falls back to time.Now.
func NewRemember(fs FS, path string, now func() time.Time) *Remember {
	if path == "" {
		path = DefaultMemoryPath
	}
	if now == nil {
		now = time.Now
	}
	return &Remember{fs: fs, path: path, now: now}
}

// Name implements Tool.
func (t *Remember) Name() string { return "remember" }

// Description implements Tool.
func (t *Remember) Description() string {
	return "Append a timestamped fact to the memory note (knowledge/memory.md by default), creating it when absent."
}

// Schema implements Tool.
func (t *Remember) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fact": map[string]any{
				"type":        "string",
				"description": "The fact to remember.",
			},
		},
		"required": []string{"fact"},
	}
}

// Run implements Tool.
func (t *Remember) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	fact, err := stringArg(args, "fact")
	if err != nil {
		return "", err
	}
	rel, err := cleanRel(t.path)
	if err != nil {
		return "", err
	}
	line := fmt.Sprintf("- %s %s\n", t.now().Format(time.RFC3339), fact)
	var data []byte
	existing, err := t.fs.ReadFile(rel)
	switch {
	case err == nil:
		data = existing
	case errors.Is(err, fs.ErrNotExist):
		// A fresh memory note gets frontmatter so it parses in the wiki.
		note := Note{Title: "Memory", Date: t.now().Format("2006-01-02")}
		data = FormatNote(note, "")
	default:
		return "", fmt.Errorf("remember: %w", err)
	}
	data = append(data, []byte(line)...)
	if dir := filepath.Dir(rel); dir != "." {
		if err := t.fs.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("remember: %w", err)
		}
	}
	if err := t.fs.WriteFile(rel, data, 0o644); err != nil {
		return "", fmt.Errorf("remember: %w", err)
	}
	return fmt.Sprintf("remembered to %s", rel), nil
}
