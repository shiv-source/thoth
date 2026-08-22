# Components (web/src)

Scope: feature components live in `web/src/components/<feature>/`, cross-cutting
primitives in `web/src/shared/`, route-level views in `web/src/pages/<name>/`,
and the app shell in `web/src/app/` — App.tsx / main.tsx / theme.ts / index.css
are covered by docs/frontend.md. Each entry: path, purpose, props/api, canonical.
Skipped: `*.test.tsx` — co-located Vitest tests, skipped by convention. All
chrome renders through antd v6 components; icons come from @ant-design/icons
(aria-hidden, decorative).

## App shell & navigation
## DevBanner
- path: web/src/components/layout/DevBanner.tsx
- purpose: Full-width antd Alert (banner mode → warning type + icon) shown while serve --dev is running; App renders it above the shell when the health slice's `dev` is true; shows the full commit id from `health.commit`; icon + message centered as one group (root flex + justify center, section flex none)
- props/api: `DevBanner({ dev, commit })` — null when dev is false
- canonical: DevBanner.tsx:8

## AppSider
- path: web/src/components/layout/AppSider.tsx
- purpose: App navigation — Fraunces brand wordmark, antd Menu over the five views (routes through useView), health footer (Badge status + version from the health slice)
- props/api: `AppSider()` — no props
- canonical: AppSider.tsx:27

## Sidebar
- path: web/src/components/layout/Sidebar.tsx
- purpose: Chat history column (antd Layout.Sider) — Chats label, New chat primary button, day-grouped conversation Lists; deletes via text danger button; toasts via App.useApp().message
- props/api: `Sidebar()` — no props; internal ChatsList()
- canonical: Sidebar.tsx:180

## AppHeader
- path: web/src/shared/AppHeader.tsx
- purpose: Per-view header — title, notification bell (Badge count + Popover → NotificationPanel), optional settings button; popover visibility lives in the ui slice, Escape closes
- props/api: `AppHeader({ title, onOpenSettings? })`
- canonical: AppHeader.tsx:12

## Chat
## ChatPage
- path: web/src/pages/chat/ChatPage.tsx
- purpose: Main chat view — owns the WebSocket, Alert banners (connection/thinking/tool status), message list, Composer; token-usage footer via UsageLine
- props/api: `ChatPage({ onOpenSettings })`
- canonical: ChatPage.tsx:19

## Composer
- path: web/src/components/chat/Composer.tsx
- purpose: Chat input — antd Input.TextArea autoSize (Enter sends, Shift+Enter newline), primary Send / default Stop buttons; draft is deliberately local state
- props/api: `Composer({ onSend, onCancel, streaming })`
- canonical: Composer.tsx:5

## MessageItem
- path: web/src/components/chat/MessageItem.tsx
- purpose: One memoized chat message — user bubble plain text, assistant bubble Markdown + antd Tooltip + CopyButton + streaming caret
- props/api: `MessageItem({ message: ChatMessage, streaming? })` — React.memo
- canonical: MessageItem.tsx:10

## UsageLine
- path: web/src/components/chat/UsageLine.tsx
- purpose: A completed turn's token breakdown as a muted footer line under the final assistant message; renders nothing when usage is absent
- props/api: `UsageLine({ usage: TokenUsage })`
- canonical: UsageLine.tsx:6

## Markdown
- path: web/src/shared/Markdown.tsx
- purpose: GFM renderer with Shiki code blocks (via CodeBlock) in the shared prose wrapper (light only)
- props/api: `Markdown({ children, trailing?, className = '' })`
- canonical: Markdown.tsx:29

## CodeBlock
- path: web/src/shared/CodeBlock.tsx
- purpose: Fenced code block via Shiki (github-light, module-level 200-entry cache) + copy button, plain <pre> fallback
- props/api: `CodeBlock({ code, lang })`
- canonical: CodeBlock.tsx:24

## CopyButton
- path: web/src/shared/CopyButton.tsx
- purpose: Shared copy control — antd text Button, clipboard write, check flip for 2 s, optional message.success toast; layout via caller className
- props/api: `CopyButton({ text, label, toast?, className? })`
- canonical: CopyButton.tsx:8

