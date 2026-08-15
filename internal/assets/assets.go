// Package assets holds static data files served by the API. models.json is
// the single source for the Settings model picker — edit it (no code change)
// to adjust the offered models.
package assets

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed models.json
var modelsJSON []byte

// Option is one selectable model for the CLI's --model flag.
type Option struct {
	// Value is the --model argument ("" = the CLI's own default).
	Value    string `json:"value"`
	Label    string `json:"label"`
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
