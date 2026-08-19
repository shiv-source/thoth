# Graph Report - thoth  (2026-08-19)

## Corpus Check
- 245 files · ~120,608 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1723 nodes · 4021 edges · 101 communities (76 shown, 25 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 545 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `3ec01b86`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- testDeps
- Deps
- New
- Open
- WriterFunc
- Run
- healthyThothDir
- useAppSelector
- dependencies
- SettingsView.tsx
- devDependencies
- package.json
- DashboardView.tsx
- Shared checklist — both layers (yes/no; any "no" gets fixed before the PR)
- setup-go-web composite action
- Sidebar.tsx
- Hub
- compilerOptions
- WikiTree.tsx
- Components (web/src/components)
- newRootCmd
- documentation.md
- PersistentClient
- index.ts
- serve.go
- Toolchain versions (go.mod / package.json authoritative)
- compilerOptions
- CLAUDE.md - Thoth repository rulebook
- ParseLine
- Workflows
- go.md
- Development - toolchain, gates, CI
- Components - Go package deep dive
- New
- serve_test.go
- Indexing and search - FTS5 and the file watcher
- New
- Client
- newTestEcho
- Go packages (internal/* + cmd/thoth)
- Global Constraints
- App.tsx
- API - REST endpoints and WebSocket chat protocol
- Architecture - two layers, one binary
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
- api/models_test.go
- Redux store (web/src/store)
- Thoth Project Skills Suite — Design
- web/package.json
- react.md
- testing.T
- putSettingsReq
- Quality gates — how this repo verifies work
- Code quality — the pre-PR gate
- OpenRepo
- newLoggingServer
- Frontend patterns — the cross-cutting conventions
- Scaffold
- events.go
- The claude blast wall (internal/claude)
- Persistence — thoth.db, migrations, index
- Hooks (web/src/hooks)
- Labels — the three-tier GitHub label set
- token-guard.sh
- Open
- mockAxios.ts
- renderWithStore
- Repo
- react-dom
- remark-gfm
- @types/react
- typescript
- vite
- vitest
- context.Context
- README.md
- setup.sh
- @testing-library/jest-dom
- ExpandHome
- NotificationToasts.tsx
- os/exec.Cmd
- graph-check.sh
- main-guard.sh
- CLIClient
- ChatSocket
- globals
- react-redux

## God Nodes (most connected - your core abstractions)
1. `testDeps()` - 88 edges
2. `New()` - 81 edges
3. `Deps` - 38 edges
4. `Components (web/src/components)` - 38 edges
5. `useAppSelector` - 35 edges
6. `Run()` - 34 edges
7. `Open()` - 33 edges
8. `react` - 33 edges
9. `newServer()` - 32 edges
10. `WriterFunc` - 32 edges

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
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/wikiSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/notificationsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/settingsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/noteSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/chatSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/connectionSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/conversationsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/doctorSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/gitSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/healthSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/searchHistorySlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/searchSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/uiSlice.ts -> web/src/store/index.ts`

## Hyperedges (group relationships)
- **The four app-layer components of the single binary** — docs_architecture_app_layer, docs_components_api_pkg, docs_components_claude_pkg, docs_components_index_pkg [EXTRACTED 1.00]
- **Cross-compile build matrix (linux/darwin/windows)** — github_workflows_ci_build_linux, github_workflows_ci_build_darwin, github_workflows_ci_build_windows [EXTRACTED 1.00]
- **Single required CI gate pattern** — github_workflows_ci_pr_final_gate, github_workflows_ci_final_gate, github_workflows_final_gate_report, github_workflows_final_gate_single_check [EXTRACTED 1.00]
- **The five CI quality gates every commit must pass** — docs_development_quality_gates, docs_development_gate_vet, docs_development_gate_race, docs_development_gate_coverage, docs_development_gate_crosscompile, docs_development_gate_frontend [EXTRACTED 1.00]
- **Shared quality gates (backend + frontend)** — github_workflows_quality_backend_test, github_workflows_quality_backend_lint, github_workflows_quality_frontend_test, github_workflows_quality_frontend_lint, github_workflows_quality_frontend_typecheck [EXTRACTED 1.00]
- **The rulebook-driven wiki filing system** — internal_wiki_templates_claude_claude, internal_wiki_templates_claude_save_protocol, docs_knowledge_base_wiki_layout, docs_knowledge_base_frontmatter, docs_architecture_knowledge_layer [INFERRED 0.85]

## Communities (101 total, 25 thin omitted)

### Community 0 - "testDeps"
Cohesion: 0.24
Nodes (28): github.com/gorilla/websocket.Conn, TestWikiChangedFrameReachesSocket(), readMsg(), TestChatCancelBeforeSendIsNoop(), TestChatCancelStopsInFlightTurn(), TestChatForwardsThinkingFrames(), TestChatHubCancellationEndsTurns(), TestChatNewChatCancelsBusyTurn() (+20 more)

### Community 1 - "Deps"
Cohesion: 0.05
Nodes (60): claudeState, Deps, githubIdentity, healthResponse, modelGroup, modelInput, settingsDTO, wikiState (+52 more)

### Community 2 - "New"
Cohesion: 0.14
Nodes (19): TestConversationsStoreError(), TestCreateConversationRejectsEmptyTitle(), TestGetConversationFound(), TestGetConversationNotFound(), TestListDirsEndpoint(), TestListDirsEndpointErrors(), TestHealth(), TestHealthCommit() (+11 more)

### Community 3 - "Open"
Cohesion: 0.05
Nodes (64): echo.MiddlewareFunc, github.com/fsnotify/fsnotify.Op, log/slog.Logger, sync.Mutex, dbLike, Note, Result, watchConfig (+56 more)

### Community 4 - "WriterFunc"
Cohesion: 0.09
Nodes (61): Option, StartOption, New(), TestDirProviderOverridesStaticDir(), TestFakeClient(), TestFakeClientPropagatesError(), TestFakeClientPropagatesWriterError(), TestFakeClientRecordsResume() (+53 more)

### Community 5 - "Run"
Cohesion: 0.11
Nodes (50): doctorRunner, net.Listener, failed(), fileExists(), newDoctorCmd(), resolveThothDir(), runDoctor(), checkAPI() (+42 more)

### Community 7 - "useAppSelector"
Cohesion: 0.19
Nodes (16): AppHeader(), ChatPanel(), createSocket(), Composer(), NotificationPanel(), SearchPanel(), useSearch(), useAppDispatch (+8 more)

### Community 8 - "dependencies"
Cohesion: 0.07
Nodes (27): @ant-design/icons, antd, axios, chart.js, @fontsource-variable/fraunces, react, react-chartjs-2, react-markdown (+19 more)

### Community 9 - "SettingsView.tsx"
Cohesion: 0.06
Nodes (58): DoctorCheck, GitHubIdentity, GitHubRepo, http, LLMModel, Message, ModelGroup, ModelInput (+50 more)

### Community 10 - "devDependencies"
Cohesion: 0.08
Nodes (25): eslint, eslint-config-prettier, @eslint/js, jsdom, oxlint, @tailwindcss/typography, @testing-library/react, @testing-library/user-event (+17 more)

### Community 11 - "package.json"
Cohesion: 0.06
Nodes (35): husky, lint-staged, author, bugs, url, description, devDependencies, husky (+27 more)

### Community 12 - "DashboardView.tsx"
Cohesion: 0.13
Nodes (22): react, DashboardView, ActivityChart(), ChatActivityChart(), mockActivity, mockChatActivity, mockInbox, mockMeetings (+14 more)

### Community 13 - "Shared checklist — both layers (yes/no; any "no" gets fixed before the PR)"
Cohesion: 0.33
Nodes (5): Correctness, Naming & style, Shared checklist — both layers (yes/no; any "no" gets fixed before the PR), Structure, Tests

### Community 14 - "setup-go-web composite action"
Cohesion: 0.11
Nodes (26): golangci-lint v2 config, Frontend embed build (make web), setup-go-web composite action, Frozen-lockfile install, setup-web composite action, Pull request template, build-darwin job, build-linux job (+18 more)

### Community 15 - "Sidebar.tsx"
Cohesion: 0.22
Nodes (15): Conversation, ChatsList(), groupByDay(), relativeDate(), Sidebar(), navigate(), conversationsSlice, ConversationsState (+7 more)

### Community 16 - "Hub"
Cohesion: 0.14
Nodes (14): clientMsg, Hub, serverMsg, turn, context.CancelFunc, strings.Builder, TestHubBroadcastDeliversToClients(), TestHubBroadcastDropsSlowClient() (+6 more)

### Community 17 - "compilerOptions"
Cohesion: 0.08
Nodes (24): DOM, src, vite/client, compilerOptions, allowArbitraryExtensions, allowImportingTsExtensions, jsx, lib (+16 more)

### Community 18 - "WikiTree.tsx"
Cohesion: 0.20
Nodes (17): TreeNode, NotesView, NotesView(), toTreeData(), WikiDataNode, WikiTree(), selectNotesExpandedKeys(), collectTreeInfo() (+9 more)

### Community 19 - "Components (web/src/components)"
Cohesion: 0.05
Nodes (38): ActivityChart, App shell & navigation, AppHeader, AppSider, Charts (Chart.js), chartSetup.ts, Chat, ChatActivityChart (+30 more)

### Community 20 - "newRootCmd"
Cohesion: 0.13
Nodes (18): main(), github.com/spf13/cobra.Command, newInitCmd(), TestInitCommandErrorOnUnwritableTarget(), TestInitCommandExpandsTildeInTarget(), TestInitCommandTooManyArgs(), TestInitCommandUsesDefaultPath(), Execute() (+10 more)

### Community 22 - "PersistentClient"
Cohesion: 0.17
Nodes (12): PersistentClient, poolEntry, proc, startConfig, CLIClient, bufio.Writer, io.ReadCloser, os.File (+4 more)

### Community 23 - "index.ts"
Cohesion: 0.10
Nodes (34): SearchResult, mocks, renderChatHook(), toolLabel(), useChat(), mocks, renderSearchHook(), AppDispatch (+26 more)

### Community 24 - "serve.go"
Cohesion: 0.13
Nodes (17): rootHolder, sync.RWMutex, defaultWikiPath(), echo.Echo, newRootHolder(), onSettingsSaved(), resolveClaudeBin(), resolveWikiPath() (+9 more)

### Community 25 - "Toolchain versions (go.mod / package.json authoritative)"
Cohesion: 0.13
Nodes (20): Chart.js, Cobra 1.10 - Go CLI framework, Echo 4.15 - Go web framework, fsnotify 1.10 - file watcher, gorilla/websocket 1.5, React 19.2, Redux Toolkit, SQLite + FTS5 (modernc.org/sqlite 1.56) (+12 more)

### Community 26 - "compilerOptions"
Cohesion: 0.10
Nodes (19): node, vite.config.ts, compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection (+11 more)

### Community 27 - "CLAUDE.md - Thoth repository rulebook"
Cohesion: 0.16
Nodes (19): Blast wall - all Claude CLI flags live only in client.go, Branch workflow - never commit to main directly, CLAUDE.md - Thoth repository rulebook, Memory and resource safety rules (no leaks), Code rules: DRY, SOLID, KISS, YAGNI, small functions, Runtime data: ~/.thoth (thoth.db + wiki/), Claude Code CLI - driven headless per conversation, Two interfaces, one contract (dashboard and terminal) (+11 more)

### Community 28 - "ParseLine"
Cohesion: 0.29
Nodes (11): ParseLine(), TestParseLineAssistantText(), TestParseLineAssistantWithEmptyText(), TestParseLineAssistantWithoutMessage(), TestParseLineIgnoresStringShapedMessage(), TestParseLineIgnoresUnknown(), TestParseLineRejectsGarbage(), TestParseLineResult() (+3 more)

### Community 29 - "Workflows"
Cohesion: 0.11
Nodes (18): 10. Diagnose/repair an install, 11. Cut a release, 1. Add a REST endpoint, 2. Extend the WS protocol, 3. Add a store migration, 4. Change claude CLI flags (BLAST WALL), 5. Add a settings key, 6. Extend the wiki contract (+10 more)

### Community 31 - "Development - toolchain, gates, CI"
Cohesion: 0.17
Nodes (13): CI-enforced quality gates (make check), CONTRIBUTING.md - contribution workflow, Additive migrations rule (never edit an applied migration), PR and review workflow (conventional commits, squash-merge), CI workflows (quality.yml, ci.yml, ci-pr.yml, final-gate.yml), Development - toolchain, gates, CI, Gate: 80 percent coverage floor on internal and cmd, Gate: five cross-compile targets (+5 more)

### Community 32 - "Components - Go package deep dive"
Cohesion: 0.28
Nodes (13): CLI - serve, init, version, doctor commands, thoth doctor - six install checks, thoth doctor --fix repair mode, Components - Go package deep dive, internal/doctor - shared install checks, internal/github - identity and git sync, internal/settings - settings KV repo, Documentation hub (index.md) (+5 more)

### Community 33 - "New"
Cohesion: 0.22
Nodes (12): ensureWiki(), Wiki, New(), TestVisible(), TestWikiNotExists(), TestWikiReadAndTree(), TestWikiReadMissingNote(), TestWikiTreeErrorOnMissingRoot() (+4 more)

### Community 34 - "serve_test.go"
Cohesion: 0.23
Nodes (12): Option, ModelOptions(), TestModelOptionsParse(), TestModelOptionsSplitShape(), devCommit(), ensureModels(), openTestRepos(), TestDevCommit() (+4 more)

### Community 35 - "Indexing and search - FTS5 and the file watcher"
Cohesion: 0.25
Nodes (11): Project invariants (files as source of truth, percent-w errors, no globals), App layer - single Go binary, Data contract: files are the source of truth, thoth.db is derived, thoth serve command, internal/api - the Echo server, internal/index - search and sync, useSearch - debounced, supersede-guarded search, bm25 ranking with title weighted 8x (+3 more)

### Community 36 - "New"
Cohesion: 0.14
Nodes (27): profileStub, net/http.HandlerFunc, net/http/httptest.Server, doJSON(), githubStub(), TestConnectGitHub(), TestConnectGitHubRejectedToken(), TestConnectGitHubRequiresToken() (+19 more)

### Community 37 - "Client"
Cohesion: 0.29
Nodes (6): getResult, Profile, Repository, net/http.Client, Client, primaryEmail()

### Community 38 - "newTestEcho"
Cohesion: 0.29
Nodes (8): echo.Echo, Register(), echo.Echo, newTestEcho(), TestRegisterFallsBackToIndexForMissingPaths(), TestRegisterReturns404ForUnknownAPIPaths(), TestRegisterServesExistingAsset(), TestRegisterServesIndexAtRoot()

### Community 39 - "Go packages (internal/* + cmd/thoth)"
Cohesion: 0.13
Nodes (14): cmd/thoth, Go packages (internal/* + cmd/thoth), internal/api, internal/assets, internal/claude — the blast wall, internal/cli, internal/config, internal/doctor (+6 more)

### Community 40 - "Global Constraints"
Cohesion: 0.13
Nodes (14): Global Constraints, Task 10: react/references/patterns.md — the cross-cutting conventions, Task 11: CLAUDE.md — the one pointer line, Task 12: Full-suite verification + PR, Task 1: go/SKILL.md — the backend procedure skill, Task 2: go/references/packages.md — the package index, Task 3: go/references/claude-blast-wall.md — the version-sensitive zone, Task 4: go/references/persistence.md — thoth.db and migrations (+6 more)

### Community 41 - "App.tsx"
Cohesion: 0.06
Nodes (43): oxc, typescript, warn, plugins, rules, react/only-export-components, react/rules-of-hooks, $schema (+35 more)

### Community 43 - "API - REST endpoints and WebSocket chat protocol"
Cohesion: 0.39
Nodes (8): API - REST endpoints and WebSocket chat protocol, Resume with 500-message replay ring, Per-conversation Claude CLI session pool, Supersede-on-send and cancel chat semantics, WebSocket chat protocol (/ws), internal/store - conversations and messages, conversations table (claude_session_id), messages table (chat transcript)

### Community 44 - "Architecture - two layers, one binary"
Cohesion: 0.43
Nodes (8): Architecture - two layers, one binary, Knowledge layer - plain markdown wiki you own, internal/wiki - the file contract, Frontmatter contract (title required), Knowledge base - the wiki directory, Wiki folder layout (8 folders), Wiki rulebook template (CLAUDE.md in wiki root), The save protocol (folder map, frontmatter, confirm)

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

### Community 61 - "api/models_test.go"
Cohesion: 0.22
Nodes (18): groupBody, modelBody, net/http.Handler, net/http/httptest.ResponseRecorder, decodeGroups(), doModelsRequest(), TestModelsCreate(), TestModelsCreateDuplicate() (+10 more)

### Community 63 - "Redux store (web/src/store)"
Cohesion: 0.11
Nodes (17): chat, connection, conversations, doctor, git, health, hooks.ts, index.ts (+9 more)

### Community 64 - "Thoth Project Skills Suite — Design"
Cohesion: 0.17
Nodes (11): Approach: C — hybrid, Decisions log, Inventory, Maintenance, Out of scope, Purpose, Reference file anatomy, SKILL.md anatomy (+3 more)

### Community 65 - "web/package.json"
Cohesion: 0.17
Nodes (11): name, private, scripts, build, dev, lint, preview, test (+3 more)

### Community 67 - "testing.T"
Cohesion: 0.18
Nodes (23): testing.T, TestGitSetupReportsSanitizedFailure(), TestGitSetupRequiresURL(), TestGitSetupRunsAgainstWiki(), writeFakeGit(), Repo, openTestRepo(), TestModelSettingRoundTrip() (+15 more)

### Community 68 - "putSettingsReq"
Cohesion: 0.23
Nodes (16): net/http.Request, net/http.ResponseWriter, getSettingsReq(), putSettingsReq(), TestConversationsEndpoints(), TestDeleteConversationEndpoint(), TestSettingsAPIKeyNeverEchoed(), TestSettingsAPIKeyRoundTrip() (+8 more)

### Community 69 - "Quality gates — how this repo verifies work"
Cohesion: 0.20
Nodes (9): Commit hygiene, Concurrency, Coverage, Cross-compile, Dependency bumps, Lint, make check — everything CI enforces, locally, Quality gates — how this repo verifies work (+1 more)

### Community 70 - "Code quality — the pre-PR gate"
Cohesion: 0.18
Nodes (10): 1. Run the quality gates, 2. Walk the review checklist, 3. Triage a failing gate, Canonical docs, Code quality — the pre-PR gate, Gotchas, Key files, Maintenance (+2 more)

### Community 71 - "OpenRepo"
Cohesion: 0.36
Nodes (14): executeDoctor(), healthyEnv(), serveThothOnFixedPort(), TestDoctorDetectsBusyPort(), TestDoctorDetectsMissingClaude(), TestDoctorDetectsMissingIndexTables(), TestDoctorDetectsNonWALDatabase(), TestDoctorFixesMissingDefaultWiki() (+6 more)

### Community 72 - "newLoggingServer"
Cohesion: 0.43
Nodes (6): bytes.Buffer, echo.Echo, newLoggingServer(), TestRequestLogsAPIPaths(), TestRequestLogsFailureWithErr(), TestRequestLogSkipsNonAPIPaths()

### Community 73 - "Frontend patterns — the cross-cutting conventions"
Cohesion: 0.20
Nodes (9): Ant Design first, Design tokens, Frontend patterns — the cross-cutting conventions, Package discipline, Routing, State placement, Test doubles (web/src/test), The API boundary (zod) (+1 more)

### Community 74 - "Scaffold"
Cohesion: 0.29
Nodes (7): Folders(), Scaffold(), TestScaffoldCreatesSkeletonAndRulebook(), TestScaffoldErrorWhenParentIsFile(), TestScaffoldIsIdempotent(), TestScaffoldKeepsExistingCLAUDE(), Rulebook()

### Community 75 - "events.go"
Cohesion: 0.36
Nodes (6): Event, EventType, rawBlock, rawLine, rawMsg, encoding/json.RawMessage

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

### Community 81 - "Open"
Cohesion: 0.07
Nodes (33): Repo, database/sql.DB, time.Time, Auth, OpenRepo(), Repo, openTestRepo(), saved() (+25 more)

### Community 82 - "mockAxios.ts"
Cohesion: 0.11
Nodes (23): api, mocks, NoteViewer(), mocks, mocks, renderWikiTree(), treeResponse, mocks (+15 more)

### Community 83 - "renderWithStore"
Cohesion: 0.04
Nodes (35): mocks, renderPanel(), cache, CodeBlock(), highlight(), mocks, renderBlock(), CopyButton() (+27 more)

### Community 92 - "context.Context"
Cohesion: 0.23
Nodes (7): ctxAwareFake, hangClient, staleLockClient, Call, context.Context, EventWriter, FakeClient

### Community 96 - "ExpandHome"
Cohesion: 0.32
Nodes (6): settleWikiPath(), ExpandHome(), TestExpandHome(), TestExpandHomeBareTilde(), TestToTilde(), ToTilde()

### Community 97 - "NotificationToasts.tsx"
Cohesion: 0.21
Nodes (10): NOTIFICATION_ICONS, NOTIFICATION_PALETTE, NotificationIcon(), NotificationToasts(), initialState, Notification, NotificationKind, notificationsSlice (+2 more)

### Community 98 - "os/exec.Cmd"
Cohesion: 0.25
Nodes (5): os/exec.Cmd, setProcessGroup(), CLIClient, killProcess(), setProcessGroup()

### Community 104 - "CLIClient"
Cohesion: 0.19
Nodes (5): stderrTail, os.Process, CLIClient, CLIClient, killProcess()

### Community 106 - "ChatSocket"
Cohesion: 0.07
Nodes (17): freshSocket(), chatIdFromPath(), ConversationRouteOptions, getConversation, renderRoute(), useConversationRoute(), ChatMessage, connectionSlice (+9 more)

## Knowledge Gaps
- **408 isolated node(s):** `@playwright/mcp`, `@ant-design/cli`, `github.com/shiv-source/thoth`, `clientMsg`, `modelInput` (+403 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **25 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Deps` connect `Deps` to `testDeps`, `New`, `New`, `Open`, `Hub`, `Open`, `Repo`, `context.Context`?**
  _High betweenness centrality (0.026) - this node is a cross-community bridge._
- **Why does `testDeps()` connect `testDeps` to `Deps`, `New`, `testing.T`, `New`, `Open`, `healthyThothDir`, `OpenRepo`, `New`, `newLoggingServer`, `putSettingsReq`, `Hub`, `Open`, `api/models_test.go`?**
  _High betweenness centrality (0.017) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `testDeps`, `Deps`, `testing.T`, `New`, `putSettingsReq`, `healthyThothDir`, `WriterFunc`, `newLoggingServer`, `api/models_test.go`?**
  _High betweenness centrality (0.014) - this node is a cross-community bridge._
- **Are the 77 inferred relationships involving `testDeps()` (e.g. with `TestHubBroadcastDeliversToClients()` and `TestHubBroadcastDropsSlowClient()`) actually correct?**
  _`testDeps()` has 77 INFERRED edges - model-reasoned connections that need verification._
- **Are the 76 inferred relationships involving `New()` (e.g. with `TestWikiChangedFrameReachesSocket()` and `TestChatCancelBeforeSendIsNoop()`) actually correct?**
  _`New()` has 76 INFERRED edges - model-reasoned connections that need verification._
- **What connects `@playwright/mcp`, `@ant-design/cli`, `github.com/shiv-source/thoth` to the rest of the system?**
  _408 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Deps` be split into smaller, more focused modules?**
  _Cohesion score 0.05401234567901234 - nodes in this community are weakly interconnected._