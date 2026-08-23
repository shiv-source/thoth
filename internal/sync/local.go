package sync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/wiki"
)

// localDriver writes timestamped zip snapshots of the wiki to a local backup
// folder — the first-class, always-available destination for users who do not
// trust external servers. The target must be outside the wiki tree (a
// self-referential mirror would recurse forever).
type localDriver struct{}

func (d *localDriver) Kind() Kind { return KindLocal }

func (d *localDriver) Fields() []Field {
	return []Field{{Key: "path", Label: "Backup folder", Required: true}}
}

func (d *localDriver) Verify(_ context.Context, cfg Config) (Identity, error) {
	if cfg["path"] == "" {
		return Identity{}, errors.New("backup folder is required")
	}
	return Identity{}, nil
}

func (d *localDriver) Targets(context.Context, Config) ([]Target, error) { return nil, nil }

// Push writes thoth-wiki-YYYYMMDD-HHMMSS.zip into the backup folder. The
// folder is created on demand; point-in-time snapshots keep deleted files
// recoverable. Errors are sanitized fixed messages.
func (d *localDriver) Push(_ context.Context, cfg Config, root string, _ Identity) error {
	raw := cfg["path"]
	if raw == "" {
		return errors.New("no backup folder configured — set one in Settings")
	}
	dir, err := config.ExpandHome(raw)
	if err != nil {
		return errors.New("could not resolve the backup folder")
	}
	if pathContains(root, dir) {
		return errors.New("the backup folder must be outside the wiki")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.New("could not create the backup folder — check the path is writable")
	}
	name := "thoth-wiki-" + time.Now().UTC().Format("20060102-150405") + ".zip"
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return errors.New("could not write the backup file — check the folder is writable")
	}
	defer func() { _ = f.Close() }()
	if err := wiki.New(root).ExportTo(f, wiki.ExportOptions{}); err != nil {
		return errors.New("could not create the wiki archive")
	}
	return f.Close()
}

// pathContains reports whether child is equal to parent or inside it.
func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
