# Frontend

React 19 + TypeScript (strict) + Vite + Tailwind CSS v4, bundled and embedded into the Go binary at build time. Package manager: **pnpm**.

## Structure

```
web/src/
├── api/client.ts        # typed REST client, zod-validated responses
├── ws/chat.ts           # ChatSocket: protocol frames, reconnect/resume
├── hooks/               # useChat, useSearch
├── components/          # ChatPanel, Composer, MessageItem, TopBar,
│                        # SearchPanel, NoteViewer, SettingsModal, Sidebar
└── index.css            # Tailwind v4 @theme tokens
```

## Components

| Component | Role |
|---|---|
| `Sidebar` | Brand, live search, recursive collapsible wiki tree (chevron folders, doc glyphs, indent guides); clicking a note opens the viewer panel |
| `SearchPanel` | Debounced search (300 ms), highlighted snippets; ↑/↓ + Enter open the highlighted note, Esc clears |
| `TopBar` | Conversation title (first message-derived), "New chat" (local reset), gear → settings modal |
| `ChatPanel` | Owns the socket lifecycle (created in an effect, closed on unmount); message list + scroll; tool status line ("Reading `path`") while a tool runs |
| `Composer` | Textarea (Enter = send, Shift+Enter = newline); Send **and** Stop — sending while streaming supersedes the running turn |
| `MessageItem` | User bubbles (plain text) vs assistant (react-markdown + GFM) with a streaming caret |
| `NoteViewer` | Slide-over markdown panel (Esc or ✕ closes); "Copy raw" copies the note to the clipboard |
| `SettingsModal` | The 6 config fields with save feedback; backdrop, ✕, and Esc close |

## Hooks

- **useChat** — messages + streaming + conversationId state; maps server frames to UI messages; `error` frames render as a visible ⚠️ message; tracks `conversation_id` from `turn_done`; `tool_activity` exposes `lastTool` (path when the input JSON carries one, else the tool name), cleared on `turn_done`/`error`; `reset()` clears everything locally without a server call
- **useSearch** — 300 ms debounce with a sequence guard, so slow older responses can't overwrite newer ones

## WebSocket client

`ChatSocket` sends `send`/`cancel`/`resume`, forwards `assistant_*`/`tool_activity`/`turn_done`/`error` frames, and reconnects exactly once after 1 s — sending `resume` from `onopen` so the turn re-syncs.

## Design system

Tokens in `index.css` (`@theme`) resolve to CSS custom properties that flip under `prefers-color-scheme`, so one semantic class (`bg-surface`, `text-ink`, `border-line`) works in both themes: slate + white surfaces in light, slate night in dark, emerald accent (`#059669` → `#34d399`). The five semantic color groups are app/surface/raised, line (borders), subtle/ink/heading (text). Display type is Fraunces (self-hosted via `@fontsource-variable` — no runtime network); body is the system stack. The typography plugin powers `prose-*` content styles with Fraunces headings.

Dark mode follows the OS — no toggle needed in a local app.

## Development

`pnpm dev` runs Vite with `/api` and `/ws` proxied to `127.0.0.1:8333`, so the frontend can be developed against a live server. For production, `make web` builds and syncs `web/dist` into `internal/webui/dist` for embedding.
