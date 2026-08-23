# Components (web/src)

Scope: feature components live in `web/src/components/<feature>/`, cross-cutting
primitives in `web/src/shared/`, route-level views in `web/src/pages/<name>/`,
and the app shell in `web/src/app/` — App.tsx / main.tsx / theme.tsx / index.css
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
- purpose: App navigation — the `Logo` brand lockup, antd Menu over the five views (routes through useView), health footer (HealthFooter); column sits on the surface color (no border, defined by contrast)
- props/api: `AppSider()` — no props
- canonical: AppSider.tsx:26

## Logo
- path: web/src/shared/Logo.tsx
- purpose: The brand lockup — `LogoMark` plus the Fraunces wordmark; the wordmark carries the accessible name (mark is decorative)
- props/api: `Logo({ size = 28, wordmark = true, className? })`
- canonical: Logo.tsx:6

## LogoMark
- path: web/src/shared/LogoMark.tsx
- purpose: The Thoth owl mark — SVG accent tile (hover→active blue gradient from the antd tokens, unique gradient id per mount) with a white owl silhouette and amber beak
- props/api: `LogoMark({ size = 28 })`
- canonical: LogoMark.tsx:10

## HealthFooter
- path: web/src/components/layout/HealthFooter.tsx
- purpose: AppSider's bottom status bar — Badge status dot + one-line reason (All systems go / API key not configured / Wiki missing / Server unreachable) and the app version, both from the health slice
- props/api: `HealthFooter()` — no props
- canonical: HealthFooter.tsx:7

## Sidebar
- path: web/src/components/layout/Sidebar.tsx
- purpose: Chat history column (antd Layout.Sider) — Chats label, New chat primary button, and the grouped conversation list (ChatsList)
- props/api: `Sidebar()` — no props
- canonical: Sidebar.tsx:9

## ChatsList
- path: web/src/components/layout/ChatsList.tsx
- purpose: The day-grouped conversation history (one antd List per Today/Yesterday/Previous 7 days/Older); re-fetches on URL change, keeps the active item scrolled into view; deletes via text danger button with toast feedback
- props/api: `ChatsList()` — no props; internal groupByDay + relativeDate helpers
- canonical: ChatsList.tsx:14

## AppHeader
- path: web/src/shared/AppHeader.tsx
- purpose: Per-view header — title, notification bell (Badge count + Popover → NotificationPanel), optional settings button; popover visibility lives in the ui slice, Escape closes
- props/api: `AppHeader({ title, onOpenSettings? })`
- canonical: AppHeader.tsx:12

## Chat
## ChatPage
- path: web/src/pages/chat/ChatPage.tsx
- purpose: Main chat view — owns the WebSocket, compact StatusPill chips (connection/thinking/tool), the `ChatEmptyState` hero when empty, message list, Composer (with the default-model chip); token-usage footer via UsageLine
- props/api: `ChatPage({ onOpenSettings, onOpenNote })`
- canonical: ChatPage.tsx:19

## ChatEmptyState
- path: web/src/components/chat/ChatEmptyState.tsx
- purpose: Empty-conversation hero — `Logo` lockup, "Ask anything" headline, and suggested-prompt chips that send a conversation directly
- props/api: `ChatEmptyState({ onSend })`
- canonical: ChatEmptyState.tsx:15

## StatusPill
- path: web/src/components/chat/StatusPill.tsx
- purpose: Compact centered status chip — info tone (spinning accent icon, thinking/tool activity) or warning tone (amber dot, connection trouble); children carry the message
- props/api: `StatusPill({ tone = 'info', children })`
- canonical: StatusPill.tsx:8

## Composer
- path: web/src/components/chat/Composer.tsx
- purpose: Chat input — antd Input.TextArea autoSize (Enter sends, Shift+Enter newline), primary Send / default Stop buttons, two-row bar with a hint line and a model chip; draft is deliberately local state
- props/api: `Composer({ onSend, onCancel, streaming, model? })`
- canonical: Composer.tsx:7

