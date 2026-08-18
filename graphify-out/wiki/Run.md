# Run()

> God node · 31 connections · `internal/doctor/doctor.go`

**Community:** [Doctor Diagnostics](Doctor_Diagnostics.md)

## Connections by Relation

### calls
- .runChecks() `EXTRACTED`
- ExpandHome() `EXTRACTED`
- TestRunAPIHealthy() `INFERRED`
- TestRunAPIPartialFailures() `INFERRED`
- TestRunClaudeLoginUnknown() `INFERRED`
- TestRunClaudeVersionFailure() `INFERRED`
- checkWiki() `EXTRACTED`
- TestRunAPIWebsocketFails() `INFERRED`
- TestRunDatabaseMissingTables() `INFERRED`
- TestRunHealthy() `INFERRED`
- TestRunIndexOutOfSync() `INFERRED`
- TestRunNonThothProcessOnPort() `INFERRED`
- TestRunNonWALDatabase() `INFERRED`
- TestRunWikiMissingFolders() `INFERRED`
- TestRunWikiMissingNoteIsSkipped() `INFERRED`
- doctorHandler() `EXTRACTED`
- checkClaude() `EXTRACTED`
- checkIndex() `EXTRACTED`
- TestRunCorruptDatabase() `INFERRED`
- TestRunDefaultWikiPathWhenDatabaseMissing() `INFERRED`

### contains
- doctor/doctor.go `EXTRACTED`

### references
- context.Context `EXTRACTED`
- log/slog.Logger `EXTRACTED`
- Check `EXTRACTED`

---

*Part of the graphify knowledge wiki. See [index](index.md) to navigate.*