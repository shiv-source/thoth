package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/config"
)

// listDirs returns the immediate subdirectories of the given path (absolute
// paths, lexically sorted), powering the Settings directory picker. It is
// localhost-bound like every endpoint; 400 when the path is missing or not a
// readable directory.
func listDirs(c echo.Context, d Deps) error {
	raw := c.QueryParam("path")
	if raw == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "path is required"})
	}
	dir, err := config.ExpandHome(raw)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot expand path"})
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "not a readable directory"})
	}
	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(dirs)
	return c.JSON(http.StatusOK, map[string]any{"dirs": dirs})
}
