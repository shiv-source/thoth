// Package openai is a Provider over the OpenAI Chat Completions API (POST
// /v1/chat/completions with stream:true). Because it targets the OpenAI wire
// shape, it also serves every OpenAI-compatible provider — DeepSeek, Qwen,
// GLM, Mistral, xAI and friends — via WithBaseURL. It imports only the public
// agent API and the shared SSE transport; no third-party SDKs.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/shiv-source/thoth/agent"
	"github.com/shiv-source/thoth/agent/transport"
)

const (
	defaultBaseURL   = "https://api.openai.com"
	defaultModel     = "gpt-5.6-mini"
	defaultMaxTokens = 4096
	chatPath         = "/v1/chat/completions"
)

// Option configures a Client.
type Option func(*options)

type options struct {
	baseURL    string
	model      string
	maxTokens  int
	httpClient *http.Client
	headers    map[string]string
}

// WithBaseURL overrides the API base URL (default https://api.openai.com).
// OpenAI-compatible providers pass their own origin here.
func WithBaseURL(u string) Option {
	return func(o *options) { o.baseURL = u }
}

// WithModel sets the model id (default gpt-5.6-mini).
func WithModel(m string) Option {
	return func(o *options) { o.model = m }
}

// WithMaxTokens sets the max_tokens bound used when the Request leaves it zero
// (default 4096).
func WithMaxTokens(n int) Option {
	return func(o *options) { o.maxTokens = n }
}

// WithHTTPClient overrides the HTTP client used for requests (default
// http.DefaultClient).
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) { o.httpClient = c }
}

// WithHeaders adds extra request headers sent on every request (e.g. Portkey's
// x-portkey-* routing headers). nil or an empty map adds nothing.
func WithHeaders(headers map[string]string) Option {
	return func(o *options) { o.headers = headers }
}

// Client is an OpenAI-compatible provider. Create one with New; it is safe for
// concurrent use.
type Client struct {
	apiKey    string
	baseURL   string
	model     string
	maxTokens int
	http      *http.Client
	headers   map[string]string
}

