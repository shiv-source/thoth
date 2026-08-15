package claude

// ModelOption is one selectable model for the CLI's --model flag.
type ModelOption struct {
	// Value is the --model argument ("" = the CLI's own default).
	Value string `json:"value"`
	Label string `json:"label"`
}

// ModelOptions returns the models offered by the Settings UI. The empty
// value keeps the CLI's configured default; the rest are the stable full
// model ids (aliases like opus/sonnet/fable also work but resolve to the
// latest model of that family, so the pinned ids are what the picker
// offers).
func ModelOptions() []ModelOption {
	return []ModelOption{
		{Value: "", Label: "Default (CLI)"},
		{Value: "claude-opus-5", Label: "Opus — deepest reasoning"},
		{Value: "claude-sonnet-5", Label: "Sonnet — balanced"},
		{Value: "claude-haiku-4-5-20251001", Label: "Haiku — fastest"},
		{Value: "claude-fable-5", Label: "Fable"},
	}
}
