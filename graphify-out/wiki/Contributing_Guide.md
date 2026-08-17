# Contributing Guide

> 13 nodes · cohesion 0.17

## Key Concepts

- **Development - toolchain, gates, CI** (11 connections) — `docs/development.md`
- **CONTRIBUTING.md - contribution workflow** (5 connections) — `CONTRIBUTING.md`
- **SQL migrations gated on PRAGMA user_version** (3 connections) — `docs/schema.md`
- **CI-enforced quality gates (make check)** (2 connections) — `CONTRIBUTING.md`
- **Additive migrations rule (never edit an applied migration)** (2 connections) — `CONTRIBUTING.md`
- **PR and review workflow (conventional commits, squash-merge)** (2 connections) — `CONTRIBUTING.md`
- **CI workflows (quality.yml, ci.yml, ci-pr.yml, final-gate.yml)** (2 connections) — `docs/development.md`
- **Five quality gates (make check)** (2 connections) — `docs/development.md`
- **Gate: 80 percent coverage floor on internal and cmd** (1 connections) — `docs/development.md`
- **Gate: five cross-compile targets** (1 connections) — `docs/development.md`
- **Gate: frontend typecheck, lint, test, build** (1 connections) — `docs/development.md`
- **Gate: go test -race** (1 connections) — `docs/development.md`
- **Gate: gofmt and go vet clean** (1 connections) — `docs/development.md`

## Relationships

- [Repo Governance Rules](Repo_Governance_Rules.md) (3 shared connections)
- [Package Docs](Package_Docs.md) (3 shared connections)
- [Toolchain Citations](Toolchain_Citations.md) (1 shared connections)
- [WS Chat Protocol](WS_Chat_Protocol.md) (1 shared connections)

## Source Files

- `CONTRIBUTING.md`
- `docs/development.md`
- `docs/schema.md`

## Audit Trail

- EXTRACTED: 15 (71%)
- INFERRED: 6 (29%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*