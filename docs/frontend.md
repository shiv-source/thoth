# Frontend

React 19 + TypeScript (strict) + Vite + **Ant Design v6** + Tailwind CSS v4 (layout utilities only), bundled and embedded into the Go binary at build time. Package manager: **pnpm**.

## Structure

```
web/src/
├── app/App.tsx          # app shell: layout, lazy page routing, health gate
├── api/client.ts        # typed REST client, zod-validated responses
├── ws/chat.ts           # ChatSocket: protocol frames, reconnect/resume
├── ws/protocol.ts       # zod schemas + inferred ServerMessage/TokenUsage types
├── ws/events.ts         # ServerEvent/ClientEvent — the wire frame names
├── hooks/               # useChat, useSearch, useConversationRoute, useView,
│                        #   useViewShortcuts
├── store/               # Redux Toolkit: 13 slices + typed hooks (below)
├── pages/               # one folder per route-level view
│   ├── chat/            #   ChatPage
│   ├── dashboard/       #   DashboardPage (+ its dashboardMock until endpoints land)
│   ├── notes/           #   NotesPage
│   ├── search/          #   SearchPage
│   ├── settings/        #   SettingsPage dispatcher → Settings{General,Providers,Git,Doctor}Page
│   │                    #   (shared SettingsShell + useSettingsForm/settingsBody)
│   └── setup/           #   SetupPage
├── components/          # feature components, grouped by owner page
│   ├── layout/          #   AppSider, Sidebar, DevBanner
│   ├── chat/            #   Composer, MessageItem, UsageLine
│   ├── dashboard/       #   charts + useThemeColors
│   ├── notes/           #   NoteViewer, WikiTree
│   └── search/          #   SearchPanel
├── shared/              # cross-cutting UI primitives: AppHeader, Markdown,
│                        #   CodeBlock, CopyButton, NotificationPanel/Toasts,
│                        #   notifications, WikiPathInput
├── utils/chart.ts       # Chart.js registration (side-effect import)
├── theme.ts             # the single antd ThemeConfig (blue primary, light-only)
└── index.css            # Tailwind v4 @theme tokens bridging antd CSS vars
```

## Components

All UI chrome renders through antd components (Layout, Menu, Button, Badge,
Popover, Tooltip, List, Empty, Skeleton, Tabs, Form, Select, AutoComplete,
Input, Switch, Alert, Drawer, Tree, Card, Statistic, Progress, Result,
message). Icons come from `@ant-design/icons` (aria-hidden, decorative).

**File structure rules** (the react skill §1a is the authority): one
component per file; `.tsx` for any JSX-bearing file, `.ts` for pure logic
(hooks without render, helpers, client.ts, slices); PascalCase component
files, camelCase logic files; named exports only; explicit props types
(derive wire types from zod via `z.infer`); pages are a dispatcher +
one file per sub-page + shared components, never a monolith.
Tailwind utilities handle layout/spacing only — colors always via the
semantic tokens.

