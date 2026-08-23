package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
)

// providerInput is the POST/PUT body for /api/providers. The api_key is
// write-only: an empty value on PUT leaves the stored key untouched.
// custom_headers is an object of header name → value (empty clears all).
type providerInput struct {
	Name          string            `json:"name"`
	BaseURL       string            `json:"base_url"`
	APIKey        string            `json:"api_key"`
	CustomHeaders map[string]string `json:"custom_headers"`
}

// providerDTO is the wire shape of /api/providers. has_api_key reports
// whether a key is stored — the key itself is never echoed back to the UI.
type providerDTO struct {
	ID            int64             `json:"id"`
	Name          string            `json:"name"`
	BaseURL       string            `json:"base_url"`
	CustomHeaders map[string]string `json:"custom_headers"`
	HasAPIKey     bool              `json:"has_api_key"`
	ModelCount    int               `json:"model_count"`
}

// providers returns the providers table A→Z, each with its model count. The
// list is the source for the Settings → Providers panels: providers are
// created before their models, so the list includes rows with zero models.
func providers(c echo.Context, d Deps) error {
	list, err := d.Store.ListProviders()
	if err != nil {
		return internalError(c, d, "list providers", err)
	}
	out := make([]providerDTO, 0, len(list))
	for _, p := range list {
		out = append(out, providerDTOFromStore(p))
	}
	return c.JSON(http.StatusOK, map[string]any{"providers": out})
}

func createProvider(c echo.Context, d Deps) error {
	var in providerInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if in.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	p, err := d.Store.CreateProvider(in.Name, in.BaseURL, in.APIKey)
	if errors.Is(err, store.ErrProviderExists) {
		return c.JSON(http.StatusConflict, map[string]string{"error": "a provider with this name already exists"})
	}
	if err != nil {
		return internalError(c, d, "create provider", err)
	}
	if err := d.Store.SetProviderHeaders(p.ID, in.CustomHeaders); err != nil {
		return internalError(c, d, "set provider headers", err)
	}
	p.Headers = in.CustomHeaders
	return c.JSON(http.StatusOK, providerDTOFromStore(p))
}

func updateProvider(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "provider not found"})
	}
	var in providerInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if in.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	old, err := d.Store.Provider(id)
	if err != nil {
		return providerStoreError(c, d, err, "read provider")
	}
	// The api_key is write-only: an empty value leaves the stored key
	// untouched (the base_url always round-trips — empty clears back to the
	// default endpoint).
	apiKey := old.APIKey
	if in.APIKey != "" {
		apiKey = in.APIKey
	}
	if err := d.Store.UpdateProvider(id, in.Name, in.BaseURL, apiKey); err != nil {
		return providerStoreError(c, d, err, "update provider")
	}
	if err := d.Store.SetProviderHeaders(id, in.CustomHeaders); err != nil {
		return internalError(c, d, "set provider headers", err)
	}
	updated, err := d.Store.Provider(id)
	if err != nil {
		return providerStoreError(c, d, err, "read provider")
	}
	return c.JSON(http.StatusOK, providerDTOFromStore(updated))
}

func deleteProvider(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "provider not found"})
	}
	// The selected-model setting is cleared before the provider goes, when
	// the deletion removes the model it points at — the reverse order could
	// leave --model pointing at a deleted row and fail every turn.
	if err := clearSelectedModelOfProvider(d, id); err != nil {
		return internalError(c, d, "read model setting", err)
	}
	if err := d.Store.DeleteProvider(id); err != nil {
		return providerStoreError(c, d, err, "delete provider")
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// clearSelectedModelOfProvider clears the settings model key when the model
// it selects is about to be deleted along with its provider.
func clearSelectedModelOfProvider(d Deps, providerID int64) error {
	selected, _, err := d.Settings.Setting(settings.KeyModel)
	if err != nil || selected == "" {
		return err
	}
	models, err := d.Store.ListModels()
	if err != nil {
		return err
	}
	for _, m := range models {
		if m.ProviderID == providerID && m.Value == selected {
			return d.Settings.SetSetting(settings.KeyModel, "")
		}
	}
	return nil
}

// providerDTOFromStore maps a store row to the wire shape, hiding the key.
func providerDTOFromStore(p store.Provider) providerDTO {
	return providerDTO{
		ID: p.ID, Name: p.Name, BaseURL: p.BaseURL,
		CustomHeaders: p.Headers, HasAPIKey: p.APIKey != "", ModelCount: p.ModelCount,
	}
}

// providerStoreError maps the providers sentinel errors to 409/404; anything
// else is an internal error.
func providerStoreError(c echo.Context, d Deps, err error, op string) error {
	switch {
	case errors.Is(err, store.ErrProviderExists):
		return c.JSON(http.StatusConflict, map[string]string{"error": "a provider with this name already exists"})
	case errors.Is(err, store.ErrProviderNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": "provider not found"})
	default:
		return internalError(c, d, op, err)
	}
}
