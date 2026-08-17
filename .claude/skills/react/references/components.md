# Components (web/src/components)

Scope: this directory only — App.tsx / main.tsx / index.css live in web/src/
and are covered by docs/frontend.md. Each entry: path, purpose, props/api,
canonical. Skipped: *.test.tsx — co-located Vitest tests; skipped by convention.

## App shell & navigation
## NavRail
- path: web/src/components/NavRail.tsx
- purpose: Persistent left rail switching dashboard/chat/notes/search views + settings button; routes through useView
- props/api: `NavRail()` — no props
- canonical: NavRail.tsx:14

## Sidebar
- path: web/src/components/Sidebar.tsx
- purpose: Left rail — branding, New chat button, day-grouped conversation history (ChatsList), health/version footer
- props/api: `Sidebar({ health, loading })`
- canonical: Sidebar.tsx:184

## TopBar
- path: web/src/components/TopBar.tsx
- purpose: Header bar — title, unread-count notification bell opening NotificationPanel, optional settings button
- props/api: `TopBar({ title, onOpenSettings? })`
- canonical: TopBar.tsx:9

## Chat
## ChatPanel
- path: web/src/components/ChatPanel.tsx
- purpose: Main chat view — owns the WebSocket, connection-status banner, thinking/tool status lines, message list, Composer
- props/api: `ChatPanel({ onOpenSettings })`
- canonical: ChatPanel.tsx:18

## Composer
- path: web/src/components/Composer.tsx
- purpose: Chat input form — Enter-to-send textarea, Send/Stop toggle
- props/api: `Composer({ onSend, onCancel, streaming })`
- canonical: Composer.tsx:3

## MessageItem
- path: web/src/components/MessageItem.tsx
- purpose: One chat message — user bubble plain-text, assistant bubble Markdown + copy button + streaming caret
- props/api: `MessageItem({ message: ChatMessage, streaming? })`
- canonical: MessageItem.tsx:6

## Markdown
- path: web/src/components/Markdown.tsx
- purpose: GFM renderer with Shiki code blocks (via CodeBlock) in a shared prose wrapper
- props/api: `Markdown({ children, trailing?, className = '' })`
- canonical: Markdown.tsx:28

## CodeBlock
- path: web/src/components/CodeBlock.tsx
- purpose: Fenced code block via Shiki (github-dark, module-level 200-entry cache) + copy button, plain <pre> fallback
- props/api: `CodeBlock({ code, lang })`
- canonical: CodeBlock.tsx:24

## CopyButton
- path: web/src/components/CopyButton.tsx
- purpose: Copies text to clipboard, flips to a check for 2 s, optional success toast
- props/api: `CopyButton({ text, label, toast?, className? })`
- canonical: CopyButton.tsx:9

## Wiki & notes
## NotesView
- path: web/src/components/NotesView.tsx
- purpose: Browse-and-read wiki surface — WikiTree left, NoteViewer or EmptyState right
- props/api: `NotesView({ openPath, onOpenNote, onOpenSettings })`
- canonical: NotesView.tsx:12

## WikiTree
- path: web/src/components/WikiTree.tsx
- purpose: Wiki directory as a folder tree via Tree; fetches on mount, focus, and chat-turn end
- props/api: `WikiTree({ openPath, onOpenNote, expandedKeys, onExpandedChange, onDirsChange? })`
- canonical: WikiTree.tsx:13

## Tree
- path: web/src/components/Tree.tsx
- purpose: Reusable accessible folder tree — roving tabIndex keyboard nav, memoized rows, controlled or internal expansion
- props/api: `Tree<T>({ nodes, getKey, getLabel, isDir, getChildren, renderIcon, renderTrailing?, renderTooltip?, onSelect, selectedKey, expandedKeys?, onExpandedChange? })` + `TreeProps<T>`
- canonical: Tree.tsx:109

## NoteViewer
- path: web/src/components/NoteViewer.tsx
- purpose: Side-panel viewer — fetches a note by path, renders Markdown, copy-raw + close
- props/api: `NoteViewer({ path, onClose })`
- canonical: NoteViewer.tsx:7

## Search
## SearchPanel
- path: web/src/components/SearchPanel.tsx
- purpose: Wiki search synced to the URL ?q= — debounced results, keyboard nav, recent-search history
- props/api: `SearchPanel({ onOpen: (path: string) => void })`
- canonical: SearchPanel.tsx:9

## SearchView
- path: web/src/components/SearchView.tsx
- purpose: Full-page search surface — TopBar plus the shared SearchPanel
- props/api: `SearchView({ onOpenNote, onOpenSettings })`
- canonical: SearchView.tsx:6

## Settings & dashboard
## SettingsView
- path: web/src/components/SettingsView.tsx
- purpose: Tabbed settings screen (General, Doctor, Git) — wiki path/model form, doctor checks, GitHub connect + repo sync
- props/api: `SettingsView()` — no props; the tab rides the URL segment
- canonical: SettingsView.tsx:49

## DashboardView
- path: web/src/components/DashboardView.tsx
- purpose: Dashboard landing — greeting, stat tiles, action buttons, mock overview/insights cards + charts
- props/api: `DashboardView({ onOpenSettings })`
- canonical: DashboardView.tsx:64

