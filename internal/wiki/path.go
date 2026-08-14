package wiki

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SafePath joins rel onto root and rejects anything that escapes root
// (.., absolute paths, symlink tricks are handled by callers via root checks).
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
	return full, nil
}
