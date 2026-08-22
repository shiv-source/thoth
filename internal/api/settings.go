package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/settings"
)

// providerDTO is the per-provider credential state in the settings DTO,
// keyed by the provider's llm_models label (e.g. "DeepSeek"). The api key is
// write-only: GET reports only whether one is stored, and PUT treats an
// empty value as "leave unchanged". base_url is never secret and
// round-trips; empty means the provider's default endpoint.
type providerDTO struct {
	HasAPIKey bool   `json:"has_api_key"`
	APIKey    string `json:"api_key"`
	BaseURL   string `json:"base_url"`
}

// settingsDTO is the wire shape of GET/PUT /api/settings. Every field lives
// in the settings table in thoth.db. Per-provider api keys are write-only:
// GET reports only whether one is stored (has_api_key), and PUT treats an
// empty api_key as "leave unchanged" — the secret is never echoed back to
// the UI. There is no shared key: credentials belong to each provider.
type settingsDTO struct {
	WikiPath    string                 `json:"wiki_path"`
	WikiFolders []string               `json:"wiki_folders"`
	Model       string                 `json:"model"`
	Providers   map[string]providerDTO `json:"providers"`
	RepoURL     string                 `json:"repo_url"`
	SyncEnabled bool                   `json:"sync_enabled"`
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
	providers, err := readProviderConfigs(d)
	if err != nil {
		return internalError(c, d, "read provider configs", err)
	}
	repoURL, _, err := d.Settings.Setting(settings.KeyRepoURL)
	if err != nil {
		return internalError(c, d, "read repo_url", err)
	}
	syncEnabled, err := d.Settings.SyncEnabled()
	if err != nil {
		return internalError(c, d, "read sync_enabled", err)
	}
	return c.JSON(http.StatusOK, settingsDTO{
		WikiPath: wikiPath, WikiFolders: wikiFolders, Model: model,
		Providers: providers, RepoURL: repoURL, SyncEnabled: syncEnabled,
	})
}

// readProviderConfigs returns the per-provider credential state for every
// provider present in the llm_models registry, keyed by the provider's exact
// label (the picker group name, so the UI can join on it). Empty map when the
// registry has no rows.
func readProviderConfigs(d Deps) (map[string]providerDTO, error) {
	models, err := d.Store.ListModels()
	if err != nil {
		return nil, err
	}
	out := make(map[string]providerDTO)
	for _, m := range models {
		if m.Provider == "" {
			continue
		}
		if _, seen := out[m.Provider]; seen {
			continue
		}
		apiKey, _, err := d.Settings.Setting(settings.ProviderAPIKeyKey(m.Provider))
		if err != nil {
			return nil, fmt.Errorf("read provider api key: %w", err)
		}
		baseURL, _, err := d.Settings.Setting(settings.ProviderBaseURLKey(m.Provider))
		if err != nil {
			return nil, fmt.Errorf("read provider base url: %w", err)
		}
		out[m.Provider] = providerDTO{HasAPIKey: apiKey != "", BaseURL: baseURL}
	}
	return out, nil
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
	for name, pc := range next.Providers {
		// base_url round-trips (an empty value clears back to the default
		// endpoint); the per-provider api key is write-only, so an empty
		// value leaves the stored key untouched.
		if err := d.Settings.SetSetting(settings.ProviderBaseURLKey(name), pc.BaseURL); err != nil {
			return internalError(c, d, "set provider base url", err)
		}
		if pc.APIKey != "" {
			if err := d.Settings.SetSetting(settings.ProviderAPIKeyKey(name), pc.APIKey); err != nil {
				return internalError(c, d, "set provider api key", err)
			}
		}
	}
	if err := d.Settings.SetSetting(settings.KeyRepoURL, next.RepoURL); err != nil {
		return internalError(c, d, "set repo_url", err)
	}
	if err := d.Settings.SetSetting(settings.KeySyncEnabled, map[bool]string{true: "true", false: "false"}[next.SyncEnabled]); err != nil {
		return internalError(c, d, "set sync_enabled", err)
	}
	return c.JSON(http.StatusOK, next)
}
