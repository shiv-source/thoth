package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/assets"
)

// models returns the model list the Settings UI offers. The list itself
// lives in internal/assets/models.json; the chosen value persists under the
// model settings key and is enforced via --model at boot.
func models(c echo.Context, d Deps) error {
	opts, err := assets.ModelOptions()
	if err != nil {
		return internalError(c, d, "models", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"models": opts})
}
