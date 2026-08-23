package sync

import (
	"fmt"
	"net/http"

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
