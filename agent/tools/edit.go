package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// EditFile is the "edit_file" tool: it replaces a single occurrence of
// old_text with new_text in a file. It refuses ambiguous edits — zero or
// multiple occurrences of old_text both error.
type EditFile struct {
	fs FS
}

// NewEditFile returns the edit_file tool backed by fs.
func NewEditFile(fs FS) *EditFile { return &EditFile{fs: fs} }

// Name implements Tool.
func (t *EditFile) Name() string { return "edit_file" }

// Description implements Tool.
func (t *EditFile) Description() string {
	return "Replace a single occurrence of old_text with new_text in a file. Errors when old_text is absent or appears multiple times. Path is relative to the wiki root."
}

// Schema implements Tool.
func (t *EditFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the file to edit.",
			},
			"old_text": map[string]any{
				"type":        "string",
				"description": "The exact text to replace (must occur exactly once).",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "The replacement text.",
			},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}

// Run implements Tool.
func (t *EditFile) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := stringArg(args, "path")
	if err != nil {
		return "", err
	}
	oldText, err := stringArg(args, "old_text")
	if err != nil {
		return "", err
	}
	newText, err := stringArg(args, "new_text")
	if err != nil {
		return "", err
	}
	if oldText == "" {
		return "", errors.New("edit_file: old_text must not be empty")
	}
	rel, err := cleanRel(path)
	if err != nil {
		return "", err
	}
	data, err := t.fs.ReadFile(rel)
	if err != nil {
		return "", fmt.Errorf("edit_file: %w", err)
	}
	content := string(data)
	count := strings.Count(content, oldText)
	switch count {
	case 0:
		return "", fmt.Errorf("edit_file: old_text not found in %s", rel)
	case 1:
	default:
		return "", fmt.Errorf("edit_file: old_text occurs %d times in %s; refusing ambiguous edit", count, rel)
	}
	content = strings.Replace(content, oldText, newText, 1)
	if err := t.fs.WriteFile(rel, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("edit_file: %w", err)
	}
	return fmt.Sprintf("edited %s", rel), nil
}

// AppendFile is the "append_file" tool: it appends content to a file, creating
// the file (and its parent directories) when absent.
type AppendFile struct {
	fs FS
}

// NewAppendFile returns the append_file tool backed by fs.
func NewAppendFile(fs FS) *AppendFile { return &AppendFile{fs: fs} }

// Name implements Tool.
func (t *AppendFile) Name() string { return "append_file" }

// Description implements Tool.
func (t *AppendFile) Description() string {
	return "Append content to a file, creating it if it does not exist. Path is relative to the wiki root."
}

// Schema implements Tool.
func (t *AppendFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the file to append to.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to append.",
			},
		},
		"required": []string{"path", "content"},
	}
}

// Run implements Tool.
func (t *AppendFile) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := stringArg(args, "path")
	if err != nil {
		return "", err
	}
	content, err := stringArg(args, "content")
	if err != nil {
		return "", err
	}
	rel, err := cleanRel(path)
	if err != nil {
		return "", err
	}
	var data []byte
	existing, err := t.fs.ReadFile(rel)
	switch {
	case err == nil:
		data = existing
	case errors.Is(err, fs.ErrNotExist):
		data = nil
	default:
		return "", fmt.Errorf("append_file: %w", err)
	}
	data = append(data, []byte(content)...)
	if dir := filepath.Dir(rel); dir != "." {
		if err := t.fs.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("append_file: %w", err)
		}
	}
	if err := t.fs.WriteFile(rel, data, 0o644); err != nil {
		return "", fmt.Errorf("append_file: %w", err)
	}
	return fmt.Sprintf("appended to %s", rel), nil
}

// RenameFile is the "rename_file" tool: it moves a file (or directory) within
// the root. Both the source and destination are bounded to the root.
type RenameFile struct {
	fs FS
}

// NewRenameFile returns the rename_file tool backed by fs.
func NewRenameFile(fs FS) *RenameFile { return &RenameFile{fs: fs} }

// Name implements Tool.
func (t *RenameFile) Name() string { return "rename_file" }

// Description implements Tool.
func (t *RenameFile) Description() string {
	return "Move a file or directory to a new path within the wiki. Both paths are relative to the wiki root."
}

// Schema implements Tool.
func (t *RenameFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the file or directory to move.",
			},
			"new_path": map[string]any{
				"type":        "string",
				"description": "Relative destination path.",
			},
		},
		"required": []string{"path", "new_path"},
	}
}

// Run implements Tool.
func (t *RenameFile) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := stringArg(args, "path")
	if err != nil {
		return "", err
	}
	newPath, err := stringArg(args, "new_path")
	if err != nil {
		return "", err
	}
	rel, err := cleanRel(path)
	if err != nil {
		return "", err
	}
	newRel, err := cleanRel(newPath)
	if err != nil {
		return "", err
	}
	if dir := filepath.Dir(newRel); dir != "." {
		if err := t.fs.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("rename_file: %w", err)
		}
	}
	if err := t.fs.Rename(rel, newRel); err != nil {
		return "", fmt.Errorf("rename_file: %w", err)
	}
	return fmt.Sprintf("moved %s to %s", rel, newRel), nil
}

// DeleteFile is the "delete_file" tool: it removes a file from the wiki.
type DeleteFile struct {
	fs FS
}

// NewDeleteFile returns the delete_file tool backed by fs.
func NewDeleteFile(fs FS) *DeleteFile { return &DeleteFile{fs: fs} }

// Name implements Tool.
func (t *DeleteFile) Name() string { return "delete_file" }

// Description implements Tool.
func (t *DeleteFile) Description() string {
	return "Delete a file. Path is relative to the wiki root."
}

// Schema implements Tool.
func (t *DeleteFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the file to delete.",
			},
		},
		"required": []string{"path"},
	}
}

// Run implements Tool.
func (t *DeleteFile) Run(ctx context.Context, args map[string]any) (string, error) {
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
	if err := t.fs.Remove(rel); err != nil {
		return "", fmt.Errorf("delete_file: %w", err)
	}
	return fmt.Sprintf("deleted %s", rel), nil
}
