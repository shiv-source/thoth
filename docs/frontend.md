# Frontend

React 19 + TypeScript (strict) + Vite + Tailwind CSS v4, bundled and embedded into the Go binary at build time. Package manager: **pnpm**.

## Structure

```
web/src/
├── api/client.ts        # typed REST client, zod-validated responses
├── ws/chat.ts           # ChatSocket: protocol frames, reconnect/resume
├── hooks/               # useChat, useSearch
├── components/          # ChatPanel, Composer, MessageItem,
│                        # SearchPanel, NoteViewer, SettingsPanel, Sidebar
└── index.css            # Tailwind v4 @theme tokens
```

## Components

| Component | Role |
|---|---|
| `ChatPanel` | Owns the socket lifecycle (created in an effect, closed on unmount); message list + scroll |
| `Composer` | Textarea (Enter = send, Shift+Enter = newline); Send **and** Stop — sending while streaming supersedes the running turn |
| `MessageItem` | User bubbles (plain text) vs assistant (react-markdown + GFM) with a streaming caret |
| `SearchPanel` | Debounced search, highlighted snippets, opens the viewer |
| `NoteViewer` | Modal markdown preview; backdrop click closes |
| `SettingsPanel` | The 6 config fields with save feedback |
| `Sidebar` | Brand, tabs (search/tree/settings), two-level wiki tree |

## Hooks

- **useChat** — messages + streaming + conversationId state; maps server frames to UI messages; `error` frames render as a visible ⚠️ message; tracks `conversation_id` from `turn_done`
- **useSearch** — 300 ms debounce with a sequence guard, so slow older responses can't overwrite newer ones

## WebSocket client

`ChatSocket` sends `send`/`cancel`/`resume`, forwards `assistant_*`/`tool_activity`/`turn_done`/`error` frames, and reconnects exactly once after 1 s — sending `resume` from `onopen` so the turn re-syncs.

## Design system

Tokens in `index.css` (`@theme`): warm paper palette (`#fbf8f1`…), ink neutrals, amber accent (`#d97706` family), and a night mode via `prefers-color-scheme`. Display type is Fraunces (self-hosted via `@fontsource-variable` — no runtime network); body is the system stack. The typography plugin powers `prose-*` content styles.

Dark mode follows the OS — no toggle needed in a local app.

## Development

`pnpm dev` runs Vite with `/api` and `/ws` proxied to `127.0.0.1:8333`, so the frontend can be developed against a live server. For production, `make web` builds and syncs `web/dist` into `internal/webui/dist` for embedding.
