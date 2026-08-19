package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The server is localhost-only with a fixed address; every user-facing
// setting lives in the settings table (internal/settings).
const (
	DefaultHost = "127.0.0.1"
	DefaultPort = 8333
	// DevPort is what serve --dev binds (make dev); it leaves DefaultPort
	// free for a running instance.
	DevPort = 8334
)

// ExpandHome expands a leading ~/ (or bare ~) to the user's home directory.
func ExpandHome(p string) (string, error) {
	if p == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand home: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand home: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(p, "~/")), nil
	}
	return p, nil
}

// ToTilde is ExpandHome's inverse for storage: a path under the home
// directory becomes ~-prefixed (the settings table's display convention);
// anything else passes through unchanged.
func ToTilde(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(filepath.Separator)) {
		return filepath.Join("~", strings.TrimPrefix(p, home))
	}
	return p
}
