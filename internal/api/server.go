package api

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/claude"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/webui"
	"github.com/shiv-source/thoth/internal/wiki"
)

type Deps struct {
	Log             *slog.Logger
	Config          *config.Config
	Store           *store.Store
	Claude          claude.Client
	Wiki            *wiki.Wiki
	Index           *index.Index
	OnSettingsSaved func(config.Config) error
}

// New builds the Echo server with all routes. API routes are registered
// before the webui wildcard so they always win.
func New(d Deps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.GET("/api/health", func(c echo.Context) error { return health(c, d) })
	e.GET("/api/search", func(c echo.Context) error { return search(c, d) })
	e.GET("/api/notes", func(c echo.Context) error { return note(c, d) })
	e.GET("/api/wiki/tree", func(c echo.Context) error { return tree(c, d) })
	// (settings, conversations, and /ws arrive in Tasks 13–15)

	webui.Register(e, d.Log)
	return e
}
