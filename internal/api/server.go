package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-warehouse/events"
	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/github"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/webui"
	"github.com/shiv-source/thoth/internal/wiki"
)

type Deps struct {
	Log             *slog.Logger
	Store           *store.Store
	Claude          Client
	GitHub          *github.Service
	Settings        *settings.Repo
	DataDir         string       // thoth dir (~/.thoth) — the doctor handler probes it
	DoctorAddr      string       // host:port for the doctor's api/websocket probes ("" → 127.0.0.1:8333); tests point it at a free port
	DoctorHTTP      *http.Client // HTTP client for the doctor's provider probe (nil → http.DefaultClient); tests stub the endpoint
	DoctorBaseURL   string       // provider base URL the doctor's provider probe targets ("" → the provider default); tests point it at a stub
	Version         string       // build version, shown in /api/health and the UI footer
	Dev             bool         // serve --dev — exposed via /api/health so the UI can show the dev banner
	Commit          string       // full git commit id the server runs from (dev only), shown in the dev banner
	DefaultWikiPath string       // the mode's wiki default in tilde form (~/.thoth/wiki, or ~/.thoth/dev/wiki in dev) — the settings hint reads it
	Wiki            *wiki.Wiki
	Index           *index.Index
	OnSettingsSaved func(wikiPath string) error
	Ctx             context.Context // cancelled on shutdown; reaps in-flight turns
	// Events is the in-process event bus. When set, wiki change batches are
	// forwarded to connected clients as wiki_changed frames; nil disables
	// the push (tests without a bus).
	Events *events.Bus
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
	e.Use(requestLog(d.Log))

	e.GET("/api/health", func(c echo.Context) error { return health(c, d) })
	e.GET("/api/doctor", func(c echo.Context) error { return doctorHandler(c, d) })
	e.POST("/api/git/setup", func(c echo.Context) error { return gitSetup(c, d) })
	e.GET("/api/search", func(c echo.Context) error { return search(c, d) })
	e.GET("/api/notes", func(c echo.Context) error { return note(c, d) })
	e.GET("/api/wiki/tree", func(c echo.Context) error { return tree(c, d) })
	e.GET("/api/fs/dirs", func(c echo.Context) error { return listDirs(c, d) })
	e.GET("/api/settings", func(c echo.Context) error { return getSettings(c, d) })
	e.PUT("/api/settings", func(c echo.Context) error { return putSettings(c, d) })
	e.GET("/api/models", func(c echo.Context) error { return models(c, d) })
	e.POST("/api/models", func(c echo.Context) error { return createModel(c, d) })
	e.PUT("/api/models/:id", func(c echo.Context) error { return updateModel(c, d) })
	e.DELETE("/api/models/:id", func(c echo.Context) error { return deleteModel(c, d) })
	e.GET("/api/conversations", func(c echo.Context) error { return listConversations(c, d) })
	e.POST("/api/conversations", func(c echo.Context) error { return createConversation(c, d) })
	e.GET("/api/conversations/:id", func(c echo.Context) error { return getConversation(c, d) })
	e.DELETE("/api/conversations/:id", func(c echo.Context) error { return deleteConversation(c, d) })
	e.POST("/api/github/auth", func(c echo.Context) error { return connectGitHub(c, d) })
	e.GET("/api/github/auth", func(c echo.Context) error { return getGitHubAuth(c, d) })
	e.DELETE("/api/github/auth", func(c echo.Context) error { return disconnectGitHub(c, d) })
	e.GET("/api/github/repos", func(c echo.Context) error { return listGitHubRepos(c, d) })

	hub := NewHub(d.Claude, d.Store, d.Log, d.ctx())
	if d.Events != nil {
		// Forward wiki change batches to every connected client, so the UI
		// refetches the tree only when files actually changed.
		if err := events.Subscribe(d.Events, d.ctx(), func(e events.Event[wiki.Changed]) {
			hub.Broadcast(serverMsg{Type: "wiki_changed", Changes: e.Data.Changes})
		}); err != nil {
			d.Log.Error("subscribe wiki events", "err", err)
		}
	}
	e.GET("/ws", hub.chat)

	webui.Register(e, d.Log)
	return e, hub
}
