# Frontend

React 19 + TypeScript (strict) + Vite + **Ant Design v6** + Tailwind CSS v4 (layout utilities only), bundled and embedded into the Go binary at build time. Package manager: **pnpm**.

## Structure

```
web/src/
├── app/App.tsx          # app shell: layout, lazy page routing, health gate
├── api/client.tsx        # typed REST client, zod-validated responses
├── ws/chat.tsx           # ChatSocket: protocol frames, reconnect/resume
├── ws/protocol.tsx       # zod schemas + inferred ServerMessage/TokenUsage types
├── ws/events.tsx         # ServerEvent/ClientEvent — the wire frame names
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
├── utils/chart.tsx       # Chart.js registration (side-effect import)
├── utils/time.tsx         # relativeDate — "3 days ago" labels (conversations, dashboard)
├── theme.tsx             # the single antd ThemeConfig (blue primary, light-only)
└── index.css            # Tailwind v4 @theme tokens bridging antd CSS vars
```

## Components

All UI chrome renders through antd components (Layout, Menu, Button, Badge,
Popover, Tooltip, List, Empty, Skeleton, Tabs, Form, Select, AutoComplete,
Input, Switch, Alert, Drawer, Tree, Card, Statistic, Progress, Result,
message). Icons come from `@ant-design/icons` (aria-hidden, decorative).

**File structure rules** (the react skill §1a is the authority): one
component per file — a second component (even a page-local footer, avatar,
or toast) gets its own file, only pure helpers/types a component alone
uses may share its file; **`.tsx` only** — every source file under `web/src`
is `.tsx`, JSX or not (components, hooks, helpers, slices, client, theme,
test doubles); PascalCase component files, camelCase logic files; named
exports only; explicit props types (derive wire types from zod via
`z.infer`); pages are a dispatcher + one file per sub-page + shared
components, never a monolith.
Tailwind utilities handle layout/spacing only — colors always via the
semantic tokens.

