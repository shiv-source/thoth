package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/claude"
)

// models returns the model list the Settings UI offers (static list, no
// settings table involvement — the chosen value lives under the model key).
func models(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{"models": claude.ModelOptions()})
}