## SetupScreen
- path: web/src/components/SetupScreen.tsx
- purpose: Full-screen blocker listing install problems with fix commands + Re-check button
- props/api: `SetupScreen({ health, loading, onRecheck })`
- canonical: SetupScreen.tsx:31

## Card
- path: web/src/components/Card.tsx
- purpose: App-wide section panel with uppercase kicker title
- props/api: `Card({ title, children })`
- canonical: Card.tsx:5

## Notifications
## NotificationPanel
- path: web/src/components/NotificationPanel.tsx
- purpose: Dropdown bell panel — notification list, mark-all-read, per-item dismiss, empty state
- props/api: `NotificationPanel({ onClose })` — reads selectNotifications, dispatches markAllRead/dismissNotification
- canonical: NotificationPanel.tsx:8

## NotificationToasts
- path: web/src/components/NotificationToasts.tsx
- purpose: NEW notifications (not seen at mount) as transient toast cards top-left, auto-dismissed after 5 s
- props/api: `NotificationToasts()` — internal ToastCard({kind, title, body, onDismiss})
- canonical: NotificationToasts.tsx:14

## notifications.tsx
- path: web/src/components/notifications.tsx
- purpose: Single source for per-kind notification emoji icons, shared by panel + toasts
- props/api: `NotificationIcon({kind})` + `NOTIFICATION_ICONS: Record<NotificationKind, string>`
- canonical: notifications.tsx:13

## Shared UI
## EmptyState
- path: web/src/components/EmptyState.tsx
- purpose: Centered placeholder — emoji icon, title, optional hint
- props/api: `EmptyState({ icon, title, hint, className = '' })`
- canonical: EmptyState.tsx:4

## IconButton
- path: web/src/components/IconButton.tsx
- purpose: Subtle hover button used across the chrome; aria-label required
- props/api: `IconButton({ label, onClick, className = '', children })`
- canonical: IconButton.tsx:4

## Tooltip
- path: web/src/components/Tooltip.tsx
- purpose: Radix UI tooltip wrapper — accessible, collision-flipping, the app's dark bubble styling
- props/api: `Tooltip({ label, children, side = 'top', align = 'center' })`
- canonical: Tooltip.tsx:7

## Toast
- path: web/src/components/Toast.tsx
- purpose: Toast system — provider renders a bottom-center stack; auto-dismiss after 3 s or on click
- props/api: `ToastProvider({ children })` + `useToast(): { toast(message, kind?) }` + type `ToastKind`
- canonical: Toast.tsx:19

## Charts (Chart.js)
## chartSetup.ts
- path: web/src/components/chartSetup.ts
- purpose: Side-effect module registering all Chart.js pieces (bar/line/doughnut/arc/category/linear/tooltip) exactly once
- props/api: no exports — Chart.register(...) on import
- canonical: chartSetup.ts:17

## ActivityChart
- path: web/src/components/ActivityChart.tsx
- purpose: Single-series emerald bar chart of notes per day, theme-colored, tooltip with date counts
- props/api: `ActivityChart({ counts: number[] })`
- canonical: ActivityChart.tsx:11

## ChatActivityChart
- path: web/src/components/ChatActivityChart.tsx
- purpose: Single-series line chart of chat messages per day, hidden value axis, hover-only points
- props/api: `ChatActivityChart({ counts })` — counts: number[]
- canonical: ChatActivityChart.tsx:10

## NotesByKindChart
- path: web/src/components/NotesByKindChart.tsx
- purpose: Doughnut of note counts by kind, series-palette hues, legend list below
- props/api: `NotesByKindChart({ slices })` — slices: {kind, count}[]
- canonical: NotesByKindChart.tsx:11

## NotesByFolderChart
- path: web/src/components/NotesByFolderChart.tsx
- purpose: Horizontal bar chart of notes per top-level wiki folder, single emerald series
- props/api: `NotesByFolderChart({ rows })` — rows: {folder, count}[]
- canonical: NotesByFolderChart.tsx:9

## useThemeColors.ts
- path: web/src/components/useThemeColors.ts
- purpose: Chart colors read from --thoth-* CSS variables, re-renders on OS theme flips
- props/api: `useThemeColors(): ChartColors` + `chartColors()` + `interface ChartColors {accent, accentHover, subtle, ink, surface, series: string[]}`
- canonical: useThemeColors.ts:36

## Data
## dashboardMock.ts
- path: web/src/components/dashboardMock.ts
- purpose: Mock data for dashboard tiles until the real index endpoints land
- props/api: data only — mockInbox, mockMeetings, mockTodos, mockRecent, mockTags, mockActivity, mockChatActivity, mockNotesByKind, mockNotesByFolder, mockStats
- canonical: dashboardMock.ts:7

## Intentional skips
- *.test.tsx — co-located Vitest tests; skipped by convention
- App.tsx / main.tsx / index.css — live in web/src/, outside this dir; see docs/frontend.md structure section

Stale if: a file appears in web/src/components without an entry or a named
skip, a component's props change, or docs/frontend.md's component table
gains a row this index lacks.