## MessageItem
- path: web/src/components/chat/MessageItem.tsx
- purpose: One memoized chat message — user bubble plain text, assistant bubble Markdown + Tooltip + CopyButton + streaming caret; 2xl bubbles, soft shadow on assistant rows
- props/api: `MessageItem({ message: ChatMessage, streaming?, onOpenNote? })` — React.memo
- canonical: MessageItem.tsx:10

## AssistantIcon
- path: web/src/components/chat/AssistantIcon.tsx
- purpose: The small `LogoMark` owl tile to the left of every assistant message, tying the assistant to the product
- props/api: `AssistantIcon()` — no props
- canonical: AssistantIcon.tsx:5

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
- purpose: Browse-and-read wiki surface — WikiTree left (expand/collapse-all toggle dispatches the ui slice), NoteViewer inline in the content area or the branded `EmptyState`; ancestors of the open note auto-expand
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
- purpose: Wiki search synced to the URL ?q= — antd Input.Search, debounced useSearch into the search slice, antd List results (server-sanitized <mark> snippets), keyboard nav via the ui slice, branded EmptyState, recent-search history (Button rows + Clear)
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
- purpose: Segment dispatcher — maps `/settings/<section>` (defaulting to general) to one of the four route-level sub-pages (`SettingsGeneralPage`, `SettingsProvidersPage`, `SettingsGitPage`, `SettingsDoctorPage`), sharing `SettingsShell` (header + a left rail `Menu`, active item pill via the `.settings-menu` CSS rule, section rides the URL)
- props/api: `SettingsPage()` — no props
- canonical: SettingsPage.tsx:21

## SettingsShell
- path: web/src/pages/settings/SettingsShell.tsx
- purpose: The shared settings layout — AppHeader, one-line description, the left rail `Menu` (one entry per `/settings/<section>`), and the section page as the scrolling content column
- props/api: `SettingsShell({ active, children })`
- canonical: SettingsShell.tsx:23

## useSettingsForm
- path: web/src/pages/settings/useSettingsForm.tsx
- purpose: The shared settings sub-page machinery — one Form seeded from the store's settings (`setFieldsValue`), a `save` that merges only the page's fields through `settingsBody`, the transient saved/error status banner state, and the settings fetch on mount; each sub-page owns its own instance
- props/api: `useSettingsForm()` → `{ form, status, saving, hasError, save }`
- canonical: useSettingsForm.tsx:14

