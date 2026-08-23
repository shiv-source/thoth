package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/settings"
)

// settingsDTO is the wire shape of GET/PUT /api/settings. Every field lives
// in the settings table in thoth.db. Provider credentials are managed through
// their own /api/providers endpoints (the key is write-only there), and sync
// destinations through /api/sync/connections — neither carries per-provider
// state here.
type settingsDTO struct {
	WikiPath         string   `json:"wiki_path"`
	WikiFolders      []string `json:"wiki_folders"`
	Model            string   `json:"model"`
	ContextInjection bool     `json:"context_injection"`
}

func getSettings(c echo.Context, d Deps) error {
	wikiPath, _, err := d.Settings.Setting(settings.KeyWikiPath)
	if err != nil {
		return internalError(c, d, "read wiki_path", err)
	}
	wikiFolders, err := d.Settings.Folders()
	if err != nil {
		return internalError(c, d, "read wiki_folders", err)
	}
	if wikiFolders == nil {
		wikiFolders = []string{} // JSON must be [] not null — the UI types it as an array
	}
	model, _, err := d.Settings.Setting(settings.KeyModel)
	if err != nil {
		return internalError(c, d, "read model", err)
	}
	contextInjection, err := d.Settings.ContextInjection()
	if err != nil {
		return internalError(c, d, "read context_injection", err)
	}
	return c.JSON(http.StatusOK, settingsDTO{
		WikiPath: wikiPath, WikiFolders: wikiFolders, Model: model,
		ContextInjection: contextInjection,
	})
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
	if err := d.Settings.SetSetting(settings.KeyWikiFolders, strings.Join(next.WikiFolders, ",")); err != nil {
		return internalError(c, d, "set wiki_folders", err)
	}
	if err := d.Settings.SetSetting(settings.KeyModel, next.Model); err != nil {
		return internalError(c, d, "set model", err)
	}
	if err := d.Settings.SetSetting(settings.KeyContextInjection, boolString(next.ContextInjection)); err != nil {
		return internalError(c, d, "set context_injection", err)
	}
	return c.JSON(http.StatusOK, next)
}

// boolString renders a boolean setting value the way the settings table
// stores it: the literals "true"/"false".
func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
