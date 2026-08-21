package tools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/wiki"
)

// WriteNote is the "write_note" tool: it scaffolds a note's YAML frontmatter
// from title, date, tags and type, then writes the note atomically, creating
// parent directories as needed. Notes written here parse in the wiki — the
// frontmatter is rendered by wiki.FormatNote, the single source for the note
// format.
type WriteNote struct {
	fs  agenttools.FS
	now func() time.Time
}

// NewWriteNote returns the write_note tool backed by fs. now supplies the note
// date; nil falls back to time.Now (tests inject a fixed clock).
func NewWriteNote(fs agenttools.FS, now func() time.Time) *WriteNote {
	if now == nil {
		now = time.Now
	}
	return &WriteNote{fs: fs, now: now}
}

// Name implements agenttools.Tool.
func (t *WriteNote) Name() string { return "write_note" }

// Description implements agenttools.Tool.
func (t *WriteNote) Description() string {
	return "Write a new wiki note with YAML frontmatter (title, date, tags, type). Path is relative to the wiki root; must end in .md or .markdown."
}

// Schema implements agenttools.Tool.
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
				"description": "Optional frontmatter type. Defaults to the top-level folder's type per the wiki rule.",
			},
		},
		"required": []string{"path", "title", "body"},
	}
}

// Run implements agenttools.Tool.
func (t *WriteNote) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := agenttools.StringArg(args, "path")
	if err != nil {
		return "", err
	}
	title, err := agenttools.StringArg(args, "title")
	if err != nil {
		return "", err
	}
	body, err := agenttools.StringArg(args, "body")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(title) == "" {
		return "", errors.New("write_note: title must not be empty")
	}
	rel, err := agenttools.CleanRel(path)
	if err != nil {
		return "", err
	}
	if !wiki.IsMarkdownPath(rel) {
		return "", fmt.Errorf("write_note: path %q must end in .md or .markdown", rel)
	}
	typ, err := agenttools.StringArgDefault(args, "type", "")
	if err != nil {
		return "", err
	}
	if typ == "" {
		typ = noteTypeFor(rel)
	}
	tags, err := agenttools.StringSliceArg(args, "tags")
	if err != nil {
		return "", err
	}
	meta := wiki.NoteMeta{
		Title: title,
		Date:  t.now().Format("2006-01-02"),
		Kind:  typ,
		Tags:  tags,
	}
	content := wiki.FormatNote(meta, body)
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
	fs       agenttools.FS
	maxBytes int
}

// NewReadNote returns the read_note tool backed by fs. A non-positive maxBytes
// falls back to the default read cap.
func NewReadNote(fs agenttools.FS, maxBytes int) *ReadNote {
	if maxBytes <= 0 {
		maxBytes = agenttools.MaxReadBytes
	}
	return &ReadNote{fs: fs, maxBytes: maxBytes}
}

// Name implements agenttools.Tool.
func (t *ReadNote) Name() string { return "read_note" }

// Description implements agenttools.Tool.
func (t *ReadNote) Description() string {
	return "Read a wiki note: its title, date, type and tags, followed by its body. Path is relative to the wiki root."
}

// Schema implements agenttools.Tool.
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

// Run implements agenttools.Tool.
func (t *ReadNote) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := agenttools.StringArg(args, "path")
	if err != nil {
		return "", err
	}
	rel, err := agenttools.CleanRel(path)
	if err != nil {
		return "", err
	}
	data, err := t.fs.ReadFile(rel)
	if err != nil {
		return "", fmt.Errorf("read_note: %w", err)
	}
	meta, body, err := wiki.ParseNote(data)
	if err != nil {
		return "", fmt.Errorf("read_note: %w", err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "title: %s\n", meta.Title)
	if meta.Date != "" {
		fmt.Fprintf(&sb, "date: %s\n", meta.Date)
	}
	if meta.Kind != "" {
		fmt.Fprintf(&sb, "type: %s\n", meta.Kind)
	}
	if len(meta.Tags) > 0 {
		fmt.Fprintf(&sb, "tags: [%s]\n", strings.Join(meta.Tags, ", "))
	}
	sb.WriteString("---\n")
	sb.Write(body)
	if len(body) > 0 && body[len(body)-1] != '\n' {
		sb.WriteByte('\n')
	}
	out := sb.String()
	if len(out) > t.maxBytes {
		return out[:t.maxBytes] + agenttools.TruncationMarker(t.maxBytes), nil
	}
	return out, nil
}
