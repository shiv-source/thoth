# Redux store (web/src/store)

Redux Toolkit owns the server-backed and shared state. Each slice lives in
slices/ with its actions, selectors, and thunks co-located; makeStore()
wires them; hooks.ts exports the typed hooks — always use
useAppDispatch/useAppSelector, never the bare versions.

## index.ts
- path: web/src/store/index.ts
- purpose: makeStore() wires all slices; exports RootState and AppDispatch
- canonical: web/src/store/index.ts

## hooks.ts
- path: web/src/store/hooks.ts
- purpose: The typed hooks — useAppDispatch / useAppSelector
- canonical: web/src/store/hooks.ts

## Slices (web/src/store/slices/)

## health
- path: healthSlice.ts
- purpose: Server health, fetched at boot (main.tsx), re-checked by the setup screen
- canonical: healthSlice.ts · docs/frontend.md §State

## settings
- path: settingsSlice.ts
- purpose: Settings loaded when the settings view mounts, saved through the slice (submit button reflects saving)
- canonical: settingsSlice.ts · docs/frontend.md §State

## conversations
- path: conversationsSlice.ts
- purpose: Conversation list — refetched on URL changes and new-chat; deletes filter the list in the slice
- canonical: conversationsSlice.ts · docs/frontend.md §State

## chat
- path: chatSlice.ts
- purpose: The live conversation — messages, streaming, thinking, lastTool, conversationId — fed by WS frames via useChat
- canonical: chatSlice.ts · docs/frontend.md §State

## connection
- path: connectionSlice.ts
- purpose: The WebSocket status, reported by ChatSocket and read by ChatPanel
- canonical: connectionSlice.ts · docs/frontend.md §State

## notifications
- path: notificationsSlice.ts
- purpose: Notification items (id/kind/title/body/read), capped ring of 50 — ephemeral UI state, never persisted
- props/api: actions notify({kind,title,body?}), markNotificationRead(id), markAllRead(), dismissNotification(id); selectors selectNotifications, selectUnreadCount
- canonical: notificationsSlice.ts:24

## searchHistory
- path: searchHistorySlice.ts
- purpose: The last committed searches (strings) — cap 10, deduped, most-recent-first; lazy localStorage load + persistSearchHistory middleware writes back
- props/api: actions commitSearch(q), clearSearchHistory(); selector selectSearchHistory
- canonical: searchHistorySlice.ts:24 · docs/frontend.md §State

Stale if: a slice appears or disappears in web/src/store/slices/, makeStore
wiring changes, or docs/frontend.md's state list gains an entry this
index lacks.
