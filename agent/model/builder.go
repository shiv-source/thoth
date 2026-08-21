package model

import "fmt"

// Builder accumulates the streaming deltas of one assistant turn into the
// message the loop acts on. Consecutive text and thinking deltas fold into a
// single block; tool_use_input deltas refresh the tracked tool call. Tool
// input stays as raw JSON until Message, so fragments that only become valid
// at the end of the turn are not lost.
type Builder struct {
	blocks  []Block
	cur     BlockType // block kind currently being accumulated
	curIdx  int
	toolIdx map[string]int
	input   map[string]string // tool_use_id -> accumulated raw JSON fragment
}

// NewBuilder returns an empty builder.
func NewBuilder() *Builder {
	return &Builder{curIdx: -1, toolIdx: map[string]int{}, input: map[string]string{}}
}

// Add folds one delta into the accumulated message. Stop deltas are ignored;
// the loop ends the turn on them.
func (b *Builder) Add(d Delta) {
	switch d.Kind {
	case DeltaText:
		if b.cur == BlockText {
			b.blocks[b.curIdx].Text += d.Text
			return
		}
		b.cur, b.curIdx = BlockText, len(b.blocks)
		b.blocks = append(b.blocks, NewTextBlock(d.Text))
	case DeltaThinking:
		if b.cur == BlockThinking {
			b.blocks[b.curIdx].Text += d.Text
			return
		}
		b.cur, b.curIdx = BlockThinking, len(b.blocks)
		b.blocks = append(b.blocks, NewThinkingBlock(d.Text))
	case DeltaToolInput:
		idx, ok := b.toolIdx[d.ToolUseID]
		if !ok {
			idx = len(b.blocks)
			b.toolIdx[d.ToolUseID] = idx
			b.blocks = append(b.blocks, NewToolUseBlock(d.ToolUseID, d.ToolName, nil))
		}
		b.blocks[idx].ToolName = d.ToolName
		b.input[d.ToolUseID] = d.Input
		b.cur = "" // a tool call closes the run of text/thinking blocks
	}
}

// Message returns the accumulated assistant message, with tool inputs parsed
// from their raw JSON fragments. A fragment that never became valid JSON is an
// error (the turn ended mid-argument).
func (b *Builder) Message() (Message, error) {
	blocks := make([]Block, 0, len(b.blocks))
	for _, blk := range b.blocks {
		if blk.Type == BlockToolUse {
			var input map[string]any
			if err := decode([]byte(b.input[blk.ID]), &input); err != nil {
				return Message{}, fmt.Errorf("model: tool %q input is not valid JSON: %w", blk.ToolName, err)
			}
			blk.Input = input
		}
		blocks = append(blocks, blk)
	}
	return Message{Role: RoleAssistant, Content: blocks}, nil
}
