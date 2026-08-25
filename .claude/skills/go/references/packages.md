# Go packages (internal/* + cmd/thoth) — key exports

Purpose, prose, and canonical files for each package: **docs/components.md**.
This file keeps only the exported-symbol lists agents grep for. Every package
in internal/ plus cmd/thoth is listed — a missing package means this index is stale.

| Package | Key exports |
|---|---|
| cmd/thoth | main (calls internal/cli Execute) |
| internal/cli | Execute; serve helpers: ensureWiki, resolveClaudeBin, onSettingsSaved, serveUntilShutdown |
| internal/claude (blast wall) | Client (Start(ctx, sessionID, prompt, w EventWriter) error), CLIClient, PersistentClient, Event, ParseLine, FakeClient |
| internal/wiki | Scaffold, ParseNote, SafePath, Wiki (New/Exists/Read/Tree), Rulebook |
| internal/index | Index, Sync, Watch, Search — Open, Upsert, Delete, DeletePrefix |
| internal/assets | ModelOptions |
| internal/store | Store (Open, ListConversations, Messages, Close), EnsureMetadata |
| internal/api | Deps, New(d Deps) *echo.Echo, Hub |
| internal/config | DefaultHost, DefaultPort (127.0.0.1:8333), ExpandHome |
| internal/doctor | Run(ctx, Options) []Check; Check{Name, OK, Message} |
| internal/github | Client, Auth, Repo, Service |
| internal/settings | Repo, OpenRepo(path); SyncEnabled/SyncState/SetSyncResult; ProviderConfig(provider), ProviderAPIKeyKey/ProviderBaseURLKey |
| internal/webui | Register |

Stale if: a new package appears in internal/, or an export above is renamed.
