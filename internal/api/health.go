package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
)

type healthResponse struct {
	Status          string       `json:"status"`
	Backend         backendState `json:"backend"`
	Wiki            wikiState    `json:"wiki"`
	Version         string       `json:"version"`
	Dev             bool         `json:"dev"`
	Commit          string       `json:"commit"`
	DefaultWikiPath string       `json:"default_wiki_path"`
}

// backendState reports the native agent backend every turn runs through:
// whether an API key is configured, the selected model, and its provider.
type backendState struct {
	Name             string `json:"name"`
	APIKeyConfigured bool   `json:"api_key_configured"`
	Model            string `json:"model"`
	Provider         string `json:"provider"`
}

type wikiState struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func health(c echo.Context, d Deps) error {
	return c.JSON(http.StatusOK, healthResponse{
		Status:          "ok",
		Backend:         backend(d),
		Wiki:            wikiState{Path: d.Wiki.Root(), Exists: d.Wiki.Exists()},
		Version:         d.Version,
		Dev:             d.Dev,
		Commit:          d.Commit,
		DefaultWikiPath: d.DefaultWikiPath,
	})
}

// backend reports the native backend state. The API key is write-only, so
// only its presence is exposed; the model comes from the settings table and
// the provider from the model's llm_models row. Unreadable settings degrade
// to "not configured" rather than failing the health probe.
func backend(d Deps) backendState {
	model, _, _ := d.Settings.Setting(settings.KeyModel)
	apiKey, _, _ := d.Settings.Setting(settings.KeyAPIKey)
	return backendState{
		Name:             "thoth-agent",
		APIKeyConfigured: apiKey != "",
		Model:            model,
		Provider:         providerName(d.Store, model),
	}
}

// providerName returns the provider registered for model, or "" when the
// model is unset or absent from the registry.
func providerName(st *store.Store, model string) string {
	if model == "" || st == nil {
		return ""
	}
	models, err := st.ListModels()
	if err != nil {
		return ""
	}
	for _, m := range models {
		if m.Value == model {
			return m.Provider
		}
	}
	return ""
}
