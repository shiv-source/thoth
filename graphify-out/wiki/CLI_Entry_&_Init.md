# CLI Entry & Init

> 23 nodes · cohesion 0.13

## Key Concepts

- **newRootCmd()** (16 connections) — `internal/cli/root.go`
- **github.com/spf13/cobra.Command** (6 connections)
- **newInitCmd()** (5 connections) — `internal/cli/init.go`
- **TestServeErrorWhenWikiScaffoldFails()** (5 connections) — `internal/cli/serve_test.go`
- **version_test.go** (5 connections) — `internal/cli/version_test.go`
- **init_test.go** (4 connections) — `internal/cli/init_test.go`
- **root.go** (4 connections) — `internal/cli/root.go`
- **Execute()** (4 connections) — `internal/cli/root.go`
- **captureStdout()** (4 connections) — `internal/cli/version_test.go`
- **TestVersionCommand()** (4 connections) — `internal/cli/version_test.go`
- **TestInitCommandErrorOnUnwritableTarget()** (3 connections) — `internal/cli/init_test.go`
- **TestInitCommandExpandsTildeInTarget()** (3 connections) — `internal/cli/init_test.go`
- **TestInitCommandTooManyArgs()** (3 connections) — `internal/cli/init_test.go`
- **TestInitCommandUsesDefaultPath()** (3 connections) — `internal/cli/init_test.go`
- **newVersionCmd()** (3 connections) — `internal/cli/root.go`
- **newServeCmd()** (3 connections) — `internal/cli/serve.go`
- **TestExecuteReturnsErrorForUnknownCommand()** (3 connections) — `internal/cli/version_test.go`
- **TestRootHasExpectedSubcommands()** (3 connections) — `internal/cli/version_test.go`
- **TestRootRejectsUnknownCommand()** (3 connections) — `internal/cli/version_test.go`
- **main()** (2 connections) — `cmd/thoth/main.go`
- **Version()** (2 connections) — `internal/cli/root.go`
- **main.go** (1 connections) — `cmd/thoth/main.go`
- **init.go** (1 connections) — `internal/cli/init.go`

## Relationships

- [GitHub Identity & Stdlib](GitHub_Identity_&_Stdlib.md) (10 shared connections)
- [Serve Command](Serve_Command.md) (4 shared connections)
- [Doctor Tests](Doctor_Tests.md) (3 shared connections)
- [Doctor Diagnostics](Doctor_Diagnostics.md) (2 shared connections)
- [Config Expansion](Config_Expansion.md) (1 shared connections)
- [Wiki Scaffolding](Wiki_Scaffolding.md) (1 shared connections)
- [GitHub Repo Storage](GitHub_Repo_Storage.md) (1 shared connections)

## Source Files

- `cmd/thoth/main.go`
- `internal/cli/init.go`
- `internal/cli/init_test.go`
- `internal/cli/root.go`
- `internal/cli/serve.go`
- `internal/cli/serve_test.go`
- `internal/cli/version_test.go`

## Audit Trail

- EXTRACTED: 41 (73%)
- INFERRED: 15 (27%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*