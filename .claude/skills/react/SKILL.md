---
name: react
description: >-
  Thoth React frontend — components, hooks, Redux slices, REST client,
  WS chat client, Vitest tests, Tailwind v4 design tokens.
---

# React frontend (web/src) — procedures & expertise

## When to use
- Adding or changing components, hooks, or Redux slices under web/src/
- Wiring new REST calls (web/src/api/client.ts — zod at the boundary)
- Touching the WS chat client (web/src/ws/chat.ts — types mirrored in internal/api/chat.go)
- Frontend tests (Vitest + the web/src/test doubles)
- Not backend work — use the `go` skill
- Not note-taking behavior — that's the wiki rulebook (~/.thoth/wiki/CLAUDE.md)

## Key files
- web/src/components/ — one component per file, co-located .test.tsx; App.tsx/main.tsx compose them
- web/src/hooks/ — useChat, useSearch, useConversationRoute, useView, useViewShortcuts
- web/src/store/ — Redux Toolkit: index.ts (makeStore), hooks.ts (typed hooks), slices/ (one per feature)
- web/src/api/client.ts — typed REST client (axios + zod)
- web/src/ws/chat.ts — ChatSocket: protocol frames, reconnect/resume
- web/src/test/ — mockAxios, fakeWS, renderWithStore, setup
- web/src/index.css — Tailwind v4 @theme tokens

## Workflows

### 1. Add a component
1. One component per file in web/src/components/<Name>.tsx; icons from lucide-react
2. Style with semantic tokens (bg-surface, text-ink, border-line) — no raw hex (see references/patterns.md)
3. Co-locate the test <Name>.test.tsx using the renderWithStore/mockAxios doubles
4. Hover hints use the Tooltip wrapper; icon-only buttons use IconButton
5. Update docs/frontend.md's component table AND references/components.md in the same commit

### 2. Add a Redux slice
1. Create web/src/store/slices/<name>Slice.ts — actions, selectors, thunks co-located
2. Wire it in web/src/store/index.ts (makeStore)
3. Use the typed hooks (useAppDispatch/useAppSelector from store/hooks.ts) — never bare useDispatch/useSelector
4. Only shared or screen-spanning state lives in the store; component-local state stays in hooks/components (docs/frontend.md)
5. Co-locate the slice test; update references/store.md in the same commit

### 3. Add a hook
1. New file web/src/hooks/useX.ts, exported as a named function
2. Every useEffect subscription/timer/socket gets a cleanup that runs on unmount (CLAUDE.md memory rule)
3. Co-locate the test; update references/hooks.md + docs/frontend.md in the same commit

### 4. Wire an API call
1. Add or extend the endpoint in web/src/api/client.ts with a zod schema — validation at the boundary, zero any (CLAUDE.md invariant)
2. Server side must match: use the `go` skill for internal/api; DTOs on both sides
3. Test with mockAxios — assert the parsed payload, not the transport
4. Update docs/api.md in the same commit

### 5. Test a component/slice
1. Use the doubles in web/src/test/ (mockAxios, fakeWS, renderWithStore, setup) — never hand-rolled mocks of the app itself
2. Assert real outcomes: what renders, what's dispatched, what the user sees
3. Run: pnpm test (Vitest) — pnpm only, never npm
4. Every behavior change ships with a test (CLAUDE.md)

### 6. Touch the WS client
1. CHANGE BOTH SIDES: web/src/ws/chat.ts (client types) AND internal/api/chat.go (server frames) — they must match
2. Frames: send/cancel/resume/open/new_chat out; assistant_*/tool_activity/turn_done/error in (docs/api.md)
3. Reconnect behavior: exactly once after 1 s, resume from onopen — changing it changes chat recovery semantics
4. Test with fakeWS; update docs/api.md in the same commit

## Gotchas
- pnpm only — never npm; the workspace lockfile (root pnpm-lock.yaml) is committed
- TS strict, zero any — eslint enforces; zod at the API boundary
- make web is REQUIRED before go build/test — frontend changes don't reach the binary without it
- WS is chat-only transport; REST for everything else
- Design tokens flip under prefers-color-scheme; dark mode follows the OS — no toggle
- Every useEffect has cleanup; no setInterval without clearInterval

## Canonical docs
- docs/frontend.md — structure, components, hooks, state, design system
- docs/api.md — REST endpoints + WS protocol (both sides)
- docs/architecture.md — the two layers

## Maintenance
Derived view — after a behavior change, update this skill + docs/ in the
same commit, then run `graphify update .`. Stale if: a workflow's file
paths stop resolving, or docs/frontend.md gains a component this skill's
workflow list doesn't mention.
