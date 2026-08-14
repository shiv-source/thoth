package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/config"
)

func getSettings(c echo.Context, d Deps) error {
	return c.JSON(http.StatusOK, d.Config)
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
	*d.Config = next
	if d.ConfigPath != "" {
		if err := config.Save(d.ConfigPath, next); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	if d.OnSettingsSaved != nil {
		if err := d.OnSettingsSaved(next); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}
	return c.JSON(http.StatusOK, next)
}
