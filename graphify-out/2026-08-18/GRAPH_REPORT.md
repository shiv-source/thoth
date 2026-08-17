# Graph Report - thoth  (2026-08-18)

## Corpus Check
- 221 files · ~99,696 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1492 nodes · 3311 edges · 96 communities (77 shown, 19 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 443 edges (avg confidence: 0.82)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `835d23d2`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- testDeps
- Deps
- renderWithStore
- Open
- WriterFunc
- Run
- Store
- index.ts
- dependencies
- client.ts
- devDependencies
- package.json
- DashboardView.tsx
- New
- setup-go-web composite action
- healthSlice.test.ts
- Hub
- compilerOptions
- App.tsx
- Components (web/src/components)
- newRootCmd
- Sidebar.tsx
- PersistentClient
- ChatSocket
- react
- Toolchain versions (go.mod / package.json authoritative)
- compilerOptions
- CLAUDE.md - Thoth repository rulebook
- runServe
- Workflows
- Tree.tsx
- Development - toolchain, gates, CI
- Components - Go package deep dive
- New
- CLIClient
- Indexing and search - FTS5 and the file watcher
- Client
- context.Context
- newTestEcho
- Go packages (internal/* + cmd/thoth)
- Global Constraints
- testing.T
- plugins
- API - REST endpoints and WebSocket chat protocol
- Architecture - two layers, one binary
- startupBanner
- ParseLine
- Git workflow — contribution procedures & expectations
- ResizeObserverStub
- playwright
- web workspace package
- React frontend (web/src) — procedures & expertise
- tsconfig.json
- pre-commit
- fake_claude.sh
- report job (summary + gate)
- github.com/shiv-source/thoth
- Index
- Open
- Redux store (web/src/store)
- Thoth Project Skills Suite — Design
- web/package.json
- useView.ts
- SettingsView.test.tsx
- github/client_test.go
- Quality gates — how this repo verifies work
- ParseNote
- cli/doctor_test.go
- New
- Frontend patterns — the cross-cutting conventions
- Scaffold
- useConversationRoute.ts
- The claude blast wall (internal/claude)
- Persistence — thoth.db, migrations, index
- Hooks (web/src/hooks)
- Labels — the three-tier GitHub label set
- newLoggingServer
- ChartStub
- ExpandHome
- eslint
- TestGitSetupReportsSanitizedFailure
- react-chartjs-2
- react-dom
- remark-gfm
- @types/react
- typescript
- vite
- vitest
- requestLog
- README.md
- setup.sh
- @testing-library/jest-dom

## God Nodes (most connected - your core abstractions)
1. `testDeps()` - 68 edges
2. `New()` - 63 edges
3. `Components (web/src/components)` - 44 edges
4. `Deps` - 32 edges
5. `react` - 32 edges
6. `Run()` - 31 edges
7. `WriterFunc` - 30 edges
8. `Open()` - 30 edges
9. `Open()` - 29 edges
10. `newServer()` - 28 edges

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
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/chatSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/connectionSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/conversationsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/healthSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/notificationsSlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/searchHistorySlice.ts -> web/src/store/index.ts`
- 2-file cycle: `web/src/store/index.ts -> web/src/store/slices/settingsSlice.ts -> web/src/store/index.ts`

## Hyperedges (group relationships)
- **The four app-layer components of the single binary** — docs_architecture_app_layer, docs_components_api_pkg, docs_components_claude_pkg, docs_components_index_pkg [EXTRACTED 1.00]
- **Cross-compile build matrix (linux/darwin/windows)** — github_workflows_ci_build_linux, github_workflows_ci_build_darwin, github_workflows_ci_build_windows [EXTRACTED 1.00]
- **Single required CI gate pattern** — github_workflows_ci_pr_final_gate, github_workflows_ci_final_gate, github_workflows_final_gate_report, github_workflows_final_gate_single_check [EXTRACTED 1.00]
- **The five CI quality gates every commit must pass** — docs_development_quality_gates, docs_development_gate_vet, docs_development_gate_race, docs_development_gate_coverage, docs_development_gate_crosscompile, docs_development_gate_frontend [EXTRACTED 1.00]
- **Shared quality gates (backend + frontend)** — github_workflows_quality_backend_test, github_workflows_quality_backend_lint, github_workflows_quality_frontend_test, github_workflows_quality_frontend_lint, github_workflows_quality_frontend_typecheck [EXTRACTED 1.00]
- **The rulebook-driven wiki filing system** — internal_wiki_templates_claude_claude, internal_wiki_templates_claude_save_protocol, docs_knowledge_base_wiki_layout, docs_knowledge_base_frontmatter, docs_architecture_knowledge_layer [INFERRED 0.85]

## Communities (96 total, 19 thin omitted)

### Community 0 - "testDeps"
Cohesion: 0.21
Nodes (29): github.com/gorilla/websocket.Conn, net/http.Handler, readMsg(), TestChatCancelBeforeSendIsNoop(), TestChatCancelStopsInFlightTurn(), TestChatForwardsThinkingFrames(), TestChatHubCancellationEndsTurns(), TestChatNewChatCancelsBusyTurn() (+21 more)

### Community 1 - "Deps"
Cohesion: 0.06
Nodes (49): claudeState, Deps, githubIdentity, healthResponse, settingsDTO, wikiState, Option, createConversation() (+41 more)

### Community 2 - "renderWithStore"
Cohesion: 0.08
Nodes (23): SearchResult, mocks, renderPanel(), mocks, renderBlock(), conversations, mocks, renderDashboard() (+15 more)

### Community 3 - "Open"
Cohesion: 0.12
Nodes (31): TestApply(), TestApplyClosedIndexLogsAndContinues(), TestApplyPathOutsideRoot(), TestApplyUnreadablePath(), TestWatchErrorOnMissingRoot(), TestWatchReturnsOnCancel(), Open(), discardLog() (+23 more)

### Community 4 - "WriterFunc"
Cohesion: 0.12
Nodes (50): Option, StartOption, New(), TestDirProviderOverridesStaticDir(), TestFakeClient(), TestFakeClientPropagatesError(), TestFakeClientPropagatesWriterError(), TestFakeClientRecordsResume() (+42 more)

### Community 5 - "Run"
Cohesion: 0.11
Nodes (46): doctorRunner, net.Listener, failed(), fileExists(), newDoctorCmd(), resolveThothDir(), runDoctor(), checkAPI() (+38 more)

### Community 6 - "Store"
Cohesion: 0.06
Nodes (23): Repo, database/sql.DB, time.Time, Auth, OpenRepo(), Repo, openTestRepo(), saved() (+15 more)

### Community 7 - "index.ts"
Cohesion: 0.12
Nodes (29): renderChatHook(), toolLabel(), useChat(), AppDispatch, AppStore, makeStore(), chatSlice, ChatState (+21 more)

### Community 8 - "dependencies"
Cohesion: 0.07
Nodes (27): axios, chart.js, @fontsource-variable/fraunces, lucide-react, @radix-ui/react-tooltip, react, react-markdown, react-redux (+19 more)

### Community 9 - "client.ts"
Cohesion: 0.09
Nodes (26): api, DoctorCheck, GitHubIdentity, GitHubRepo, http, Message, ModelOption, Note (+18 more)

### Community 10 - "devDependencies"
Cohesion: 0.08
Nodes (25): eslint-config-prettier, @eslint/js, globals, jsdom, oxlint, @tailwindcss/typography, @testing-library/react, @testing-library/user-event (+17 more)

### Community 11 - "package.json"
Cohesion: 0.06
Nodes (35): husky, lint-staged, author, bugs, url, description, devDependencies, husky (+27 more)

### Community 12 - "DashboardView.tsx"
Cohesion: 0.14
Nodes (20): ActivityChart(), Card(), ChatActivityChart(), mockActivity, mockChatActivity, mockInbox, mockMeetings, mockNotesByFolder (+12 more)

### Community 13 - "New"
Cohesion: 0.18
Nodes (20): net/http.Request, net/http.ResponseWriter, TestConversationsStoreError(), TestCreateConversationRejectsEmptyTitle(), TestGetConversationFound(), TestGetConversationNotFound(), TestModelsEndpoint(), New() (+12 more)

### Community 14 - "setup-go-web composite action"
Cohesion: 0.11
Nodes (26): golangci-lint v2 config, Frontend embed build (make web), setup-go-web composite action, Frozen-lockfile install, setup-web composite action, Pull request template, build-darwin job, build-linux job (+18 more)

### Community 15 - "healthSlice.test.ts"
Cohesion: 0.16
Nodes (15): Health, App(), Problem, problemsFromHealth(), SetupScreen(), store, fetchHealth, healthSlice (+7 more)

### Community 16 - "Hub"
Cohesion: 0.17
Nodes (12): clientMsg, Hub, serverMsg, turn, context.CancelFunc, strings.Builder, allowLocalOrigin(), echo.Context (+4 more)

### Community 17 - "compilerOptions"
Cohesion: 0.08
Nodes (24): DOM, src, vite/client, compilerOptions, allowArbitraryExtensions, allowImportingTsExtensions, jsx, lib (+16 more)

### Community 18 - "App.tsx"
Cohesion: 0.12
Nodes (25): ChatPanel(), createSocket(), Composer(), EmptyState(), IconButton(), NotificationPanel(), NOTIFICATION_ICONS, NotificationIcon() (+17 more)

### Community 19 - "Components (web/src/components)"
Cohesion: 0.04
Nodes (44): ActivityChart, App shell & navigation, Card, Charts (Chart.js), chartSetup.ts, Chat, ChatActivityChart, ChatPanel (+36 more)

### Community 20 - "newRootCmd"
Cohesion: 0.15
Nodes (17): main(), github.com/spf13/cobra.Command, TestInitCommandErrorOnUnwritableTarget(), TestInitCommandExpandsTildeInTarget(), TestInitCommandTooManyArgs(), TestInitCommandUsesDefaultPath(), Execute(), newRootCmd() (+9 more)

### Community 21 - "Sidebar.tsx"
Cohesion: 0.11
Nodes (22): Conversation, ChatsList(), groupByDay(), relativeDate(), Sidebar(), conversations, healthy, mocks (+14 more)

### Community 22 - "PersistentClient"
Cohesion: 0.18
Nodes (12): PersistentClient, poolEntry, proc, startConfig, CLIClient, bufio.Writer, io.ReadCloser, os.File (+4 more)

### Community 23 - "ChatSocket"
Cohesion: 0.11
Nodes (3): freshSocket(), FakeWS, ChatSocket

### Community 24 - "react"
Cohesion: 0.10
Nodes (21): react, cache, CodeBlock(), highlight(), CopyButton(), renderCopy(), components, Markdown() (+13 more)

### Community 25 - "Toolchain versions (go.mod / package.json authoritative)"
Cohesion: 0.13
Nodes (20): Chart.js, Cobra 1.10 - Go CLI framework, Echo 4.15 - Go web framework, fsnotify 1.10 - file watcher, gorilla/websocket 1.5, React 19.2, Redux Toolkit, SQLite + FTS5 (modernc.org/sqlite 1.56) (+12 more)

### Community 26 - "compilerOptions"
Cohesion: 0.10
Nodes (19): node, vite.config.ts, compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection (+11 more)

### Community 27 - "CLAUDE.md - Thoth repository rulebook"
Cohesion: 0.16
Nodes (19): Blast wall - all Claude CLI flags live only in client.go, Branch workflow - never commit to main directly, CLAUDE.md - Thoth repository rulebook, Memory and resource safety rules (no leaks), Code rules: DRY, SOLID, KISS, YAGNI, small functions, Runtime data: ~/.thoth (thoth.db + wiki/), Claude Code CLI - driven headless per conversation, Two interfaces, one contract (dashboard and terminal) (+11 more)

### Community 28 - "runServe"
Cohesion: 0.19
Nodes (14): rootHolder, log/slog.Logger, sync.RWMutex, isTerminal(), echo.Echo, newRootHolder(), onSettingsSaved(), openIndex() (+6 more)

### Community 29 - "Workflows"
Cohesion: 0.12
Nodes (15): 1. Add a REST endpoint, 2. Extend the WS protocol, 3. Add a store migration, 4. Change claude CLI flags (BLAST WALL), 5. Add a settings key, 6. Extend the wiki contract, 7. Bump a dependency, 8. Add a doctor install check (+7 more)

### Community 30 - "Tree.tsx"
Cohesion: 0.20
Nodes (6): Node, sample, Tree(), TreeProps, TreeRow, TreeRowProps

### Community 31 - "Development - toolchain, gates, CI"
Cohesion: 0.17
Nodes (13): CI-enforced quality gates (make check), CONTRIBUTING.md - contribution workflow, Additive migrations rule (never edit an applied migration), PR and review workflow (conventional commits, squash-merge), CI workflows (quality.yml, ci.yml, ci-pr.yml, final-gate.yml), Development - toolchain, gates, CI, Gate: 80 percent coverage floor on internal and cmd, Gate: five cross-compile targets (+5 more)

### Community 32 - "Components - Go package deep dive"
Cohesion: 0.28
Nodes (13): CLI - serve, init, version, doctor commands, thoth doctor - six install checks, thoth doctor --fix repair mode, Components - Go package deep dive, internal/doctor - shared install checks, internal/github - identity and git sync, internal/settings - settings KV repo, Documentation hub (index.md) (+5 more)

### Community 33 - "New"
Cohesion: 0.24
Nodes (10): ensureWiki(), Wiki, New(), TestWikiNotExists(), TestWikiReadAndTree(), TestWikiReadMissingNote(), TestWikiTreeErrorOnMissingRoot(), TestWikiTreeErrorOnUnreadableSubdir() (+2 more)

### Community 34 - "CLIClient"
Cohesion: 0.11
Nodes (10): stderrTail, os/exec.Cmd, os.Process, CLIClient, CLIClient, killProcess(), setProcessGroup(), CLIClient (+2 more)

### Community 35 - "Indexing and search - FTS5 and the file watcher"
Cohesion: 0.25
Nodes (11): Project invariants (files as source of truth, percent-w errors, no globals), App layer - single Go binary, Data contract: files are the source of truth, thoth.db is derived, thoth serve command, internal/api - the Echo server, internal/index - search and sync, useSearch - debounced, supersede-guarded search, bm25 ranking with title weighted 8x (+3 more)

### Community 36 - "Client"
Cohesion: 0.29
Nodes (6): getResult, Profile, Repository, net/http.Client, Client, primaryEmail()

### Community 37 - "context.Context"
Cohesion: 0.21
Nodes (8): ctxAwareFake, hangClient, staleLockClient, Call, context.Context, sync.Mutex, EventWriter, FakeClient

### Community 38 - "newTestEcho"
Cohesion: 0.29
Nodes (8): echo.Echo, Register(), echo.Echo, newTestEcho(), TestRegisterFallsBackToIndexForMissingPaths(), TestRegisterReturns404ForUnknownAPIPaths(), TestRegisterServesExistingAsset(), TestRegisterServesIndexAtRoot()

### Community 39 - "Go packages (internal/* + cmd/thoth)"
Cohesion: 0.13
Nodes (14): cmd/thoth, Go packages (internal/* + cmd/thoth), internal/api, internal/assets, internal/claude — the blast wall, internal/cli, internal/config, internal/doctor (+6 more)

### Community 40 - "Global Constraints"
Cohesion: 0.13
Nodes (14): Global Constraints, Task 10: react/references/patterns.md — the cross-cutting conventions, Task 11: CLAUDE.md — the one pointer line, Task 12: Full-suite verification + PR, Task 1: go/SKILL.md — the backend procedure skill, Task 2: go/references/packages.md — the package index, Task 3: go/references/claude-blast-wall.md — the version-sensitive zone, Task 4: go/references/persistence.md — thoth.db and migrations (+6 more)

### Community 41 - "testing.T"
Cohesion: 0.20
Nodes (19): testing.T, TestNoteEndpoint(), TestNoteEndpointMissingNote(), TestNoteEndpointRequiresPath(), TestSearchEndpoint(), TestSearchEndpointDefaultsBadLimit(), TestSearchEndpointIndexError(), TestSearchEndpointRequiresQuery() (+11 more)

### Community 42 - "plugins"
Cohesion: 0.22
Nodes (8): oxc, typescript, warn, plugins, rules, react/only-export-components, react/rules-of-hooks, $schema

### Community 43 - "API - REST endpoints and WebSocket chat protocol"
Cohesion: 0.39
Nodes (8): API - REST endpoints and WebSocket chat protocol, Resume with 500-message replay ring, Per-conversation Claude CLI session pool, Supersede-on-send and cancel chat semantics, WebSocket chat protocol (/ws), internal/store - conversations and messages, conversations table (claude_session_id), messages table (chat transcript)

### Community 44 - "Architecture - two layers, one binary"
Cohesion: 0.43
Nodes (8): Architecture - two layers, one binary, Knowledge layer - plain markdown wiki you own, internal/wiki - the file contract, Frontmatter contract (title required), Knowledge base - the wiki directory, Wiki folder layout (8 folders), Wiki rulebook template (CLAUDE.md in wiki root), The save protocol (folder map, frontmatter, confirm)

### Community 45 - "startupBanner"
Cohesion: 0.53
Nodes (5): startupBanner(), TestStartupBannerAddsColorOnlyWhenAsked(), TestStartupBannerContainsTheFacts(), TestStartupBannerFormatsIPv6Hosts(), TestStartupBannerShowsTheBigWordmark()

### Community 46 - "ParseLine"
Cohesion: 0.16
Nodes (17): Event, EventType, rawBlock, rawLine, rawMsg, encoding/json.RawMessage, ParseLine(), TestParseLineAssistantText() (+9 more)

### Community 47 - "Git workflow — contribution procedures & expectations"
Cohesion: 0.14
Nodes (13): 1. Start a change (branch), 2. Commit, 3. Open a PR, 4. Label issues and PRs, 5. Design doc first (large or cross-package changes), 6. Merge is human-only — squash by default, Canonical docs, Git workflow — contribution procedures & expectations (+5 more)

### Community 49 - "playwright"
Cohesion: 0.50
Nodes (3): npx, playwright, @playwright/mcp

### Community 50 - "web workspace package"
Cohesion: 0.50
Nodes (4): web workspace package, pnpm workspace root, Thoth web entry (index.html), React app entry (src/main.tsx)

### Community 51 - "React frontend (web/src) — procedures & expertise"
Cohesion: 0.14
Nodes (13): 1. Add a component, 2. Add a Redux slice, 3. Add a hook, 4. Wire an API call, 5. Test a component/slice, 6. Touch the WS client, Canonical docs, Gotchas (+5 more)

### Community 61 - "Index"
Cohesion: 0.24
Nodes (7): dbLike, Note, Result, del(), Index, upsert(), Index

### Community 62 - "Open"
Cohesion: 0.24
Nodes (12): healthyThothDir(), TestDoctorEndpointHealthy(), Open(), TestClosedStoreErrors(), TestConversationRoundTrip(), TestConversationSessionIDRoundTrip(), TestDeleteConversation(), TestEnsureMetadataSeedsOnce() (+4 more)

### Community 63 - "Redux store (web/src/store)"
Cohesion: 0.17
Nodes (11): chat, connection, conversations, health, hooks.ts, index.ts, notifications, Redux store (web/src/store) (+3 more)

### Community 64 - "Thoth Project Skills Suite — Design"
Cohesion: 0.17
Nodes (11): Approach: C — hybrid, Decisions log, Inventory, Maintenance, Out of scope, Purpose, Reference file anatomy, SKILL.md anatomy (+3 more)

### Community 65 - "web/package.json"
Cohesion: 0.17
Nodes (11): name, private, scripts, build, dev, lint, preview, test (+3 more)

### Community 66 - "useView.ts"
Cohesion: 0.22
Nodes (11): NavRail(), VIEWS, navigateNote(), navigateSegment(), navigateView(), setPath(), useView(), View (+3 more)

### Community 67 - "SettingsView.test.tsx"
Cohesion: 0.17
Nodes (5): connected, emptyGitHub, mocks, renderSettings(), settings

### Community 68 - "github/client_test.go"
Cohesion: 0.21
Nodes (15): profileStub, net/http.HandlerFunc, Client, newStubClient(), TestFetchProfileContextDeadline(), TestFetchProfileEmailsBestEffort(), TestFetchProfileMalformedUserBody(), TestFetchProfileNetworkErrorIsSanitized() (+7 more)

### Community 69 - "Quality gates — how this repo verifies work"
Cohesion: 0.20
Nodes (9): Commit hygiene, Concurrency, Coverage, Cross-compile, Dependency bumps, Lint, make check — everything CI enforces, locally, Quality gates — how this repo verifies work (+1 more)

### Community 70 - "ParseNote"
Cohesion: 0.33
Nodes (8): ParseNote(), TestParseNote(), TestParseNoteClosedAtEOF(), TestParseNoteRejectsBadYAML(), TestParseNoteRejectsMissingFrontmatter(), TestParseNoteRejectsMissingTitle(), TestParseNoteRejectsUnclosedFrontmatter(), NoteMeta

### Community 71 - "cli/doctor_test.go"
Cohesion: 0.36
Nodes (14): executeDoctor(), healthyEnv(), serveThothOnFixedPort(), TestDoctorDetectsBusyPort(), TestDoctorDetectsMissingClaude(), TestDoctorDetectsMissingIndexTables(), TestDoctorDetectsNonWALDatabase(), TestDoctorFixesMissingDefaultWiki() (+6 more)

### Community 72 - "New"
Cohesion: 0.29
Nodes (13): net/http/httptest.ResponseRecorder, net/http/httptest.Server, doJSON(), githubStub(), TestConnectGitHub(), TestConnectGitHubRejectedToken(), TestConnectGitHubRequiresToken(), TestConnectGitHubUpstreamError() (+5 more)

### Community 73 - "Frontend patterns — the cross-cutting conventions"
Cohesion: 0.22
Nodes (8): Design tokens, Frontend patterns — the cross-cutting conventions, Package discipline, Routing, State placement, Test doubles (web/src/test), The API boundary (zod), The WS protocol (ChatSocket)

### Community 74 - "Scaffold"
Cohesion: 0.29
Nodes (7): Folders(), Scaffold(), TestScaffoldCreatesSkeletonAndRulebook(), TestScaffoldErrorWhenParentIsFile(), TestScaffoldIsIdempotent(), TestScaffoldKeepsExistingCLAUDE(), Rulebook()

### Community 75 - "useConversationRoute.ts"
Cohesion: 0.31
Nodes (6): chatIdFromPath(), ConversationRouteOptions, getConversation, renderRoute(), useConversationRoute(), ChatMessage

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
Cohesion: 0.40
Nodes (4): Areas (package-aligned), Labels — the three-tier GitHub label set, Priority (issues only), Types (mirror the conventional-commit prefixes)

### Community 80 - "newLoggingServer"
Cohesion: 0.43
Nodes (6): bytes.Buffer, echo.Echo, newLoggingServer(), TestRequestLogsAPIPaths(), TestRequestLogsFailureWithErr(), TestRequestLogSkipsNonAPIPaths()

### Community 82 - "ExpandHome"
Cohesion: 0.33
Nodes (4): newInitCmd(), ExpandHome(), TestExpandHome(), TestExpandHomeBareTilde()

### Community 84 - "TestGitSetupReportsSanitizedFailure"
Cohesion: 0.60
Nodes (4): TestGitSetupReportsSanitizedFailure(), TestGitSetupRequiresURL(), TestGitSetupRunsAgainstWiki(), writeFakeGit()

## Knowledge Gaps
- **369 isolated node(s):** `npx`, `@playwright/mcp`, `github.com/shiv-source/thoth`, `clientMsg`, `CLIClient` (+364 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **19 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Deps` connect `Deps` to `testDeps`, `New`, `context.Context`, `Store`, `New`, `Hub`, `runServe`, `Index`?**
  _High betweenness centrality (0.031) - this node is a cross-community bridge._
- **Why does `testDeps()` connect `testDeps` to `Deps`, `New`, `Open`, `Store`, `cli/doctor_test.go`, `New`, `testing.T`, `New`, `newLoggingServer`, `TestGitSetupReportsSanitizedFailure`, `Open`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **Why does `newServer()` connect `Deps` to `testDeps`, `newTestEcho`, `New`, `Hub`, `requestLog`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **Are the 59 inferred relationships involving `testDeps()` (e.g. with `TestChatCancelBeforeSendIsNoop()` and `TestChatCancelStopsInFlightTurn()`) actually correct?**
  _`testDeps()` has 59 INFERRED edges - model-reasoned connections that need verification._
- **Are the 58 inferred relationships involving `New()` (e.g. with `TestChatCancelBeforeSendIsNoop()` and `TestChatCancelStopsInFlightTurn()`) actually correct?**
  _`New()` has 58 INFERRED edges - model-reasoned connections that need verification._
- **What connects `npx`, `@playwright/mcp`, `github.com/shiv-source/thoth` to the rest of the system?**
  _369 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Deps` be split into smaller, more focused modules?**
  _Cohesion score 0.06394230769230769 - nodes in this community are weakly interconnected._