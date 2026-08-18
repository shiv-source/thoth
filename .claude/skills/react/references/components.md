# Components (web/src/components)

Scope: this directory only — App.tsx / main.tsx / theme.ts / index.css live
in web/src/ and are covered by docs/frontend.md. Each entry: path, purpose,
props/api, canonical. Skipped: *.test.tsx — co-located Vitest tests; skipped
by convention. All chrome renders through antd v6 components; icons come
from lucide-react (aria-hidden).

## App shell & navigation
## AppSider
- path: web/src/components/AppSider.tsx
- purpose: App navigation — Fraunces brand wordmark, antd Menu over the five views (routes through useView), health footer (Badge status + version from the health slice)
- props/api: `AppSider()` — no props
- canonical: AppSider.tsx:26

## AppHeader
- path: web/src/components/AppHeader.tsx
- purpose: Per-view header — title, notification bell (Badge count + Popover → NotificationPanel), optional settings button; popover visibility lives in the ui slice, Escape closes
- props/api: `AppHeader({ title, onOpenSettings? })`
- canonical: AppHeader.tsx:12

## Sidebar
- path: web/src/components/Sidebar.tsx
- purpose: Chat history column (antd Layout.Sider) — Chats label, New chat primary button, day-grouped conversation Lists; deletes via text danger button; toasts via App.useApp().message
- props/api: `Sidebar()` — no props; internal ChatsList()
- canonical: Sidebar.tsx:189

## Chat
## ChatPanel
- path: web/src/components/ChatPanel.tsx
- purpose: Main chat view — owns the WebSocket, Alert banners (connection/thinking/tool status), message list, Composer
- props/api: `ChatPanel({ onOpenSettings })`
- canonical: ChatPanel.tsx:18

## Composer
- path: web/src/components/Composer.tsx
- purpose: Chat input — antd Input.TextArea autoSize (Enter sends, Shift+Enter newline), primary Send / default Stop buttons; draft is deliberately local state
- props/api: `Composer({ onSend, onCancel, streaming })`
- canonical: Composer.tsx:5

## MessageItem
- path: web/src/components/MessageItem.tsx
- purpose: One memoized chat message — user bubble plain text, assistant bubble Markdown + antd Tooltip + CopyButton + streaming caret
- props/api: `MessageItem({ message: ChatMessage, streaming? })` — React.memo
- canonical: MessageItem.tsx:9

## Markdown
- path: web/src/components/Markdown.tsx
- purpose: GFM renderer with Shiki code blocks (via CodeBlock) in the shared prose wrapper (light only)
- props/api: `Markdown({ children, trailing?, className = '' })`
- canonical: Markdown.tsx:29

## CodeBlock
- path: web/src/components/CodeBlock.tsx
- purpose: Fenced code block via Shiki (github-light, module-level 200-entry cache) + copy button, plain <pre> fallback
- props/api: `CodeBlock({ code, lang })`
- canonical: CodeBlock.tsx:24

## CopyButton
- path: web/src/components/CopyButton.tsx
- purpose: Shared copy control — antd text Button, clipboard write, check flip for 2 s, optional message.success toast; layout via caller className
- props/api: `CopyButton({ text, label, toast?, className? })`
- canonical: CopyButton.tsx:9

## Wiki & notes
## NotesView
- path: web/src/components/NotesView.tsx
- purpose: Browse-and-read wiki surface — WikiTree left (expand/collapse-all toggle dispatches the ui slice), NoteViewer inline in the content area or Empty placeholder; ancestors of the open note auto-expand
- props/api: `NotesView({ openPath, onOpenNote, onOpenSettings })`
- canonical: NotesView.tsx:14

## WikiTree
- path: web/src/components/WikiTree.tsx
- purpose: Wiki directory via antd Tree.DirectoryTree (virtual={false}, motion={false}); data from the wiki slice, expansion from the ui slice; refetches on mount, chat-turn end, window focus; per-folder file-count Tooltips in the title renderer
- props/api: `WikiTree({ openPath, onOpenNote })`
- canonical: WikiTree.tsx:41

## NoteViewer
- path: web/src/components/NoteViewer.tsx
- purpose: Inline note reader filling the content area — note content from the note slice (stale-path responses discarded), Skeleton/Alert states, Copy raw; Esc or ✕ routes through onClose (the open note is the URL, not an overlay)
- props/api: `NoteViewer({ path: string, onClose })`
- canonical: NoteViewer.tsx:11

## Search
## SearchPanel
- path: web/src/components/SearchPanel.tsx
- purpose: Wiki search synced to the URL ?q= — antd Input.Search, debounced useSearch into the search slice, antd List results (server-sanitized <mark> snippets), keyboard nav via the ui slice, Empty state, recent-search history (Button rows + Clear)
- props/api: `SearchPanel({ onOpen: (path: string) => void })`
- canonical: SearchPanel.tsx:14

## SearchView
- path: web/src/components/SearchView.tsx
- purpose: Full-page search surface — AppHeader plus the shared SearchPanel
- props/api: `SearchView({ onOpenNote, onOpenSettings })`
- canonical: SearchView.tsx:6

