package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxReadBytes is the default cap on read_file output: the tool returns only
// the first maxReadBytes bytes of an oversized file, with a truncation marker.
const maxReadBytes = 128 * 1024

// truncationMarker reports the marker appended when read_file returns only the
// head of a larger file.
func truncationMarker(maxBytes int) string {
	return fmt.Sprintf("\n\n[output truncated: file exceeds %d bytes]", maxBytes)
}

// FS is the minimal filesystem surface the file tools need. The default
// implementation (OSFS) binds every relative path to a root directory and
// rejects anything that escapes it; hosts inject their own implementation to
// add further constraints (e.g. a wiki SafePath boundary).
type FS interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(name string) ([]fs.DirEntry, error)
	Stat(name string) (fs.FileInfo, error)
	Remove(name string) error
	Rename(oldPath, newPath string) error
}

// OSFS is the default FS. Every operation resolves its relative name against
// a fixed root and fails with a clear error when the name is absolute,
// contains ".." segments, or resolves through a symlink to somewhere outside
// root.
type OSFS struct {
	root string // canonical (symlink-free) form of the injected root
}

// NewOSFS returns an OSFS bound to root. Root is canonicalized so symlink
// comparisons are exact; it must exist.
func NewOSFS(root string) (*OSFS, error) {
	if root == "" {
		return nil, errors.New("tools: root must not be empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("tools: resolve root: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("tools: resolve root: %w", err)
	}
	return &OSFS{root: canonical}, nil
}

// Root returns the canonical root directory the OSFS is bound to.
func (f *OSFS) Root() string { return f.root }

// resolve validates that name is a relative path staying inside root and
// returns the absolute path to operate on. It rejects absolute paths and ".."
// segments syntactically, then follows symlinks — resolving the deepest
// existing ancestor when the final segment does not exist yet — and refuses
// any path whose real target lies outside root.
func (f *OSFS) resolve(name string) (string, error) {
	clean, err := cleanRel(name)
	if err != nil {
		return "", err
	}
	full := filepath.Join(f.root, clean)
	if err := f.bounded(full); err != nil {
		return "", err
	}
	return full, nil
}

// bounded reports whether full (or, when it does not exist, its deepest
// existing ancestor) resolves to a real location inside root.
func (f *OSFS) bounded(full string) error {
	target := full
	for {
		resolved, err := filepath.EvalSymlinks(target)
		if err == nil {
			if !f.within(resolved) {
				return fmt.Errorf("tools: %q resolves outside the root via a symlink", filepath.ToSlash(full))
			}
			return nil
		}
		parent := filepath.Dir(target)
		if parent == target {
			return fmt.Errorf("tools: cannot resolve %q: %w", filepath.ToSlash(full), err)
		}
		target = parent
	}
}

// within reports whether resolved points at root or at something beneath it.
func (f *OSFS) within(resolved string) bool {
	rel, err := filepath.Rel(f.root, resolved)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// cleanRel validates that name is a non-empty relative path with no ".."
// segments.
func cleanRel(name string) (string, error) {
	if name == "" {
		return "", errors.New("tools: path must not be empty")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("tools: absolute path %q not allowed", name)
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("tools: path %q escapes the root", name)
	}
	return clean, nil
}

// ReadFile implements FS.
func (f *OSFS) ReadFile(name string) ([]byte, error) {
	full, err := f.resolve(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("tools: read %q: %w", name, err)
	}
	return data, nil
}

// WriteFile implements FS. The write is atomic: data is written to a temp file
// in the target directory and renamed into place, so a failed write never
// leaves a partial file at name.
func (f *OSFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	full, err := f.resolve(name)
	if err != nil {
		return err
	}
	if err := AtomicWrite(full, data, perm); err != nil {
		return fmt.Errorf("tools: write %q: %w", name, err)
	}
	return nil
}

// AtomicWrite writes data to path via a temp file in the target directory and
// a rename, so a failed write never leaves a partial file at path. path must
// already be resolved and bounded by the caller.
func AtomicWrite(path string, data []byte, perm fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".thoth-write-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		defer func() { _ = tmp.Close() }()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		defer func() { _ = tmp.Close() }()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename into place: %w", err)
	}
	return nil
}

// MkdirAll implements FS.
func (f *OSFS) MkdirAll(path string, perm fs.FileMode) error {
	full, err := f.resolve(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(full, perm); err != nil {
		return fmt.Errorf("tools: mkdir %q: %w", path, err)
	}
	return nil
}

// ReadDir implements FS.
func (f *OSFS) ReadDir(name string) ([]fs.DirEntry, error) {
	full, err := f.resolve(name)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, fmt.Errorf("tools: list %q: %w", name, err)
	}
	return entries, nil
}

// Stat implements FS.
func (f *OSFS) Stat(name string) (fs.FileInfo, error) {
	full, err := f.resolve(name)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(full)
	if err != nil {
		return nil, fmt.Errorf("tools: stat %q: %w", name, err)
	}
	return fi, nil
}

// Remove implements FS. Removing a symlink removes the link itself, never
// the target, because resolve bounds the link name rather than following it
// for deletion.
func (f *OSFS) Remove(name string) error {
	full, err := f.resolve(name)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil {
		return fmt.Errorf("tools: remove %q: %w", name, err)
	}
	return nil
}

