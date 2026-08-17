# Hooks (web/src/hooks)

Each entry: path, purpose, props/api, canonical. Coverage: all five hooks;
a missing hook means this index is stale.

## useChat
- path: web/src/hooks/useChat.ts
- purpose: Adapts the chat slice to the WebSocket — WS frames become store dispatches, send/cancel/load/reset touch the socket; conversation state survives component remounts in the chat slice
- props/api: `useChat(socket: ChatSocket | null) => { messages, streaming, conversationId, lastTool, thinking, thinkingText, send, cancel, load, reset }`; re-exports `ChatMessage`
- canonical: useChat.ts:29 · docs/frontend.md §Hooks

## useSearch
- path: web/src/hooks/useSearch.ts
- purpose: Debounced (300 ms) search with abort + sequence guards so slow responses can't overwrite newer ones
- props/api: `useSearch(query: string) => { results: SearchResult[], loading: boolean }`
- canonical: useSearch.ts:4 · docs/frontend.md §Hooks

## useConversationRoute
- path: web/src/hooks/useConversationRoute.ts
- purpose: Keeps the URL /chat/<uuid> and the active conversation in sync (URL → state on popstate, state → URL via pushState)
- props/api: `useConversationRoute({ socket, conversationId, load, reset, onError })`; also exports `navigate(path: string)`
- canonical: useConversationRoute.ts:37

## useView
- path: web/src/hooks/useView.ts
- purpose: Pathname-based view routing — maps the URL to a view plus segment/query, push URLs with popstate dispatch
- props/api: `useView(): View` · `useViewRoute(): ViewRoute` · `navigateView(v)` · `navigateSegment(v, segment)` · `navigateNote(path)`; `View = 'chat' | 'notes' | 'dashboard' | 'search' | 'settings'`
- canonical: useView.ts:73

## useViewShortcuts
- path: web/src/hooks/useViewShortcuts.ts
- purpose: Binds Cmd/Ctrl+1..4 to dashboard/chat/notes/search and Cmd/Ctrl+K to search
- props/api: `useViewShortcuts(): void` — window listener removed on unmount
- canonical: useViewShortcuts.ts:9

Stale if: a new hook appears in web/src/hooks, a signature above changes,
or docs/frontend.md's hooks section gains an entry this index lacks.
