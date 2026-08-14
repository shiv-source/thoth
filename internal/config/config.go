package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	WikiPath       string `toml:"wiki_path" json:"wiki_path"`
	Host           string `toml:"host" json:"host"`
	Port           int    `toml:"port" json:"port"`
	ClaudeBin      string `toml:"claude_bin" json:"claude_bin"`
	PermissionMode string `toml:"permission_mode" json:"permission_mode"`
	Model          string `toml:"model" json:"model"`
	GitRemoteURL   string `toml:"git_remote_url" json:"git_remote_url"`
}

func Default() Config {
	return Config{
		WikiPath: "~/.thoth/wiki",
		Host:     "127.0.0.1",
		Port:     8333,
	}
}

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

// Load reads path; a missing file yields Default().
func Load(path string) (Config, error) {
	c := Default()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("load config: %w", err)
	}
	if err := toml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("parse config %s: %w", path, err)
	}
	return c, nil
}

// Save writes c to path, creating parent directories.
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	b, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
