package api

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
)

// modelInput is the POST/PUT body for /api/models. Value is the --model
// argument; name is the display name; tag and provider are optional display
// fields.
type modelInput struct {
	Value    string `json:"value"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Provider string `json:"provider"`
}

// modelGroup is the GET /api/models wire shape: models grouped by provider,
// providers sorted case-insensitively A→Z and models in seed (id) order.
type modelGroup struct {
	Provider string           `json:"provider"`
	Models   []store.LLMModel `json:"models"`
}

// models returns the llm_models table grouped by provider, seeded from
// assets/models.json on first boot and edited by the user afterwards. The
// settings model key stores the chosen value and is enforced via --model at
// boot.
func models(c echo.Context, d Deps) error {
	list, err := d.Store.ListModels()
	if err != nil {
		return internalError(c, d, "list models", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"groups": groupModels(list)})
}

// groupModels groups the flat list by provider (first-seen model order per
// provider, which is the seed order) and sorts the groups A→Z.
func groupModels(list []store.LLMModel) []modelGroup {
	byProvider := map[string][]store.LLMModel{}
	var providers []string
	for _, m := range list {
		if _, ok := byProvider[m.Provider]; !ok {
			providers = append(providers, m.Provider)
		}
		byProvider[m.Provider] = append(byProvider[m.Provider], m)
	}
	sort.Slice(providers, func(i, j int) bool {
		return strings.ToLower(providers[i]) < strings.ToLower(providers[j])
	})
	groups := make([]modelGroup, 0, len(providers))
	for _, p := range providers {
		groups = append(groups, modelGroup{Provider: p, Models: byProvider[p]})
	}
	return groups
}

func createModel(c echo.Context, d Deps) error {
	var in modelInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if in.Value == "" || in.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "value and name are required"})
	}
	m, err := d.Store.CreateModel(in.Value, in.Name, in.Tag, in.Provider)
	if errors.Is(err, store.ErrModelExists) {
		return c.JSON(http.StatusConflict, map[string]string{"error": "a model with this value already exists"})
	}
	if err != nil {
		return internalError(c, d, "create model", err)
	}
	return c.JSON(http.StatusOK, m)
}

func updateModel(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "model not found"})
	}
	var in modelInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if in.Value == "" || in.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "value and name are required"})
	}
	old, err := d.Store.Model(id)
	if err != nil {
		return modelStoreError(c, d, err, "read model")
	}
	if err := d.Store.UpdateModel(id, in.Value, in.Name, in.Tag, in.Provider); err != nil {
		return modelStoreError(c, d, err, "update model")
	}
	// A renamed value follows the selected-model setting, so the --model
	// flag keeps pointing at a row that exists. Written after the row:
	// a stale pointer self-heals on the next save, a 409 above leaves
	// everything untouched.
	if in.Value != old.Value {
		if selected, _, err := d.Settings.Setting(settings.KeyModel); err != nil {
			return internalError(c, d, "read model setting", err)
		} else if selected == old.Value {
			if err := d.Settings.SetSetting(settings.KeyModel, in.Value); err != nil {
				return internalError(c, d, "set model setting", err)
			}
		}
	}
	return c.JSON(http.StatusOK, store.LLMModel{ID: id, Value: in.Value, Name: in.Name, Tag: in.Tag, Provider: in.Provider})
}

func deleteModel(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "model not found"})
	}
	m, err := d.Store.Model(id)
	if err != nil {
		return modelStoreError(c, d, err, "read model")
	}
	// The selected-model setting is cleared before the row goes: if the
	// delete then failed, the worst case is a user re-selecting a model
	// that still exists — the reverse order could leave --model pointing
	// at a deleted row and fail every turn until re-saved.
	if selected, _, err := d.Settings.Setting(settings.KeyModel); err != nil {
		return internalError(c, d, "read model setting", err)
	} else if selected == m.Value {
		if err := d.Settings.SetSetting(settings.KeyModel, ""); err != nil {
			return internalError(c, d, "clear model setting", err)
		}
	}
	if err := d.Store.DeleteModel(id); err != nil {
		return modelStoreError(c, d, err, "delete model")
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// modelStoreError maps the llm_models sentinel errors to 409/404; anything
// else is an internal error.
func modelStoreError(c echo.Context, d Deps, err error, op string) error {
	switch {
	case errors.Is(err, store.ErrModelExists):
		return c.JSON(http.StatusConflict, map[string]string{"error": "a model with this value already exists"})
	case errors.Is(err, store.ErrModelNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": "model not found"})
	default:
		return internalError(c, d, op, err)
	}
}
