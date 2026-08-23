package api

import (
	"context"
	"embed"
	"log/slog"
	"net/http"

	"github.com/go-warehouse/events"
	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	syncsvc "github.com/shiv-source/thoth/internal/sync"
	"github.com/shiv-source/thoth/internal/webui"
	"github.com/shiv-source/thoth/internal/wiki"
)

//go:embed all:docs
var docsFS embed.FS

// APIVersion is the URL-path version segment for the REST and WebSocket
// surfaces. Every route lives under /api<APIVersion>/... (and /ws<APIVersion>);
// bumping it is a deliberate breaking change, so a new version is a one-file
// edit here plus the callers that must move with it.
const APIVersion = "/v1"

type Deps struct {
	Log             *slog.Logger
	Store           *store.Store
	Claude          Client
	Sync            *syncsvc.Service
	Settings        *settings.Repo
	DataDir         string       // thoth dir (~/.thoth) — the doctor handler probes it
	DoctorAddr      string       // host:port for the doctor's api/websocket probes ("" → 127.0.0.1:8333); tests point it at a free port
	DoctorHTTP      *http.Client // HTTP client for the doctor's provider probe (nil → http.DefaultClient); tests stub the endpoint
	DoctorBaseURL   string       // provider base URL the doctor's provider probe targets ("" → the provider default); tests point it at a stub
	Version         string       // build version, shown in /api/v1/health and the UI footer
	Dev             bool         // serve --dev — exposed via /api/v1/health so the UI can show the dev banner
	Commit          string       // full git commit id the server runs from (dev only), shown in the dev banner
	DefaultWikiPath string       // the mode's wiki default in tilde form (~/.thoth/wiki, or ~/.thoth/dev/wiki in dev) — the settings hint reads it
	ServeAPIDocs    bool         // serve --dev without --no-api-docs — registers the API reference routes (/api/docs, its assets, /swagger.json); absent when false
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

	e.GET("/api"+APIVersion+"/health", func(c echo.Context) error { return health(c, d) })
	e.GET("/api"+APIVersion+"/doctor", func(c echo.Context) error { return doctorHandler(c, d) })
	e.GET("/api"+APIVersion+"/search", func(c echo.Context) error { return search(c, d) })
	e.GET("/api"+APIVersion+"/notes", func(c echo.Context) error { return note(c, d) })
	e.POST("/api"+APIVersion+"/notes", func(c echo.Context) error { return createNote(c, d) })
	e.GET("/api"+APIVersion+"/wiki/tree", func(c echo.Context) error { return tree(c, d) })
	e.GET("/api"+APIVersion+"/wiki/export", func(c echo.Context) error { return exportWiki(c, d) })
	e.POST("/api"+APIVersion+"/wiki/import", func(c echo.Context) error { return importWiki(c, d) })
	e.GET("/api"+APIVersion+"/fs/dirs", func(c echo.Context) error { return listDirs(c, d) })
	e.GET("/api"+APIVersion+"/settings", func(c echo.Context) error { return getSettings(c, d) })
	e.PUT("/api"+APIVersion+"/settings", func(c echo.Context) error { return putSettings(c, d) })
	e.GET("/api"+APIVersion+"/models", func(c echo.Context) error { return models(c, d) })
	e.POST("/api"+APIVersion+"/models", func(c echo.Context) error { return createModel(c, d) })
	e.PUT("/api"+APIVersion+"/models/:id", func(c echo.Context) error { return updateModel(c, d) })
	e.DELETE("/api"+APIVersion+"/models/:id", func(c echo.Context) error { return deleteModel(c, d) })
	e.GET("/api"+APIVersion+"/providers", func(c echo.Context) error { return providers(c, d) })
	e.POST("/api"+APIVersion+"/providers", func(c echo.Context) error { return createProvider(c, d) })
	e.PUT("/api"+APIVersion+"/providers/:id", func(c echo.Context) error { return updateProvider(c, d) })
	e.DELETE("/api"+APIVersion+"/providers/:id", func(c echo.Context) error { return deleteProvider(c, d) })
	e.GET("/api"+APIVersion+"/conversations", func(c echo.Context) error { return listConversations(c, d) })
	e.POST("/api"+APIVersion+"/conversations", func(c echo.Context) error { return createConversation(c, d) })
	e.GET("/api"+APIVersion+"/conversations/:id", func(c echo.Context) error { return getConversation(c, d) })
	e.DELETE("/api"+APIVersion+"/conversations/:id", func(c echo.Context) error { return deleteConversation(c, d) })
	e.GET("/api"+APIVersion+"/sync/providers", func(c echo.Context) error { return listSyncProviders(c, d) })
	e.POST("/api"+APIVersion+"/sync/providers", func(c echo.Context) error { return createSyncProvider(c, d) })
	e.PUT("/api"+APIVersion+"/sync/providers/:id", func(c echo.Context) error { return updateSyncProvider(c, d) })
	e.DELETE("/api"+APIVersion+"/sync/providers/:id", func(c echo.Context) error { return deleteSyncProvider(c, d) })
	e.GET("/api"+APIVersion+"/sync/connections", func(c echo.Context) error { return listConnections(c, d) })
	e.POST("/api"+APIVersion+"/sync/connections", func(c echo.Context) error { return connect(c, d) })
	e.GET("/api"+APIVersion+"/sync/connections/:id", func(c echo.Context) error { return getConnection(c, d) })
	e.PUT("/api"+APIVersion+"/sync/connections/:id", func(c echo.Context) error { return updateConnection(c, d) })
	e.DELETE("/api"+APIVersion+"/sync/connections/:id", func(c echo.Context) error { return disconnect(c, d) })
	e.GET("/api"+APIVersion+"/sync/connections/:id/targets", func(c echo.Context) error { return listTargets(c, d) })
	e.POST("/api"+APIVersion+"/sync/connections/:id/push", func(c echo.Context) error { return pushConnection(c, d) })
	e.GET("/api"+APIVersion+"/sync/connections/:id/snapshots", func(c echo.Context) error { return listSnapshots(c, d) })
	e.POST("/api"+APIVersion+"/sync/connections/:id/restore", func(c echo.Context) error { return restoreConnection(c, d) })
	e.POST("/api"+APIVersion+"/sync/connections/:id/active", func(c echo.Context) error { return setActiveConnection(c, d) })

	// The API reference page is a dev-only convenience: serve --dev exposes
	// it (/api/docs, its assets, and /swagger.json), --dev --no-api-docs (and
	// any non-dev serve) leaves every route unregistered so nothing leaks in
	// normal use.
	if d.ServeAPIDocs {
		sub, err := docsSub()
		if err != nil {
			d.Log.Error("embed api docs", "err", err)
		} else {
			registerAPIDocs(e, d, sub)
		}
	}

	hub := NewHub(d.Claude, d.Store, d.Log, d.ctx())
	if d.Events != nil {
		// Forward wiki change batches to every connected client, so the UI
		// refetches the tree only when files actually changed.
		if err := events.Subscribe(d.Events, d.ctx(), func(e events.Event[wiki.Changed]) {
			hub.Broadcast(serverMsg{Type: "wiki_changed", Changes: e.Data.Changes})
		}); err != nil {
			d.Log.Error("subscribe wiki events", "err", err)
		}
		// Forward auto-sync results to every connected client, so a scheduled
		// push's outcome surfaces as a notification without polling.
		if err := events.Subscribe(d.Events, d.ctx(), func(e events.Event[syncsvc.Result]) {
			hub.Broadcast(serverMsg{
				Type: "sync_result",
				SyncResult: &syncResultFrame{
					ConnectionID: e.Data.ConnectionID,
					Name:         e.Data.Name,
					OK:           e.Data.OK,
					Error:        e.Data.Error,
				},
			})
		}); err != nil {
			d.Log.Error("subscribe sync events", "err", err)
		}
	}
	e.GET("/ws"+APIVersion, hub.chat)

	webui.Register(e, d.Log)
	return e, hub
}
