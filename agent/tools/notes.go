package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// singularNoteType mirrors the wiki's mechanical rule (internal/wiki.noteType):
// a folder's note type is its name with a trailing "s" trimmed (meetings ->
// meeting). It is the default `type:` for notes written to that folder.
func singularNoteType(folder string) string {
	return strings.TrimSuffix(folder, "s")
}

// noteTypeFor derives the default frontmatter type for a note path: the
// top-level folder's singularized name (meetings/x.md -> meeting). A path at
// the root has no folder, so its type stays empty.
func noteTypeFor(rel string) string {
	top := rel
	if i := strings.IndexByte(rel, '/'); i >= 0 {
		top = rel[:i]
	}
	return singularNoteType(top)
}

// WriteNote is the "write_note" tool: it scaffolds a note's YAML frontmatter
// from title, date, tags and type, then writes the note atomically, creating
// parent directories as needed. Notes written here parse in the wiki.
type WriteNote struct {
	fs  FS
	now func() time.Time
}

// NewWriteNote returns the write_note tool backed by fs. now supplies the
// note date; nil falls back to time.Now (tests inject a fixed clock).
func NewWriteNote(fs FS, now func() time.Time) *WriteNote {
	if now == nil {
		now = time.Now
	}
	return &WriteNote{fs: fs, now: now}
}

// Name implements Tool.
func (t *WriteNote) Name() string { return "write_note" }

// Description implements Tool.
func (t *WriteNote) Description() string {
	return "Write a new wiki note with YAML frontmatter (title, date, tags, type). Path is relative to the wiki root; must end in .md or .markdown."
}

// Schema implements Tool.
func (t *WriteNote) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the note to write (e.g. meetings/2026-01-02-x.md).",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "The note title (required; becomes the frontmatter title).",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "The note body in markdown.",
			},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional frontmatter tags.",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Optional frontmatter type. Defaults to the top-level folder's singularized name.",
			},
		},
		"required": []string{"path", "title", "body"},
	}
}

// Run implements Tool.
func (t *WriteNote) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := stringArg(args, "path")
	if err != nil {
		return "", err
	}
	title, err := stringArg(args, "title")
	if err != nil {
		return "", err
	}
	body, err := stringArg(args, "body")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("write_note: title must not be empty")
	}
	rel, err := cleanRel(path)
	if err != nil {
		return "", err
	}
	if !HasMarkdownExt(rel) {
		return "", fmt.Errorf("write_note: path %q must end in .md or .markdown", rel)
	}
	typ, err := stringArgDefault(args, "type", "")
	if err != nil {
		return "", err
	}
	if typ == "" {
		typ = noteTypeFor(rel)
	}
	tags, err := stringSliceArg(args, "tags")
	if err != nil {
		return "", err
	}
	note := Note{
		Title: title,
		Date:  t.now().Format("2006-01-02"),
		Type:  typ,
		Tags:  tags,
	}
	content := FormatNote(note, body)
	if dir := filepath.Dir(rel); dir != "." {
		if err := t.fs.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("write_note: %w", err)
		}
	}
	if err := t.fs.WriteFile(rel, content, 0o644); err != nil {
		return "", fmt.Errorf("write_note: %w", err)
	}
	return fmt.Sprintf("wrote note %s", rel), nil
}

// ReadNote is the "read_note" tool: it reads a note and returns its parsed
// frontmatter metadata plus body — metadata as readable fields, not raw YAML.
// Output is size-capped like read_file.
type ReadNote struct {
	fs       FS
	maxBytes int
}

// NewReadNote returns the read_note tool backed by fs. A non-positive maxBytes
// falls back to the default read cap.
func NewReadNote(fs FS, maxBytes int) *ReadNote {
	if maxBytes <= 0 {
		maxBytes = maxReadBytes
	}
	return &ReadNote{fs: fs, maxBytes: maxBytes}
}

// Name implements Tool.
func (t *ReadNote) Name() string { return "read_note" }

// Description implements Tool.
func (t *ReadNote) Description() string {
	return "Read a wiki note: its title, date, type and tags, followed by its body. Path is relative to the wiki root."
}

// Schema implements Tool.
func (t *ReadNote) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the note to read.",
			},
		},
		"required": []string{"path"},
	}
}

// Run implements Tool.
func (t *ReadNote) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := stringArg(args, "path")
	if err != nil {
		return "", err
	}
	rel, err := cleanRel(path)
	if err != nil {
		return "", err
	}
	data, err := t.fs.ReadFile(rel)
	if err != nil {
		return "", fmt.Errorf("read_note: %w", err)
	}
	note, body, err := ParseNote(data)
	if err != nil {
		return "", fmt.Errorf("read_note: %w", err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "title: %s\n", note.Title)
	if note.Date != "" {
		fmt.Fprintf(&sb, "date: %s\n", note.Date)
	}
	if note.Type != "" {
		fmt.Fprintf(&sb, "type: %s\n", note.Type)
	}
	if len(note.Tags) > 0 {
		fmt.Fprintf(&sb, "tags: [%s]\n", strings.Join(note.Tags, ", "))
	}
	sb.WriteString("---\n")
	sb.Write(body)
	if len(body) > 0 && body[len(body)-1] != '\n' {
		sb.WriteByte('\n')
	}
	out := sb.String()
	if len(out) > t.maxBytes {
		return out[:t.maxBytes] + truncationMarker(t.maxBytes), nil
	}
	return out, nil
}
