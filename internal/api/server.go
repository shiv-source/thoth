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
	ConfigPath      string
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
	e.GET("/api/settings", func(c echo.Context) error { return getSettings(c, d) })
	e.PUT("/api/settings", func(c echo.Context) error { return putSettings(c, d) })
	e.GET("/api/conversations", func(c echo.Context) error { return listConversations(c, d) })
	e.POST("/api/conversations", func(c echo.Context) error { return createConversation(c, d) })
	e.GET("/api/conversations/:id", func(c echo.Context) error { return getConversation(c, d) })
	// (/ws arrives in Task 13)

	webui.Register(e, d.Log)
	return e
}
