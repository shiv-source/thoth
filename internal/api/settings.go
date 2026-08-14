package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/config"
)

func getSettings(c echo.Context, d Deps) error {
	// Copy under the read lock: the JSON marshal must not race a concurrent
	// putSettings assignment.
	d.ConfigMu.RLock()
	cfg := *d.Config
	d.ConfigMu.RUnlock()
	return c.JSON(http.StatusOK, cfg)
}

func putSettings(c echo.Context, d Deps) error {
	var next config.Config
	if err := c.Bind(&next); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if next.WikiPath == "" || next.Host == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "wiki_path and host are required"})
	}
	if next.Port < 1 || next.Port > 65535 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "port must be 1-65535"})
	}
	// The callback runs before the save so a wiki-path change starts
	// rebuilding the index immediately; the in-memory config is only swapped
	// once both succeeded. A failed save after a successful callback
	// self-heals on retry: the callback sees the path already current,
	// no-ops, and the save succeeds.
	if d.OnSettingsSaved != nil {
		if err := d.OnSettingsSaved(next); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}
	if d.ConfigPath != "" {
		if err := config.Save(d.ConfigPath, next); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}
	d.ConfigMu.Lock()
	*d.Config = next
	d.ConfigMu.Unlock()
	return c.JSON(http.StatusOK, next)
}