## Wiki & notes
## NotesPage
- path: web/src/pages/notes/NotesPage.tsx
- purpose: Browse-and-read wiki surface — WikiTree left (expand/collapse-all toggle dispatches the ui slice), NoteViewer inline in the content area or Empty placeholder; ancestors of the open note auto-expand
- props/api: `NotesPage({ openPath, onOpenNote, onOpenSettings })`
- canonical: NotesPage.tsx:13

## WikiTree
- path: web/src/components/notes/WikiTree.tsx
- purpose: Wiki directory via antd Tree.DirectoryTree (virtual={false}, motion={false}); data from the wiki slice, expansion from the ui slice; refetches on mount, chat-turn end, window focus; per-folder file-count Tooltips in the title renderer
- props/api: `WikiTree({ openPath, onOpenNote })`
- canonical: WikiTree.tsx:44

## NoteViewer
- path: web/src/components/notes/NoteViewer.tsx
- purpose: Inline note reader filling the content area — note content from the note slice (stale-path responses discarded), Skeleton/Alert states, Copy raw; only `.md`/`.markdown` paths are previewed (case-insensitive), other file types get a "can't be previewed" state; Esc or ✕ routes through onClose (the open note is the URL, not an overlay)
- props/api: `NoteViewer({ path: string, onClose })`
- canonical: NoteViewer.tsx:37

## Search
## SearchPanel
- path: web/src/components/search/SearchPanel.tsx
- purpose: Wiki search synced to the URL ?q= — antd Input.Search, debounced useSearch into the search slice, antd List results (server-sanitized <mark> snippets), keyboard nav via the ui slice, Empty state, recent-search history (Button rows + Clear)
- props/api: `SearchPanel({ onOpen: (path: string) => void })`
- canonical: SearchPanel.tsx:10

## SearchPage
- path: web/src/pages/search/SearchPage.tsx
- purpose: Full-page search surface — AppHeader plus the shared SearchPanel
- props/api: `SearchPage({ onOpenNote, onOpenSettings })`
- canonical: SearchPage.tsx:6

## Settings & dashboard
## SettingsPage
- path: web/src/pages/settings/SettingsPage.tsx
- purpose: antd Tabs (left rail, tabPlacement="start"; active pill via the .settings-tabs CSS rule + theme.ts Tabs tokens) over General/Providers/Doctor/Git — each tab one Card with icon'd SectionHeading sections, Dividers, and responsive two-column section grids: General (wiki path WikiPathInput + a two-field Provider/Model cascade Select + scaffold folders tag Select, save loading, saved/error Alerts); Providers (a Collapse panel per provider — header with name + model-count/key/endpoint Tags, body with Base URL + write-only per-provider API key Inputs with Configured/Not set Tags and the provider's models Table + Add/Edit Modal pre-filling the provider + Popconfirm delete); Doctor (pass Progress summary, Run checks, CheckRow status rows); Git (PAT Input.Password connect, account Avatar row + scope Tags, AutoComplete repo picker with public-repo guard, Switch auto-sync, Initialize & Push). Async state in the settings/doctor/git slices; tab rides the URL segment
- props/api: `SettingsPage()` — no props; internal GeneralTab/DoctorTab/GitTab + SectionHeading/CheckRow/GitAccountSection/GitRemoteSection/GitSyncSection helpers
- canonical: SettingsPage.tsx:83

## DashboardPage
- path: web/src/pages/dashboard/DashboardPage.tsx
- purpose: Landing — greeting, Statistic KPI tiles, quick-action Buttons, Overview cards (inbox/meetings/todos+Progress/recent notes/real recent chats/tags), Insights (four Chart.js charts, blue palette). Mock data lives in dashboardMock.ts, tagged with its issue until the index endpoints land
- props/api: `DashboardPage({ onOpenSettings })`
- canonical: DashboardPage.tsx:60

## SetupPage
- path: web/src/pages/setup/SetupPage.tsx
- purpose: antd Result listing install problems (per-problem Alerts with fix commands) + Re-check primary button
- props/api: `SetupPage({ health, loading, onRecheck })`
- canonical: SetupPage.tsx:32