// Rename implements FS. Both names resolve within the root, so a move can
// never leave the root or cross into it from outside.
func (f *OSFS) Rename(oldPath, newPath string) error {
	oldFull, err := f.resolve(oldPath)
	if err != nil {
		return err
	}
	newFull, err := f.resolve(newPath)
	if err != nil {
		return err
	}
	if err := os.Rename(oldFull, newFull); err != nil {
		return fmt.Errorf("tools: rename %q: %w", oldPath, err)
	}
	return nil
}

// ReadFile is the "read_file" tool: it returns a file's contents as text,
// capped at a maximum byte size with an explicit truncation marker.
type ReadFile struct {
	fs       FS
	maxBytes int
}

// NewReadFile returns the read_file tool backed by fs. A non-positive maxBytes
// falls back to the 128 KiB default cap.
func NewReadFile(fs FS, maxBytes int) *ReadFile {
	if maxBytes <= 0 {
		maxBytes = maxReadBytes
	}
	return &ReadFile{fs: fs, maxBytes: maxBytes}
}

// Name implements Tool.
func (t *ReadFile) Name() string { return "read_file" }

// Description implements Tool.
func (t *ReadFile) Description() string {
	return "Read the contents of a file as text. Path is relative to the workspace root."
}

// Schema implements Tool.
func (t *ReadFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the file to read.",
			},
		},
		"required": []string{"path"},
	}
}

// Run implements Tool.
func (t *ReadFile) Run(ctx context.Context, args map[string]any) (string, error) {
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
		return "", fmt.Errorf("read_file: %w", err)
	}
	if len(data) > t.maxBytes {
		return string(data[:t.maxBytes]) + truncationMarker(t.maxBytes), nil
	}
	return string(data), nil
}

// WriteFile is the "write_file" tool: it writes content to a file, creating
// parent directories as needed. The write is atomic, so a failure leaves no
// partial file.
type WriteFile struct {
	fs FS
}

// NewWriteFile returns the write_file tool backed by fs.
func NewWriteFile(fs FS) *WriteFile { return &WriteFile{fs: fs} }

// Name implements Tool.
func (t *WriteFile) Name() string { return "write_file" }

// Description implements Tool.
func (t *WriteFile) Description() string {
	return "Write content to a file, creating parent directories as needed. Path is relative to the workspace root."
}

// Schema implements Tool.
func (t *WriteFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the file to write.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Full contents to write to the file.",
			},
		},
		"required": []string{"path", "content"},
	}
}

// Run implements Tool.
func (t *WriteFile) Run(ctx context.Context, args map[string]any) (string, error) {
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
	if dir := filepath.Dir(rel); dir != "." {
		if err := t.fs.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("write_file: %w", err)
		}
	}
	if err := t.fs.WriteFile(rel, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write_file: %w", err)
	}
	return fmt.Sprintf("wrote %s", rel), nil
}

// List is the "list" tool: it lists a directory relative to the injected root,
// one entry per line, sorted lexicographically, with each name flagged as a
// directory or a file.
type List struct {
	fs FS
}

// NewList returns the list tool backed by fs.
func NewList(fs FS) *List { return &List{fs: fs} }

// Name implements Tool.
func (t *List) Name() string { return "list" }

// Description implements Tool.
func (t *List) Description() string {
	return "List the entries of a directory. Path is relative to the workspace root."
}

// Schema implements Tool.
func (t *List) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path of the directory to list. Defaults to the workspace root.",
			},
		},
	}
}

// Run implements Tool.
func (t *List) Run(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := stringArgDefault(args, "path", ".")
	if err != nil {
		return "", err
	}
	rel, err := cleanRel(path)
	if err != nil {
		return "", err
	}
	entries, err := t.fs.ReadDir(rel)
	if err != nil {
		return "", fmt.Errorf("list: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var sb strings.Builder
	for _, e := range entries {
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		sb.WriteString(e.Name())
		sb.WriteByte('\t')
		sb.WriteString(kind)
		sb.WriteByte('\n')
	}
	return strings.TrimSuffix(sb.String(), "\n"), nil
}

// stringArg returns the string value of key in args, erroring when it is
// absent or not a string.
func stringArg(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("tools: missing argument %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("tools: argument %q must be a string", key)
	}
	return s, nil
}

// stringArgDefault returns the string value of key in args, or def when it is
// absent.
func stringArgDefault(args map[string]any, key, def string) (string, error) {
	if _, ok := args[key]; !ok {
		return def, nil
	}
	return stringArg(args, key)
}

// stringSliceArg returns the value of key in args as a slice of strings,
// accepting either a []string or a []any of strings (how JSON-decoded args
// arrive). A missing key returns nil without error; a wrong type errors.
func stringSliceArg(args map[string]any, key string) ([]string, error) {
	v, ok := args[key]
	if !ok {
		return nil, nil
	}
	switch t := v.(type) {
	case []string:
		return t, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("tools: argument %q must be an array of strings", key)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("tools: argument %q must be an array of strings", key)
	}
}

// intArg returns the integer value of key in args. JSON-decoded numbers arrive
// as float64; a missing key or non-numeric value errors.
func intArg(args map[string]any, key string) (int, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("tools: missing argument %q", key)
	}
	switch t := v.(type) {
	case float64:
		return int(t), nil
	case int:
		return t, nil
	default:
		return 0, fmt.Errorf("tools: argument %q must be a number", key)
	}
}

// intArgDefault returns the integer value of key in args, or def when absent.
func intArgDefault(args map[string]any, key string, def int) (int, error) {
	if _, ok := args[key]; !ok {
		return def, nil
	}
	return intArg(args, key)
}
