# WS Chat Protocol

> 8 nodes · cohesion 0.39

## Key Concepts

- **API - REST endpoints and WebSocket chat protocol** (12 connections) — `docs/api.md`
- **WebSocket chat protocol (/ws)** (7 connections) — `docs/api.md`
- **conversations table (claude_session_id)** (5 connections) — `docs/schema.md`
- **internal/store - conversations and messages** (4 connections) — `docs/components.md`
- **Per-conversation Claude CLI session pool** (3 connections) — `docs/api.md`
- **Supersede-on-send and cancel chat semantics** (3 connections) — `docs/api.md`
- **messages table (chat transcript)** (3 connections) — `docs/schema.md`
- **Resume with 500-message replay ring** (2 connections) — `docs/api.md`

## Relationships

- [Package Docs](Package_Docs.md) (7 shared connections)
- [Repo Governance Rules](Repo_Governance_Rules.md) (5 shared connections)
- [Toolchain Citations](Toolchain_Citations.md) (2 shared connections)
- [Knowledge Layer Docs](Knowledge_Layer_Docs.md) (1 shared connections)
- [Architecture Invariants](Architecture_Invariants.md) (1 shared connections)
- [Contributing Guide](Contributing_Guide.md) (1 shared connections)

## Source Files

- `docs/api.md`
- `docs/components.md`
- `docs/schema.md`

## Audit Trail

- EXTRACTED: 14 (50%)
- INFERRED: 14 (50%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*