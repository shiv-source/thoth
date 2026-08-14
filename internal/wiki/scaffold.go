package wiki

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var folders = []string{
	"inbox", "meetings", "projects", "links", "setup", "knowledge", "todos", "daily",
}

// Scaffold creates the wiki folder skeleton and CLAUDE.md under dir.
// It never overwrites an existing CLAUDE.md.
func Scaffold(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}
	for _, f := range folders {
		p := filepath.Join(dir, f)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("scaffold: %w", err)
		}
		// .gitkeep so empty folders survive the wiki's git repo.
		if _, err := os.Stat(filepath.Join(p, ".gitkeep")); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(filepath.Join(p, ".gitkeep"), nil, 0o644); err != nil {
				return fmt.Errorf("scaffold: %w", err)
			}
		}
	}
	rp := filepath.Join(dir, "CLAUDE.md")
	if _, err := os.Stat(rp); err == nil {
		return nil // respect an existing rulebook
	}
	if err := os.WriteFile(rp, []byte(Rulebook()), 0o644); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}
	return nil
}