## Notifications
## NotificationPanel
- path: web/src/shared/NotificationPanel.tsx
- purpose: Bell Popover content — header with mark-all-read/close, antd List of notifications, Empty state, per-item dismiss
- props/api: `NotificationPanel({ onClose })` — reads selectNotifications, dispatches markAllRead/dismissNotification
- canonical: NotificationPanel.tsx:10

## NotificationToasts
- path: web/src/shared/NotificationToasts.tsx
- purpose: NEW notifications (not seen at mount) as transient antd Alerts top-left, auto-dismissed after 5 s; close dispatches dismiss
- props/api: `NotificationToasts()` — internal ToastAlert({kind, title, body, onDismiss})
- canonical: NotificationToasts.tsx:14

## notifications.tsx
- path: web/src/shared/notifications.tsx
- purpose: Single source for per-kind notification icons — antd icons inside tinted circular Avatars (semantic token pair per kind), shared by panel + toasts
- props/api: `NotificationIcon({kind})` + `NOTIFICATION_ICONS: Record<NotificationKind, ComponentType>`
- canonical: notifications.tsx:8

## Shared inputs
## WikiPathInput
- path: web/src/shared/WikiPathInput.tsx
- purpose: Wiki path field — FolderOpenOutlined prefix opens a directory browser Modal backed by GET /api/fs/dirs (enter a subdirectory, Up to the parent, OK fills the field); the value stays hand-editable
- props/api: `WikiPathInput({ value, onChange })`
- canonical: WikiPathInput.tsx:10

## Charts (Chart.js)
## chart.ts
- path: web/src/utils/chart.ts
- purpose: Side-effect module registering all Chart.js pieces (bar/line/doughnut/arc/category/linear/tooltip) exactly once
- props/api: no exports — Chart.register(...) on import
- canonical: chart.ts:17

## ActivityChart
- path: web/src/components/dashboard/ActivityChart.tsx
- purpose: Single-series bar chart of notes per day, blue palette, tooltip with date counts; canvas role="img" + aria-label
- props/api: `ActivityChart({ counts: number[] })`
- canonical: ActivityChart.tsx:11

## ChatActivityChart
- path: web/src/components/dashboard/ChatActivityChart.tsx
- purpose: Single-series line chart of chat messages per day, hidden value axis, hover-only points
- props/api: `ChatActivityChart({ counts })` — counts: number[]
- canonical: ChatActivityChart.tsx:10

## NotesByKindChart
- path: web/src/components/dashboard/NotesByKindChart.tsx
- purpose: Doughnut of note counts by kind, series palette, legend list below
- props/api: `NotesByKindChart({ slices })` — slices: {kind, count}[]
- canonical: NotesByKindChart.tsx:11

## NotesByFolderChart
- path: web/src/components/dashboard/NotesByFolderChart.tsx
- purpose: Horizontal bar chart of notes per top-level wiki folder, blue palette
- props/api: `NotesByFolderChart({ rows })` — rows: {folder, count}[]
- canonical: NotesByFolderChart.tsx:9

## useThemeColors.ts
- path: web/src/components/dashboard/useThemeColors.ts
- purpose: Chart colors derived from the antd theme tokens via theme.useToken() (accent/hover/subtle/ink/surface); series hues read from the :root categorical palette
- props/api: `useThemeColors(): ChartColors` + `interface ChartColors {accent, accentHover, subtle, ink, surface, series: string[]}`
- canonical: useThemeColors.ts:4

## Data
## dashboardMock.ts
- path: web/src/pages/dashboard/dashboardMock.ts
- purpose: Mock data for dashboard tiles until the real index endpoints land
- props/api: data only — mockInbox, mockMeetings, mockTodos, mockRecent, mockTags, mockActivity, mockChatActivity, mockNotesByKind, mockNotesByFolder, mockStats
- canonical: dashboardMock.ts:7

## Intentional skips
- `*.test.tsx` — co-located Vitest tests; skipped by convention
- `app/App.tsx` / `main.tsx` / `theme.ts` / `index.css` — app shell and global styles; see docs/frontend.md structure section

Stale if: a file appears in web/src/components, web/src/shared, or
web/src/pages without an entry or a named skip, a component's props change, or
docs/frontend.md's component table gains a row this index lacks.