// New returns an OpenAI-compatible client for apiKey. Configure it with the
// With* options.
func New(apiKey string, opts ...Option) *Client {
	o := options{
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		maxTokens:  defaultMaxTokens,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{
		apiKey: apiKey, baseURL: o.baseURL, model: o.model, maxTokens: o.maxTokens, http: o.httpClient, headers: o.headers,
	}
}

var _ agent.Provider = (*Client)(nil)

// Stream starts a turn and returns a stream of normalized deltas. The caller
// must Close the stream; cancelling ctx aborts the underlying request.
func (c *Client) Stream(ctx context.Context, req agent.Request) (agent.Stream, error) {
	wire, err := buildRequest(c, req)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("openai: encode request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+chatPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	httpReq.Header.Set("authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("content-type", "application/json")
	for name, value := range c.headers {
		httpReq.Header.Set(name, value)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("openai: request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai: api returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return newStream(resp.Body), nil
}

// wireRequest is the OpenAI Chat Completions request body.
type wireRequest struct {
	Model      string        `json:"model"`
	Messages   []wireMessage `json:"messages"`
	Tools      []wireTool    `json:"tools,omitempty"`
	ToolChoice string        `json:"tool_choice,omitempty"`
	MaxTokens  int           `json:"max_tokens,omitempty"`
	Stream     bool          `json:"stream"`
}

type wireMessage struct {
	Role       string         `json:"role"`
	Content    any            `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
}

type wireTextPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type wireTool struct {
	Type     string       `json:"type"`
	Function wireFunction `json:"function"`
}

type wireFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type wireToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function wireCallFunc `json:"function"`
}

type wireCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// buildRequest maps a normalized Request onto the OpenAI wire shape. The
// system prompt becomes the first system message; tool results become
// role:"tool" messages keyed by their tool_call_id.
func buildRequest(c *Client, req agent.Request) (wireRequest, error) {
	messages, err := wireMessages(req.Messages)
	if err != nil {
		return wireRequest{}, err
	}
	if req.System != "" {
		messages = append([]wireMessage{{Role: "system", Content: req.System}}, messages...)
	}
	w := wireRequest{
		Model:     c.model,
		Messages:  messages,
		MaxTokens: req.MaxTokens,
		Stream:    true,
	}
	if w.MaxTokens == 0 {
		w.MaxTokens = c.maxTokens
	}
	if len(req.Tools) > 0 {
		w.Tools = make([]wireTool, 0, len(req.Tools))
		w.ToolChoice = "auto"
		for _, t := range req.Tools {
			w.Tools = append(w.Tools, wireTool{
				Type:     "function",
				Function: wireFunction{Name: t.Name, Description: t.Description, Parameters: t.Schema},
			})
		}
	}
	return w, nil
}

func wireMessages(msgs []agent.Message) ([]wireMessage, error) {
	out := make([]wireMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case agent.RoleUser, agent.RoleAssistant:
			w, err := wireTurnMessage(m)
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		case agent.RoleTool:
			for _, b := range m.Content {
				if b.Type != agent.BlockToolResult {
					continue
				}
				out = append(out, wireMessage{Role: "tool", ToolCallID: b.ToolUseID, Content: b.Text})
			}
		default:
			return nil, fmt.Errorf("openai: unknown role %q", m.Role)
		}
	}
	return out, nil
}

// wireTurnMessage maps a user or assistant message: text blocks become content
// parts, tool_use blocks become function tool_calls (assistant only), and
// thinking blocks are dropped — reasoning has no OpenAI wire form. Tool results
// ride in separate tool messages, so a tool_result block here is unexpected.
func wireTurnMessage(m agent.Message) (wireMessage, error) {
	var parts []any
	var calls []wireToolCall
	for _, b := range m.Content {
		switch b.Type {
		case agent.BlockText:
			parts = append(parts, wireTextPart{Type: "text", Text: b.Text})
		case agent.BlockThinking:
			// reasoning is not representable on the OpenAI wire; skipped
		case agent.BlockToolUse:
			if m.Role != agent.RoleAssistant {
				return wireMessage{}, fmt.Errorf("openai: tool_use block in %s message", m.Role)
			}
			args, err := toolArguments(b.Input)
			if err != nil {
				return wireMessage{}, err
			}
			calls = append(calls, wireToolCall{
				ID:       b.ID,
				Type:     "function",
				Function: wireCallFunc{Name: b.ToolName, Arguments: args},
			})
		case agent.BlockToolResult:
			return wireMessage{}, fmt.Errorf("openai: tool_result block in %s message", m.Role)
		default:
			return wireMessage{}, fmt.Errorf("openai: unknown block type %q", b.Type)
		}
	}
	w := wireMessage{Role: m.Role}
	if len(parts) > 0 {
		w.Content = parts
	}
	if len(calls) > 0 {
		w.ToolCalls = calls
	}
	return w, nil
}

// toolArguments renders a tool_use input as the JSON-encoded arguments string
// the OpenAI wire requires.
func toolArguments(input map[string]any) (string, error) {
	if input == nil {
		input = map[string]any{}
	}
	b, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("openai: encode tool arguments: %w", err)
	}
	return string(b), nil
}

// stopReason maps an OpenAI finish_reason onto the loop's normalized stop
// reason, keeping the Anthropic and OpenAI providers symmetric.
func stopReason(finish string) string {
	switch finish {
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	default: // "stop", "content_filter", ...
		return "end_turn"
	}
}

// chunk is the streaming chat.completion.chunk wire shape. Fields that appear
// only on some providers (reasoning_content, top-level usage) decode to zero
// values when absent.
type chunk struct {
	Choices []struct {
		Delta struct {
			Content          string          `json:"content"`
			ReasoningContent string          `json:"reasoning_content"`
			ToolCalls        []toolCallDelta `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type toolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// stream decodes the OpenAI SSE response into normalized deltas. Tool call
// arguments arrive as split JSON fragments indexed by their tool_call index;
// they accumulate per index and are re-emitted whole on each fragment, exactly
// like the Anthropic provider. A tool call declared but never given arguments
// closes as an empty object. The stop delta is held until the stream ends
// (EOF or "[DONE]") so a trailing usage chunk is captured first — OpenAI sends
// usage in a final chunk after the finish_reason chunk.
type stream struct {
	frames       *transport.SSEReader
	body         io.Closer
	usage        agent.Usage
	pendingStop  string
	queue        []agent.Delta
	toolInput    map[int]string
	toolID       map[int]string
	toolName     map[int]string
	inputEmitted map[int]bool
	ended        bool
}

func newStream(body io.ReadCloser) *stream {
	return &stream{
		frames:       transport.NewSSEReader(body),
		body:         body,
		toolInput:    map[int]string{},
		toolID:       map[int]string{},
		toolName:     map[int]string{},
		inputEmitted: map[int]bool{},
	}
}

// Next returns the next delta in order, then io.EOF when the turn ends.
func (s *stream) Next() (agent.Delta, error) {
	if s.ended {
		return agent.Delta{}, io.EOF
	}
	for {
		if len(s.queue) > 0 {
			d := s.queue[0]
			s.queue = s.queue[1:]
			if d.Kind == agent.DeltaStop {
				s.ended = true
			}
			return d, nil
		}
		f, err := s.frames.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if s.pendingStop != "" {
					s.finishTurn()
					continue // drain the freshly queued deltas
				}
				s.ended = true
				return agent.Delta{}, io.EOF
			}
			s.ended = true
			return agent.Delta{}, fmt.Errorf("openai: sse: %w", err)
		}
		if err := s.handle(f); err != nil {
			s.ended = true
			return agent.Delta{}, err
		}
	}
}

// finishTurn queues the trailing tool-call closures and the stop delta once
// the stream has ended; usage has already been captured from the final chunk.
func (s *stream) finishTurn() {
	for idx, id := range s.toolID {
		if id != "" && s.toolName[idx] != "" && !s.inputEmitted[idx] {
			s.queue = append(s.queue, agent.ToolInputDelta(id, s.toolName[idx], "{}"))
		}
	}
	s.queue = append(s.queue, agent.StopDelta(s.pendingStop))
	s.pendingStop = ""
}

// Usage is valid once Next has returned io.EOF or a stop delta.
func (s *stream) Usage() agent.Usage { return s.usage }

// Close releases the response body.
func (s *stream) Close() error { return s.body.Close() }

func (s *stream) handle(f transport.Frame) error {
	var c chunk
	if err := f.Decode(&c); err != nil {
		return fmt.Errorf("openai: %w", err)
	}
	if c.Usage != nil {
		s.usage.InputTokens = c.Usage.PromptTokens
		s.usage.OutputTokens = c.Usage.CompletionTokens
	}
	if c.Error != nil {
		return fmt.Errorf("openai: %s: %s", c.Error.Type, c.Error.Message)
	}
	for _, ch := range c.Choices {
		if d := ch.Delta.Content; d != "" {
			s.queue = append(s.queue, agent.TextDelta(d))
		}
		if d := ch.Delta.ReasoningContent; d != "" {
			s.queue = append(s.queue, agent.ThinkingDelta(d))
		}
		for _, tc := range ch.Delta.ToolCalls {
			idx := tc.Index
			if tc.ID != "" {
				s.toolID[idx] = tc.ID
			}
			if tc.Function.Name != "" {
				s.toolName[idx] = tc.Function.Name
			}
			if tc.Function.Arguments == "" {
				continue
			}
			if s.toolID[idx] == "" || s.toolName[idx] == "" {
				return fmt.Errorf("openai: tool call arguments before id/name at index %d", idx)
			}
			s.toolInput[idx] += tc.Function.Arguments
			s.inputEmitted[idx] = true
			s.queue = append(s.queue, agent.ToolInputDelta(s.toolID[idx], s.toolName[idx], s.toolInput[idx]))
		}
		if fr := ch.FinishReason; fr != "" {
			s.pendingStop = stopReason(fr)
		}
	}
	return nil
}
