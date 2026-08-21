// Package model is the normalized conversation model every provider maps to
// and from: messages of content blocks (text | tool_use | tool_result |
// thinking), the streaming Delta fragments, and the Builder that accumulates
// deltas back into a message. It has no dependencies on the rest of the agent
// library.
package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Role constants for Message.Role.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool" // a message carrying one or more tool_result blocks
)

// BlockType enumerates the normalized content-block shapes every provider
// maps to and from.
type BlockType string

const (
	BlockText       BlockType = "text"
	BlockToolUse    BlockType = "tool_use"
	BlockToolResult BlockType = "tool_result"
	BlockThinking   BlockType = "thinking"
)

// Block is one content block inside a Message. Exactly one of the fields is
// meaningful per Type:
//
//   - text / thinking: Text
//   - tool_use: ID, ToolName, Input
//   - tool_result: ToolUseID (the tool_use it answers), Text, IsError
type Block struct {
	Type      BlockType
	Text      string
	ID        string         // tool_use: tool call id (correlates with its result)
	ToolName  string         // tool_use: tool invoked
	Input     map[string]any // tool_use: tool arguments
	ToolUseID string         // tool_result: the tool_use id being answered
	IsError   bool           // tool_result: whether the tool failed
}

// NewTextBlock returns a text block with the given content.
func NewTextBlock(text string) Block {
	return Block{Type: BlockText, Text: text}
}

// NewThinkingBlock returns a thinking block with the given reasoning text.
func NewThinkingBlock(text string) Block {
	return Block{Type: BlockThinking, Text: text}
}

// NewToolUseBlock returns a tool_use block invoking name with input.
func NewToolUseBlock(id, name string, input map[string]any) Block {
	return Block{Type: BlockToolUse, ID: id, ToolName: name, Input: input}
}

// NewToolResultBlock returns a tool_result block answering the tool_use with
// the given id.
func NewToolResultBlock(toolUseID, content string, isError bool) Block {
	return Block{Type: BlockToolResult, ToolUseID: toolUseID, Text: content, IsError: isError}
}

// ToolResult renders the tool_result block that answers this tool_use block.
func (b Block) ToolResult(content string, isError bool) Block {
	return NewToolResultBlock(b.ID, content, isError)
}

// wireBlock is the canonical JSON shape a Block serializes to; ParseBlock
// accepts it (and, for thinking, the CLI's text-bearing variant).
type wireBlock struct {
	Type      BlockType       `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// MarshalJSON renders the block in the canonical wire shape so it round-trips
// through ParseBlock without loss.
func (b Block) MarshalJSON() ([]byte, error) {
	w := wireBlock{Type: b.Type}
	switch b.Type {
	case BlockText:
		w.Text = b.Text
	case BlockThinking:
		w.Thinking = b.Text
	case BlockToolUse:
		w.ID = b.ID
		w.Name = b.ToolName
		if b.Input != nil {
			raw, err := json.Marshal(b.Input)
			if err != nil {
				return nil, fmt.Errorf("model: marshal tool input: %w", err)
			}
			w.Input = raw
		}
	case BlockToolResult:
		w.ToolUseID = b.ToolUseID
		w.Content = b.Text
		w.IsError = b.IsError
	default:
		return nil, fmt.Errorf("model: cannot marshal unknown block type %q", b.Type)
	}
	return json.Marshal(w)
}

// UnmarshalJSON lets Block decode directly out of a JSON array of blocks.
func (b *Block) UnmarshalJSON(raw []byte) error {
	parsed, err := ParseBlock(raw)
	if err != nil {
		return err
	}
	*b = parsed
	return nil
}

// ParseBlock builds a Block from one raw wire block. Thinking blocks accept
// both the "thinking" key (Anthropic wire) and the "text" key (CLI stream).
func ParseBlock(raw []byte) (Block, error) {
	var w wireBlock
	if err := decode(raw, &w); err != nil {
		return Block{}, fmt.Errorf("model: parse block: %w", err)
	}
	switch w.Type {
	case BlockText:
		return NewTextBlock(w.Text), nil
	case BlockThinking:
		text := w.Thinking
		if text == "" {
			text = w.Text
		}
		return NewThinkingBlock(text), nil
	case BlockToolUse:
		input := map[string]any{}
		if len(w.Input) > 0 {
			if err := decode(w.Input, &input); err != nil {
				return Block{}, fmt.Errorf("model: parse tool input: %w", err)
			}
		}
		return NewToolUseBlock(w.ID, w.Name, input), nil
	case BlockToolResult:
		return NewToolResultBlock(w.ToolUseID, w.Content, w.IsError), nil
	default:
		return Block{}, fmt.Errorf("model: unknown block type %q", w.Type)
	}
}

// Message is one turn of the conversation history.
type Message struct {
	Role    string
	Content []Block
}

// ParseMessage builds a Message from a raw {"role","content"} JSON object.
func ParseMessage(raw []byte) (Message, error) {
	var m struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return Message{}, fmt.Errorf("model: parse message: %w", err)
	}
	msg := Message{Role: m.Role}
	if len(m.Content) > 0 {
		if err := json.Unmarshal(m.Content, &msg.Content); err != nil {
			return Message{}, fmt.Errorf("model: parse message content: %w", err)
		}
	}
	return msg, nil
}

// MarshalJSON renders the message so it round-trips through ParseMessage.
func (m Message) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Role    string  `json:"role"`
		Content []Block `json:"content"`
	}{Role: m.Role, Content: m.Content})
}

// decode unmarshals raw into v, preserving numbers so tool inputs round-trip
// losslessly.
func decode(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	return dec.Decode(v)
}
