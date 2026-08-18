# GitHub API Client

> 11 nodes · cohesion 0.29

## Key Concepts

- **Client** (8 connections) — `internal/github/client.go`
- **github/client.go** (6 connections) — `internal/github/client.go`
- **.FetchProfile()** (5 connections) — `internal/github/client.go`
- **.get()** (5 connections) — `internal/github/client.go`
- **.FetchRepos()** (4 connections) — `internal/github/client.go`
- **getResult** (2 connections) — `internal/github/client.go`
- **Profile** (2 connections) — `internal/github/client.go`
- **Repository** (2 connections) — `internal/github/client.go`
- **net/http.Client** (2 connections)
- **primaryEmail()** (2 connections) — `internal/github/client.go`
- **.WithBaseURL()** (1 connections) — `internal/github/client.go`

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (4 shared connections)
- [Claude Test Fakes](Claude_Test_Fakes.md) (3 shared connections)

## Source Files

- `internal/github/client.go`

## Audit Trail

- EXTRACTED: 23 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*