package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/github"
)

// settingsDTO is the wire shape of GET/PUT /api/settings: the config fields
// plus repo_url, which lives in the github_auth table (DB), not config.toml.
type settingsDTO struct {
	config.Config
	RepoURL string `json:"repo_url"`
}

func getSettings(c echo.Context, d Deps) error {
	// Copy under the read lock: the JSON marshal must not race a concurrent
	// putSettings assignment.
	d.ConfigMu.RLock()
	cfg := *d.Config
	d.ConfigMu.RUnlock()
	a, _, err := d.GitHub.Repo.Get()
	if err != nil {
		return internalError(c, d, "read github auth", err)
	}
	return c.JSON(http.StatusOK, settingsDTO{Config: cfg, RepoURL: a.RepoURL})
}

func putSettings(c echo.Context, d Deps) error {
	var next settingsDTO
	if err := c.Bind(&next); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if next.WikiPath == "" || next.Host == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "wiki_path and host are required"})
	}
	if next.Port < 1 || next.Port > 65535 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "port must be 1-65535"})
	}
	// The repo URL lives in the DB: setting it requires a connected GitHub
	// account, except for an empty value (nothing to store) so ordinary saves
	// by unconnected users keep working.
	err := d.GitHub.Repo.SetRepoURL(next.RepoURL)
	switch {
	case err == nil:
	case errors.Is(err, github.ErrGitHubAuthNotFound) && next.RepoURL == "":
	case errors.Is(err, github.ErrGitHubAuthNotFound):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "connect GitHub first"})
	default:
		return internalError(c, d, "set repo url", err)
	}
	// The callback runs before the save so a wiki-path change starts
	// rebuilding the index immediately; the in-memory config is only swapped
	// once both succeeded. A failed save after a successful callback
	// self-heals on retry: the callback sees the path already current,
	// no-ops, and the save succeeds.
	if d.OnSettingsSaved != nil {
		if err := d.OnSettingsSaved(next.Config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}
	if d.ConfigPath != "" {
		if err := config.Save(d.ConfigPath, next.Config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}
	d.ConfigMu.Lock()
	*d.Config = next.Config
	d.ConfigMu.Unlock()
	return c.JSON(http.StatusOK, next)
}
