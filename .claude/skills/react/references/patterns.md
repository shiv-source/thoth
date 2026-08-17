# Frontend patterns — the cross-cutting conventions

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
- renderWithStore — renders with the real store provider
- setup — Vitest setup file
- Rule: use these; never hand-roll mocks of the app itself (CLAUDE.md)

## State placement
- Shared or screen-spanning data → Redux slices
- Component-local state (form fields, tree expansion, debounce, openNote) → hooks/components
- canonical: docs/frontend.md §State

## Design tokens
- Tokens in web/src/index.css (@theme) resolve to CSS custom properties that flip under prefers-color-scheme — one semantic class works in both themes
- The five semantic groups: app/surface/raised (bg-surface…), line (border-line…), subtle/ink/heading (text-ink…); emerald accent (#059669 → #34d399)
- Use semantic classes; no raw hex in components; dark mode follows the OS — no toggle
- Display type: Fraunces (self-hosted via @fontsource-variable — no runtime network); body: system stack
- canonical: web/src/index.css · docs/frontend.md §Design system

## Routing
- Hand-rolled: useView maps the pathname to a view; useConversationRoute keeps /chat/<uuid> synced; SearchPanel rides ?q=
- canonical: web/src/hooks/useView.ts · web/src/hooks/useConversationRoute.ts

## Package discipline
- pnpm only — never npm; workspace root; lockfile committed; save-exact
- make web syncs web/dist into internal/webui/dist — required before go build/test
- canonical: CLAUDE.md §Toolchain

Stale if: a zod schema changes shape without a client.ts update, the WS
frame set changes, a new test double appears in web/src/test, or new
semantic tokens land in index.css without an entry above.
