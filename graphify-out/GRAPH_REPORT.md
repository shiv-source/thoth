# Graph Report - thoth  (2026-08-21)

## Corpus Check
- 307 files · ~160,910 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2412 nodes · 5902 edges · 138 communities (109 shown, 29 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 768 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `b62ee815`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- New
- NewToolUseBlock
- putSettingsReq
- Troubleshooting & FAQ
- NewPersistent
- doctor/doctor_test.go
- serve_test.go
- NewTextBlock
- dependencies
- index.ts
- devDependencies
- package.json
- openai.go
- Shared checklist — both layers (yes/no; any "no" gets fixed before the PR)
- setup-go-web composite action
- dependencies
- Hub
- compilerOptions
- sse_test.go
- Components (web/src/components)
- client.ts
- documentation.md
- PersistentClient
- Client
- Scaffold
- Toolchain versions (go.mod / package.json authoritative)
- compilerOptions
- CLAUDE.md - Thoth repository rulebook
- DashboardView.tsx
- Workflows
- go.md
- Development - toolchain, gates, CI
- Components - Go package deep dive
- lib.mjs
- Open
- Indexing and search - FTS5 and the file watcher
- New
- devDependencies
- Open
- Go packages (internal/* + cmd/thoth)
- SearchPanel.tsx
- scripts
- development
- API - REST endpoints and WebSocket chat protocol
- Architecture - two layers, one binary
- App.tsx
- pr.sh
- Git workflow — contribution procedures & expectations
- ResizeObserverStub
- antd
- web workspace package
- React frontend (web/src) — procedures & expertise
- tsconfig.json
- pre-commit
- fake_claude.sh
- report job (summary + gate)
- github.com/shiv-source/thoth
- openTestRepo
- Sidebar.test.tsx
- Redux store (web/src/store)
- AppHeader.tsx
- web/package.json
- react.md
- Agent
- openai_test.go
- Quality gates — how this repo verifies work
- Code quality — the pre-PR gate
- Registry
- react
- Frontend patterns — the cross-cutting conventions
- renderWithStore
- docs-site/package.json
- The claude blast wall (internal/claude)
- Persistence — thoth.db, migrations, index
- Hooks (web/src/hooks)
- Labels — the three-tier GitHub label set
- token-guard.sh
- testing.T
- NoteViewer.tsx
- Index
- cli/doctor_test.go
- watcher_test.go
- file.go
- remark-gfm
- docusaurus.config.ts
- typescript
- history_test.go
- log/slog.Logger
- SettingsView.test.tsx
- README.md
- setup.sh
- Deps
- ParseNote
- github.com/shiv-source/thoth/agent.Request
- CLIClient
- react-dom
- graph-check.sh
- main-guard.sh
- Validate
- wiki.go
- NewSearch
- context.Context
- WikiTree.tsx
- file_test.go
- sidebars.ts
- os/exec.Cmd
- plugins
- react-markdown
- @tailwindcss/typography
- Watch
- openModels
- eventRecorder
- NewRegistry
- New
- @types/react-dom
- ModelOptions
- Repo
- NewHub
- watcher.go
- registry
- github.com/shiv-source/thoth/agent.Delta
- fakeTool
- FakeClient
- jsdom
- newLoggingServer
- TestGitSetupReportsSanitizedFailure
- OSFS
- testDeps
- ScaffoldWithOptions
- anthropic.go
- SafePath
- vite
- @easyops-cn/docusaurus-search-local
- @types/node

## God Nodes (most connected - your core abstractions)
1. `testDeps()` - 97 edges
2. `New()` - 85 edges
3. `Deps` - 39 edges
4. `Open()` - 39 edges
5. `NewPersistent()` - 38 edges
6. `Components (web/src/components)` - 38 edges
7. `WriterFunc` - 36 edges
8. `Open()` - 36 edges
9. `useAppSelector` - 35 edges
10. `react` - 34 edges

## Surprising Connections (you probably didn't know these)
- `Runtime data: ~/.thoth (thoth.db + wiki/)` --conceptually_related_to--> `Database schema - thoth.db tables`  [INFERRED]
  CLAUDE.md → docs/schema.md
- `CI-enforced quality gates (make check)` --semantically_similar_to--> `Five quality gates (make check)`  [INFERRED] [semantically similar]
  CONTRIBUTING.md → docs/development.md
- `Additive migrations rule (never edit an applied migration)` --semantically_similar_to--> `SQL migrations gated on PRAGMA user_version`  [INFERRED] [semantically similar]
  CONTRIBUTING.md → docs/schema.md
- `Project invariants (files as source of truth, percent-w errors, no globals)` --semantically_similar_to--> `Data contract: files are the source of truth, thoth.db is derived`  [INFERRED] [semantically similar]
  CLAUDE.md → docs/architecture.md
- `TestWriterFuncAdapter()` --calls--> `WriterFunc`  [INFERRED]
  internal/claude/events_test.go → agent/events/events.go

## Import Cycles
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/noteSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/doctorSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/searchHistorySlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/wikiSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/searchSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/notificationsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/connectionSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/gitSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/chatSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/conversationsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/healthSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/settingsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/uiSlice.ts -> web/src/store/index.ts`

## Hyperedges (group relationships)
- **The four app-layer components of the single binary** — docs_architecture_app_layer, docs_components_api_pkg, docs_components_claude_pkg, docs_components_index_pkg [EXTRACTED 1.00]
- **Cross-compile build matrix (linux/darwin/windows)** — github_workflows_ci_build_linux, github_workflows_ci_build_darwin, github_workflows_ci_build_windows [EXTRACTED 1.00]
- **Single required CI gate pattern** — github_workflows_ci_pr_final_gate, github_workflows_ci_final_gate, github_workflows_final_gate_report, github_workflows_final_gate_single_check [EXTRACTED 1.00]
- **The five CI quality gates every commit must pass** — docs_development_quality_gates, docs_development_gate_vet, docs_development_gate_race, docs_development_gate_coverage, docs_development_gate_crosscompile, docs_development_gate_frontend [EXTRACTED 1.00]
- **Shared quality gates (backend + frontend)** — github_workflows_quality_backend_test, github_workflows_quality_backend_lint, github_workflows_quality_frontend_test, github_workflows_quality_frontend_lint, github_workflows_quality_frontend_typecheck [EXTRACTED 1.00]
- **The rulebook-driven wiki filing system** — internal_wiki_templates_claude_claude, internal_wiki_templates_claude_save_protocol, docs_knowledge_base_wiki_layout, docs_knowledge_base_frontmatter, docs_architecture_knowledge_layer [INFERRED 0.85]

## Communities (138 total, 29 thin omitted)

### Community 0 - "New"
Cohesion: 0.22
Nodes (29): github.com/gorilla/websocket.Conn, TestWikiChangedFrameReachesSocket(), readMsg(), TestChatCancelBeforeSendIsNoop(), TestChatCancelStopsInFlightTurn(), TestChatForwardsThinkingFrames(), TestChatHubCancellationEndsTurns(), TestChatNewChatCancelsBusyTurn() (+21 more)

### Community 1 - "NewToolUseBlock"
Cohesion: 0.37
Nodes (15): NewToolUseBlock(), New(), readFixture(), streamTurn(), TestStreamAccumulatesToolUseMessage(), TestStreamBuildsRequest(), TestStreamCancel(), TestStreamDefaultsMaxTokens() (+7 more)

### Community 2 - "putSettingsReq"
Cohesion: 0.17
Nodes (20): fixtureHandler, net/http.Request, net/http.ResponseWriter, allowLocalOrigin(), getSettingsReq(), putSettingsReq(), TestConversationsEndpoints(), TestDeleteConversationEndpoint() (+12 more)

### Community 3 - "Troubleshooting & FAQ"
Cohesion: 0.06
Nodes (33): First run, Getting started, Install Thoth, Next steps, What you need, Your first conversation, A note I edited by hand isn't showing up, "claude" is not found (+25 more)

### Community 4 - "NewPersistent"
Cohesion: 0.07
Nodes (77): Option, StartOption, Event, EventType, WriterFunc, Client, New(), TestDirProviderOverridesStaticDir() (+69 more)

### Community 5 - "doctor/doctor_test.go"
Cohesion: 0.05
Nodes (78): doctorRunner, main(), github.com/spf13/cobra.Command, net.Listener, failed(), fileExists(), newDoctorCmd(), resolveThothDir() (+70 more)

### Community 6 - "serve_test.go"
Cohesion: 0.35
Nodes (15): prewarmPool(), cliSpawnCount(), openTestRepos(), TestEnsureModelsKeepsUserRows(), TestEnsureWikiErrorWhenAttachmentsBlocked(), TestEnsureWikiRecreatesReservedAttachments(), TestPrewarmPoolEmptyDBLeavesPoolEmpty(), TestPrewarmPoolExistingConversation() (+7 more)

### Community 7 - "NewTextBlock"
Cohesion: 0.06
Nodes (64): Accumulate(), Block, Message, NewBuilder(), TestBuilderAccumulatesText(), TestBuilderAccumulatesThinking(), TestBuilderAccumulatesToolInput(), TestBuilderIgnoresStop() (+56 more)

### Community 8 - "dependencies"
Cohesion: 0.07
Nodes (27): @ant-design/icons, antd, axios, chart.js, react-chartjs-2, react-redux, @reduxjs/toolkit, shiki (+19 more)

### Community 9 - "index.ts"
Cohesion: 0.07
Nodes (57): DoctorCheck, GitHubIdentity, GitHubRepo, LLMModel, ModelGroup, ModelInput, Settings, SettingsView (+49 more)

### Community 10 - "devDependencies"
Cohesion: 0.08
Nodes (25): eslint, eslint-config-prettier, @eslint/js, globals, oxlint, @testing-library/jest-dom, @testing-library/react, @testing-library/user-event (+17 more)

### Community 11 - "package.json"
Cohesion: 0.05
Nodes (37): husky, lint-staged, author, bugs, url, description, devDependencies, husky (+29 more)

### Community 12 - "openai.go"
Cohesion: 0.17
Nodes (19): buildRequest(), wireMessage, wireTool, toolArguments(), wireMessages(), wireTurnMessage(), WithHTTPClient(), WithMaxTokens() (+11 more)

### Community 13 - "Shared checklist — both layers (yes/no; any "no" gets fixed before the PR)"
Cohesion: 0.33
Nodes (5): Correctness, Naming & style, Shared checklist — both layers (yes/no; any "no" gets fixed before the PR), Structure, Tests

### Community 14 - "setup-go-web composite action"
Cohesion: 0.11
Nodes (26): golangci-lint v2 config, Frontend embed build (make web), setup-go-web composite action, Frozen-lockfile install, setup-web composite action, Pull request template, build-darwin job, build-linux job (+18 more)

### Community 15 - "dependencies"
Cohesion: 0.10
Nodes (21): clsx, dependencies, clsx, @docusaurus/core, @docusaurus/faster, @docusaurus/preset-classic, @docusaurus/theme-mermaid, @fontsource-variable/fraunces (+13 more)

### Community 16 - "Hub"
Cohesion: 0.08
Nodes (18): clientEntry, clientMsg, Hub, serverMsg, turn, fakePrewarmStore, context.CancelFunc, github.com/shiv-source/thoth/internal/claude.WriterFunc (+10 more)

### Community 17 - "compilerOptions"
Cohesion: 0.08
Nodes (24): DOM, src, vite/client, compilerOptions, allowArbitraryExtensions, allowImportingTsExtensions, jsx, lib (+16 more)

### Community 18 - "sse_test.go"
Cohesion: 0.16
Nodes (16): NewSSEReader(), readAllFrames(), TestFrameDecode(), TestSSEReaderBlankLinesIgnored(), TestSSEReaderChunkBoundaries(), TestSSEReaderCommentsIgnored(), TestSSEReaderDONETerminator(), TestSSEReaderEOFMidFrame() (+8 more)

### Community 19 - "Components (web/src/components)"
Cohesion: 0.05
Nodes (38): ActivityChart, App shell & navigation, AppHeader, AppSider, Charts (Chart.js), chartSetup.ts, Chat, ChatActivityChart (+30 more)

### Community 20 - "client.ts"
Cohesion: 0.09
Nodes (20): Conversation, Health, http, Message, Note, TreeNodeSchema, TreeNodeShape, SetupScreen (+12 more)

### Community 22 - "PersistentClient"
Cohesion: 0.20
Nodes (9): poolEntry, proc, CLIClient, bufio.Writer, os.File, time.Duration, time.Timer, openDebugDump() (+1 more)

### Community 23 - "Client"
Cohesion: 0.33
Nodes (5): getResult, Profile, Repository, Client, primaryEmail()

### Community 24 - "Scaffold"
Cohesion: 0.14
Nodes (23): folderMap(), noteType(), NoteTypes(), NoteTypesFor(), EnsureReservedDir(), Scaffold(), TestEnsureReservedDirCreatesWhenMissing(), TestEnsureReservedDirKeepsExistingContents() (+15 more)

### Community 25 - "Toolchain versions (go.mod / package.json authoritative)"
Cohesion: 0.13
Nodes (20): Chart.js, Cobra 1.10 - Go CLI framework, Echo 4.15 - Go web framework, fsnotify 1.10 - file watcher, gorilla/websocket 1.5, React 19.2, Redux Toolkit, SQLite + FTS5 (modernc.org/sqlite 1.56) (+12 more)

### Community 26 - "compilerOptions"
Cohesion: 0.10
Nodes (19): node, vite.config.ts, compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection (+11 more)

### Community 27 - "CLAUDE.md - Thoth repository rulebook"
Cohesion: 0.16
Nodes (19): Blast wall - all Claude CLI flags live only in client.go, Branch workflow - never commit to main directly, CLAUDE.md - Thoth repository rulebook, Memory and resource safety rules (no leaks), Code rules: DRY, SOLID, KISS, YAGNI, small functions, Runtime data: ~/.thoth (thoth.db + wiki/), Claude Code CLI - driven headless per conversation, Two interfaces, one contract (dashboard and terminal) (+11 more)

### Community 28 - "DashboardView.tsx"
Cohesion: 0.09
Nodes (22): Feature, features, DashboardView, ActivityChart(), ChatActivityChart(), mockActivity, mockChatActivity, mockInbox (+14 more)

### Community 29 - "Workflows"
Cohesion: 0.11
Nodes (18): 10. Diagnose/repair an install, 11. Cut a release, 1. Add a REST endpoint, 2. Extend the WS protocol, 3. Add a store migration, 4. Change claude CLI flags (BLAST WALL), 5. Add a settings key, 6. Extend the wiki contract (+10 more)

### Community 31 - "Development - toolchain, gates, CI"
Cohesion: 0.17
Nodes (13): CI-enforced quality gates (make check), CONTRIBUTING.md - contribution workflow, Additive migrations rule (never edit an applied migration), PR and review workflow (conventional commits, squash-merge), CI workflows (quality.yml, ci.yml, ci-pr.yml, final-gate.yml), Development - toolchain, gates, CI, Gate: 80 percent coverage floor on internal and cmd, Gate: five cross-compile targets (+5 more)

### Community 32 - "Components - Go package deep dive"
Cohesion: 0.28
Nodes (13): CLI - serve, init, version, doctor commands, thoth doctor - six install checks, thoth doctor --fix repair mode, Components - Go package deep dive, internal/doctor - shared install checks, internal/github - identity and git sync, internal/settings - settings KV repo, Documentation hub (index.md) (+5 more)

### Community 33 - "lib.mjs"
Cohesion: 0.24
Nodes (11): main(), writeStepOutput(), ALLOWED_KINDS, computeLabels(), loadConfig(), missingLabels(), normalizeLabel(), parseFields() (+3 more)

### Community 34 - "Open"
Cohesion: 0.13
Nodes (23): database/sql.DB, healthyThothDir(), TestDoctorEndpointHealthy(), applyMigration(), migrate(), splitStatements(), TestUpgradeRenamesDescriptionToTag(), OpenDB() (+15 more)

### Community 35 - "Indexing and search - FTS5 and the file watcher"
Cohesion: 0.25
Nodes (11): Project invariants (files as source of truth, percent-w errors, no globals), App layer - single Go binary, Data contract: files are the source of truth, thoth.db is derived, thoth serve command, internal/api - the Echo server, internal/index - search and sync, useSearch - debounced, supersede-guarded search, bm25 ranking with title weighted 8x (+3 more)

### Community 36 - "New"
Cohesion: 0.14
Nodes (26): profileStub, net/http.HandlerFunc, doJSON(), githubStub(), TestConnectGitHub(), TestConnectGitHubRejectedToken(), TestConnectGitHubRequiresToken(), TestConnectGitHubUpstreamError() (+18 more)

### Community 37 - "devDependencies"
Cohesion: 0.18
Nodes (11): devDependencies, @docusaurus/module-type-aliases, @docusaurus/tsconfig, @docusaurus/types, @types/react, typescript, @types/react, typescript (+3 more)

### Community 38 - "Open"
Cohesion: 0.27
Nodes (16): TestApply(), TestApplyClosedIndexLogsAndContinues(), TestApplyPathOutsideRoot(), TestApplyUnreadablePath(), TestWatchErrorOnMissingRoot(), TestWatchReturnsOnCancel(), Open(), discardLog() (+8 more)

### Community 39 - "Go packages (internal/* + cmd/thoth)"
Cohesion: 0.13
Nodes (14): cmd/thoth, Go packages (internal/* + cmd/thoth), internal/api, internal/assets, internal/claude — the blast wall, internal/cli, internal/config, internal/doctor (+6 more)

### Community 40 - "SearchPanel.tsx"
Cohesion: 0.10
Nodes (23): SearchResult, SearchView, SearchPanel(), mocks, mocks, renderSearchHook(), useSearch(), loadHistory() (+15 more)

### Community 41 - "scripts"
Cohesion: 0.18
Nodes (11): scripts, build, clear, deploy, docusaurus, serve, start, swizzle (+3 more)

### Community 42 - "development"
Cohesion: 0.22
Nodes (9): browserslist, development, production, >0.5%, last 3 chrome version, last 3 firefox version, last 5 safari version, not dead (+1 more)

### Community 43 - "API - REST endpoints and WebSocket chat protocol"
Cohesion: 0.39
Nodes (8): API - REST endpoints and WebSocket chat protocol, Resume with 500-message replay ring, Per-conversation Claude CLI session pool, Supersede-on-send and cancel chat semantics, WebSocket chat protocol (/ws), internal/store - conversations and messages, conversations table (claude_session_id), messages table (chat transcript)

### Community 44 - "Architecture - two layers, one binary"
Cohesion: 0.43
Nodes (8): Architecture - two layers, one binary, Knowledge layer - plain markdown wiki you own, internal/wiki - the file contract, Frontmatter contract (title required), Knowledge base - the wiki directory, Wiki folder layout (8 folders), Wiki rulebook template (CLAUDE.md in wiki root), The save protocol (folder map, frontmatter, confirm)

### Community 45 - "App.tsx"
Cohesion: 0.11
Nodes (35): App(), AppSider(), HealthFooter(), ITEMS, ChatPanel(), createSocket(), Composer(), DashboardView() (+27 more)

### Community 46 - "pr.sh"
Cohesion: 0.33
Nodes (14): check_worktree(), derive_labels(), derive_title(), die(), label_known(), load_label_sets(), main(), parse_branch() (+6 more)

### Community 47 - "Git workflow — contribution procedures & expectations"
Cohesion: 0.14
Nodes (13): 1. Start a change (branch), 2. Commit, 3. Open a PR, 4. Label issues and PRs, 5. Design doc first (large or cross-package changes), 6. Merge is human-only — squash by default, Canonical docs, Git workflow — contribution procedures & expectations (+5 more)

### Community 49 - "antd"
Cohesion: 0.40
Nodes (5): npx, antd, playwright, @ant-design/cli, @playwright/mcp

### Community 50 - "web workspace package"
Cohesion: 0.50
Nodes (4): web workspace package, pnpm workspace root, Thoth web entry (index.html), React app entry (src/main.tsx)

### Community 51 - "React frontend (web/src) — procedures & expertise"
Cohesion: 0.12
Nodes (15): 1. Add a component, 2. Add a Redux slice, 3. Add a hook, 4. Wire an API call, 5. Test a component/slice, 6. Touch the WS client, 7. Bump a frontend dependency, Canonical docs (+7 more)

### Community 61 - "openTestRepo"
Cohesion: 0.23
Nodes (10): Repo, Auth, OpenRepo(), Repo, openTestRepo(), saved(), TestRepoClear(), TestRepoClosedErrors() (+2 more)

### Community 62 - "Sidebar.test.tsx"
Cohesion: 0.20
Nodes (6): conversations, mocks, older, renderSidebar(), today, yesterday

### Community 63 - "Redux store (web/src/store)"
Cohesion: 0.11
Nodes (17): chat, connection, conversations, doctor, git, health, hooks.ts, index.ts (+9 more)

### Community 64 - "AppHeader.tsx"
Cohesion: 0.16
Nodes (14): AppHeader(), NotificationPanel(), NOTIFICATION_ICONS, NOTIFICATION_PALETTE, NotificationIcon(), NotificationToasts(), initialState, Notification (+6 more)

### Community 65 - "web/package.json"
Cohesion: 0.17
Nodes (11): name, private, scripts, build, dev, lint, preview, test (+3 more)

### Community 67 - "Agent"
Cohesion: 0.16
Nodes (27): Agent, Options, Block, Message, New(), NewBuilder(), NewTextBlock(), NewThinkingBlock() (+19 more)

### Community 68 - "openai_test.go"
Cohesion: 0.38
Nodes (15): Client, New(), readFixture(), streamTurn(), TestStreamAccumulatesMultiToolUseMessage(), TestStreamBuildsRequest(), TestStreamCancel(), TestStreamDefaultsMaxTokens() (+7 more)

### Community 69 - "Quality gates — how this repo verifies work"
Cohesion: 0.20
Nodes (9): Commit hygiene, Concurrency, Coverage, Cross-compile, Dependency bumps, Lint, make check — everything CI enforces, locally, Quality gates — how this repo verifies work (+1 more)

### Community 70 - "Code quality — the pre-PR gate"
Cohesion: 0.18
Nodes (10): 1. Run the quality gates, 2. Walk the review checklist, 3. Triage a failing gate, Canonical docs, Code quality — the pre-PR gate, Gotchas, Key files, Maintenance (+2 more)

### Community 72 - "react"
Cohesion: 0.08
Nodes (24): react, api, freshSocket(), toolLabel(), useChat(), chatIdFromPath(), ConversationRouteOptions, getConversation (+16 more)

### Community 73 - "Frontend patterns — the cross-cutting conventions"
Cohesion: 0.20
Nodes (9): Ant Design first, Design tokens, Frontend patterns — the cross-cutting conventions, Package discipline, Routing, State placement, Test doubles (web/src/test), The API boundary (zod) (+1 more)

### Community 74 - "renderWithStore"
Cohesion: 0.06
Nodes (35): mocks, mocks, renderPanel(), mocks, renderBlock(), conversations, mocks, renderDashboard() (+27 more)

### Community 75 - "docs-site/package.json"
Cohesion: 0.33
Nodes (5): engines, node, name, private, version

### Community 76 - "The claude blast wall (internal/claude)"
Cohesion: 0.29
Nodes (6): client.go — the flags, Client interface & FakeClient, events.go — stream parsing, persistent.go — the process pool, Process kill, The claude blast wall (internal/claude)

### Community 77 - "Persistence — thoth.db, migrations, index"
Cohesion: 0.29
Nodes (6): Migrations (internal/store/migrations/), Ownership (who owns which table), Persistence — thoth.db, migrations, index, Rules that matter here, Settings keys (0007), The index (internal/index)

### Community 78 - "Hooks (web/src/hooks)"
Cohesion: 0.29
Nodes (6): Hooks (web/src/hooks), useChat, useConversationRoute, useSearch, useView, useViewShortcuts

### Community 79 - "Labels — the three-tier GitHub label set"
Cohesion: 0.33
Nodes (5): Areas (package-aligned), Kept GitHub defaults (outside the three-tier model), Labels — the three-tier GitHub label set, Priority (issues only), Types (mirror the conventional-commit prefixes)

### Community 81 - "testing.T"
Cohesion: 0.12
Nodes (36): TestEventShapeUnchanged(), TestEventTypeValues(), TestWriterFuncAdapter(), testing.T, ParseLine(), TestParseLineAssistantText(), TestParseLineAssistantWithEmptyText(), TestParseLineAssistantWithoutMessage() (+28 more)

### Community 82 - "NoteViewer.tsx"
Cohesion: 0.12
Nodes (20): cache, CodeBlock(), highlight(), CopyButton(), renderCopy(), components, Markdown(), MessageItem (+12 more)

### Community 83 - "Index"
Cohesion: 0.07
Nodes (43): Client, collect, Option, ctxAwareFake, github.com/shiv-source/thoth/agent.Provider, github.com/shiv-source/thoth/internal/claude.Event, github.com/shiv-source/thoth/internal/claude.EventWriter, dbLike (+35 more)

### Community 84 - "cli/doctor_test.go"
Cohesion: 0.38
Nodes (13): executeDoctor(), healthyEnv(), serveThothOnFixedPort(), TestDoctorDetectsBusyPort(), TestDoctorDetectsMissingClaude(), TestDoctorDetectsMissingIndexTables(), TestDoctorDetectsNonWALDatabase(), TestDoctorFixesMissingDefaultWiki() (+5 more)

### Community 85 - "watcher_test.go"
Cohesion: 0.28
Nodes (13): bytes.Buffer, sync.Mutex, lockedBuffer, newPublishingWatcher(), TestWatchAttachmentChangesPublishNothing(), TestWatchPublishesChangeBatch(), TestWatchPublishesDirectoryRemoval(), TestWatchPublishesNoDotfileNoise() (+5 more)

### Community 86 - "file.go"
Cohesion: 0.18
Nodes (7): cleanRel(), stringArg(), stringArgDefault(), truncationMarker(), TestStringArg(), ReadFile, WriteFile

### Community 90 - "history_test.go"
Cohesion: 0.24
Nodes (21): fakeSummarizer, CacheMarkers(), Cap(), Message, hasOrphanedResult(), lastUserTurn(), nthUserTurn(), previousUser() (+13 more)

### Community 91 - "log/slog.Logger"
Cohesion: 0.11
Nodes (22): prewarmStore, rootHolder, echo.MiddlewareFunc, log/slog.Logger, sync.RWMutex, requestLog(), devCommit(), ensureWiki() (+14 more)

### Community 92 - "SettingsView.test.tsx"
Cohesion: 0.15
Nodes (6): connected, devHealth, emptyGitHub, mocks, renderSettings(), settings

### Community 95 - "Deps"
Cohesion: 0.05
Nodes (65): claudeState, Deps, githubIdentity, healthResponse, modelGroup, modelInput, settingsDTO, wikiState (+57 more)

### Community 96 - "ParseNote"
Cohesion: 0.33
Nodes (8): ParseNote(), TestParseNote(), TestParseNoteClosedAtEOF(), TestParseNoteRejectsBadYAML(), TestParseNoteRejectsMissingFrontmatter(), TestParseNoteRejectsMissingTitle(), TestParseNoteRejectsUnclosedFrontmatter(), NoteMeta

### Community 97 - "github.com/shiv-source/thoth/agent.Request"
Cohesion: 0.24
Nodes (6): cancelProvider, script, scriptedProvider, github.com/shiv-source/thoth/agent.Request, github.com/shiv-source/thoth/agent.Stream, fakeProvider

### Community 98 - "CLIClient"
Cohesion: 0.24
Nodes (4): startConfig, stderrTail, CLIClient, turnFailure()

### Community 102 - "Validate"
Cohesion: 0.43
Nodes (5): TestValidate(), TestValidateReportsMultipleProblems(), topFolder(), Validate(), Problem

### Community 103 - "wiki.go"
Cohesion: 0.25
Nodes (11): checkMalformed(), countNotes(), TestCheckMalformed(), Index, Hidden(), Indexable(), IsMarkdownPath(), isReservedPath() (+3 more)

### Community 104 - "NewSearch"
Cohesion: 0.19
Nodes (10): NewSearch(), TestSearchArgValidation(), TestSearchDefaultLimit(), TestSearchEmpty(), TestSearchErrorPropagates(), TestSearchFormat(), TestSearchToolEnforcesLimit(), Result (+2 more)

### Community 105 - "context.Context"
Cohesion: 0.22
Nodes (9): Agent, Block, Message, requestTools(), writeDelta(), writeToolEvent(), hangClient, EventWriter (+1 more)

### Community 106 - "WikiTree.tsx"
Cohesion: 0.11
Nodes (27): TreeNode, NotesView, NotesView(), toTreeData(), WikiDataNode, WikiTree(), RootState, connectionSlice (+19 more)

### Community 107 - "file_test.go"
Cohesion: 0.22
Nodes (18): NewList(), NewOSFS(), NewReadFile(), NewWriteFile(), newTestFS(), TestListTool(), TestNewOSFSValidation(), TestOSFSRejectsTraversal() (+10 more)

### Community 109 - "os/exec.Cmd"
Cohesion: 0.18
Nodes (8): os/exec.Cmd, os.Process, CLIClient, killProcess(), setProcessGroup(), CLIClient, killProcess(), setProcessGroup()

### Community 110 - "plugins"
Cohesion: 0.22
Nodes (8): oxc, typescript, warn, plugins, rules, react/only-export-components, react/rules-of-hooks, $schema

### Community 113 - "Watch"
Cohesion: 0.18
Nodes (13): github.com/fsnotify/fsnotify.Event, github.com/fsnotify/fsnotify.Watcher, fakeWatcher, fsnotifyAdapter, Index, newFakeWatcher(), TestWatchIndexesAttachment(), TestWatchIndexesMarkdownExtension() (+5 more)

### Community 114 - "openModels"
Cohesion: 0.36
Nodes (9): openModels(), TestCreateModelDuplicateValue(), TestDeleteModelNotFound(), TestListModelsSeedOrder(), TestMigrationSeedsAPIKeyRow(), TestModelByID(), TestModelsCRUD(), TestUpdateModelNotFound() (+1 more)

### Community 116 - "NewRegistry"
Cohesion: 0.26
Nodes (8): NewRegistry(), TestRegistryDuplicateErrors(), TestRegistryEmptyList(), TestRegistryGetUnknown(), TestRegistryRegisterGetList(), TestRegistryRejectsNilAndEmptyName(), TestToolSchemas(), toolStub

### Community 117 - "New"
Cohesion: 0.25
Nodes (10): IsImagePath(), New(), TestIndexable(), TestIsImagePath(), TestVisible(), TestWikiNotExists(), TestWikiReadAndTree(), TestWikiReadMissingNote() (+2 more)

### Community 119 - "ModelOptions"
Cohesion: 0.33
Nodes (7): Option, ModelOptions(), TestModelOptionsParse(), TestModelOptionsSplitShape(), ensureModels(), TestEnsureModelsReseedsEmptyTable(), TestEnsureModelsSeedsFirstBoot()

### Community 120 - "Repo"
Cohesion: 0.19
Nodes (6): defaultWikiPath(), resolveWikiPath(), settleWikiPath(), TestDefaultWikiPath(), TestResolveWikiPath(), Repo

### Community 121 - "NewHub"
Cohesion: 0.25
Nodes (8): TestHubBroadcastDeliversToClients(), TestHubBroadcastDropsSlowClient(), NewHub(), TestHubFireAwaySkipsWhenClientActive(), TestHubRelaxCancelledOnReturn(), TestHubRelaxFlushesWhenLastClientGoesAway(), TestHubRelaxOnDisconnect(), TestHubSessionIDUsesRotated()

### Community 122 - "watcher.go"
Cohesion: 0.39
Nodes (7): github.com/fsnotify/fsnotify.Op, fileWatcher, watchConfig, WatchOption, opName(), WithPublisher(), withWatcher()

### Community 123 - "registry"
Cohesion: 0.43
Nodes (5): indexSearch(), registry(), TestIndexSearchTool(), TestWikiToolsBoundToSafePath(), TestWriteFileProducesParseableNote()

### Community 124 - "github.com/shiv-source/thoth/agent.Delta"
Cohesion: 0.10
Nodes (15): fakeProvider, stream, ThinkingDelta(), fakeStream, stopReason(), scriptedStream, Frame, SSEReader (+7 more)

### Community 126 - "FakeClient"
Cohesion: 0.50
Nodes (3): staleLockClient, Call, FakeClient

### Community 128 - "newLoggingServer"
Cohesion: 0.53
Nodes (5): echo.Echo, newLoggingServer(), TestRequestLogsAPIPaths(), TestRequestLogsFailureWithErr(), TestRequestLogSkipsNonAPIPaths()

### Community 129 - "TestGitSetupReportsSanitizedFailure"
Cohesion: 0.60
Nodes (4): TestGitSetupReportsSanitizedFailure(), TestGitSetupRequiresURL(), TestGitSetupRunsAgainstWiki(), writeFakeGit()

### Community 130 - "OSFS"
Cohesion: 0.22
Nodes (3): io/fs.FileMode, List, OSFS

### Community 132 - "testDeps"
Cohesion: 0.10
Nodes (39): groupBody, modelBody, net/http.Handler, net/http/httptest.ResponseRecorder, TestConversationsStoreError(), TestCreateConversationRejectsEmptyTitle(), TestGetConversationFound(), TestGetConversationNotFound() (+31 more)

### Community 133 - "ScaffoldWithOptions"
Cohesion: 0.31
Nodes (6): Init(), TestInitCreatesRepository(), TestInitErrorsWithoutGit(), gitInit(), ScaffoldWithOptions(), ScaffoldOptions

### Community 134 - "anthropic.go"
Cohesion: 0.10
Nodes (28): buildRequest(), Client, wireMessage, wireTool, newStream(), wireBlock(), wireBlocks(), wireMessages() (+20 more)

### Community 135 - "SafePath"
Cohesion: 0.24
Nodes (4): wikiFS, io/fs.DirEntry, SafePath(), TestSafePath()

## Knowledge Gaps
- **470 isolated node(s):** `ALLOWED_KINDS`, `config`, `@playwright/mcp`, `@ant-design/cli`, `Summarizer` (+465 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **29 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Deps` connect `Deps` to `New`, `NewPersistent`, `testDeps`, `context.Context`, `Hub`, `Index`, `PersistentClient`, `Repo`, `log/slog.Logger`?**
  _High betweenness centrality (0.019) - this node is a cross-community bridge._
- **Why does `CLIClient` connect `CLIClient` to `watcher_test.go`, `NewPersistent`, `os/exec.Cmd`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._
- **Why does `Hub` connect `Hub` to `NewPersistent`, `context.Context`, `watcher_test.go`, `PersistentClient`, `NewHub`, `log/slog.Logger`, `Deps`?**
  _High betweenness centrality (0.013) - this node is a cross-community bridge._
- **Are the 85 inferred relationships involving `testDeps()` (e.g. with `TestHubBroadcastDeliversToClients()` and `TestHubBroadcastDropsSlowClient()`) actually correct?**
  _`testDeps()` has 85 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `New()` (e.g. with `TestWikiChangedFrameReachesSocket()` and `TestChatCancelBeforeSendIsNoop()`) actually correct?**
  _`New()` has 80 INFERRED edges - model-reasoned connections that need verification._
- **What connects `ALLOWED_KINDS`, `config`, `@playwright/mcp` to the rest of the system?**
  _470 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Troubleshooting & FAQ` be split into smaller, more focused modules?**
  _Cohesion score 0.05714285714285714 - nodes in this community are weakly interconnected._