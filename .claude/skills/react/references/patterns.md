# Frontend patterns — the cross-cutting conventions

## Ant Design first
- All UI chrome renders through antd v6 components (Layout, Menu, Button,
  Badge, Popover, Tooltip, List, Empty, Skeleton, Tabs, Form, Select,
  AutoComplete, Input, Switch, Alert, Drawer-free inline panels, Tree,
  Card, Statistic, Progress, Result, App.useApp().message)
- Icons come from @ant-design/icons (aria-hidden for decorative use —
  antd icons default to role="img" + aria-label, so always pass
  aria-hidden when the surrounding element already carries the name)
- Tailwind utilities are for layout/spacing only; never raw hex or palette
  classes in components — colors via the semantic tokens
- Check the antd MCP (or ant.design) for a component's API before writing
  custom UI; virtual={false}/motion={false} on Tree/Select/AutoComplete —
  small local lists and jsdom compatibility
- canonical: web/src/theme.ts · docs/frontend.md §Design system

## The API boundary (zod)
- web/src/api/client.ts is the typed REST client: axios + zod, every
  response parsed by a zod schema — validation at the boundary
- TS strict, zero any (CLAUDE.md invariant); DTOs must match the Go side
- Tests use mockAxios and assert the parsed payload, not the transport
- canonical: web/src/api/client.ts · docs/api.md

## The WS protocol (ChatSocket)
- Out: send {text}, cancel, resume {conversation_id}, open {conversation_id}, new_chat
- In: assistant_start, assistant_thinking {text}, assistant_delta {text}, tool_activity {tool, detail}, turn_done {conversation_id}, error {message}
- Reconnects exactly once after 1 s, sending resume from onopen so the turn re-syncs
- open pins the server-side conversation and never becomes the reconnect-resume id
- Server message types in internal/api/chat.go must match web/src/ws/chat.ts — CHANGE BOTH SIDES
- canonical: web/src/ws/chat.ts · docs/api.md §WebSocket chat

## Test doubles (web/src/test)
- mockAxios — the axios mock for client.ts consumers
- fakeWS — the scripted WebSocket double for ChatSocket consumers
- renderWithStore — renders with a fresh store + the antd App wrapper
  (mirrors main.tsx, so App.useApp().message works in tests)
- setup — Vitest setup file (stubs ResizeObserver/matchMedia — antd needs both)
- Rule: use these; never hand-roll mocks of the app itself (CLAUDE.md)
- jsdom quirks: antd motion callbacks never complete — assert store state
  for close transitions; prefer findBy* over getBy* for async renders

## State placement
- Server-backed, shared, and screen-spanning data → Redux slices
  (health, settings, conversations, chat, connection, notifications,
  searchHistory, ui, wiki, note, search, doctor, git)
- Deliberately NOT in Redux: per-keystroke drafts (Composer text — a
  dispatch per character is an anti-pattern), antd Form values
  (rc-field-form owns them; seed via setFieldsValue), toasts
  (App.useApp().message), the ChatSocket instance (non-serializable),
  and the URL (routing source of truth)
- canonical: docs/frontend.md §State

## Design tokens
- web/src/theme.ts holds the single antd ThemeConfig — blue primary
  #1677ff, borderRadius 6, cssVar: {} (antd emits --ant-* variables),
  hashed: false; LIGHT THEME ONLY — no dark mode, no OS-scheme flipping
- index.css @theme tokens bridge to the antd variables
  (--color-accent: var(--ant-color-primary, #1677ff)); neutrals mirror
  the antd light palette; chart series are the blue family
- Use semantic classes (bg-surface, text-ink, border-line, bg-accent…);
  no raw hex in components
- Display type: Fraunces (self-hosted via @fontsource-variable — no
  runtime network), brand wordmark only; body/headings: antd's font stack
- canonical: web/src/theme.ts · web/src/index.css · docs/frontend.md §Design system

## Routing
- Hand-rolled: useView maps the pathname to a view; useConversationRoute keeps /chat/<uuid> synced; SearchPanel rides ?q=; the open note rides /notes/<path>
- canonical: web/src/hooks/useView.ts · web/src/hooks/useConversationRoute.ts

## Package discipline
- pnpm only — never npm; workspace root; lockfile committed; save-exact
- antd + @ant-design/icons are direct deps; icons only from @ant-design/icons
- make web syncs web/dist into internal/webui/dist — required before go build/test
- canonical: CLAUDE.md §Toolchain

Stale if: a zod schema changes shape without a client.ts update, the WS
frame set changes, a new test double appears in web/src/test, new
semantic tokens land in index.css without an entry above, or the antd
theme/icon conventions change.
