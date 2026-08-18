# Repo Governance Rules

> 19 nodes · cohesion 0.16

## Key Concepts

- **CLAUDE.md - Thoth repository rulebook** (13 connections) — `CLAUDE.md`
- **Security - threat model and mechanisms** (10 connections) — `docs/security.md`
- **internal/claude - the blast wall package** (8 connections) — `docs/components.md`
- **PersistentClient - long-lived CLI process pool** (6 connections) — `docs/components.md`
- **Claude Code CLI - driven headless per conversation** (5 connections) — `docs/architecture.md`
- **README.md - Thoth overview** (5 connections) — `README.md`
- **Threat model - local-only, single-user, enforced in code** (4 connections) — `docs/security.md`
- **Deliberate security trade-offs (no auth, skip-permissions, plaintext PAT)** (4 connections) — `docs/security.md`
- **Blast wall - all Claude CLI flags live only in client.go** (3 connections) — `CLAUDE.md`
- **allowLocalOrigin - WebSocket origin check** (3 connections) — `docs/security.md`
- **Thoth - local-first personal knowledge base** (3 connections) — `README.md`
- **Memory and resource safety rules (no leaks)** (2 connections) — `CLAUDE.md`
- **Runtime data: ~/.thoth (thoth.db + wiki/)** (2 connections) — `CLAUDE.md`
- **Two interfaces, one contract (dashboard and terminal)** (2 connections) — `docs/architecture.md`
- **SafePath - path traversal guard** (2 connections) — `docs/components.md`
- **No secrets in the wiki - placeholders only** (2 connections) — `internal/wiki/templates/CLAUDE.md`
- **Branch workflow - never commit to main directly** (1 connections) — `CLAUDE.md`
- **Code rules: DRY, SOLID, KISS, YAGNI, small functions** (1 connections) — `CLAUDE.md`
- **FakeClient - scripted-event test double** (1 connections) — `docs/components.md`

## Relationships

- [Knowledge Layer Docs](Knowledge_Layer_Docs.md) (7 shared connections)
- [Package Docs](Package_Docs.md) (6 shared connections)
- [WS Chat Protocol](WS_Chat_Protocol.md) (5 shared connections)
- [Contributing Guide](Contributing_Guide.md) (3 shared connections)
- [Architecture Invariants](Architecture_Invariants.md) (2 shared connections)

## Source Files

- `CLAUDE.md`
- `README.md`
- `docs/architecture.md`
- `docs/components.md`
- `docs/security.md`
- `internal/wiki/templates/CLAUDE.md`

## Audit Trail

- EXTRACTED: 29 (58%)
- INFERRED: 21 (42%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*