| Component | Role |
|---|---|
| `DevBanner` | Full-width antd `Alert` (banner mode, warning, content centered as one group) rendered above the shell while `serve --dev` is running — `App` shows it when the health slice's `dev` is true, with the full commit id from `health.commit` |
| `AppSider` | App shell navigation: brand wordmark (Fraunces), antd `Menu` (Dashboard/Chat/Notes/Search/Settings) routed through `useView`, health footer (`Badge status` + version) reading the health slice |
| `AppHeader` | Per-view header: title, notification bell (`Badge count` + `Popover` → NotificationPanel, visibility in the ui slice, Esc closes), optional settings button |
| `Sidebar` | Chat view's history column (antd `Layout.Sider`): "New chat" primary button + day-grouped conversation `List`s; deletes via text button + antd Tooltip; toasts via `App.useApp().message` |
| `ChatPage` | Owns the socket lifecycle (created in an effect, closed on unmount); antd `Alert` banners for connection/thinking/tool status; message list + scroll; toasts via `App.useApp().message` |
| `Composer` | antd `Input.TextArea autoSize` (Enter = send, Shift+Enter = newline) + primary Send / Stop buttons — sending while streaming supersedes the turn; the draft is deliberately local state |
| `MessageItem` | Memoized row: user bubble vs assistant (react-markdown + GFM) with streaming caret; antd Tooltip + CopyButton for the copy action |
| `Markdown` | GFM renderer with Shiki code blocks (via CodeBlock) in the shared prose wrapper (light only) |
| `CodeBlock` | Fenced code via Shiki (`github-light`, module-level 200-entry cache) + copy button, plain `<pre>` fallback |
| `CopyButton` | Shared copy control: antd text Button, clipboard write, check flip for 2 s, optional `message.success` toast |
| `NotesPage` | Browse-and-read surface: wiki tree left (expand/collapse-all toggle dispatches the ui slice), note reader rendered inline in the content area (the URL `/notes/<path>` owns the open note); `Empty` placeholder |
| `WikiTree` | The wiki directory rendered through antd `Tree.DirectoryTree` (`virtual={false}`, `motion={false}` — small local tree, jsdom-compatible); tree data from the wiki slice, expansion from the ui slice; refetches when the WS connection (re)connects, on window focus; per-folder file-count Tooltips (an unreadable directory instead shows a warning Tooltip with its per-node error). Per-change refetches ride the `wiki_changed` frame (see useChat), so no polling lives here |
| `NoteViewer` | Inline note reader filling the content area (Esc or ✕ closes — the open note is the URL, not an overlay): note content from the note slice (stale-path responses discarded), Skeleton loading, Alert errors, Copy raw. Markdown paths (`.md`/`.markdown`, case-insensitive, matching `wiki.IsMarkdownPath`) are fetched and rendered as Markdown; image attachments (`.png/.jpg/.jpeg/.gif/.svg/.webp`, matching `wiki.IsImagePath`) render inline from the `/api/notes` raw bytes; any other attachment (scripts, PDFs, …) keeps a "can't be previewed" state with a Download action in the header |
| `SearchPanel` | antd `Input.Search` synced to the URL `?q=` — 300 ms debounced `useSearch` dispatches into the search slice, results in an antd `List` (server-sanitized `<mark>` snippets), keyboard nav via the ui slice, `Empty` state, recent-search history (`Button` rows + Clear) |
| `SettingsPage` | Segment dispatcher: renders one of four route-level sub-pages (`/settings/<section>`) — `SettingsGeneralPage`, `SettingsProvidersPage`, `SettingsGitPage`, `SettingsDoctorPage` — sharing `SettingsShell` (header + a left rail `Menu`; the active item gets a primary pill via the `.settings-menu` CSS rule; the section rides the URL `/settings/<section>`; the rail stays fixed while the content scrolls). Each sub-page owns its own `Form` via `useSettingsForm` (fetch + seed + `save` merging only its own fields through `settingsBody`, so the PUT payload is always complete). Each section is one Card: **General** (`WikiPathInput` with a clickable folder picker + a two-field Provider/Model cascade `Select` (the Provider select is view state, the Model select the saved form field) + scaffold folders tag `Select`, save with loading state, saved/error `Alert`s), **Providers** (a `Collapse` panel per provider whose header carries name + model-count/key/endpoint `Tag`s and whose body holds the provider's Base URL + API key credential fields plus its models `Table` with colored tag chips and an `Empty` state; Add/Edit `Modal` with a combobox tag `AutoComplete` pre-fills the provider from the panel it opened in, per-row `Popconfirm` delete; mutations refetch the grouped registry and settings), **Doctor** (pass `Progress` summary, Run checks button, `CheckRow` status rows), **Git remote** (`Input.Password` PAT connect, account section with `Avatar` + scope `Tag`s, `AutoComplete` repo picker with private-repo guard, `Switch` auto-sync, Initialize & Push) — all async state in the git/settings/doctor slices |
| `WikiPathInput` | The wiki path field: a `FolderOpenOutlined` prefix icon opens a directory browser `Modal` backed by `GET /api/fs/dirs` (enter a subdirectory, `Up` to the parent, OK fills the field); the value stays hand-editable at all times |
| `DashboardPage` | Landing: greeting, four `Statistic` KPI tiles, quick-action `Button`s, **Overview** cards (inbox, meetings, todos with `Progress`, recent notes, real recent chats, tag `Button`s) and **Insights** (four Chart.js charts, blue palette). Mock data lives in `pages/dashboard/dashboardMock.ts`, tagged with its issue until the index endpoints land |
| `ActivityChart` | Single-series mini bar chart (Chart.js): notes/day, last 7 days — blue palette, canvas `role="img"` + aria-label; colors read from the CSS variables; the chart instance is destroyed on unmount |
| `SetupPage` | antd `Result` shown when `/api/health` reports problems: per-problem `Alert`s with exact fix commands, Re-check primary button (loading). Problems derive from the health schema — server unreachable, no provider API key (`health.backend.api_key_configured`), or missing wiki (`health.wiki.exists`); `App` shows it whenever `health.backend.api_key_configured` is false, unless the user is already on Settings |
| `NotificationPanel` | The bell `Popover` content: header with mark-all-read/close buttons, antd `List` of notifications, `Empty` state, per-item dismiss |
| `NotificationToasts` | NEW notifications (not seen at mount) as transient antd `Alert`s top-left, auto-dismissed after 5 s, close dispatches dismiss |

## Hooks

- **useChat** — thin adapter over the chat slice: maps server frames to chat actions, `send`/`cancel` call the socket; the conversation state itself (messages, streaming, conversationId, thinking, lastTool) lives in the Redux `chat` slice and survives component remounts. A `wiki_changed` frame dispatches the wiki slice's `fetchTree` (server-pushed refetch instead of turn-end polling). `load(messages, conversationId)` replaces the whole conversation (history fetch — local only, the caller pins the server side with `socket.open`); `reset()` clears locally and sends `new_chat` to unpin the server
- **useSearch** — 300 ms debounce with abort (AbortController); dispatches `searchNotes` into the search slice (the slice's query guard drops stale responses); clearing the query dispatches `clearSearch`

## State

Redux Toolkit owns the server-backed, shared, and screen-spanning state. Slices live in `store/slices/` with their thunks/actions and selectors co-located; `makeStore()` wires them and `store/hooks.ts` exports the typed `useAppDispatch`/`useAppSelector`:

- **health** — fetched at boot (`main.tsx`), re-checked by the setup page; the slice holds the `{status, backend:{name, api_key_configured, model, provider}, wiki:{path, exists}, version, dev, commit, default_wiki_path}` shape — `App`/`SetupPage` gate on `backend.api_key_configured` + `wiki.exists`, `DevBanner` reads `dev`/`commit`, `AppSider`'s footer reads `version`
- **settings** — loaded on mount, saved through the slice (submit button reflects `saving`); also holds the `/api/models` picker list
- **conversations** — refetched on URL changes and when a new chat is created; deletes filter the list in the slice
- **chat** — the live conversation (messages, streaming, thinking, lastTool, conversationId), fed by WS frames via `useChat`
- **connection** — the WebSocket status, reported by `ChatSocket` and read by `ChatPage` (and `WikiTree`, which seeds the tree on the reconnect edge)
- **notifications** — the capped ring of 50; panel + toasts both consume it
- **searchHistory** — committed searches (cap 10, deduped, most-recent first); loaded from `localStorage` at store creation and written back by `persistSearchHistory` middleware on every commit/clear
- **ui** — screen-spanning chrome: notification-panel open, notes-tree expansion, search keyboard selection
- **wiki** — the wiki tree (`fetchTree`; refetch on WS connect/reconnect, `wiki_changed` frames, and window focus)
- **note** — the open note's content (`fetchNote(path)`; stale-path responses discarded)
- **search** — live search results (`searchNotes` with signal; abort is not an error)
- **doctor** — the doctor check rows (`runDoctor`)
- **git** — GitHub auth/repos/connect/push/disconnect (server messages surface as errors)

Deliberate exceptions (documented in `references/patterns.md`): per-keystroke drafts (Composer) stay local, antd Form values are owned by rc-field-form, toasts use `App.useApp().message`, the `ChatSocket` instance is non-serializable and lives in ChatPage, and the URL remains the routing source of truth.

## WebSocket client

`ChatSocket` sends `send`/`cancel`/`resume`/`open` (`open` pins the server-side conversation without replay and never becomes the reconnect-resume id), forwards `assistant_*`/`tool_activity`/`turn_done`/`error` frames (the optional `usage` breakdown on `turn_done`, and on persisted messages loaded from history, feeds the token-usage footer under the last message), and reconnects exactly once after 1 s — sending `resume` from `onopen` so the turn re-syncs. It also sends a `presence` frame (`ChatPage` wires it to the Page Visibility API): a hidden tab reports `active:false`. The frame is kept for protocol compatibility — Thoth Agent keeps no idle processes to flush, so the server ignores it — and the last reported presence is re-sent from `onopen`.

## Design system

**Ant Design v6 is the design system**; `web/src/theme.ts` holds the single `ThemeConfig` — modern blue primary (`#1677ff`), `borderRadius: 6`, `cssVar: {}` (antd emits `--ant-*` CSS variables **scoped under the ConfigProvider's css-var class, not on `:root`**), `hashed: false`. **Light theme only — no dark mode.** The Tailwind tokens in `index.css` use `@theme inline` so the semantic classes (`bg-surface`, `text-ink`, `border-line`) inline their `var(--ant-*, fallback)` reference and resolve per element, inside antd's scope — never define them on `:root`. Icons are 16px app-wide (Menu `iconSize`/Button `onlyIconSize` tokens + a global `.anticon` rule); cards carry the tertiary elevation (`box-shadow: var(--ant-box-shadow-tertiary)`); per-view titles are `text-base font-semibold`; the settings rail is a `Menu` whose active-pill fill is scoped by the `.settings-menu` rule in index.css. antd components reset their own margins, so vertical rhythm around them uses the container's `gap` (antd `Flex vertical gap`) — never `space-y-*` on antd children. Neutral values mirror the antd light palette (surface white, layout `#f5f5f5`, border `#d9d9d9`). Display type is Fraunces (self-hosted via `@fontsource-variable` — no runtime network), used for the brand wordmark; body/headings use antd's font stack. The typography plugin powers `prose-*` content styles; chart series use a blue family (`#1677ff`, `#0958d9`, `#91caff`, `#ffc53d`) and chart component colors come from `theme.useToken()` via `useThemeColors`.

## Development

`pnpm dev` runs Vite with `/api` and `/ws` proxied to `127.0.0.1:8333`, so the frontend can be developed against a live server. For production, `make web` builds and syncs `web/dist` into `internal/webui/dist` for embedding. Tests render via `renderWithStore` (fresh store + antd `App` wrapper — mirrors `main.tsx`); antd's rc-motion never completes under jsdom, so tests assert store state for close transitions and set `virtual={false}`/`motion={false}` on list components.
