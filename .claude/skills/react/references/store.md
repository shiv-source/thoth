# Redux store (web/src/store)

The slice index lives in **docs/frontend.md § State** — the authoritative
table of every slice under `web/src/store/slices/` (purpose, actions,
selectors, thunks). Read it there, not here.

Rules that still live here (the react skill workflow 2 points at them):
- Redux Toolkit owns server-backed, shared, and screen-spanning state;
  component-local state stays in hooks/components (patterns.md § State placement).
- `makeStore()` wires the slices (`index.tsx`); always use
  `useAppDispatch`/`useAppSelector` from `store/hooks.tsx`, never the bare versions.

Stale if: a slice appears or disappears without a docs/frontend.md § State update.