## SaveFooter
- path: web/src/pages/settings/components/SaveFooter.tsx
- purpose: The settings save bar shared by SettingsGeneralPage — the transient saved/error feedback `Alert` (from useSettingsForm's status) with the submit `Button` beside it, separated by a hairline; one convention, one place
- props/api: `SaveFooter({ status, saving, hasError, className? })`
- canonical: SaveFooter.tsx:7

## GitConnectSection
- path: web/src/pages/settings/components/GitConnectSection.tsx
- purpose: The not-yet-connected Git page card — PAT `Input.Password` (local state owned by the page) + Connect button; the token is stored server-side in the github_auth row
- props/api: `GitConnectSection({ token, onTokenChange, connecting, error, onConnect })`
- canonical: GitConnectSection.tsx:8

## DashboardPage
- path: web/src/pages/dashboard/DashboardPage.tsx
- purpose: Landing — greeting header, quick-action Buttons, the StatTile KPI row, the Overview widget cards, and the Insights ChartCard rows; sections separated by the shared `SectionHeader` kicker. The page wires mock data (pages/dashboard/dashboardMock.tsx, tagged with its issue until the index endpoints land) into the widget components; each widget is its own component
- props/api: `DashboardPage({ onOpenSettings })`
- canonical: DashboardPage.tsx:39

## SectionHeader
- path: web/src/shared/SectionHeader.tsx
- purpose: Page-section kicker — a small accent tick + uppercase micro-label (dashboard Overview/Insights)
- props/api: `SectionHeader({ children })`
- canonical: SectionHeader.tsx:7

## StatTile
- path: web/src/components/dashboard/StatTile.tsx
- purpose: KPI tile — tinted accent icon chip + label + large value + optional trend delta (a leading `+` tints the delta success)
- props/api: `StatTile({ icon, label, value, delta? })`
- canonical: StatTile.tsx:7

## InboxCard
- path: web/src/components/dashboard/InboxCard.tsx
- purpose: Overview "Inbox needs attention" — waiting-capture count + inbox/ file names
- props/api: `InboxCard({ count, files })`
- canonical: InboxCard.tsx:6

## MeetingsCard
- path: web/src/components/dashboard/MeetingsCard.tsx
- purpose: Overview "Today's meetings" — time chip + title/path rows, hover-reveal clock icon
- props/api: `MeetingsCard({ meetings })` — meetings: {time, title, path}[]
- canonical: MeetingsCard.tsx:13

## TodosCard
- path: web/src/components/dashboard/TodosCard.tsx
- purpose: Overview "Open todos" — checklist rows + done-progress bar
- props/api: `TodosCard({ todos })` — todos: {text, done}[]
- canonical: TodosCard.tsx:11

## RecentNotesCard
- path: web/src/components/dashboard/RecentNotesCard.tsx
- purpose: Overview "Recent notes" — most recently touched wiki paths
- props/api: `RecentNotesCard({ notes })` — notes: string[]
- canonical: RecentNotesCard.tsx:5

## RecentChatsCard
- path: web/src/components/dashboard/RecentChatsCard.tsx
- purpose: Overview "Recent chats" — the latest conversations, each opening its /chat/<id> on click
- props/api: `RecentChatsCard({ chats, onOpen })` — chats: Conversation[]; onOpen: (id) => void
- canonical: RecentChatsCard.tsx:6

## TagsCard
- path: web/src/components/dashboard/TagsCard.tsx
- purpose: Overview "Tags" — the wiki's tags as round chips jumping to the search view
- props/api: `TagsCard({ tags, onOpen })`
- canonical: TagsCard.tsx:6

## ChartCard
- path: web/src/components/dashboard/ChartCard.tsx
- purpose: Insights widget shell — a titled card wrapping one chart plus its mock-data footnote, so the four Insights cards stay identical
- props/api: `ChartCard({ title, note, children })`
- canonical: ChartCard.tsx:6

## SetupPage
- path: web/src/pages/setup/SetupPage.tsx
- purpose: antd Result listing install problems (per-problem Alerts with fix commands) + Re-check primary button; the hero icon is the `LogoMark` owl
- props/api: `SetupPage({ health, loading, onRecheck })`
- canonical: SetupPage.tsx:32

## EmptyState
- path: web/src/shared/EmptyState.tsx
- purpose: Branded empty-state placeholder — soft icon circle + title + optional description/action; replaces antd's stock gray Empty (Notes, search, notifications, providers, sync)
- props/api: `EmptyState({ icon?, title, description?, action?, className? })`
- canonical: EmptyState.tsx:8

## Notifications
## NotificationPanel
- path: web/src/shared/NotificationPanel.tsx
- purpose: Bell Popover content — header with mark-all-read/close, antd List of notifications, branded EmptyState, per-item dismiss
- props/api: `NotificationPanel({ onClose })` — reads selectNotifications, dispatches markAllRead/dismissNotification
- canonical: NotificationPanel.tsx:10

## NotificationToasts
- path: web/src/shared/NotificationToasts.tsx
- purpose: NEW notifications (not seen at mount) as transient antd Alerts top-left, each rendered by ToastAlert; close dispatches dismiss
- props/api: `NotificationToasts()`
- canonical: NotificationToasts.tsx:8

## ToastAlert
- path: web/src/shared/ToastAlert.tsx
- purpose: One transient notification toast — antd Alert (NotificationIcon, title/body, closable) that auto-dismisses via onDismiss after 5 s
- props/api: `ToastAlert({ kind, title, body?, onDismiss })`
- canonical: ToastAlert.tsx:8

## notifications.tsx
- path: web/src/shared/notifications.tsx
- purpose: Single source for per-kind notification icons — antd icons inside tinted circular Avatars (semantic token pair per kind), shared by panel + toasts
- props/api: `NotificationIcon({kind})` + `NOTIFICATION_ICONS: Record<NotificationKind, ComponentType>`
- canonical: notifications.tsx:8

## Shared inputs
## WikiPathInput
- path: web/src/shared/WikiPathInput.tsx
- purpose: Wiki path field — FolderOpenOutlined prefix opens the DirBrowserModal; the value stays hand-editable at all times
- props/api: `WikiPathInput({ value, onChange })`
- canonical: WikiPathInput.tsx:7

## DirBrowserModal
- path: web/src/shared/DirBrowserModal.tsx
- purpose: Directory picker behind WikiPathInput — Modal backed by GET /api/fs/dirs (enter a subdirectory, Up to the parent, OK reports the choice via onSelect); loads the starting directory (initial, home fallback) on open
- props/api: `DirBrowserModal({ open, initial, onCancel, onSelect })`
- canonical: DirBrowserModal.tsx:10

## Charts (Chart.js)
## chart.tsx
- path: web/src/utils/chart.tsx
- purpose: Side-effect module registering all Chart.js pieces (bar/line/doughnut/arc/category/linear/tooltip/filler) exactly once
- props/api: no exports — Chart.register(...) on import
- canonical: chart.tsx:17

## chartTheme
- path: web/src/components/dashboard/chartTheme.tsx
- purpose: Dashboard chart helpers — `verticalGradient` (top-to-bottom fill for bars/line areas, solid fallback before layout) and `tooltipStyle` (shared light tooltip chrome: white card, hairline border, soft shadow)
- props/api: `verticalGradient(context, top, bottom)` · `tooltipStyle`
- canonical: chartTheme.tsx:6

## ActivityChart
- path: web/src/components/dashboard/ActivityChart.tsx
- purpose: Single-series bar chart of notes per day, blue palette with a vertical gradient fill + shared light tooltip; canvas role="img" + aria-label
- props/api: `ActivityChart({ counts: number[] })`
- canonical: ActivityChart.tsx:11

## ChatActivityChart
- path: web/src/components/dashboard/ChatActivityChart.tsx
- purpose: Single-series line chart of chat messages per day, hidden value axis, hover-only points, soft area fill
- props/api: `ChatActivityChart({ counts })` — counts: number[]
- canonical: ChatActivityChart.tsx:10

## NotesByKindChart
- path: web/src/components/dashboard/NotesByKindChart.tsx
- purpose: Doughnut of note counts by kind, series palette, legend list below, shared light tooltip
- props/api: `NotesByKindChart({ slices })` — slices: {kind, count}[]
- canonical: NotesByKindChart.tsx:11

## NotesByFolderChart
- path: web/src/components/dashboard/NotesByFolderChart.tsx
- purpose: Horizontal bar chart of notes per top-level wiki folder, blue palette with a vertical gradient fill
- props/api: `NotesByFolderChart({ rows })` — rows: {folder, count}[]
- canonical: NotesByFolderChart.tsx:9

## useThemeColors.tsx
- path: web/src/components/dashboard/useThemeColors.tsx
- purpose: Chart colors derived from the antd theme tokens via theme.useToken() (accent/hover/subtle/ink/surface); series hues read from the :root categorical palette
- props/api: `useThemeColors(): ChartColors` + `interface ChartColors {accent, accentHover, subtle, ink, surface, series: string[]}`
- canonical: useThemeColors.tsx:4

## Data
## dashboardMock.tsx
- path: web/src/pages/dashboard/dashboardMock.tsx
- purpose: Mock data for dashboard tiles until the real index endpoints land
- props/api: data only — mockInbox, mockMeetings, mockTodos, mockRecent, mockTags, mockActivity, mockChatActivity, mockNotesByKind, mockNotesByFolder, mockStats
- canonical: dashboardMock.tsx:7

## Intentional skips
- `*.test.tsx` — co-located Vitest tests; skipped by convention
- `app/App.tsx` / `main.tsx` / `theme.tsx` / `index.css` — app shell and global styles; see docs/frontend.md structure section

Stale if: a file appears in web/src/components, web/src/shared, or
web/src/pages without an entry or a named skip, a component's props change, or
docs/frontend.md's component table gains a row this index lacks.
