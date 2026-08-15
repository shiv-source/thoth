package webui

import (
	"embed"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/labstack/echo/v4"
)

//go:embed all:dist
var dist embed.FS

// Register serves the embedded SPA with a fallback to index.html so
// client-side routes and deep links resolve.
func Register(e *echo.Echo, _ *slog.Logger) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // embed misconfiguration is a build-time bug
	}
	// c.File serves from Echo.Filesystem, so point it at the embedded dist.
	e.Filesystem = sub
	e.GET("/*", func(c echo.Context) error {
		p := strings.TrimPrefix(c.Request().URL.Path, "/")
		// The API surface stays JSON: unknown /api paths are a 404, never the
		// SPA shell.
		if p == "api" || strings.HasPrefix(p, "api/") {
			return echo.ErrNotFound
		}
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(sub, p); err == nil {
			return c.File(p)
		}
		return c.File("index.html")
	})
}
