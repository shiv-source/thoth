# React frontend checklist (web/src)

Walk shared.md first; this file adds the frontend-specific items.
Each item cites its rule source in the code-rules skill.

- [ ] TS `strict`, zero `any` (eslint enforces) (code-rules skill § Invariants)
- [ ] zod validates every API boundary response (code-rules skill § Invariants)
- [ ] Every useEffect subscription/timer/socket has cleanup; no setInterval without clearInterval (code-rules skill § Memory)
- [ ] Components are reusable and composable — small, props-driven; shared pieces extracted to hooks/components, not duplicated (code-rules skill § Code Rules: Modular & composable)
- [ ] Composition over inheritance — no class hierarchies; shared logic lives in custom hooks (code-rules skill § Code Rules: Patterns over novelty)
- [ ] WS message types match `internal/api/chat.go` when the chat client changes (code-rules skill § Invariants)

Canonical: code-rules skill § Invariants · § Memory · § Code Rules

Stale if: code-rules skill's TS/frontend rules change without this file following.
