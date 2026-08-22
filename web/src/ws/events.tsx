// WS frame "type" values — the single source of truth for the chat protocol's
// event names on the web side. The server (internal/api/chat.go) reads and
// writes the same strings; docs/api.md is the authoritative protocol reference,
// and the values below must never drift from it.
//
// Two directions:
//   - ServerEvent — frames the server pushes to the client (ServerMessage).
//   - ClientEvent — frames the client sends to the server (ChatSocket).
//
// Each const object doubles as its own type: `(typeof ServerEvent)['TurnDone']`
// is the literal wire string, and `ServerEvent` (the type) is the union of all
// of them. Using the constants everywhere (never a bare literal) keeps the
// wire names in exactly one place and gives autocomplete + rename refactors
// for free.

export const ServerEvent = {
    AssistantStart: 'assistant_start',
    AssistantThinking: 'assistant_thinking',
    AssistantDelta: 'assistant_delta',
    ToolActivity: 'tool_activity',
    TurnDone: 'turn_done',
    WikiChanged: 'wiki_changed',
    Error: 'error'
} as const

export type ServerEvent = (typeof ServerEvent)[keyof typeof ServerEvent]

export const ClientEvent = {
    Send: 'send',
    Cancel: 'cancel',
    Resume: 'resume',
    Open: 'open',
    NewChat: 'new_chat',
    Presence: 'presence'
} as const

export type ClientEvent = (typeof ClientEvent)[keyof typeof ClientEvent]
