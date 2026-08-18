# CI Actions

> 26 nodes · cohesion 0.11

## Key Concepts

- **setup-go-web composite action** (7 connections) — `.github/actions/setup-go-web/action.yml`
- **final-gate job (ci)** (7 connections) — `.github/workflows/ci.yml`
- **setup-web composite action** (6 connections) — `.github/actions/setup-web/action.yml`
- **quality job (ci)** (6 connections) — `.github/workflows/ci.yml`
- **final-gate workflow** (4 connections) — `.github/workflows/final-gate.yml`
- **Pull request template** (3 connections) — `.github/pull_request_template.md`
- **build-darwin job** (3 connections) — `.github/workflows/ci.yml`
- **build-linux job** (3 connections) — `.github/workflows/ci.yml`
- **build-windows job** (3 connections) — `.github/workflows/ci.yml`
- **frontend-build job** (3 connections) — `.github/workflows/ci.yml`
- **final-gate job (ci-pr)** (3 connections) — `.github/workflows/ci-pr.yml`
- **Single required CI check pattern** (3 connections) — `.github/workflows/final-gate.yml`
- **backend-lint job** (3 connections) — `.github/workflows/quality.yml`
- **ci-pr workflow** (2 connections) — `.github/workflows/ci-pr.yml`
- **quality job (ci-pr)** (2 connections) — `.github/workflows/ci-pr.yml`
- **backend-test job** (2 connections) — `.github/workflows/quality.yml`
- **80% coverage floor** (2 connections) — `.github/workflows/quality.yml`
- **quality workflow (shared gates)** (2 connections) — `.github/workflows/quality.yml`
- **golangci-lint v2 config** (1 connections) — `.golangci.yml`
- **Frontend embed build (make web)** (1 connections) — `.github/actions/setup-go-web/action.yml`
- **Frozen-lockfile install** (1 connections) — `.github/actions/setup-web/action.yml`
- **ci workflow (main push)** (1 connections) — `.github/workflows/ci.yml`
- **frontend-lint job** (1 connections) — `.github/workflows/quality.yml`
- **frontend-test job** (1 connections) — `.github/workflows/quality.yml`
- **frontend-typecheck job** (1 connections) — `.github/workflows/quality.yml`
- *... and 1 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `.github/actions/setup-go-web/action.yml`
- `.github/actions/setup-web/action.yml`
- `.github/pull_request_template.md`
- `.github/workflows/ci-pr.yml`
- `.github/workflows/ci.yml`
- `.github/workflows/final-gate.yml`
- `.github/workflows/quality.yml`
- `.golangci.yml`

## Audit Trail

- EXTRACTED: 34 (94%)
- INFERRED: 2 (6%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*