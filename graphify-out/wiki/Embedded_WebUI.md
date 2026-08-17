# Embedded WebUI

> 10 nodes · cohesion 0.29

## Key Concepts

- **newTestEcho()** (7 connections) — `internal/webui/webui_test.go`
- **Register()** (5 connections) — `internal/webui/embed.go`
- **webui_test.go** (5 connections) — `internal/webui/webui_test.go`
- **TestRegisterFallsBackToIndexForMissingPaths()** (3 connections) — `internal/webui/webui_test.go`
- **TestRegisterReturns404ForUnknownAPIPaths()** (3 connections) — `internal/webui/webui_test.go`
- **TestRegisterServesExistingAsset()** (3 connections) — `internal/webui/webui_test.go`
- **TestRegisterServesIndexAtRoot()** (3 connections) — `internal/webui/webui_test.go`
- **embed.go** (1 connections) — `internal/webui/embed.go`
- **echo.Echo** (1 connections)
- **echo.Echo** (1 connections)

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (4 shared connections)
- [API Server & DTOs](API_Server_&_DTOs.md) (1 shared connections)
- [Serve Command](Serve_Command.md) (1 shared connections)

## Source Files

- `internal/webui/embed.go`
- `internal/webui/webui_test.go`

## Audit Trail

- EXTRACTED: 18 (95%)
- INFERRED: 1 (5%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*