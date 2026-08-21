package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	agentlib "github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/provider/anthropic"
	"github.com/shiv-source/thoth/agent/provider/openai"
	agenttools "github.com/shiv-source/thoth/agent/tools"
	"github.com/shiv-source/thoth/internal/index"
	"github.com/shiv-source/thoth/internal/store"
	"github.com/shiv-source/thoth/internal/wiki"
)

// defaultHistoryCap is the HistoryCap used when Options leaves it zero: the
// last 20 user-initiated turns are replayed to the model (T7).
const defaultHistoryCap = 20

// Option configures a Client.
type Option func(*options)

type options struct {
	provider        agentlib.Provider
	logger          *slog.Logger
	folders         []string
	historyCap      int
	maxIterations   int
	maxOutputTokens int
}

// WithProvider overrides the provider New would choose from the model id
// (tests, custom providers).
func WithProvider(p agentlib.Provider) Option { return func(o *options) { o.provider = p } }

// WithLogger sets the logger the agent loop diagnostics go to.
func WithLogger(l *slog.Logger) Option { return func(o *options) { o.logger = l } }

// WithFolders sets the folder set the rulebook fallback renders when the
// wiki has no CLAUDE.md (default: the scaffold folder set).
func WithFolders(folders []string) Option { return func(o *options) { o.folders = folders } }

// WithHistoryCap caps the replayed history to the last n user-initiated
// turns; zero selects the default of 20.
func WithHistoryCap(n int) Option { return func(o *options) { o.historyCap = n } }

// WithMaxIterations bounds the tool loop's provider turns (zero selects the
// agent library default).
func WithMaxIterations(n int) Option { return func(o *options) { o.maxIterations = n } }

// WithMaxOutputTokens bounds each provider turn's output tokens.
func WithMaxOutputTokens(n int) Option { return func(o *options) { o.maxOutputTokens = n } }

// Client is the Thoth host layer on the reusable agent library: the chat
// seam the api Hub depends on (Start with an EventWriter), driving
// agent.Agent instead of the CLI. sessionID is treated as the conversation
// id for history lookup.
type Client struct {
	model           string
	provider        agentlib.Provider
	wiki            *wiki.Wiki
	store           *store.Store
	folders         []string
	tools           *agenttools.Registry
	logger          *slog.Logger
	historyCap      int
	maxIterations   int
	maxOutputTokens int
}

// New wires a Client from the model id and API key plus the wiki (tools and
// rulebook), store (history) and index (search) dependencies. The provider is
// chosen from the model id — claude-* models use the Anthropic provider,
// gpt-* models the OpenAI-compatible provider — unless WithProvider overrides
// it. Unknown model ids error; broader provider configuration lands with T12.
func New(model, apiKey string, w *wiki.Wiki, st *store.Store, ix *index.Index, opts ...Option) (*Client, error) {
	if model == "" {
		return nil, errors.New("agent: model is required")
	}
	if w == nil {
		return nil, errors.New("agent: wiki is required")
	}
	if st == nil {
		return nil, errors.New("agent: store is required")
	}
	o := options{folders: wiki.Folders(), historyCap: defaultHistoryCap}
	for _, opt := range opts {
		opt(&o)
	}
	prov := o.provider
	if prov == nil {
		var err error
		prov, err = providerFor(model, apiKey)
		if err != nil {
			return nil, err
		}
	}
	logger := o.logger
	if logger == nil {
		logger = slog.Default()
	}
	reg, err := registry(w.Root, ix)
	if err != nil {
		return nil, fmt.Errorf("agent: build tools: %w", err)
	}
	return &Client{
		model:           model,
		provider:        prov,
		wiki:            w,
		store:           st,
		folders:         o.folders,
		tools:           reg,
		logger:          logger,
		historyCap:      o.historyCap,
		maxIterations:   o.maxIterations,
		maxOutputTokens: o.maxOutputTokens,
	}, nil
}

// providerFor maps a model id to the provider client that serves it: the
// claude-* family is Anthropic, gpt-* is OpenAI-compatible. Model families
// without a concrete provider here error until provider settings land (T12).
func providerFor(model, apiKey string) (agentlib.Provider, error) {
	switch {
	case strings.HasPrefix(model, "claude-"):
		return anthropic.New(apiKey, anthropic.WithModel(model)), nil
	case strings.HasPrefix(model, "gpt-"):
		return openai.New(apiKey, openai.WithModel(model)), nil
	default:
		return nil, fmt.Errorf("agent: no provider for model %q", model)
	}
}

// Start runs one turn for conversation sessionID, streaming events to w. It
// builds a fresh agent.Agent per turn: the system prompt is re-read from the
// rulebook so edits apply without restart, and the loop is single-turn, while
// the Hub serves many conversations at once.
func (c *Client) Start(ctx context.Context, sessionID, prompt string, w agentlib.EventWriter) error {
	if w == nil {
		return errors.New("agent: EventWriter is required")
	}
	system, err := SystemPrompt(c.wiki, c.folders)
	if err != nil {
		return err
	}
	ag, err := agentlib.New(agentlib.Options{
		Provider:        c.provider,
		Model:           c.model,
		System:          system,
		History:         History(c.store),
		HistoryCap:      c.historyCap,
		Tools:           c.tools,
		MaxIterations:   c.maxIterations,
		MaxOutputTokens: c.maxOutputTokens,
		Logger:          c.logger,
	})
	if err != nil {
		return err
	}
	return ag.Start(ctx, sessionID, prompt, w)
}
