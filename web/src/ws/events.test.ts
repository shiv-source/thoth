import { describe, expect, it } from 'vitest'
import { ClientEvent, ServerEvent } from './events'

// The enum values are the wire strings of the chat protocol — they must
// never drift from the server contract (internal/api/chat.go · docs/api.md).
// Pin them so a rename can't silently change the bytes on the wire.
describe('ServerEvent wire names', () => {
    it('matches the frames the server can push', () => {
        expect(ServerEvent).toEqual({
            AssistantStart: 'assistant_start',
            AssistantThinking: 'assistant_thinking',
            AssistantDelta: 'assistant_delta',
            ToolActivity: 'tool_activity',
            TurnDone: 'turn_done',
            WikiChanged: 'wiki_changed',
            Error: 'error'
        })
    })
})

describe('ClientEvent wire names', () => {
    it('matches the frames the client can send', () => {
        expect(ClientEvent).toEqual({
            Send: 'send',
            Cancel: 'cancel',
            Resume: 'resume',
            Open: 'open',
            NewChat: 'new_chat',
            Presence: 'presence'
        })
    })
})
