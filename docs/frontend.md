# Frontend

React 19 + TypeScript (strict) + Vite + Tailwind CSS v4, bundled and embedded into the Go binary at build time. Package manager: **pnpm**.

## Structure

```
web/src/
├── api/client.ts        # typed REST client, zod-validated responses
├── ws/chat.ts           # ChatSocket: protocol frames, reconnect/resume
├── hooks/               # useChat, useSearch, useConversationRoute
├── store/               # Redux Toolkit: slices (health, settings,
│                        # conversations, chat, connection) + typed hooks
├── components/          # ChatPanel, Composer, MessageItem, TopBar,
│                        # Sidebar (Chats + Wiki), SearchPanel, NoteViewer,
│                        # SettingsView, Card, DashboardView, ActivityChart,
│                        # dashboardMock, SetupScreen, Toast, Sidebar
└── index.css            # Tailwind v4 @theme tokens
```

## Components

| Component | Role |
|---|---|
| `Sidebar` | Brand, live search, recursive collapsible wiki tree (chevron folders, doc glyphs, indent guides); clicking a note opens the viewer panel |
| `SearchPanel` | Debounced search (300 ms), highlighted snippets; ↑/↓ + Enter open the highlighted note, Esc clears. The query rides the URL (`?q=`, replaceState while typing — reload/back/forward restore it), and committed searches (Enter, or opening a result) land in the Redux `searchHistory` slice — shown as "Recent searches" (with Clear) when the box is empty, on the search page only |
| `TopBar` | Conversation title (first message-derived), conversation-history dropdown, "New chat" (local reset), gear → settings view |

| `ChatPanel` | Owns the socket lifecycle (created in an effect, closed on unmount); message list + scroll; tool status line ("Reading `path`") while a tool runs; surfaces the one-shot reconnect failure as an error toast |
| `Composer` | Textarea (Enter = send, Shift+Enter = newline); Send **and** Stop — sending while streaming supersedes the running turn |
| `Sidebar` | Brand header, then a **Chats** section (New chat + conversation history grouped Today/Yesterday/Previous 7 days/Older, dates on hover, active rail, loading/error/empty states, re-fetches on URL change) above the **Wiki** section (search + `WikiTree`) |
| `MessageItem` | User bubbles (plain text) vs assistant (react-markdown + GFM) with a streaming caret |
| `NoteViewer` | Slide-over markdown panel (Esc or ✕ closes); "Copy raw" copies the note to the clipboard (+ success toast) |
| `Tree` | Reusable, accessible folder tree: collapsible chevrons, ARIA tree roles, Arrow/Enter keyboard navigation, generic over the node type (icons from `lucide-react`); rows are memoized — a selection change re-renders only the affected rows |
| `WikiTree` | The wiki directory rendered through `Tree` (folders start collapsed; clicking a file opens the note; loading/error/empty states) |
| `SettingsView` | Routed settings page (`/settings/<tab>`, tab rides the URL): left nav rail with icons (**General** / **Git remote** / **Doctor**), content in `Card` sections. **General** (wiki path + AI-model select side by side, fed by `GET /api/models` — both persisted to the settings table; the model applies after restart), **Doctor** (shared check suite via `GET /api/doctor`, green ✓ / red ✗ rows, "Run checks"), **Git remote** (connect with a PAT → account card with avatar/name/email + Disconnect; repo URL + an auto-sync toggle persisted to the settings table; "Initialize & Push" calls `POST /api/git/setup`) |
| `Card` | App-wide section idiom: a `bg-surface` panel with an uppercase kicker title, composed by Dashboard and Settings views |
| `DashboardView` | The launcher + resume home: greeting, four KPI tiles (icon chip + number), quick-action buttons, then **Overview** (inbox, meetings with time chips, todos with checkboxes + progress bar, recent notes, real recent chats, tags) and **Insights** (four Chart.js charts: notes/week bars, chat-activity line, notes-by-kind doughnut with legend, notes-by-folder bars). Mock data lives in `dashboardMock.ts`, tagged with its issue until the index endpoints land |
| `ActivityChart` | Single-series mini bar chart (Chart.js): notes/day, last 7 days — one emerald hue, thin rounded bars, built-in tooltip, canvas `role="img"` + aria-label. Colors are read from the Tailwind CSS variables and re-applied on `prefers-color-scheme` changes; the chart instance is destroyed on unmount |
| `SetupScreen` | Full-zone card shown while `/api/health` reports the Claude CLI missing: problem checklist with exact fix commands, "Re-check" button (spinner while rechecking) |
| `Toast` | `ToastProvider` + `useToast()`; fixed bottom-center stack (z-50), surface cards with emerald (success) / red (error) dots, auto-dismiss after 3 s, click to close |

## Hooks

- **useChat** — thin adapter over the chat slice: maps server frames to chat actions, `send`/`cancel` call the socket; the conversation state itself (messages, streaming, conversationId, thinking, lastTool) lives in the Redux `chat` slice and survives component remounts. `load(messages, conversationId)` replaces the whole conversation (history fetch — local only, the caller pins the server side with `socket.open`); `reset()` clears locally and sends `new_chat` to unpin the server
- **useSearch** — 300 ms debounce with a sequence guard, so slow older responses can't overwrite newer ones; superseded requests are aborted (AbortController), and clearing the query cancels the in-flight request and resets loading

## State

Redux Toolkit owns the server-backed and shared state. Slices live in `store/slices/` with their thunks/actions and selectors co-located; `makeStore()` wires them and `store/hooks.ts` exports the typed `useAppDispatch`/`useAppSelector`:

- **health** — fetched at boot (`main.tsx`), re-checked by the setup screen
- **settings** — loaded when the settings view mounts, saved through the slice (the submit button reflects `saving`)
- **conversations** — refetched on URL changes and when a new chat is created; deletes filter the list in the slice
- **chat** — the live conversation (messages, streaming, thinking, lastTool, conversationId), fed by WS frames via `useChat`
- **connection** — the WebSocket status, reported by `ChatSocket` and read by `ChatPanel`
- **searchHistory** — committed searches (cap 10, deduped, most-recent first); loaded from `localStorage` at store creation and written back by `persistSearchHistory` middleware on every commit/clear

Component-local state — form fields while editing, tree expansion, search debounce, `openNote` — stays in hooks/components; only shared or screen-spanning data lives in the store.

## WebSocket client

`ChatSocket` sends `send`/`cancel`/`resume`/`open` (`open` pins the server-side conversation without replay and never becomes the reconnect-resume id), forwards `assistant_*`/`tool_activity`/`turn_done`/`error` frames, and reconnects exactly once after 1 s — sending `resume` from `onopen` so the turn re-syncs.

## Design system

Tokens in `index.css` (`@theme`) resolve to CSS custom properties that flip under `prefers-color-scheme`, so one semantic class (`bg-surface`, `text-ink`, `border-line`) works in both themes: slate + white surfaces in light, slate night in dark, emerald accent (`#059669` → `#34d399`). The five semantic color groups are app/surface/raised, line (borders), subtle/ink/heading (text). Display type is Fraunces (self-hosted via `@fontsource-variable` — no runtime network); body is the system stack. The typography plugin powers `prose-*` content styles with Fraunces headings.

Dark mode follows the OS — no toggle needed in a local app.

## Development

`pnpm dev` runs Vite with `/api` and `/ws` proxied to `127.0.0.1:8333`, so the frontend can be developed against a live server. For production, `make web` builds and syncs `web/dist` into `internal/webui/dist` for embedding.
