package api

import (
	"context"
	"log/slog"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/claude"
	"github.com/shiv-source/thoth/internal/config"
	"github.com/shiv-source/thoth/internal/github"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/webui"
	"github.com/shiv-source/thoth/internal/wiki"
)

type Deps struct {
	Log             *slog.Logger
	Config          *config.Config
	ConfigPath      string
	ConfigMu        *sync.RWMutex // guards *Config (shared with serve)
	Store           *store.Store
	Claude          claude.Client
	GitHub          *github.Service
	Wiki            *wiki.Wiki
	Index           *index.Index
	OnSettingsSaved func(config.Config) error
	Ctx             context.Context // cancelled on shutdown; reaps in-flight turns
}

// ctx returns d.Ctx, defaulting to background for embedders that do not
// thread a shutdown context.
func (d Deps) ctx() context.Context {
	if d.Ctx != nil {
		return d.Ctx
	}
	return context.Background()
}

// New builds the Echo server with all routes. API routes are registered
// before the webui wildcard so they always win.
func New(d Deps) *echo.Echo {
	e, _ := newServer(d)
	return e
}

// newServer additionally hands back the chat hub so tests can inspect
// in-flight turn state.
func newServer(d Deps) (*echo.Echo, *Hub) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.GET("/api/health", func(c echo.Context) error { return health(c, d) })
	e.GET("/api/doctor", func(c echo.Context) error { return doctorHandler(c, d) })
	e.POST("/api/git/setup", func(c echo.Context) error { return gitSetup(c, d) })
	e.GET("/api/search", func(c echo.Context) error { return search(c, d) })
	e.GET("/api/notes", func(c echo.Context) error { return note(c, d) })
	e.GET("/api/wiki/tree", func(c echo.Context) error { return tree(c, d) })
	e.GET("/api/settings", func(c echo.Context) error { return getSettings(c, d) })
	e.PUT("/api/settings", func(c echo.Context) error { return putSettings(c, d) })
	e.GET("/api/conversations", func(c echo.Context) error { return listConversations(c, d) })
	e.POST("/api/conversations", func(c echo.Context) error { return createConversation(c, d) })
	e.GET("/api/conversations/:id", func(c echo.Context) error { return getConversation(c, d) })
	e.POST("/api/github/auth", func(c echo.Context) error { return connectGitHub(c, d) })
	e.GET("/api/github/auth", func(c echo.Context) error { return getGitHubAuth(c, d) })
	e.DELETE("/api/github/auth", func(c echo.Context) error { return disconnectGitHub(c, d) })
	e.GET("/api/github/repos", func(c echo.Context) error { return listGitHubRepos(c, d) })

	hub := NewHub(d.Claude, d.Store, d.Log, d.ctx())
	e.GET("/ws", hub.chat)

	webui.Register(e, d.Log)
	return e, hub
}
