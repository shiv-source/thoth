# Redux store (web/src/store)

Redux Toolkit owns the server-backed, shared, and screen-spanning state. Each
slice lives in slices/ with its actions, selectors, and thunks co-located;
makeStore() wires them; hooks.ts exports the typed hooks — always use
useAppDispatch/useAppSelector, never the bare versions.

## index.ts
- path: web/src/store/index.ts
- purpose: makeStore() wires all slices (plus the persistSearchHistory middleware); exports RootState and AppDispatch
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
- purpose: Settings loaded on mount, saved through the slice (submit button reflects saving); also holds the /api/models picker list (fetchModels)
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

## ui
- path: uiSlice.ts
- purpose: Screen-spanning chrome state — notification-panel open, notes-tree expansion, search keyboard selection
- props/api: actions setNotificationsOpen(b), setNotesExpandedKeys(keys), setSearchActive(i), setGitReposOpen(b) (unused — AutoComplete owns its dropdown); selectors selectNotificationsOpen, selectNotesExpandedKeys, selectSearchActive, selectGitReposOpen
- canonical: uiSlice.ts:16

## wiki
- path: wikiSlice.ts
- purpose: The wiki tree — fetchTree thunk; refetched on mount, window focus, and chat-turn end; exports the shared collectTreeInfo(nodes) walker (all dirs + per-dir file counts)
- props/api: thunk fetchTree(); selectors selectWikiNodes, selectWikiLoading, selectWikiError
- canonical: wikiSlice.ts:5

## note
- path: noteSlice.ts
- purpose: The open note's content — fetchNote(path); fulfilled results for a stale path are discarded (fast note switching never shows the wrong note)
- props/api: thunk fetchNote(path); selectors selectNoteContent, selectNoteLoading, selectNoteError
- canonical: noteSlice.ts:6

## search
- path: searchSlice.ts
- purpose: Live search results — searchNotes(q) with AbortSignal support; the slice's query guard drops stale responses and treats aborts as intentional, not errors; clearSearch resets
- props/api: thunk searchNotes(q, {signal}); action clearSearch(); selectors selectSearchResults, selectSearchLoading, selectSearchError
- canonical: searchSlice.ts:8

## doctor
- path: doctorSlice.ts
- purpose: The doctor check rows — runDoctor thunk (shared check suite, GET /api/doctor)
- props/api: thunk runDoctor(); selectors selectDoctorChecks, selectDoctorRunning, selectDoctorError
- canonical: doctorSlice.ts:5

## git
- path: gitSlice.ts
- purpose: GitHub wiring — fetchGitAuth (missing auth = not-connected, not an error), fetchGitRepos, connectGit (server messages surface as errors), pushWiki (ok:false business failures surface as errors), disconnectGit
- props/api: thunks fetchGitAuth(), fetchGitRepos(), connectGit(token), pushWiki(url), disconnectGit(); selectors selectGitAuth, selectGitRepos, selectGitLoading, selectGitConnecting, selectGitPushing, selectGitError
- canonical: gitSlice.ts:9

Stale if: a slice appears or disappears in web/src/store/slices/, makeStore
wiring changes, or docs/frontend.md's state list gains an entry this
index lacks.
