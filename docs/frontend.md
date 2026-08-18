# Frontend

React 19 + TypeScript (strict) + Vite + **Ant Design v6** + Tailwind CSS v4 (layout utilities only), bundled and embedded into the Go binary at build time. Package manager: **pnpm**.

## Structure

```
web/src/
├── api/client.ts        # typed REST client, zod-validated responses
├── ws/chat.ts           # ChatSocket: protocol frames, reconnect/resume
├── hooks/               # useChat, useSearch, useConversationRoute, useView,
│                        #   useViewShortcuts
├── store/               # Redux Toolkit: 13 slices + typed hooks (below)
├── theme.ts             # the single antd ThemeConfig (blue primary, light-only)
├── components/          # AppSider, AppHeader, ChatPanel, Composer,
│                        #   MessageItem, Sidebar, NotesView, WikiTree,
│                        #   NoteViewer, SearchPanel/View, SettingsView,
│                        #   DashboardView, SetupScreen, NotificationPanel,
│                        #   NotificationToasts, Markdown, CodeBlock,
│                        #   CopyButton, charts, dashboardMock
└── index.css            # Tailwind v4 @theme tokens bridging antd CSS vars
```

## Components

All UI chrome renders through antd components (Layout, Menu, Button, Badge,
Popover, Tooltip, List, Empty, Skeleton, Tabs, Form, Select, AutoComplete,
Input, Switch, Alert, Drawer, Tree, Card, Statistic, Progress, Result,
message). Icons come from `lucide-react` (aria-hidden, decorative). Tailwind
utilities handle layout/spacing only — colors always via the semantic tokens.

| Component | Role |
|---|---|
| `AppSider` | App shell navigation: brand wordmark (Fraunces), antd `Menu` (Dashboard/Chat/Notes/Search/Settings) routed through `useView`, health footer (`Badge status` + version) reading the health slice |
| `AppHeader` | Per-view header: title, notification bell (`Badge count` + `Popover` → NotificationPanel, visibility in the ui slice, Esc closes), optional settings button |
| `Sidebar` | Chat view's history column (antd `Layout.Sider`): "New chat" primary button + day-grouped conversation `List`s; deletes via text button + antd Tooltip; toasts via `App.useApp().message` |
| `ChatPanel` | Owns the socket lifecycle (created in an effect, closed on unmount); antd `Alert` banners for connection/thinking/tool status; message list + scroll; toasts via `App.useApp().message` |
| `Composer` | antd `Input.TextArea autoSize` (Enter = send, Shift+Enter = newline) + primary Send / Stop buttons — sending while streaming supersedes the turn; the draft is deliberately local state |
| `MessageItem` | Memoized row: user bubble vs assistant (react-markdown + GFM) with streaming caret; antd Tooltip + CopyButton for the copy action |
| `Markdown` | GFM renderer with Shiki code blocks (via CodeBlock) in the shared prose wrapper (light only) |
| `CodeBlock` | Fenced code via Shiki (`github-light`, module-level 200-entry cache) + copy button, plain `<pre>` fallback |
| `CopyButton` | Shared copy control: antd text Button, clipboard write, check flip for 2 s, optional `message.success` toast |
| `NotesView` | Browse-and-read surface: wiki tree left (expand/collapse-all toggle dispatches the ui slice), note reader rendered inline in the content area (the URL `/notes/<path>` owns the open note); `Empty` placeholder |
| `WikiTree` | The wiki directory rendered through antd `Tree.DirectoryTree` (`virtual={false}`, `motion={false}` — small local tree, jsdom-compatible); tree data from the wiki slice, expansion from the ui slice; refetches on mount, chat-turn end, window focus; per-folder file-count Tooltips |
| `NoteViewer` | Inline note reader filling the content area (Esc or ✕ closes — the open note is the URL, not an overlay): note content from the note slice (stale-path responses discarded), Skeleton loading, Alert errors, Copy raw |
| `SearchPanel` | antd `Input.Search` synced to the URL `?q=` — 300 ms debounced `useSearch` dispatches into the search slice, results in an antd `List` (server-sanitized `<mark>` snippets), keyboard nav via the ui slice, `Empty` state, recent-search history (`Button` rows + Clear) |
| `SettingsView` | antd `Tabs` (tabPosition left; the tab rides the URL `/settings/<tab>`): **General** (Form: wiki path `Input` + model `Select` grouped by provider, save with loading state, saved/error `Alert`s), **Doctor** (`runDoctor` slice, `List` of ✓/✗ checks, Run checks button), **Git remote** (`Input.Password` PAT connect, account card with `Avatar`, `AutoComplete` repo picker with private-repo guard, `Switch` auto-sync, Initialize & Push) — all async state in the git/settings/doctor slices |
| `DashboardView` | Landing: greeting, four `Statistic` KPI tiles, quick-action `Button`s, **Overview** cards (inbox, meetings, todos with `Progress`, recent notes, real recent chats, tag `Button`s) and **Insights** (four Chart.js charts, blue palette). Mock data lives in `dashboardMock.ts`, tagged with its issue until the index endpoints land |
| `ActivityChart` | Single-series mini bar chart (Chart.js): notes/day, last 7 days — blue palette, canvas `role="img"` + aria-label; colors read from the CSS variables; the chart instance is destroyed on unmount |
| `SetupScreen` | antd `Result` shown while `/api/health` reports problems: per-problem `Alert`s with exact fix commands, Re-check primary button (loading) |
| `NotificationPanel` | The bell `Popover` content: header with mark-all-read/close buttons, antd `List` of notifications, `Empty` state, per-item dismiss |
| `NotificationToasts` | NEW notifications (not seen at mount) as transient antd `Alert`s top-left, auto-dismissed after 5 s, close dispatches dismiss |

