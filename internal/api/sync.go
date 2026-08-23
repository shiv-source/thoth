package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/settings"
	"github.com/shiv-source/thoth/internal/store"
	syncsvc "github.com/shiv-source/thoth/internal/sync"
	"github.com/shiv-source/thoth/internal/wiki"
)

// syncProviderDTO is the catalog wire shape. fields are the driver's input
// descriptors so the UI renders the right connect form per provider; kind is
// the transport family for grouping. Secrets never appear here.
type syncProviderDTO struct {
	ID              int64           `json:"id"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Driver          string          `json:"driver"`
	Kind            syncsvc.Kind    `json:"kind"`
	BaseURL         string          `json:"base_url"`
	Protected       bool            `json:"protected"`
	Fields          []syncsvc.Field `json:"fields"`
	ConnectionCount int             `json:"connection_count"`
}

// connectionDTO is the token-free wire shape of a connection. Secret config
// fields surface as has_<key> booleans (the value never leaves the store);
// non-secret fields (repo_url, bucket, region, path, …) round-trip.
type connectionDTO struct {
	ID           int64             `json:"id"`
	ProviderID   int64             `json:"provider_id"`
	ProviderSlug string            `json:"provider_slug"`
	ProviderName string            `json:"provider_name"`
	Name         string            `json:"name"`
	Enabled      bool              `json:"enabled"`
	Protected    bool              `json:"protected"`
	Active       bool              `json:"active"`
	Identity     *syncsvc.Identity `json:"identity"`
	Config       map[string]any    `json:"config"`
	LastSyncedAt string            `json:"last_synced_at"`
	LastError    string            `json:"last_error"`
	// PushHistory is the recent sync runs (newest first), beyond the single
	// last_synced_at/last_error columns.
	PushHistory []store.PushEntry `json:"push_history"`
}

// syncProviderInput is the POST/PUT body for /api/sync/providers.
type syncProviderInput struct {
	Name    string `json:"name"`
	Driver  string `json:"driver"`
	BaseURL string `json:"base_url"`
}

// connectInput is the POST /api/sync/connections body. Config carries the
// driver's credential/target fields (token, access keys, path, …).
type connectInput struct {
	ProviderID int64             `json:"provider_id"`
	Name       string            `json:"name"`
	Config     map[string]string `json:"config"`
}

// updateConnectionInput is the PUT /api/sync/connections/:id body. Config
// secrets are write-only: an empty value leaves the stored one untouched.
type updateConnectionInput struct {
	Name    string            `json:"name"`
	Enabled *bool             `json:"enabled"`
	Config  map[string]string `json:"config"`
}

// listSyncProviders returns the sync provider catalog — the Settings sync
// page source: built-ins (github, gitlab, aws_s3, local) plus user rows.
func listSyncProviders(c echo.Context, d Deps) error {
	list, err := d.Store.ListSyncProviders()
	if err != nil {
		return internalError(c, d, "list sync providers", err)
	}
	out := make([]syncProviderDTO, 0, len(list))
	for _, p := range list {
		out = append(out, syncProviderDTOFromStore(d, p))
	}
	return c.JSON(http.StatusOK, map[string]any{"providers": out})
}

func createSyncProvider(c echo.Context, d Deps) error {
	var in syncProviderInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if in.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if d.Sync == nil || !d.Sync.KnownDriver(in.Driver) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "driver is required"})
	}
	slug, err := uniqueSyncSlug(d.Store, in.Name)
	if err != nil {
		return internalError(c, d, "make sync provider slug", err)
	}
	p, err := d.Store.CreateSyncProvider(slug, in.Name, in.Driver, in.BaseURL)
	if errors.Is(err, store.ErrSyncProviderExists) {
		return c.JSON(http.StatusConflict, map[string]string{"error": "a sync provider with this name already exists"})
	}
	if err != nil {
		return internalError(c, d, "create sync provider", err)
	}
	return c.JSON(http.StatusOK, syncProviderDTOFromStore(d, p))
}

func updateSyncProvider(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "sync provider not found"})
	}
	var in syncProviderInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if in.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if err := d.Store.UpdateSyncProvider(id, in.Name, in.BaseURL); err != nil {
		return syncProviderStoreError(c, d, err, "update sync provider")
	}
	p, err := d.Store.SyncProvider(id)
	if err != nil {
		return syncProviderStoreError(c, d, err, "read sync provider")
	}
	return c.JSON(http.StatusOK, syncProviderDTOFromStore(d, p))
}

func deleteSyncProvider(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "sync provider not found"})
	}
	if err := d.Store.DeleteSyncProvider(id); err != nil {
		return syncProviderStoreError(c, d, err, "delete sync provider")
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// listConnections returns every connection with its provider joined.
func listConnections(c echo.Context, d Deps) error {
	conns, err := d.Store.ListConnections()
	if err != nil {
		return internalError(c, d, "list connections", err)
	}
	activeID, _, _ := d.Settings.Setting(settings.KeyActiveConnection)
	out := make([]connectionDTO, 0, len(conns))
	for _, conn := range conns {
		out = append(out, connectionDTOFromStore(d, conn, activeID))
	}
	return c.JSON(http.StatusOK, map[string]any{"connections": out})
}

// connect verifies the provider credentials and stores the connection. The
// server calls the driver's Verify so a bad token fails here, not on first
// sync.
func connect(c echo.Context, d Deps) error {
	var in connectInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if in.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if in.Config == nil {
		in.Config = map[string]string{}
	}
	p, err := d.Store.SyncProvider(in.ProviderID)
	if err != nil {
		return syncProviderStoreError(c, d, err, "read sync provider")
	}
	if d.Sync == nil {
		return internalError(c, d, "connect", errors.New("missing sync service"))
	}
	drv, err := d.Sync.Driver(p)
	if err != nil {
		return internalError(c, d, "resolve sync driver", err)
	}
	if missing := requiredFieldsMissing(drv, in.Config); missing != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": missing + " is required"})
	}
	ident, err := drv.Verify(c.Request().Context(), in.Config)
	if errors.Is(err, syncsvc.ErrRejected) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "the provider rejected these credentials"})
	}
	if err != nil {
		return internalError(c, d, "verify sync credentials", err)
	}
	cfgJSON, err := syncsvc.EncodeConfig(in.Config)
	if err != nil {
		return internalError(c, d, "encode connection config", err)
	}
	identJSON, err := syncsvc.EncodeIdentity(ident)
	if err != nil {
		return internalError(c, d, "encode connection identity", err)
	}
	conn, err := d.Store.CreateConnection(p.ID, in.Name, cfgJSON, identJSON, true)
	if err != nil {
		return internalError(c, d, "create connection", err)
	}
	activeID, _, _ := d.Settings.Setting(settings.KeyActiveConnection)
	return c.JSON(http.StatusOK, connectionDTOFromStore(d, conn, activeID))
}

func getConnection(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	conn, err := d.Store.Connection(id)
	if err != nil {
		return connectionStoreError(c, d, err, "read connection")
	}
	activeID, _, _ := d.Settings.Setting(settings.KeyActiveConnection)
	return c.JSON(http.StatusOK, connectionDTOFromStore(d, conn, activeID))
}

// updateConnection edits the editable fields. Secret config values with an
// empty incoming value are left unchanged (write-only); every other field
// overwrites.
func updateConnection(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	conn, err := d.Store.Connection(id)
	if err != nil {
		return connectionStoreError(c, d, err, "read connection")
	}
	var in updateConnectionInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	name := in.Name
	if name == "" {
		name = conn.Name
	}
	enabled := conn.Enabled
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	config := conn.Config
	if in.Config != nil {
		p, err := d.Store.SyncProvider(conn.ProviderID)
		if err != nil {
			return syncProviderStoreError(c, d, err, "read sync provider")
		}
		drv, err := d.Sync.Driver(p)
		if err != nil {
			return internalError(c, d, "resolve sync driver", err)
		}
		stored, err := syncsvc.DecodeConfig(conn.Config)
		if err != nil {
			return internalError(c, d, "decode connection config", err)
		}
		config, err = syncsvc.EncodeConfig(mergeConfig(drv, stored, in.Config))
		if err != nil {
			return internalError(c, d, "encode connection config", err)
		}
	}
	if err := d.Store.UpdateConnection(id, name, config, enabled); err != nil {
		return connectionStoreError(c, d, err, "update connection")
	}
	got, err := d.Store.Connection(id)
	if err != nil {
		return connectionStoreError(c, d, err, "read connection")
	}
	activeID, _, _ := d.Settings.Setting(settings.KeyActiveConnection)
	return c.JSON(http.StatusOK, connectionDTOFromStore(d, got, activeID))
}

// disconnect removes a connection. A protected connection (the local backup)
// is 403; clearing the active setting first keeps the agent's git tools from
// resolving a dangling id.
func disconnect(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	activeID, found, _ := d.Settings.Setting(settings.KeyActiveConnection)
	if found && activeID == strconv.FormatInt(id, 10) {
		if err := d.Settings.SetSetting(settings.KeyActiveConnection, ""); err != nil {
			return internalError(c, d, "clear active connection", err)
		}
	}
	if err := d.Store.DeleteConnection(id); err != nil {
		return connectionStoreError(c, d, err, "delete connection")
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// listTargets returns selectable sync destinations for a connected account
// (repos for git providers; empty for s3/local).
func listTargets(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	conn, err := d.Store.Connection(id)
	if err != nil {
		return connectionStoreError(c, d, err, "read connection")
	}
	p, err := d.Store.SyncProvider(conn.ProviderID)
	if err != nil {
		return syncProviderStoreError(c, d, err, "read sync provider")
	}
	drv, err := d.Sync.Driver(p)
	if err != nil {
		return internalError(c, d, "resolve sync driver", err)
	}
	cfg, err := syncsvc.DecodeConfig(conn.Config)
	if err != nil {
		return internalError(c, d, "decode connection config", err)
	}
	targets, err := drv.Targets(c.Request().Context(), cfg)
	if errors.Is(err, syncsvc.ErrRejected) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "the provider rejected these credentials"})
	}
	if err != nil {
		return internalError(c, d, "list sync targets", err)
	}
	if targets == nil {
		targets = []syncsvc.Target{}
	}
	return c.JSON(http.StatusOK, map[string]any{"targets": targets})
}

// pushConnection syncs the wiki through the connection's driver and records
// the outcome on the row (last_synced_at / last_error + push history).
// Driver errors are already sanitized fixed messages, so they are safe to
// surface and store; transient failures are retried with backoff inside
// Service.Push. The push path is shared with the background scheduler, so
// both record the same state.
func pushConnection(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	conn, err := d.Store.Connection(id)
	if err != nil {
		return connectionStoreError(c, d, err, "read connection")
	}
	if err := d.Sync.Push(c.Request().Context(), conn, d.Wiki.Root()); err != nil {
		return c.JSON(http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// restoreInput is the POST /api/v1/sync/connections/:id/restore body. Key
// selects a snapshot; empty = the latest.
type restoreInput struct {
	Key string `json:"key"`
}

// connectionRestorer resolves the connection's driver and type-asserts it to
// the sync.Restorer capability. Git-kind drivers are push-only today, so they
// surface a clean 400 here. Errors are raw store/driver errors — the caller
// maps them (connectionStoreError etc.) so a successful c.JSON write inside
// the mapping cannot be mistaken for a restorer.
func connectionRestorer(d Deps, id int64) (store.Connection, syncsvc.Restorer, error) {
	conn, err := d.Store.Connection(id)
	if err != nil {
		return store.Connection{}, nil, err
	}
	p, err := d.Store.SyncProvider(conn.ProviderID)
	if err != nil {
		return conn, nil, err
	}
	drv, err := d.Sync.Driver(p)
	if err != nil {
		return conn, nil, err
	}
	restorer, ok := drv.(syncsvc.Restorer)
	if !ok {
		return conn, nil, errRestoreUnsupported
	}
	return conn, restorer, nil
}

// errRestoreUnsupported marks a connection whose driver cannot restore.
var errRestoreUnsupported = errors.New("restore is not supported for this destination")

// listSnapshots returns the restore points available for a connection (S3
// objects / local backup files), newest-first. Git-kind connections return an
// empty list — there is nothing stored to restore.
func listSnapshots(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	conn, restorer, err := connectionRestorer(d, id)
	if errors.Is(err, errRestoreUnsupported) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": errRestoreUnsupported.Error()})
	}
	if err != nil {
		return connectionStoreError(c, d, err, "read connection")
	}
	cfg, err := syncsvc.DecodeConfig(conn.Config)
	if err != nil {
		return internalError(c, d, "decode connection config", err)
	}
	snaps, err := restorer.Snapshots(c.Request().Context(), cfg)
	if err != nil {
		return internalError(c, d, "list snapshots", err)
	}
	if snaps == nil {
		snaps = []syncsvc.Snapshot{}
	}
	return c.JSON(http.StatusOK, map[string]any{"snapshots": snaps})
}

// restoreConnection downloads a stored archive (the latest, or the chosen
// snapshot key) and imports it onto the wiki via the same backup-first merge
// the upload path uses (wiki.ImportFrom), then rebuilds the index and pushes
// a wiki_changed frame. The wiki is overwritten by the archive — the merge
// takes a sibling backup first, so nothing is lost. Errors are sanitized; a
// 400 means "restore not supported" or a bad/non-wiki archive.
func restoreConnection(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	var in restoreInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	conn, restorer, err := connectionRestorer(d, id)
	if errors.Is(err, errRestoreUnsupported) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": errRestoreUnsupported.Error()})
	}
	if err != nil {
		return connectionStoreError(c, d, err, "read connection")
	}
	cfg, err := syncsvc.DecodeConfig(conn.Config)
	if err != nil {
		return internalError(c, d, "decode connection config", err)
	}
	ra, size, err := restorer.Restore(c.Request().Context(), cfg, in.Key)
	if err != nil {
		// A bad/missing snapshot key is a client error (sanitized fixed
		// messages — no credentials or key names leak).
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	imported, err := d.Wiki.ImportFrom(ra, size, d.Log)
	if err != nil {
		if errors.Is(err, wiki.ErrArchiveTooLarge) {
			return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := d.Index.Sync(d.Wiki.Root(), d.Log); err != nil {
		return internalError(c, d, "reindex after restore", err)
	}
	d.publishWikiChanged()
	return c.JSON(http.StatusOK, map[string]any{
		"files":  imported.Files,
		"backup": imported.Backup,
	})
}

// setActiveConnection marks the connection the agent's git tools default to.
func setActiveConnection(c echo.Context, d Deps) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	}
	if _, err := d.Store.Connection(id); err != nil {
		return connectionStoreError(c, d, err, "read connection")
	}
	if err := d.Settings.SetSetting(settings.KeyActiveConnection, strconv.FormatInt(id, 10)); err != nil {
		return internalError(c, d, "set active connection", err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// syncProviderDTOFromStore maps a catalog row to the wire shape, resolving
// the driver's kind and fields (degraded to zero values for an unknown
// driver — the row still lists).
func syncProviderDTOFromStore(d Deps, p store.SyncProvider) syncProviderDTO {
	dto := syncProviderDTO{
		ID: p.ID, Slug: p.Slug, Name: p.Name, Driver: p.Driver,
		BaseURL: p.BaseURL, Protected: p.Protected, ConnectionCount: p.ConnectionCount,
	}
	if d.Sync == nil {
		return dto
	}
	if drv, err := d.Sync.Driver(p); err == nil {
		dto.Kind = drv.Kind()
		dto.Fields = drv.Fields()
	}
	return dto
}

// connectionDTOFromStore maps a connection row to the token-free wire shape.
func connectionDTOFromStore(d Deps, c store.Connection, activeID string) connectionDTO {
	dto := connectionDTO{
		ID: c.ID, ProviderID: c.ProviderID, ProviderSlug: c.ProviderSlug,
		ProviderName: c.ProviderName, Name: c.Name, Enabled: c.Enabled,
		Protected: c.Protected, LastSyncedAt: c.LastSyncedAt, LastError: c.LastError,
		Active: activeID != "" && strconv.FormatInt(c.ID, 10) == activeID,
		Config: map[string]any{},
		// Non-nil so the wire shape is always [] (never null) even when the
		// history read fails — the client types it as an array.
		PushHistory: []store.PushEntry{},
	}
	if id, err := syncsvc.DecodeIdentity(c.Identity); err == nil && id != (syncsvc.Identity{}) {
		dto.Identity = &id
	}
	if d.Sync != nil {
		if p, err := d.Store.SyncProvider(c.ProviderID); err == nil {
			if drv, err := d.Sync.Driver(p); err == nil {
				if cfg, err := syncsvc.DecodeConfig(c.Config); err == nil {
					dto.Config = configWire(drv, cfg)
				}
			}
		}
	}
	if history, err := d.Store.ListPushHistory(c.ID); err == nil {
		dto.PushHistory = history
	}
	return dto
}

// configWire maps a connection's stored config onto the wire: secret fields
// become has_<key> booleans (the value never leaves the store), non-secret
// fields round-trip as themselves.
func configWire(drv syncsvc.Driver, cfg syncsvc.Config) map[string]any {
	out := make(map[string]any, len(drv.Fields()))
	for _, f := range drv.Fields() {
		if f.Secret {
			out["has_"+f.Key] = cfg[f.Key] != ""
		} else {
			out[f.Key] = cfg[f.Key]
		}
	}
	return out
}

// mergeConfig applies an incoming config over the stored one: secret fields
// with an empty value keep the stored value (write-only), every other
// provided field overwrites.
func mergeConfig(drv syncsvc.Driver, stored, incoming syncsvc.Config) syncsvc.Config {
	out := make(syncsvc.Config, len(stored))
	for k, v := range stored {
		out[k] = v
	}
	for _, f := range drv.Fields() {
		v, present := incoming[f.Key]
		if !present {
			continue
		}
		if f.Secret && v == "" {
			continue
		}
		out[f.Key] = v
	}
	return out
}

// requiredFieldsMissing returns the label of the first required field missing
// from cfg, or "" when every required field is present.
func requiredFieldsMissing(drv syncsvc.Driver, cfg map[string]string) string {
	for _, f := range drv.Fields() {
		if f.Required && strings.TrimSpace(cfg[f.Key]) == "" {
			return f.Label
		}
	}
	return ""
}

// uniqueSyncSlug derives a unique slug for a user-added sync provider: the
// name reduced to lowercase alphanumerics, suffixed with -2, -3, … on a
// collision.
func uniqueSyncSlug(st *store.Store, name string) (string, error) {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	base := b.String()
	if base == "" {
		return "", errors.New("name must contain a letter or digit")
	}
	for i := 1; ; i++ {
		slug := base
		if i > 1 {
			slug = fmt.Sprintf("%s-%d", base, i)
		}
		if _, err := st.SyncProviderBySlug(slug); errors.Is(err, store.ErrSyncProviderNotFound) {
			return slug, nil
		} else if err != nil {
			return "", err
		}
	}
}

// syncProviderStoreError maps the sync_providers sentinel errors to
// 400/403/404/409; anything else is internal.
func syncProviderStoreError(c echo.Context, d Deps, err error, op string) error {
	switch {
	case errors.Is(err, store.ErrSyncProviderNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": "sync provider not found"})
	case errors.Is(err, store.ErrSyncProviderExists):
		return c.JSON(http.StatusConflict, map[string]string{"error": "a sync provider with this name already exists"})
	case errors.Is(err, store.ErrSyncProviderProtected):
		return c.JSON(http.StatusForbidden, map[string]string{"error": "this sync provider is protected and cannot be modified or deleted"})
	case errors.Is(err, store.ErrSyncProviderInUse):
		return c.JSON(http.StatusConflict, map[string]string{"error": "this sync provider still has connections"})
	default:
		return internalError(c, d, op, err)
	}
}

// connectionStoreError maps the sync_connections sentinel errors to
// 403/404; anything else is internal.
func connectionStoreError(c echo.Context, d Deps, err error, op string) error {
	switch {
	case errors.Is(err, store.ErrConnectionNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": "connection not found"})
	case errors.Is(err, store.ErrConnectionProtected):
		return c.JSON(http.StatusForbidden, map[string]string{"error": "this connection is protected and cannot be deleted"})
	default:
		return internalError(c, d, op, err)
	}
}