| Component | Role |
|---|---|
| `DevBanner` | Full-width antd `Alert` (banner mode, warning, content centered as one group) rendered above the shell while `serve --dev` is running — `App` shows it when the health slice's `dev` is true, with the full commit id from `health.commit` |
| `AppSider` | App shell navigation: brand lockup (`Logo` mark + Fraunces wordmark), antd `Menu` (Dashboard/Chat/Notes/Search/Settings) routed through `useView`, plus the `HealthFooter` (Badge status + version) reading the health slice; the column sits on the surface color, defined by contrast rather than a border |
| `HealthFooter` | AppSider's bottom status bar: `Badge` status dot + one-line reason (All systems go / API key not configured / Wiki missing / Server unreachable) and the app version from the health slice |
| `AppHeader` | Per-view header: title, notification bell (`Badge count` + `Popover` → NotificationPanel, visibility in the ui slice, Esc closes), optional settings button |
| `Sidebar` | Chat view's history column (antd `Layout.Sider`): "New chat" primary button + the day-grouped conversation list (`ChatsList`) || `ChatsList` | The day-grouped conversation history (one antd `List` per Today/Yesterday/Previous 7 days/Older), re-fetched on URL change, active item kept in view; deletes via text button + antd Tooltip, toasts via `App.useApp().message` |
| `ChatPage` | Owns the socket lifecycle (created in an effect, closed on unmount); compact `StatusPill` chips for connection/thinking/tool status (replacing full-width banners); the brand-hero `ChatEmptyState` with suggested prompts when empty; message list + scroll; toasts via `App.useApp().message` |
| `Composer` | antd `Input.TextArea autoSize` (Enter = send, Shift+Enter = newline) + primary Send / Stop buttons in a two-row bar with a hint line and a model chip (default model from the settings slice) — sending while streaming supersedes the turn; the draft is deliberately local state |
| `MessageItem` | Memoized row: user bubble vs assistant (react-markdown + GFM) with streaming caret; 2xl bubbles with a soft shadow on assistant rows; antd Tooltip + CopyButton for the copy action; assistant rows lead with the `AssistantIcon` avatar; a `SaveAsNote` button on completed assistant rows promotes the answer into a wiki note |
| `SaveAsNote` | The chat "Save as note" action: a Tooltip'd save Button on assistant messages opens a `Modal` with a folder `Select` (configured folders from the settings slice, defaulting to the first; `virtual={false}` for jsdom-compatible tests); saving calls `api.saveNote` (`POST /api/v1/notes`), toasts the created path via the notifications slice, and refetches the wiki tree |
| `AssistantIcon` | The small `LogoMark` brand tile (the owl) shown left of every assistant message, tying the assistant to the product |
| `ChatEmptyState` | The empty-conversation hero: `Logo` lockup, "Ask anything" headline, and suggested-prompt chips that send a conversation directly |
| `StatusPill` | Compact centered status chip for the chat header — info tone (spinning accent icon, thinking/tool activity) or warning tone (amber dot, connection trouble) |
| `LogoMark` | The Thoth owl mark: an SVG accent tile (hover→active blue gradient from the antd tokens, unique gradient id per mount) with a white owl silhouette and amber beak; decorative by default |
| `Logo` | The brand lockup — `LogoMark` plus the Fraunces wordmark; the wordmark carries the accessible name |
| `EmptyState` | Branded empty-state placeholder — a soft icon circle + title + optional description/action; replaces antd's stock gray `Empty` across Notes, search, notifications, providers, and sync |
| `Markdown` | GFM renderer with Shiki code blocks (via CodeBlock) in the shared prose wrapper (light only) |
| `CodeBlock` | Fenced code via Shiki (`github-light`, module-level 200-entry cache) + copy button, plain `<pre>` fallback |
| `CopyButton` | Shared copy control: antd text Button, clipboard write, check flip for 2 s, optional `message.success` toast |
| `NotesPage` | Browse-and-read surface: wiki tree left (expand/collapse-all toggle dispatches the ui slice), note reader rendered inline in the content area (the URL `/notes/<path>` owns the open note); branded `EmptyState` placeholder |
| `WikiTree` | The wiki directory rendered through antd `Tree.DirectoryTree` (`virtual={false}`, `motion={false}` — small local tree, jsdom-compatible); tree data from the wiki slice, expansion from the ui slice; refetches when the WS connection (re)connects, on window focus; per-folder file-count Tooltips (an unreadable directory instead shows a warning Tooltip with its per-node error). Per-change refetches ride the `wiki_changed` frame (see useChat), so no polling lives here |
| `NoteViewer` | Inline note reader filling the content area (Esc or ✕ closes — the open note is the URL, not an overlay): note content from the note slice (stale-path responses discarded), Skeleton loading, Alert errors, Copy raw. Markdown paths (`.md`/`.markdown`, case-insensitive, matching `wiki.IsMarkdownPath`) are fetched and rendered as Markdown; image attachments (`.png/.jpg/.jpeg/.gif/.svg/.webp`, matching `wiki.IsImagePath`) render inline from the `/api/v1/notes` raw bytes; any other attachment (scripts, PDFs, …) keeps a "can't be previewed" state with a Download action in the header |
| `SearchPanel` | antd `Input.Search` synced to the URL `?q=` — 300 ms debounced `useSearch` dispatches into the search slice, results in an antd `List` (server-sanitized `<mark>` snippets), keyboard nav via the ui slice, branded `EmptyState`, recent-search history (`Button` rows + Clear) |
| `SettingsPage` | Segment dispatcher: renders one of four route-level sub-pages (`/settings/<section>`) — `SettingsGeneralPage`, `SettingsProvidersPage`, `SettingsSyncPage`, `SettingsDoctorPage` — sharing `SettingsShell` (header + a left rail `Menu`; the active item gets a primary pill via the `.settings-menu` CSS rule; the section rides the URL `/settings/<section>`; the rail stays fixed while the content scrolls). The General sub-page owns a `Form` via `useSettingsForm` (fetch + seed + `save` merging only its own fields through `settingsBody`, so the PUT payload is always complete) and shares the `SaveFooter` save bar; the Sync page manages its own cards/forms instead. Each section is one Card: **General** (`WikiPathInput` with the `DirBrowserModal` picker + a two-field Provider/Model cascade `Select` (the Provider select is view state, the Model select the saved form field) + scaffold folders tag `Select`), **Providers** (an Add-provider `Button` + a `Collapse` panel per provider from the `/api/v1/providers` list; the header carries name + model-count/key/endpoint `Tag`s and Edit/Delete icon actions, the body holds the provider's models `Table` with colored tag chips and an `Empty` state; the `ProviderModal` adds/edits name + base URL + write-only API key + custom request headers as key/value rows (Portkey-style gateway headers), the `ModelModal` (combobox tag `AutoComplete` + provider `Select` pre-filled from the panel) adds/edits models, per-row `Popconfirm` delete; mutations refetch the providers list and the grouped registry), **Doctor** (pass `Progress` summary, Run checks button, `CheckRow` status rows), **Sync** (`SyncConnectionCard` per connection: identity line, provider-driven editable config fields, enabled `Switch`, push/disconnect/set-active actions, a Restore modal for s3/local connections backed by `GET /api/v1/sync/connections/:id/snapshots` + `POST …/restore`, and a compact recent-run history; `SyncConnectForm` renders the connect form from the provider's field descriptors; `SyncProviderEditor` manages the catalog) — all async state in the sync/settings/doctor slices |
| `SaveFooter` | The settings save bar shared by the settings sub-pages with a `Form`: transient saved/error `Alert` feedback + the submit `Button`, separated by a hairline — one convention, one place |
| `SyncConnectForm` | The "connect a destination" card: pick a provider, fill its credential/target fields (driven by the provider's field descriptors — secrets render as `Input.Password`), name the connection, and connect (the server verifies credentials before storing) |
| `SyncConnectionCard` | One connected destination: identity line, editable non-secret config fields (from the provider's descriptors — including the auto-sync `interval` and s3's `snapshot`/`retention`), the enabled `Switch`, push/disconnect/set-active actions, a Restore modal for s3/local connections (snapshot picker + overwrite warning — a local backup is taken first), and a compact recent-run history from `connection.push_history` with colored status glyphs |
| `WikiPathInput` | The wiki path field: a `FolderOpenOutlined` prefix icon opens the `DirBrowserModal` directory picker; the value stays hand-editable at all times |
| `DirBrowserModal` | Directory picker behind WikiPathInput: `Modal` backed by `GET /api/v1/fs/dirs` (enter a subdirectory, `Up` to the parent, OK reports the choice); loads the starting directory on open |
| `DashboardPage` | Landing: greeting header, four KPI `StatTile`s, quick-action `Button`s, the **Overview** widgets (`QuickCaptureCard`, `ContinueCard`, `NeedsAttentionCard`, `TodayCard`, `TagsCard`) and the **Insights** rows (`ChartCard` wrapping four Chart.js charts, blue palette), sections separated by the shared `SectionHeader` kicker. Mock data lives in `pages/dashboard/dashboardMock.tsx`, tagged with its issue until the index endpoints land |
| `ContinueCard` | Overview "Continue where you left off": recent chats + recently touched notes merged into one recency-sorted resume `Listy` (antd `Avatar` kind tiles + `Tag` labels + relative dates), each row opening its chat or note; full-bleed hover rows |
| `NeedsAttentionCard` | Overview "Needs attention": aggregate rows (waiting captures, open todos, sync debt) with tone-tinted antd `Avatar` tiles (warning/danger via antd css-var colors), each opening the notes view; full-bleed hover rows |
| `TodayCard` | Overview "Today": antd `Timeline` (`title` = time, `content` = row) of meetings + captures with colored dots and `Tag` kind chips, each opening its note |
| `QuickCaptureCard` | Overview "Quick capture": antd `Space.Compact` single-line capture input + primary button (Enter or click) that files into inbox/ via the page's toast (mock until the capture endpoint lands) |
| `SectionHeader` | Page-section kicker — a small accent tick + uppercase micro-label (`Overview`, `Insights`), used to separate blocks on a page |
| `StatTile` | KPI tile: tinted accent icon chip + label + large value + optional trend delta (a leading `+` tints the delta success) |
| `ActivityChart` | Single-series mini bar chart (Chart.js): notes/day, last 7 days — blue palette with a vertical gradient fill and the shared light tooltip; canvas `role="img"` + aria-label; colors read from the CSS variables; the chart instance is destroyed on unmount |
| `chartTheme` | Dashboard chart helpers: `verticalGradient` (top-to-bottom fill for bars/line areas, solid fallback before layout) and `tooltipStyle` (shared light tooltip chrome — white card, hairline border, soft shadow) |
| `SetupPage` | antd `Result` shown when `/api/v1/health` reports problems: per-problem `Alert`s with exact fix commands, Re-check primary button (loading). Problems derive from the health schema — server unreachable, no provider API key (`health.backend.api_key_configured`), or missing wiki (`health.wiki.exists`); `App` shows it whenever `health.backend.api_key_configured` is false, unless the user is already on Settings. The hero icon is the `LogoMark` owl |
| `NotificationPanel` | The bell `Popover` content: header with mark-all-read/close buttons, antd `List` of notifications, branded `EmptyState`, per-item dismiss |
| `NotificationToasts` | NEW notifications (not seen at mount) as transient antd `Alert`s top-left, each rendered by `ToastAlert`, auto-dismissed after 5 s; close dispatches dismiss |
| `ToastAlert` | One transient notification toast — antd `Alert` (NotificationIcon, title/body, closable) that auto-dismisses via `onDismiss` after 5 s |

## Hooks

- **useChat** — thin adapter over the chat slice: maps server frames to chat actions, `send`/`cancel` call the socket; the conversation state itself (messages, streaming, conversationId, thinking, lastTool) lives in the Redux `chat` slice and survives component remounts. A `wiki_changed` frame dispatches the wiki slice's `fetchTree` (server-pushed refetch instead of turn-end polling); a `sync_result` frame dispatches the notifications slice's `notify` (a scheduled push's outcome surfaced as a toast). `load(messages, conversationId)` replaces the whole conversation (history fetch — local only, the caller pins the server side with `socket.open`); `reset()` clears locally and sends `new_chat` to unpin the server
- **useSearch** — 300 ms debounce with abort (AbortController); dispatches `searchNotes` into the search slice (the slice's query guard drops stale responses); clearing the query dispatches `clearSearch`

## State

Redux Toolkit owns the server-backed, shared, and screen-spanning state. Slices live in `store/slices/` with their thunks/actions and selectors co-located; `makeStore()` wires them and `store/hooks.tsx` exports the typed `useAppDispatch`/`useAppSelector`:

- **health** — fetched at boot (`main.tsx`), re-checked by the setup page; the slice holds the `{status, backend:{name, api_key_configured, model, provider}, wiki:{path, exists}, version, dev, commit, default_wiki_path}` shape — `App`/`SetupPage` gate on `backend.api_key_configured` + `wiki.exists`, `DevBanner` reads `dev`/`commit`, `AppSider`'s footer reads `version`
- **settings** — loaded on mount, saved through the slice (submit button reflects `saving`); also holds the `/api/v1/models` picker list
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

**Ant Design v6 is the design system**; `web/src/theme.tsx` holds the single `ThemeConfig` — a full token set over the brand blue `#1677ff`: cool-gray neutral ramp (layout `#f5f7fa`, border `#e3e7ee`), radius `8`/`12`, control height `36`, soft 3px blue focus ring, layered box-shadows, unified motion timings, and component tokens for Layout/Menu/Button/Card/Table/Statistic/Tree/Alert/Input. `cssVar: {}` (antd emits `--ant-*` variables **scoped under the ConfigProvider's css-var class, not on `:root`**), `hashed: false`. **Light theme only — no dark mode**; every value lives in the one ThemeConfig, so a dark theme is a pure token flip. The Tailwind tokens in `index.css` use `@theme inline` so the semantic classes (`bg-surface`, `text-ink`, `border-line`, plus `bg-elevated`/`hover`/`line-soft`/`faint` and status soft pairs) inline their `var(--ant-*, fallback)` reference and resolve per element, inside antd's scope — never define them on `:root`. Icons are 16px app-wide (Menu `iconSize`/Button `onlyIconSize` tokens); cards carry the tertiary elevation; per-view titles are `text-base font-semibold`; the settings rail is a `Menu` whose active-pill fill is scoped by the `.settings-menu` rule in index.css. antd components reset their own margins, so vertical rhythm around them uses the container's `gap` (antd `Flex vertical gap`) — never `space-y-*` on antd children. Brand: the owl mark (`LogoMark`, an SVG tile on a hover→active blue gradient with an amber beak — token colors, unique gradient id per mount) + the Fraunces wordmark (`Logo`); the mark doubles as the favicon (`public/favicon.svg`). Display type is Fraunces (self-hosted via `@fontsource-variable` — no runtime network), used for the brand wordmark and hero/empty-state headlines; body/headings use antd's font stack. The typography plugin powers `prose-*` content styles; chart series use a blue family (`#1677ff`, `#0958d9`, `#91caff`, `#ffc53d`) and chart component colors come from `theme.useToken()` via `useThemeColors`.

## Development

`pnpm dev` runs Vite with `/api` and `/ws` proxied to `127.0.0.1:8333`, so the frontend can be developed against a live server. For production, `make web` builds and syncs `web/dist` into `internal/webui/dist` for embedding. Tests render via `renderWithStore` (fresh store + antd `App` wrapper — mirrors `main.tsx`); antd's rc-motion never completes under jsdom, so tests assert store state for close transitions and set `virtual={false}`/`motion={false}` on list components.
