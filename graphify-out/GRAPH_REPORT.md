# Graph Report - thoth  (2026-08-21)

## Corpus Check
- 334 files · ~193,056 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2823 nodes · 7173 edges · 157 communities (124 shown, 33 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 1033 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `34772db4`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- wsURL
- mockAxios.ts
- WriteFile
- Troubleshooting & FAQ
- tools/conversations.go
- GitOptions
- NoteViewer.tsx
- NewTextBlock
- dependencies
- index.ts
- devDependencies
- package.json
- SafePath
- Shared checklist — both layers (yes/no; any "no" gets fixed before the PR)
- setup-go-web composite action
- dependencies
- Hub
- compilerOptions
- sse_test.go
- Components (web/src/components)
- github.com/shiv-source/thoth/agent.Delta
- documentation.md
- scaffold_test.go
- testDeps
- New
- Toolchain versions (go.mod / package.json authoritative)
- compilerOptions
- CLAUDE.md - Thoth repository rulebook
- DashboardView.tsx
- Workflows
- go.md
- Development - toolchain, gates, CI
- Components - Go package deep dive
- lib.mjs
- newRootCmd
- Indexing and search - FTS5 and the file watcher
- github/client_test.go
- devDependencies
- edit.go
- Go packages (internal/* + cmd/thoth)
- searchHistorySlice.ts
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
- agent/client.go
- report job (summary + gate)
- github.com/shiv-source/thoth
- api/models_test.go
- events.go
- Redux store (web/src/store)
- useAppSelector
- web/package.json
- react.md
- StopDelta
- openai.go
- Quality gates — how this repo verifies work
- Code quality — the pre-PR gate
- agent/history_test.go
- useChat.ts
- Frontend patterns — the cross-cutting conventions
- New
- docs-site/package.json
- The claude blast wall (internal/claude)
- Persistence — thoth.db, migrations, index
- Hooks (web/src/hooks)
- Labels — the three-tier GitHub label set
- token-guard.sh
- doctor/doctor_test.go
- renderWithStore
- Open
- client.ts
- gitTestDeps
- context.Context
- doctor/doctor.go
- docusaurus.config.ts
- typescript
- tools/git_test.go
- NewTextBlock
- NewBuilder
- README.md
- setup.sh
- Deps
- github.com/shiv-source/thoth/agent.Event
- github.com/shiv-source/thoth/agent.Request
- openStore
- Open
- graph-check.sh
- main-guard.sh
- putSettingsReq
- Migrating to Thoth Agent
- NewSearch
- git/git_test.go
- Delta
- file_test.go
- sidebars.ts
- .Start
- plugins
- react-markdown
- @tailwindcss/typography
- Watch
- searchSlice.ts
- ParseNote
- Agent
- openTestRepo
- @types/react-dom
- runServe
- Usage
- FS
- jsdom
- registry
- github.com/shiv-source/thoth/agent.Usage
- remark-gfm
- Store
- wiki.go
- .runChecks
- blockingStream
- Repo
- stream
- GitInit
- GitStatus
- anthropic.go
- worktreeChange
- startupBanner
- TestClientWiresConversationAndHealthTools
- Repo
- vite
- ExpandHome
- New
- failStream
- @types/node
- @easyops-cn/docusaurus-search-local
- fakeTool
- time.Time
- watcher_test.go
- Wiki
- index.tsx
- Index
- testing.T
- History
- watcher.go
- ListTree
- requestLog
- react-dom

## God Nodes (most connected - your core abstractions)
1. `testDeps()` - 130 edges
2. `New()` - 122 edges
3. `registry()` - 51 edges
4. `Open()` - 44 edges
5. `Deps` - 43 edges
6. `FS` - 42 edges
7. `runServe()` - 42 edges
8. `Open()` - 38 edges
9. `New()` - 38 edges
10. `Components (web/src/components)` - 38 edges

## Surprising Connections (you probably didn't know these)
- `Runtime data: ~/.thoth (thoth.db + wiki/)` --conceptually_related_to--> `Database schema - thoth.db tables`  [INFERRED]
  CLAUDE.md → docs/schema.md
- `CI-enforced quality gates (make check)` --semantically_similar_to--> `Five quality gates (make check)`  [INFERRED] [semantically similar]
  CONTRIBUTING.md → docs/development.md
- `Additive migrations rule (never edit an applied migration)` --semantically_similar_to--> `SQL migrations gated on PRAGMA user_version`  [INFERRED] [semantically similar]
  CONTRIBUTING.md → docs/schema.md
- `Project invariants (files as source of truth, percent-w errors, no globals)` --semantically_similar_to--> `Data contract: files are the source of truth, thoth.db is derived`  [INFERRED] [semantically similar]
  CLAUDE.md → docs/architecture.md
- `web/README.md - stock Vite template README` --conceptually_related_to--> `Frontend - React 19 + TS strict + Tailwind v4`  [INFERRED]
  web/README.md → docs/frontend.md

## Import Cycles
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/searchHistorySlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/gitSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/wikiSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/healthSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/chatSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/notificationsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/noteSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/doctorSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/uiSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/settingsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/connectionSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/conversationsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/searchSlice.ts -> web/src/store/index.ts`

## Hyperedges (group relationships)
- **The four app-layer components of the single binary** — docs_architecture_app_layer, docs_components_api_pkg, docs_components_claude_pkg, docs_components_index_pkg [EXTRACTED 1.00]
- **Cross-compile build matrix (linux/darwin/windows)** — github_workflows_ci_build_linux, github_workflows_ci_build_darwin, github_workflows_ci_build_windows [EXTRACTED 1.00]
- **Single required CI gate pattern** — github_workflows_ci_pr_final_gate, github_workflows_ci_final_gate, github_workflows_final_gate_report, github_workflows_final_gate_single_check [EXTRACTED 1.00]
- **The five CI quality gates every commit must pass** — docs_development_quality_gates, docs_development_gate_vet, docs_development_gate_race, docs_development_gate_coverage, docs_development_gate_crosscompile, docs_development_gate_frontend [EXTRACTED 1.00]
- **Shared quality gates (backend + frontend)** — github_workflows_quality_backend_test, github_workflows_quality_backend_lint, github_workflows_quality_frontend_test, github_workflows_quality_frontend_lint, github_workflows_quality_frontend_typecheck [EXTRACTED 1.00]
- **The rulebook-driven wiki filing system** — internal_wiki_templates_claude_claude, internal_wiki_templates_claude_save_protocol, docs_knowledge_base_wiki_layout, docs_knowledge_base_frontmatter, docs_architecture_knowledge_layer [INFERRED 0.85]

## Communities (157 total, 33 thin omitted)

### Community 0 - "wsURL"
Cohesion: 0.16
Nodes (33): github.com/gorilla/websocket.Conn, TestChatOpenConversationExistsError(), TestChatSendConversationCreateError(), TestChatTurnWriterSurfacesAgentErrorEvent(), readMsg(), TestChatAcceptsPresenceFrames(), TestChatCancelBeforeSendIsNoop(), TestChatCancelStopsInFlightTurn() (+25 more)

### Community 1 - "mockAxios.ts"
Cohesion: 0.09
Nodes (32): TreeNode, mocks, NotesView(), mocks, renderNotes(), mocks, mocks, treeResponse (+24 more)

### Community 3 - "Troubleshooting & FAQ"
Cohesion: 0.06
Nodes (33): First run, Getting started, Install Thoth, Next steps, What you need, Your first conversation, A note I edited by hand isn't showing up, Chat turns fail — no API key or provider unreachable (+25 more)

### Community 4 - "tools/conversations.go"
Cohesion: 0.09
Nodes (24): conversationStore, NewGetConversation(), NewListConversations(), NewSearchConversations(), snippet(), convTime(), TestConversationToolSchemas(), TestGetConversation() (+16 more)

### Community 5 - "GitOptions"
Cohesion: 0.12
Nodes (14): auth(), ensureRepo(), guardErr(), identity(), NewGitDiff(), openRepo(), repoPath(), GitAuth (+6 more)

### Community 6 - "NoteViewer.tsx"
Cohesion: 0.13
Nodes (19): cache, CodeBlock(), highlight(), CopyButton(), components, Markdown(), MessageItem, isImagePath() (+11 more)

### Community 7 - "NewTextBlock"
Cohesion: 0.15
Nodes (24): Block, Message, NewTextBlock(), NewThinkingBlock(), NewToolResultBlock(), NewToolUseBlock(), ParseBlock(), ParseMessage() (+16 more)

### Community 8 - "dependencies"
Cohesion: 0.07
Nodes (27): @ant-design/icons, antd, axios, chart.js, react-chartjs-2, react-redux, @reduxjs/toolkit, shiki (+19 more)

### Community 9 - "index.ts"
Cohesion: 0.07
Nodes (58): DoctorCheck, GitHubIdentity, GitHubRepo, LLMModel, ModelGroup, ModelInput, Settings, defaultFolders (+50 more)

### Community 10 - "devDependencies"
Cohesion: 0.08
Nodes (25): eslint, eslint-config-prettier, @eslint/js, globals, oxlint, @testing-library/jest-dom, @testing-library/react, @testing-library/user-event (+17 more)

### Community 11 - "package.json"
Cohesion: 0.05
Nodes (37): husky, lint-staged, author, bugs, url, description, devDependencies, husky (+29 more)

### Community 12 - "SafePath"
Cohesion: 0.10
Nodes (14): AtomicWrite(), TestAtomicWriteCreatesFile(), TestAtomicWriteFailsWhenParentIsNotDir(), wikiFS, io/fs.DirEntry, io/fs.FileInfo, io/fs.FileMode, boundedSymlink() (+6 more)

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
Cohesion: 0.15
Nodes (15): Client, clientEntry, clientMsg, Hub, serverMsg, turn, context.CancelFunc, github.com/shiv-source/thoth/agent.WriterFunc (+7 more)

### Community 17 - "compilerOptions"
Cohesion: 0.08
Nodes (24): DOM, src, vite/client, compilerOptions, allowArbitraryExtensions, allowImportingTsExtensions, jsx, lib (+16 more)

### Community 18 - "sse_test.go"
Cohesion: 0.16
Nodes (17): NewSSEReader(), readAllFrames(), TestFrameDecode(), TestSSEReaderBlankLinesIgnored(), TestSSEReaderCapsOversizedFrame(), TestSSEReaderChunkBoundaries(), TestSSEReaderCommentsIgnored(), TestSSEReaderDONETerminator() (+9 more)

### Community 19 - "Components (web/src/components)"
Cohesion: 0.05
Nodes (38): ActivityChart, App shell & navigation, AppHeader, AppSider, Charts (Chart.js), chartSetup.ts, Chat, ChatActivityChart (+30 more)

### Community 20 - "github.com/shiv-source/thoth/agent.Delta"
Cohesion: 0.26
Nodes (6): Frame, SSEReader, isDONE(), stream, bufio.Reader, github.com/shiv-source/thoth/agent.Delta

### Community 22 - "scaffold_test.go"
Cohesion: 0.11
Nodes (28): TestSystemPromptFallsBackToRulebook(), folderMap(), NoteType(), noteType(), NoteTypes(), NoteTypesFor(), EnsureGitRepo(), EnsureReservedDir() (+20 more)

### Community 23 - "testDeps"
Cohesion: 0.08
Nodes (33): backendBody, TestHealthProviderNameDegrades(), TestHealthProviderNameUnknownModel(), TestHubRecordCapsAt500(), TestSettingsKVError(), TestHubBroadcastDeliversToClients(), TestHubBroadcastDropsSlowClient(), TestWikiChangedFrameReachesSocket() (+25 more)

### Community 24 - "New"
Cohesion: 0.20
Nodes (26): doReq(), TestAllowLocalOriginRejectsMalformedOrigin(), TestConnectGitHubNotConfigured(), TestConnectGitHubSaveError(), TestDisconnectGitHubError(), TestFakeClientSurfacesWriterError(), TestGetGitHubAuthError(), TestGetSettingsProviderReadError() (+18 more)

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
Cohesion: 0.13
Nodes (22): react, DashboardView, ActivityChart(), ChatActivityChart(), mockActivity, mockChatActivity, mockInbox, mockMeetings (+14 more)

### Community 29 - "Workflows"
Cohesion: 0.10
Nodes (19): 10. Diagnose/repair an install, 11. Cut a release, 1. Add a REST endpoint, 2. Extend the WS protocol, 3. Add a store migration, 4. Change claude CLI flags (BLAST WALL), 5. Add a settings key, 6. Extend the wiki contract (+11 more)

### Community 31 - "Development - toolchain, gates, CI"
Cohesion: 0.17
Nodes (13): CI-enforced quality gates (make check), CONTRIBUTING.md - contribution workflow, Additive migrations rule (never edit an applied migration), PR and review workflow (conventional commits, squash-merge), CI workflows (quality.yml, ci.yml, ci-pr.yml, final-gate.yml), Development - toolchain, gates, CI, Gate: 80 percent coverage floor on internal and cmd, Gate: five cross-compile targets (+5 more)

### Community 32 - "Components - Go package deep dive"
Cohesion: 0.28
Nodes (13): CLI - serve, init, version, doctor commands, thoth doctor - six install checks, thoth doctor --fix repair mode, Components - Go package deep dive, internal/doctor - shared install checks, internal/github - identity and git sync, internal/settings - settings KV repo, Documentation hub (index.md) (+5 more)

### Community 33 - "lib.mjs"
Cohesion: 0.24
Nodes (11): main(), writeStepOutput(), ALLOWED_KINDS, computeLabels(), loadConfig(), missingLabels(), normalizeLabel(), parseFields() (+3 more)

### Community 34 - "newRootCmd"
Cohesion: 0.12
Nodes (21): main(), github.com/spf13/cobra.Command, TestVersion(), TestDoctorReportsProviderAuthFailure(), initFolders(), newInitCmd(), TestInitCommandErrorOnUnwritableTarget(), TestInitCommandExpandsTildeInTarget() (+13 more)

### Community 35 - "Indexing and search - FTS5 and the file watcher"
Cohesion: 0.25
Nodes (11): Project invariants (files as source of truth, percent-w errors, no globals), App layer - single Go binary, Data contract: files are the source of truth, thoth.db is derived, thoth serve command, internal/api - the Echo server, internal/index - search and sync, useSearch - debounced, supersede-guarded search, bm25 ranking with title weighted 8x (+3 more)

### Community 36 - "github/client_test.go"
Cohesion: 0.21
Nodes (15): profileStub, net/http.HandlerFunc, Client, newStubClient(), TestFetchProfileContextDeadline(), TestFetchProfileEmailsBestEffort(), TestFetchProfileMalformedUserBody(), TestFetchProfileNetworkErrorIsSanitized() (+7 more)

### Community 37 - "devDependencies"
Cohesion: 0.18
Nodes (11): devDependencies, @docusaurus/module-type-aliases, @docusaurus/tsconfig, @docusaurus/types, @types/react, typescript, @types/react, typescript (+3 more)

### Community 38 - "edit.go"
Cohesion: 0.10
Nodes (12): NewAppendFile(), NewDeleteFile(), NewEditFile(), NewRenameFile(), TestAppendFileTool(), TestDeleteFileTool(), TestEditFileTool(), TestRenameFileTool() (+4 more)

### Community 39 - "Go packages (internal/* + cmd/thoth)"
Cohesion: 0.13
Nodes (14): cmd/thoth, Go packages (internal/* + cmd/thoth), internal/api, internal/assets, internal/claude — the blast wall, internal/cli, internal/config, internal/doctor (+6 more)

### Community 40 - "searchHistorySlice.ts"
Cohesion: 0.29
Nodes (8): loadHistory(), persistSearchHistory(), SEARCH_HISTORY_KEY, SEARCH_HISTORY_MAX, searchHistorySlice, SearchHistoryState, SearchHistoryStoreShape, selectSearchHistory()

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
Cohesion: 0.08
Nodes (37): Health, App(), NotesView, SearchView, SettingsView, SetupScreen, emptyGitHub, mocks (+29 more)

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

### Community 54 - "agent/client.go"
Cohesion: 0.21
Nodes (19): Client, Option, github.com/shiv-source/thoth/agent.Provider, time.Duration, anthropicClient(), options, openaiClient(), providerFor() (+11 more)

### Community 61 - "api/models_test.go"
Cohesion: 0.24
Nodes (17): groupBody, modelBody, net/http/httptest.ResponseRecorder, decodeGroups(), doModelsRequest(), TestModelsCreate(), TestModelsCreateDuplicate(), TestModelsCreateValidation() (+9 more)

### Community 62 - "events.go"
Cohesion: 0.47
Nodes (4): TestWriterFuncAdapter(), Event, EventType, WriterFunc

### Community 63 - "Redux store (web/src/store)"
Cohesion: 0.11
Nodes (17): chat, connection, conversations, doctor, git, health, hooks.ts, index.ts (+9 more)

### Community 64 - "useAppSelector"
Cohesion: 0.10
Nodes (26): AppHeader(), ChatPanel(), createSocket(), Composer(), NotificationPanel(), NOTIFICATION_ICONS, NOTIFICATION_PALETTE, NotificationIcon() (+18 more)

### Community 65 - "web/package.json"
Cohesion: 0.17
Nodes (11): name, private, scripts, build, dev, lint, preview, test (+3 more)

### Community 67 - "StopDelta"
Cohesion: 0.19
Nodes (25): StopDelta(), TextDelta(), ToolInputDelta(), alwaysToolProvider, TestLoopHistoryErrorFailsTurn(), TestLoopHistoryErrorWithCancelledCtx(), TestLoopProviderStreamError(), TestLoopProviderStreamErrorWithCancelledCtx() (+17 more)

### Community 68 - "openai.go"
Cohesion: 0.13
Nodes (23): newStream(), buildRequest(), Client, wireMessage, wireTool, newStream(), toolArguments(), wireMessages() (+15 more)

### Community 69 - "Quality gates — how this repo verifies work"
Cohesion: 0.20
Nodes (9): Commit hygiene, Concurrency, Coverage, Cross-compile, Dependency bumps, Lint, make check — everything CI enforces, locally, Quality gates — how this repo verifies work (+1 more)

### Community 70 - "Code quality — the pre-PR gate"
Cohesion: 0.18
Nodes (10): 1. Run the quality gates, 2. Walk the review checklist, 3. Triage a failing gate, Canonical docs, Code quality — the pre-PR gate, Gotchas, Key files, Maintenance (+2 more)

### Community 71 - "agent/history_test.go"
Cohesion: 0.24
Nodes (21): fakeSummarizer, CacheMarkers(), Cap(), Message, hasOrphanedResult(), lastUserTurn(), nthUserTurn(), previousUser() (+13 more)

### Community 72 - "useChat.ts"
Cohesion: 0.07
Nodes (29): TokenUsage, UsageLine(), freshSocket(), mocks, renderChatHook(), toolLabel(), useChat(), chatIdFromPath() (+21 more)

### Community 73 - "Frontend patterns — the cross-cutting conventions"
Cohesion: 0.20
Nodes (9): Ant Design first, Design tokens, Frontend patterns — the cross-cutting conventions, Package discipline, Routing, State placement, Test doubles (web/src/test), The API boundary (zod) (+1 more)

### Community 74 - "New"
Cohesion: 0.28
Nodes (21): Accumulate(), TestStreamHandleDecodeError(), TestStreamHTTPTransportError(), TestStreamPlainEOFWithoutFinishReason(), TestStreamRequestBuildErrors(), TestStreamSSEFrameError(), TestWithHTTPClientApplies(), New() (+13 more)

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

### Community 81 - "doctor/doctor_test.go"
Cohesion: 0.16
Nodes (38): net.Listener, byName(), freeAddr(), healthyThothDir(), providerStub(), runChecks(), runProviderChecks(), seedSettingsRaw() (+30 more)

### Community 82 - "renderWithStore"
Cohesion: 0.05
Nodes (30): renderPanel(), mocks, renderBlock(), renderCopy(), ChartStub, conversations, mocks, renderDashboard() (+22 more)

### Community 83 - "Open"
Cohesion: 0.11
Nodes (37): openIndex(), healthyThothDir(), TestDoctorRepairIndexSyncFails(), TestDoctorRepairScaffoldFails(), TestOnSettingsSavedSwitchesRootTwice(), TestOnSettingsSavedSyncError(), TestOpenIndex(), executeDoctor() (+29 more)

### Community 84 - "client.ts"
Cohesion: 0.10
Nodes (24): api, Conversation, http, Message, Note, ProviderConfig, mocks, TreeNodeSchema (+16 more)

### Community 85 - "gitTestDeps"
Cohesion: 0.35
Nodes (11): net/http.Handler, TestGitSetRemoteAddPath(), TestGitSetRemoteReplacesURL(), gitSetupReq(), gitTestDeps(), initBare(), TestGitSetupEmptyTree(), TestGitSetupReportsSanitizedFailure() (+3 more)

### Community 86 - "context.Context"
Cohesion: 0.10
Nodes (15): CleanRel(), IntArg(), IntArgDefault(), StringArg(), StringArgDefault(), StringSliceArg(), TruncationMarker(), WalkFiles() (+7 more)

### Community 87 - "doctor/doctor.go"
Cohesion: 0.23
Nodes (24): Options, providerProbe, backendSettings(), checkAPI(), checkDatabase(), checkIndex(), checkMalformed(), checkProvider() (+16 more)

### Community 90 - "tools/git_test.go"
Cohesion: 0.22
Nodes (19): NewGitCommit(), NewGitInit(), NewGitLog(), NewGitPush(), NewGitStatus(), committedDir(), gitOpts(), TestGitCommitAutoInit() (+11 more)

### Community 91 - "NewTextBlock"
Cohesion: 0.23
Nodes (29): NewTextBlock(), NewToolResultBlock(), NewToolUseBlock(), TestCapNoPreviousUserWhenOrphaned(), TestStreamCacheMarkerOnToolResultBlock(), TestStreamCacheMarkerOnToolUseBlock(), TestStreamDecodeErrors(), TestStreamHTTPTransportError() (+21 more)

### Community 92 - "NewBuilder"
Cohesion: 0.21
Nodes (11): Block, Message, NewBuilder(), TestBuilderAccumulatesText(), TestBuilderAccumulatesThinking(), TestBuilderAccumulatesToolInput(), TestBuilderIgnoresStop(), TestBuilderInvalidToolInput() (+3 more)

### Community 95 - "Deps"
Cohesion: 0.05
Nodes (66): backendState, Deps, githubIdentity, healthResponse, modelGroup, modelInput, providerDTO, settingsDTO (+58 more)

### Community 96 - "github.com/shiv-source/thoth/agent.Event"
Cohesion: 0.36
Nodes (5): collect, errWriter, eventRecorder, github.com/shiv-source/thoth/agent.Event, github.com/shiv-source/thoth/agent.EventType

### Community 97 - "github.com/shiv-source/thoth/agent.Request"
Cohesion: 0.22
Nodes (8): blockingProvider, cancelProvider, script, scriptedProvider, streamProvider, github.com/shiv-source/thoth/agent.Request, github.com/shiv-source/thoth/agent.Stream, fakeProvider

### Community 98 - "openStore"
Cohesion: 0.38
Nodes (15): New(), TestClientStartCapsHistory(), TestClientStartReadsSystemPerTurn(), TestClientStartRunsTurnAgainstFakeProvider(), TestClientStartSurfacesProviderError(), TestClientStartTurnTimeout(), TestNewRejectsMissingModel(), TestNewRejectsNilStore() (+7 more)

### Community 99 - "Open"
Cohesion: 0.27
Nodes (16): TestApply(), TestApplyClosedIndexLogsAndContinues(), TestApplyPathOutsideRoot(), TestApplyUnreadablePath(), TestWatchErrorOnMissingRoot(), TestWatchReturnsOnCancel(), Open(), discardLog() (+8 more)

### Community 102 - "putSettingsReq"
Cohesion: 0.16
Nodes (22): fixtureHandler, net/http.Request, net/http.ResponseWriter, allowLocalOrigin(), getSettingsReq(), putSettingsReq(), TestConversationsEndpoints(), TestDeleteConversationEndpoint() (+14 more)

### Community 103 - "Migrating to Thoth Agent"
Cohesion: 0.20
Nodes (8): Migrating to Thoth Agent, The advantages, What changed, What is Thoth Agent?, What it means for you, Why we did it, August 2026 — Thoth Agent replaces the Claude Code CLI (epic #121), What's new

### Community 104 - "NewSearch"
Cohesion: 0.18
Nodes (12): NewSearch(), TestSearchArgValidation(), TestSearchCtxCancelled(), TestSearchDefaultLimit(), TestSearchEmpty(), TestSearchErrorPropagates(), TestSearchFormat(), TestSearchToolEnforcesLimit() (+4 more)

### Community 105 - "git/git_test.go"
Cohesion: 0.27
Nodes (17): Init(), commitFile(), Repo, initBare(), initTestRepo(), rawBranch(), rawLog(), remoteURL() (+9 more)

### Community 106 - "Delta"
Cohesion: 0.23
Nodes (17): Block, Delta, StopDelta(), TestBlockDeltas(), TestDeltaConstructors(), TextDelta(), ThinkingDelta(), ToolInputDelta() (+9 more)

### Community 107 - "file_test.go"
Cohesion: 0.08
Nodes (37): NewList(), NewOSFS(), NewReadFile(), NewWriteFile(), newTestFS(), TestListTool(), TestNewOSFSValidation(), TestOSFSMkdirAllErrorWhenParentIsFile() (+29 more)

### Community 109 - ".Start"
Cohesion: 0.29
Nodes (8): Agent, Block, Message, requestTools(), writeDelta(), writeToolEvent(), Stream, EventWriter

### Community 110 - "plugins"
Cohesion: 0.22
Nodes (8): oxc, typescript, warn, plugins, rules, react/only-export-components, react/rules-of-hooks, $schema

### Community 113 - "Watch"
Cohesion: 0.18
Nodes (13): github.com/fsnotify/fsnotify.Event, github.com/fsnotify/fsnotify.Watcher, fakeWatcher, fsnotifyAdapter, Index, newFakeWatcher(), TestWatchIndexesAttachment(), TestWatchIndexesMarkdownExtension() (+5 more)

### Community 114 - "searchSlice.ts"
Cohesion: 0.21
Nodes (13): SearchResult, mocks, renderSearchHook(), useSearch(), initialState, searchNotes, searchSlice, SearchState (+5 more)

### Community 115 - "ParseNote"
Cohesion: 0.11
Nodes (33): noteTypeFor(), TestNoteTypeFor(), walkNotes(), FormatNote(), NoteMeta, isFence(), noteDate(), noteTags() (+25 more)

### Community 116 - "Agent"
Cohesion: 0.14
Nodes (15): Agent, Options, Block, Message, New(), NewBuilder(), NewThinkingBlock(), ParseBlock() (+7 more)

### Community 117 - "openTestRepo"
Cohesion: 0.12
Nodes (20): Auth, database/sql.DB, gitToolOptions(), TestGitToolOptions(), Auth, Repo, OpenRepo(), Repo (+12 more)

### Community 119 - "runServe"
Cohesion: 0.08
Nodes (51): Option, log/slog.Logger, ModelOptions(), TestModelOptionsParse(), TestModelOptionsSplitShape(), TestEnsureModelsStoreError(), TestModelProviderStoreError(), TestRunServeBootsAndShutsDown() (+43 more)

### Community 120 - "Usage"
Cohesion: 0.22
Nodes (7): Request, Tool, Usage, Provider, Response, stubProvider, stubStream

### Community 121 - "FS"
Cohesion: 0.08
Nodes (36): NewListRecent(), NewSearchByTag(), TestWalkNotesSkipsNonNotesAndHidden(), TestWikiToolNamesAndSchemasPopulated(), TestWikiToolSchemas(), NewReadNote(), NewWriteNote(), newTestFS() (+28 more)

### Community 123 - "registry"
Cohesion: 0.19
Nodes (21): stubTool, TestConversationStoreAdapter(), TestConversationToolsRegisteredWhenConfigured(), TestConversationToolsRoundTrip(), TestDoctorHealth(), TestSystemHealthRegisteredWhenConfigured(), registry(), TestFSBackedToolsBoundAndFollowRoot() (+13 more)

### Community 124 - "github.com/shiv-source/thoth/agent.Usage"
Cohesion: 0.13
Nodes (9): fakeStream, scriptedStream, Call, ctxAwareFake, FakeClient, hangClient, scriptedHangClient, github.com/shiv-source/thoth/agent.EventWriter (+1 more)

### Community 126 - "Store"
Cohesion: 0.23
Nodes (5): encoding/json.RawMessage, Store, newID(), Conversation, Message

### Community 127 - "wiki.go"
Cohesion: 0.18
Nodes (15): countNotes(), Index, TestValidate(), TestValidateReportsMultipleProblems(), topFolder(), Validate(), Hidden(), Indexable() (+7 more)

### Community 128 - ".runChecks"
Cohesion: 0.29
Nodes (7): doctorRunner, TestResolveThothDirEmptyUsesHome(), failed(), fileExists(), newDoctorCmd(), resolveThothDir(), runDoctor()

### Community 130 - "Repo"
Cohesion: 0.22
Nodes (6): Repo, Open(), TestOpen(), Auth, Identity, git.Repository

### Community 131 - "stream"
Cohesion: 0.32
Nodes (3): stopReason(), io.Closer, stream

### Community 134 - "anthropic.go"
Cohesion: 0.12
Nodes (26): buildRequest(), Client, wireMessage, wireTool, markMessage(), wireBlock(), wireBlocks(), wireMessages() (+18 more)

### Community 135 - "worktreeChange"
Cohesion: 0.25
Nodes (10): readWorktreeFile(), worktreeChange(), writeBlob(), git.FileStatus, git.Worktree, github.com/go-git/go-git/v5/plumbing/filemode.FileMode, github.com/go-git/go-git/v5/plumbing.Hash, github.com/go-git/go-git/v5/plumbing/object.Change (+2 more)

### Community 136 - "startupBanner"
Cohesion: 0.27
Nodes (8): os.File, isTerminal(), startupBanner(), TestStartupBannerAddsColorOnlyWhenAsked(), TestStartupBannerContainsTheFacts(), TestStartupBannerFormatsIPv6Hosts(), TestStartupBannerShowsTheBigWordmark(), TestIsTerminal()

### Community 137 - "TestClientWiresConversationAndHealthTools"
Cohesion: 0.18
Nodes (9): NewSystemHealth(), TestSystemHealth(), TestSystemHealthChecksContext(), WithHealthFunc(), DoctorHealth(), TestClientWiresConversationAndHealthTools(), HealthFunc, HealthReport (+1 more)

### Community 138 - "Repo"
Cohesion: 0.14
Nodes (16): Repo, ProviderAPIKeyKey(), ProviderBaseURLKey(), providerSlug(), Repo, openTestRepo(), TestFoldersParsing(), TestModelSettingRoundTrip() (+8 more)

### Community 140 - "ExpandHome"
Cohesion: 0.38
Nodes (5): ExpandHome(), TestExpandHome(), TestExpandHomeBareTilde(), TestToTilde(), ToTilde()

### Community 141 - "New"
Cohesion: 0.17
Nodes (17): getResult, Profile, Repository, net/http/httptest.Server, doJSON(), githubStub(), TestConnectGitHub(), TestConnectGitHubRejectedToken() (+9 more)

### Community 149 - "time.Time"
Cohesion: 0.10
Nodes (8): NewGetTime(), TestGetTimeTool(), TestGetTimeToolCtxCancelled(), time.Time, GetInbox, GetTime, Remember, WriteNote

### Community 151 - "watcher_test.go"
Cohesion: 0.28
Nodes (13): bytes.Buffer, sync.Mutex, lockedBuffer, newPublishingWatcher(), TestWatchAttachmentChangesPublishNothing(), TestWatchPublishesChangeBatch(), TestWatchPublishesDirectoryRemoval(), TestWatchPublishesNoDotfileNoise() (+5 more)

### Community 152 - "Wiki"
Cohesion: 0.14
Nodes (8): RegistryOptions, sync.RWMutex, SystemPrompt(), TestSystemPromptReadsRulebook(), discardLog(), writeRulebook(), Folders(), Wiki

### Community 155 - "Index"
Cohesion: 0.29
Nodes (6): dbLike, Note, Result, del(), Index, upsert()

### Community 157 - "testing.T"
Cohesion: 0.10
Nodes (38): TestEventShapeUnchanged(), TestEventTypeValues(), testing.T, echo.Echo, newLoggingServer(), TestRequestLogsAPIPaths(), TestRequestLogsFailureWithErr(), TestRequestLogSkipsNonAPIPaths() (+30 more)

### Community 160 - "History"
Cohesion: 0.36
Nodes (7): History(), TestHistoryDropsAllTrailingUserMessages(), TestHistoryEmptyConversation(), TestHistoryMapsMessagesAndDropsTrailingUser(), TestHistorySkipsNonChatRoles(), TestHistorySoloPromptStillDrops(), TestHistoryStoreError()

### Community 161 - "watcher.go"
Cohesion: 0.39
Nodes (7): github.com/fsnotify/fsnotify.Op, fileWatcher, watchConfig, WatchOption, opName(), WithPublisher(), withWatcher()

### Community 168 - "requestLog"
Cohesion: 0.40
Nodes (4): echo.MiddlewareFunc, TestRequestLogLogsErrors(), TestRequestLogNilIsIdentity(), requestLog()

## Knowledge Gaps
- **478 isolated node(s):** `ALLOWED_KINDS`, `config`, `@playwright/mcp`, `@ant-design/cli`, `Summarizer` (+473 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **33 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Deps` connect `Deps` to `anthropic.go`, `Repo`, `testDeps`, `gitTestDeps`, `context.Context`, `runServe`, `Wiki`, `New`, `Index`, `Store`?**
  _High betweenness centrality (0.035) - this node is a cross-community bridge._
- **Why does `Watch()` connect `Watch` to `watcher.go`, `Open`, `runServe`, `context.Context`, `watcher_test.go`, `wiki.go`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **Why does `testDeps()` connect `testDeps` to `wsURL`, `Open`, `putSettingsReq`, `New`, `Hub`, `Open`, `api/models_test.go`, `gitTestDeps`, `openTestRepo`, `New`, `registry`, `testing.T`, `Deps`?**
  _High betweenness centrality (0.014) - this node is a cross-community bridge._
- **Are the 116 inferred relationships involving `testDeps()` (e.g. with `TestAllowLocalOriginRejectsMalformedOrigin()` and `TestChatOpenConversationExistsError()`) actually correct?**
  _`testDeps()` has 116 INFERRED edges - model-reasoned connections that need verification._
- **Are the 117 inferred relationships involving `New()` (e.g. with `TestAllowLocalOriginRejectsMalformedOrigin()` and `TestChatOpenConversationExistsError()`) actually correct?**
  _`New()` has 117 INFERRED edges - model-reasoned connections that need verification._
- **Are the 16 inferred relationships involving `registry()` (e.g. with `New()` and `TestConversationToolsRegisteredWhenConfigured()`) actually correct?**
  _`registry()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **What connects `ALLOWED_KINDS`, `config`, `@playwright/mcp` to the rest of the system?**
  _478 weakly-connected nodes found - possible documentation gaps or missing edges._