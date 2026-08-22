// WS frame "type" values — the single source of truth for the chat protocol's
// event names on the web side. The server (internal/api/chat.go) reads and
// writes the same strings; docs/api.md is the authoritative protocol reference,
// and the enum values below must never drift from it.
//
// Two directions:
//   - ServerEvent — frames the server pushes to the client (ServerMessage).
//   - ClientEvent — frames the client sends to the server (ChatSocket).
//
// Using the enum everywhere (never a bare literal) keeps the wire names in
// exactly one place and gives autocomplete + rename refactors for free.

// ServerEvent is the "type" discriminant of every ServerMessage the server
// can push. A malformed or unknown type is dropped defensively by ChatSocket.
export enum ServerEvent {
    AssistantStart = 'assistant_start',
    AssistantThinking = 'assistant_thinking',
    AssistantDelta = 'assistant_delta',
    ToolActivity = 'tool_activity',
    TurnDone = 'turn_done',
    WikiChanged = 'wiki_changed',
    Error = 'error'
}

// ClientEvent is the "type" field of every frame ChatSocket sends to the
// server.
export enum ClientEvent {
    Send = 'send',
    Cancel = 'cancel',
    Resume = 'resume',
    Open = 'open',
    NewChat = 'new_chat',
    Presence = 'presence'
}
