// Package assets holds static data files served by the API. models.json is
// the single source for the llm_models seed (first boot) — edit it (no code
// change) to adjust the offered models.
package assets

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed models.json
var modelsJSON []byte

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

// ModelOptions parses the embedded models.json.
func ModelOptions() ([]Option, error) {
	var payload struct {
		Models []Option `json:"models"`
	}
	if err := json.Unmarshal(modelsJSON, &payload); err != nil {
		return nil, fmt.Errorf("parse models.json: %w", err)
	}
	return payload.Models, nil
}
