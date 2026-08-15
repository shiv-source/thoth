package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/settings"
)

// settingsDTO is the wire shape of GET/PUT /api/settings. Every field lives
// in the settings table in thoth.db.
type settingsDTO struct {
	WikiPath    string `json:"wiki_path"`
	Model       string `json:"model"`
	RepoURL     string `json:"repo_url"`
	SyncEnabled bool   `json:"sync_enabled"`
}

func getSettings(c echo.Context, d Deps) error {
	wikiPath, _, err := d.Settings.Setting(settings.KeyWikiPath)
	if err != nil {
		return internalError(c, d, "read wiki_path", err)
	}
	model, _, err := d.Settings.Setting(settings.KeyModel)
	if err != nil {
		return internalError(c, d, "read model", err)
	}
	repoURL, _, err := d.Settings.Setting(settings.KeyRepoURL)
	if err != nil {
		return internalError(c, d, "read repo_url", err)
	}
	syncEnabled, err := d.Settings.SyncEnabled()
	if err != nil {
		return internalError(c, d, "read sync_enabled", err)
	}
	return c.JSON(http.StatusOK, settingsDTO{WikiPath: wikiPath, Model: model, RepoURL: repoURL, SyncEnabled: syncEnabled})
}

func putSettings(c echo.Context, d Deps) error {
	var next settingsDTO
	if err := c.Bind(&next); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if next.WikiPath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "wiki_path is required"})
	}
	// The callback runs before the writes so a wiki-path change starts
	// rebuilding the index immediately; a failed write after a successful
	// callback self-heals on retry (the callback sees the path already
	// current, no-ops, and the writes succeed).
	if d.OnSettingsSaved != nil {
		if err := d.OnSettingsSaved(next.WikiPath); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}
	if err := d.Settings.SetSetting(settings.KeyWikiPath, next.WikiPath); err != nil {
		return internalError(c, d, "set wiki_path", err)
	}
	if err := d.Settings.SetSetting(settings.KeyModel, next.Model); err != nil {
		return internalError(c, d, "set model", err)
	}
	if err := d.Settings.SetSetting(settings.KeyRepoURL, next.RepoURL); err != nil {
		return internalError(c, d, "set repo_url", err)
	}
	if err := d.Settings.SetSetting(settings.KeySyncEnabled, map[bool]string{true: "true", false: "false"}[next.SyncEnabled]); err != nil {
		return internalError(c, d, "set sync_enabled", err)
	}
	return c.JSON(http.StatusOK, next)
}