## Settings & dashboard
## SettingsView
- path: web/src/components/SettingsView.tsx
- purpose: antd Tabs (left) over General/Doctor/Git — General: Form (wiki path Input + model Select grouped by provider, save loading, saved/error Alerts); Doctor: runDoctor slice + check List; Git: PAT Input.Password connect, account card (Avatar), AutoComplete repo picker with public-repo guard, Switch auto-sync, Initialize & Push. Async state in the settings/doctor/git slices; tab rides the URL segment
- props/api: `SettingsView()` — no props; internal GeneralTab/DoctorTab/GitTab
- canonical: SettingsView.tsx:39

## DashboardView
- path: web/src/components/DashboardView.tsx
- purpose: Landing — greeting, Statistic KPI tiles, quick-action Buttons, Overview cards (inbox/meetings/todos+Progress/recent notes/real recent chats/tags), Insights (four Chart.js charts, blue palette). Mock data tagged with its issue
- props/api: `DashboardView({ onOpenSettings })`
- canonical: DashboardView.tsx:66

## SetupScreen
- path: web/src/components/SetupScreen.tsx
- purpose: antd Result listing install problems (per-problem Alerts with fix commands) + Re-check primary button
- props/api: `SetupScreen({ health, loading, onRecheck })`
- canonical: SetupScreen.tsx:31

## Notifications
## NotificationPanel
- path: web/src/components/NotificationPanel.tsx
- purpose: Bell Popover content — header with mark-all-read/close, antd List of notifications, Empty state, per-item dismiss
- props/api: `NotificationPanel({ onClose })` — reads selectNotifications, dispatches markAllRead/dismissNotification
- canonical: NotificationPanel.tsx:10

## NotificationToasts
- path: web/src/components/NotificationToasts.tsx
- purpose: NEW notifications (not seen at mount) as transient antd Alerts top-left, auto-dismissed after 5 s; close dispatches dismiss
- props/api: `NotificationToasts()` — internal ToastAlert({kind, title, body, onDismiss})
- canonical: NotificationToasts.tsx:12

## notifications.tsx
- path: web/src/components/notifications.tsx
- purpose: Single source for per-kind notification icons — antd icons inside tinted circular Avatars (semantic token pair per kind), shared by panel + toasts
- props/api: `NotificationIcon({kind})` + `NOTIFICATION_ICONS: Record<NotificationKind, ComponentType>`
- canonical: notifications.tsx:9

## Charts (Chart.js)
## chartSetup.ts
- path: web/src/components/chartSetup.ts
- purpose: Side-effect module registering all Chart.js pieces (bar/line/doughnut/arc/category/linear/tooltip) exactly once
- props/api: no exports — Chart.register(...) on import
- canonical: chartSetup.ts:17

## ActivityChart
- path: web/src/components/ActivityChart.tsx
- purpose: Single-series bar chart of notes per day, blue palette, tooltip with date counts; canvas role="img" + aria-label
- props/api: `ActivityChart({ counts: number[] })`
- canonical: ActivityChart.tsx:11

## ChatActivityChart
- path: web/src/components/ChatActivityChart.tsx
- purpose: Single-series line chart of chat messages per day, hidden value axis, hover-only points
- props/api: `ChatActivityChart({ counts })` — counts: number[]
- canonical: ChatActivityChart.tsx:10

## NotesByKindChart
- path: web/src/components/NotesByKindChart.tsx
- purpose: Doughnut of note counts by kind, series palette, legend list below
- props/api: `NotesByKindChart({ slices })` — slices: {kind, count}[]
- canonical: NotesByKindChart.tsx:11

## NotesByFolderChart
- path: web/src/components/NotesByFolderChart.tsx
- purpose: Horizontal bar chart of notes per top-level wiki folder, blue palette
- props/api: `NotesByFolderChart({ rows })` — rows: {folder, count}[]
- canonical: NotesByFolderChart.tsx:9

## useThemeColors.ts
- path: web/src/components/useThemeColors.ts
- purpose: Chart colors derived from the antd theme tokens via theme.useToken() (accent/hover/subtle/ink/surface); series hues read from the :root categorical palette
- props/api: `useThemeColors(): ChartColors` + `interface ChartColors {accent, accentHover, subtle, ink, surface, series: string[]}`
- canonical: useThemeColors.ts:19

## Data
## dashboardMock.ts
- path: web/src/components/dashboardMock.ts
- purpose: Mock data for dashboard tiles until the real index endpoints land
- props/api: data only — mockInbox, mockMeetings, mockTodos, mockRecent, mockTags, mockActivity, mockChatActivity, mockNotesByKind, mockNotesByFolder, mockStats
- canonical: dashboardMock.ts:7

## Intentional skips
- *.test.tsx — co-located Vitest tests; skipped by convention
- App.tsx / main.tsx / theme.ts / index.css — live in web/src/, outside this dir; see docs/frontend.md structure section

Stale if: a file appears in web/src/components without an entry or a named
skip, a component's props change, or docs/frontend.md's component table
gains a row this index lacks.
