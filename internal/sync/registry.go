package sync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/shiv-source/thoth/internal/github"
	"github.com/shiv-source/thoth/internal/store"
)

// Service composes the driver registry with the store — the single handle the
// API layer depends on. Drivers are stateless; the provider's base_url
// override is bound per call, so a "GitHub Enterprise" provider row gets a
// GitHub driver pointed at its endpoint without any shared state.
type Service struct {
	Store   *store.Store
	drivers map[string]func(baseURL string) Driver
}

// NewService wires the built-in drivers. hc is the HTTP client the git REST
// clients use (nil → http.DefaultClient); tests inject a stub server client.
func NewService(st *store.Store, hc *http.Client) *Service {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Service{
		Store: st,
		drivers: map[string]func(baseURL string) Driver{
			"github": func(baseURL string) Driver {
				c := github.New(hc)
				if baseURL != "" {
					c = c.WithBaseURL(baseURL)
				}
				return &gitDriver{svc: &githubService{client: c}}
			},
			"gitlab": func(baseURL string) Driver {
				return &gitDriver{svc: newGitlabService(hc, baseURL)}
			},
			"s3": func(baseURL string) Driver {
				return &s3Driver{endpoint: baseURL}
			},
			"local": func(string) Driver { return &localDriver{} },
		},
	}
}

// Driver returns the driver for a provider row, binding the provider's
// base_url override (empty = the driver's default endpoint).
func (s *Service) Driver(p store.SyncProvider) (Driver, error) {
	newDriver, ok := s.drivers[p.Driver]
	if !ok {
		return nil, fmt.Errorf("unknown sync driver %q", p.Driver)
	}
	return newDriver(p.BaseURL), nil
}

// KnownDriver reports whether driver is a registered driver id (github,
// gitlab, s3, local). The create-provider handler validates against it so a
// user-added row always resolves at runtime.
func (s *Service) KnownDriver(driver string) bool {
	_, ok := s.drivers[driver]
	return ok
}

// Push syncs the wiki at root through connection c, retrying transient
// failures with backoff, and records the outcome (last_synced_at/last_error +
// push history) on the row. It is the single push path shared by the API
// handler and the background scheduler, so both record the same state. The
// returned error is the driver's sanitized message ("" on success).
func (s *Service) Push(ctx context.Context, c store.Connection, root string) error {
	p, err := s.Store.SyncProvider(c.ProviderID)
	if err != nil {
		return err
	}
	drv, err := s.Driver(p)
	if err != nil {
		return err
	}
	cfg, err := DecodeConfig(c.Config)
	if err != nil {
		return err
	}
	ident, err := DecodeIdentity(c.Identity)
	if err != nil {
		return err
	}
	err = pushWithRetry(ctx, drv, cfg, root, ident)
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	_ = s.Store.SetConnectionSyncResult(c.ID, err == nil, detail)
	return err
}

// pushAttempts caps how many times a transient failure is retried before the
// final error is surfaced; pushBackoffBase is the exponential backoff step.
const (
	pushAttempts    = 3
	pushBackoffBase = 500 * time.Millisecond
)

// pushWithRetry runs drv.Push, retrying only ErrRetryable failures (network
// flakes, server faults) with exponential backoff. A permanent error returns
// immediately. The final transient error is annotated with the retry count so
// last_error surfaces the retry state.
func pushWithRetry(ctx context.Context, drv Driver, cfg Config, root string, ident Identity) error {
	var err error
	for attempt := 1; attempt <= pushAttempts; attempt++ {
		err = drv.Push(ctx, cfg, root, ident)
		if err == nil || !errors.Is(err, ErrRetryable) {
			return err
		}
		if attempt == pushAttempts {
			return retryable(fmt.Sprintf("%s (retried %d times)", err.Error(), pushAttempts-1))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pushBackoffBase * time.Duration(attempt)):
		}
	}
	return err
}
