// Package assets holds static data files served by the API. llm-providers.json
// is the single source for the llm_models seed (first boot) — edit it (no code
// change) to adjust the offered models. sync-providers.json is the matching
// seed for the sync_providers catalog.
package assets

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed llm-providers.json
var modelsJSON []byte

//go:embed sync-providers.json
var syncProvidersJSON []byte

// Option is one selectable model for the CLI's --model flag. Name and Tag
// are separate fields: the UI renders the tag as secondary text, and the
// seeded llm_models table keeps them in separate columns.
type Option struct {
	// Value is the --model argument.
	Value    string `json:"value"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Provider string `json:"provider"`
}

// ModelOptions parses the embedded llm-providers.json.
func ModelOptions() ([]Option, error) {
	var payload struct {
		Models []Option `json:"models"`
	}
	if err := json.Unmarshal(modelsJSON, &payload); err != nil {
		return nil, fmt.Errorf("parse llm-providers.json: %w", err)
	}
	return payload.Models, nil
}

// SyncProviderOption is one built-in sync provider for the sync_providers
// seed. Driver selects the sync implementation; Protected marks first-class
// providers the user can neither edit nor delete (the local backup).
type SyncProviderOption struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Driver    string `json:"driver"`
	BaseURL   string `json:"base_url"`
	Protected bool   `json:"protected"`
}

// SyncProviderOptions parses the embedded sync-providers.json — the built-in
// sync_providers seed, mirrored by migration 0012.
func SyncProviderOptions() ([]SyncProviderOption, error) {
	var payload struct {
		Providers []SyncProviderOption `json:"providers"`
	}
	if err := json.Unmarshal(syncProvidersJSON, &payload); err != nil {
		return nil, fmt.Errorf("parse sync-providers.json: %w", err)
	}
	return payload.Providers, nil
}
