# React frontend checklist (web/src)

Walk shared.md first; this file adds the frontend-specific items.
Each item cites its rule source in CLAUDE.md.

- [ ] TS `strict`, zero `any` (eslint enforces) (CLAUDE.md § Invariants)
- [ ] zod validates every API boundary response (CLAUDE.md § Invariants)
- [ ] Every useEffect subscription/timer/socket has cleanup; no setInterval without clearInterval (CLAUDE.md § Memory)
- [ ] Components are reusable and composable — small, props-driven; shared pieces extracted to hooks/components, not duplicated (CLAUDE.md § Code Rules: Modular & composable)
- [ ] Composition over inheritance — no class hierarchies; shared logic lives in custom hooks (CLAUDE.md § Code Rules: Patterns over novelty)
- [ ] WS message types match `internal/api/chat.go` when the chat client changes (CLAUDE.md § Invariants)

Canonical: CLAUDE.md § Invariants · § Memory · § Code Rules

Stale if: CLAUDE.md's TS/frontend rules change without this file following.
