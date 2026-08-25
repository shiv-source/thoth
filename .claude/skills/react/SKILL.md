---
name: react
description: >-
  Thoth React frontend — antd v6 components, hooks, Redux slices, REST client,
  WS chat client, Vitest tests, design tokens.
---

# React frontend (web/src) — procedures & expertise

## When to use
- Adding or changing components, hooks, or Redux slices under web/src/
- Wiring new REST calls (web/src/api/client.tsx — zod at the boundary)
- Touching the WS chat client (web/src/ws/chat.tsx — types mirrored in internal/api/chat.go)
- Frontend tests (Vitest + the web/src/test doubles)
- Not backend work — use the `go` skill
- Not note-taking behavior — that's the wiki rulebook (~/.thoth/wiki/CLAUDE.md)

## Key files
- web/src/components/ — one component per file, co-located .test.tsx; App.tsx/main.tsx compose them
- web/src/hooks/ — useChat, useSearch, useConversationRoute, useView, useViewShortcuts
- web/src/store/ — Redux Toolkit: index.tsx (makeStore), hooks.tsx (typed hooks), slices/ (one per feature)
- web/src/api/client.tsx — typed REST client (axios + zod)
- web/src/ws/chat.tsx — ChatSocket: protocol frames, reconnect/resume
- web/src/test/ — mockAxios, fakeWS, renderWithStore, setup
- web/src/theme.tsx — the single antd ThemeConfig (blue primary, light-only)
- web/src/index.css — Tailwind v4 @theme tokens bridging antd's CSS variables
- references/patterns.md — the cross-cutting conventions (file structure, antd, tokens, state placement, routing, test doubles)

## The antd MCP (check it before writing UI)

The antd MCP server (`mcp__antd__*` tools) is the first stop for any
antd question — component APIs, tokens, semantic DOM, demos:
`antd_info` (props/types/defaults per component), `antd_doc` (full
markdown docs), `antd_token` (design tokens), `antd_semantic`
(classNames/styles structure), `antd_demo` (demo source), `antd_list`.
Fallback: https://ant.design/components/overview/.

Hard-won v6 facts worth re-checking in the MCP: DirectoryTree defaults
`showIcon: true` (disable it when a custom switcherIcon replaces the
caret — otherwise double icons), `destroyOnHidden` unmounts Popover
content, Badge has a `title` prop (6.5+), Select/AutoComplete/Tree are
virtualized by default (`virtual={false}` for small local lists),
rc-motion never completes under jsdom (assert store state for closes;
`motion={false}` on Tree), antd components reset their own margins
(antd `Flex` sets `margin: 0` — vertical rhythm around them must come
from the container's own `gap`, never `space-y-*`), css-var mode scopes
`--ant-*` under the ConfigProvider wrapper class (bridge Tailwind
tokens with `@theme inline`, never `:root`), and antd 6.6 deprecates
`List` (removed next major — prefer Flex-based rows for new code).

## Workflows

### 1. Add a component
1. Check the antd MCP for the component you need; prefer antd components over custom markup.
2. One component per file in web/src/components/<Name>.tsx; icons from @ant-design/icons (aria-hidden on decorative icons). File & naming rules: references/patterns.md § File structure — one component per file, `.tsx` only, PascalCase files, named exports, explicit props, no `any`.
3. Style with semantic tokens (bg-surface, text-ink, border-line) — no raw hex (references/patterns.md).
4. Co-locate the test <Name>.test.tsx using the renderWithStore/mockAxios doubles (renderWithStore wraps antd App, so App.useApp().message works).
5. Toasts use App.useApp().message — never a custom toast system.
6. Update docs/frontend.md's component table in the same commit.

### 2. Add a Redux slice
1. Create web/src/store/slices/<name>Slice.tsx — actions, selectors, thunks co-located.
2. Wire it in web/src/store/index.tsx (makeStore).
3. Use the typed hooks (useAppDispatch/useAppSelector from store/hooks.tsx) — never bare useDispatch/useSelector.
4. Only shared or screen-spanning state lives in the store; component-local state stays in hooks/components (docs/frontend.md §State).
5. Co-locate the slice test.

### 3. Add a hook
1. New file web/src/hooks/useX.tsx, exported as a named function.
2. Every useEffect subscription/timer/socket gets a cleanup that runs on unmount (code-rules skill memory rule).
3. Co-locate the test.

### 4. Wire an API call
1. Add or extend the endpoint in web/src/api/client.tsx with a zod schema — validation at the boundary, zero any.
2. Server side must match: use the `go` skill for internal/api; DTOs on both sides.
3. Test with mockAxios — assert the parsed payload, not the transport.
4. Update docs/api.md in the same commit.

### 5. Test a component/slice
1. Use the doubles in web/src/test/ (mockAxios, fakeWS, renderWithStore, setup) — never hand-rolled mocks of the app itself.
2. Assert real outcomes: what renders, what's dispatched, what the user sees.
3. Run: pnpm test (Vitest) — pnpm only, never npm.
4. Every behavior change ships with a test (code-rules skill).

### 6. Touch the WS client
1. CHANGE BOTH SIDES: web/src/ws/chat.tsx (client types) AND internal/api/chat.go (server frames) — they must match (code-rules skill § Invariants).
2. Frames: send/cancel/resume/open/new_chat out; assistant_*/tool_activity/turn_done/error in (docs/api.md).
3. Reconnect behavior: exactly once after 1 s, resume from onopen — changing it changes chat recovery semantics.
4. Test with fakeWS; update docs/api.md in the same commit.

### 7. Bump a frontend dependency
1. `pnpm add <pkg>@latest` from the repo root (workspace proxies) — pnpm only, never npm; never hand-edit versions in web/package.json.
2. The root pnpm-lock.yaml is committed — CI verifies the bump.
3. Run `pnpm typecheck && pnpm lint && pnpm test`, then `make web` to re-sync the embed.

## Gotchas
- pnpm only — never npm; the workspace lockfile (root pnpm-lock.yaml) is committed.
- TS strict, zero any — eslint enforces; zod at the API boundary.
- Every file under web/src is `.tsx` — no `.ts` source files.
- make web is REQUIRED before go build/test — frontend changes don't reach the binary without it.
- Light theme only — no dark mode; colors flow from the antd tokens in web/src/theme.tsx.
- Every useEffect has cleanup; no setInterval without clearInterval.

## Canonical docs
- docs/frontend.md — structure, components, hooks, state, design system
- docs/api.md — REST endpoints + WS protocol (both sides)
- references/patterns.md — the cross-cutting conventions

## Maintenance
Derived view — after a behavior change, update this skill + docs/ in the
same commit. Stale if a workflow's file paths stop resolving, docs/frontend.md
gains a component/hook/store entry this skill doesn't reference, or the
file-structure rules stop matching how pages and components are laid out.