## Hooks

- **useChat** — thin adapter over the chat slice: maps server frames to chat actions, `send`/`cancel` call the socket; the conversation state itself (messages, streaming, conversationId, thinking, lastTool) lives in the Redux `chat` slice and survives component remounts. `load(messages, conversationId)` replaces the whole conversation (history fetch — local only, the caller pins the server side with `socket.open`); `reset()` clears locally and sends `new_chat` to unpin the server
- **useSearch** — 300 ms debounce with abort (AbortController); dispatches `searchNotes` into the search slice (the slice's query guard drops stale responses); clearing the query dispatches `clearSearch`

## State

Redux Toolkit owns the server-backed, shared, and screen-spanning state. Slices live in `store/slices/` with their thunks/actions and selectors co-located; `makeStore()` wires them and `store/hooks.ts` exports the typed `useAppDispatch`/`useAppSelector`:

- **health** — fetched at boot (`main.tsx`), re-checked by the setup screen
- **settings** — loaded on mount, saved through the slice (submit button reflects `saving`); also holds the `/api/models` picker list
- **conversations** — refetched on URL changes and when a new chat is created; deletes filter the list in the slice
- **chat** — the live conversation (messages, streaming, thinking, lastTool, conversationId), fed by WS frames via `useChat`
- **connection** — the WebSocket status, reported by `ChatSocket` and read by `ChatPanel`
- **notifications** — the capped ring of 50; panel + toasts both consume it
- **searchHistory** — committed searches (cap 10, deduped, most-recent first); loaded from `localStorage` at store creation and written back by `persistSearchHistory` middleware on every commit/clear
- **ui** — screen-spanning chrome: notification-panel open, notes-tree expansion, search keyboard selection
- **wiki** — the wiki tree (`fetchTree`; refetch on focus/turn-end)
- **note** — the open note's content (`fetchNote(path)`; stale-path responses discarded)
- **search** — live search results (`searchNotes` with signal; abort is not an error)
- **doctor** — the doctor check rows (`runDoctor`)
- **git** — GitHub auth/repos/connect/push/disconnect (server messages surface as errors)

Deliberate exceptions (documented in `references/patterns.md`): per-keystroke drafts (Composer) stay local, antd Form values are owned by rc-field-form, toasts use `App.useApp().message`, the `ChatSocket` instance is non-serializable and lives in ChatPanel, and the URL remains the routing source of truth.

## WebSocket client

`ChatSocket` sends `send`/`cancel`/`resume`/`open` (`open` pins the server-side conversation without replay and never becomes the reconnect-resume id), forwards `assistant_*`/`tool_activity`/`turn_done`/`error` frames, and reconnects exactly once after 1 s — sending `resume` from `onopen` so the turn re-syncs.

## Design system

**Ant Design v6 is the design system**; `web/src/theme.ts` holds the single `ThemeConfig` — modern blue primary (`#1677ff`), `borderRadius: 6`, `cssVar: {}` (antd emits `--ant-*` CSS variables), `hashed: false`. **Light theme only — no dark mode.** The Tailwind `@theme` tokens in `index.css` bridge to the antd variables (`--color-accent: var(--ant-color-primary, #1677ff)`) so the semantic classes (`bg-surface`, `text-ink`, `border-line`) stay in use for layout styling while colors have one source of truth. Neutral values mirror the antd light palette (surface white, layout `#f5f5f5`, border `#d9d9d9`). Display type is Fraunces (self-hosted via `@fontsource-variable` — no runtime network), used for the brand wordmark; body/headings use antd's font stack. The typography plugin powers `prose-*` content styles; chart series use a blue family (`#1677ff`, `#0958d9`, `#91caff`, `#ffc53d`).

## Development

`pnpm dev` runs Vite with `/api` and `/ws` proxied to `127.0.0.1:8333`, so the frontend can be developed against a live server. For production, `make web` builds and syncs `web/dist` into `internal/webui/dist` for embedding. Tests render via `renderWithStore` (fresh store + antd `App` wrapper — mirrors `main.tsx`); antd's rc-motion never completes under jsdom, so tests assert store state for close transitions and set `virtual={false}`/`motion={false}` on list components.
