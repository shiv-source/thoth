package wiki

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SafePath joins rel onto root and rejects anything that escapes root
// syntactically (.., absolute paths) or through a symlink: the path's
// deepest existing ancestor is resolved with EvalSymlinks and must land on
// root or beneath it. A rel naming a not-yet-created file still resolves via
// its deepest existing ancestor, so writes are bound too.
func SafePath(root, rel string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("path escapes wiki root: %q", rel)
	}
	full := filepath.Join(root, cleaned)
	check, err := filepath.Rel(root, full)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", rel, err)
	}
	if check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes wiki root: %q", rel)
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	if err := boundedSymlink(rootReal, full); err != nil {
		return "", err
	}
	return full, nil
}

// boundedSymlink reports whether full (or, when it does not exist yet, its
// deepest existing ancestor) resolves to a real location on or under the
// canonical root. It walks up the path so a fresh file whose parent chain
// exists — or must still be created — is checked against the nearest real
// directory.
func boundedSymlink(rootReal, full string) error {
	target := full
	for {
		resolved, err := filepath.EvalSymlinks(target)
		if err == nil {
			rel, err := filepath.Rel(rootReal, resolved)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil
			}
			return fmt.Errorf("path escapes wiki root via symlink: %q", filepath.ToSlash(full))
		}
		parent := filepath.Dir(target)
		if parent == target {
			return fmt.Errorf("resolve path %q: %w", filepath.ToSlash(full), err)
		}
		target = parent
	}
}
