# OpenCode TUI Session Dashboard

## TL;DR

> **Quick Summary**: Build a Go TUI dashboard (Bubbletea/Charm) that reads the OpenCode SQLite database and shows a global overview of all sessions across all projects, with attention signals highlighting which sessions need user interaction.
> 
> **Deliverables**:
> - Single Go binary `opencode-dashboard`
> - Split-pane TUI: project-grouped session list (left), session detail (right)
> - 5 attention signals: needs response, active now, has errors, stale work, pending todos
> - Filter bar with presets (Needs Attention, All Active, Archived) + free-text search
> - Auto-refresh every 30s, vim+arrow navigation, session launch via Enter
> 
> **Estimated Effort**: Medium-Large
> **Parallel Execution**: YES - 5 waves
> **Critical Path**: Task 1 -> Task 5 -> Task 9 -> Task 16 -> Task 18 -> Final

---

## Context

### Original Request
Build a TUI dashboard for getting a global overview of all OpenCode sessions across all projects with their current status, so the user can identify which sessions need attention.

### Interview Summary
**Key Discussions**:
- **Attention signals**: All 5 types selected - waiting for input, error/failed, stale/abandoned, recently active, pending todos
- **Layout**: Split pane with project-grouped session list (left) and full session detail (right)
- **Session rows**: Attention indicator, title/slug, last activity time, todo progress
- **Filtering**: Filter bar with presets + free-text search by title
- **Refresh**: Auto-refresh every 30s polling SQLite
- **Navigation**: Vim keys (j/k, h/l) + arrow keys, Enter to expand/launch
- **V1 scope**: Read-only + launch. No session management (delete/archive/tag).
- **Tests**: TDD approach for data layer, tmux-based QA for TUI

**Research Findings**:
- **Database**: `~/.local/share/opencode/opencode.db` (SQLite, ~1.9GB)
- **Scale**: 2,776 sessions, 63,736 messages, 3,479 todos, 24 projects
- **Critical discovery**: 81% of sessions (2,254) are child/subagent sessions (`parent_id IS NOT NULL`). Only 531 are root sessions.
- **"Waiting for input" caveat**: 99.2% of root sessions have last_role='assistant'. Signal only meaningful with a recency window.
- **"Global" project**: 713 sessions (26%) have `project_id='global'`, `worktree='/'` - not tied to a specific directory.
- **Error detection**: 1,348 error parts across 257K total parts. `json_extract` scan takes ~1.5s.
- **Core query performance**: 64ms for root sessions with project join - well within 30s refresh budget.
- **Session launch**: `opencode --session <session_id>` confirmed working.
- **Go environment**: Go 1.24.3 installed, CGO enabled, `go-sqlite3` pattern exists in whatsapp-bridge project.
- **Pre-computed stats**: `session.summary_additions/deletions/files` available for free.
- **Zero archived sessions currently**: `time_archived` filter preset will be empty but still useful for future.

### Metis Review
**Identified Gaps** (addressed):
- **Child session noise**: Defaulting to root sessions only (`parent_id IS NULL`), with child count badge on parents. [DECISION NEEDED: see below]
- **Recency window for "waiting for input"**: Applied 24h window. Sessions updated >24h ago with last_role='assistant' are not "waiting."
- **"Global" project handling**: Displayed as "Global" group at bottom of project list.
- **Error query caching**: Cache error counts between refreshes; only re-scan sessions updated since last check.
- **SQLite locking**: Open with `?mode=ro` to prevent write locks conflicting with OpenCode.
- **User message JSON size**: Never load full `message.data` blobs - use `json_extract()` for specific fields only.
- **DB schema drift**: Validate expected tables/columns on startup; graceful error on mismatch.
- **CGO vs pure Go**: Using `modernc.org/sqlite` (pure Go) for single-binary portability. [DECISION NEEDED: see below]

---

## Work Objectives

### Core Objective
Build a single-binary Go TUI that reads the OpenCode SQLite database in read-only mode and presents a split-pane dashboard showing all root sessions grouped by project, with attention signals and filtering.

### Concrete Deliverables
- `opencode-dashboard` binary (Go, single file, no runtime deps)
- Split-pane TUI with project tree (left) and session detail (right)
- 5 attention signal classifications with priority ordering
- Filter bar with 3 presets + free-text search
- 30s auto-refresh
- Session launch via `opencode --session <id>`

### Definition of Done
- [ ] `go build ./cmd/opencode-dashboard` produces working binary
- [ ] `go test ./...` passes all tests (data layer TDD + integration)
- [ ] Binary launches, shows sessions grouped by project, attention icons visible
- [ ] j/k navigates sessions, h/l switches panes, Enter launches session
- [ ] Filter presets cycle with Tab, free-text search with /
- [ ] Auto-refresh updates data every 30s without losing scroll position

### Must Have
- Root session filtering (parent_id IS NULL) by default
- All 5 attention signals with correct priority ordering
- Read-only SQLite access (?mode=ro)
- Split-pane layout with project grouping
- Session launch capability
- Vim + arrow key navigation
- Filter presets: Needs Attention, All Active, Archived
- TDD for data/business logic layer
- Auto-refresh every 30s

### Must NOT Have (Guardrails)
- **No write operations** to the SQLite database - read-only, always
- **No session management** (delete, archive, tag) - V1 is read + launch only
- **No message content display** - V1 shows metadata only (role, agent, model, timestamp)
- **No workspace/branch grouping** - workspace table is empty, defer to V2
- **No color themes or style config** - single hardcoded color scheme
- **No cobra/viper CLI framework** - single binary, no subcommands, optional `--db-path` flag only
- **No cost/token tracking** - not in scope for V1
- **No agent performance breakdowns** - not in scope for V1
- **No session comparison views** - not in scope for V1
- **No configurable thresholds** for attention signals - hardcoded constants (24h, 5min, 7d)
- **No loading of full message.data or part.data blobs** - json_extract for specific fields only
- **Data layer must NOT import Bubbletea** - pure Go with stdlib + SQLite only
- **TUI must NOT add "open in editor"** or any launch target other than `opencode --session`

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: NO (greenfield)
- **Automated tests**: TDD (RED-GREEN-REFACTOR)
- **Framework**: Go standard `testing` package + `github.com/stretchr/testify` for assertions
- **Test fixture**: Seeded SQLite database in `testdata/` built via `TestMain` helper
- **Benchmarks**: `testing.B` against real DB path, skipped with `testing.Short()`
- **TUI testing**: Agent-executed QA via tmux (launch binary, send keystrokes, assert screen content)

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Data layer**: `go test ./internal/...` with seeded fixture DB
- **Business logic**: `go test ./internal/attention/...` and `./internal/filter/...`
- **TUI**: `interactive_bash` (tmux) - launch binary, send keystrokes, capture pane, assert text
- **Integration**: Full binary launch with real DB, navigate, verify data appears

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation - all parallel, no deps):
├── Task 1: Project scaffold + go.mod + directory structure [quick]
├── Task 2: Domain types + interfaces [quick]
├── Task 3: Test fixture DB builder [quick]
└── Task 4: SQLite connection layer [quick]

Wave 2 (Data Layer - TDD, parallel, depends on Wave 1):
├── Task 5: Session repository (depends: 2, 3, 4) [unspecified-high]
├── Task 6: Message metadata repository (depends: 2, 3, 4) [unspecified-high]
├── Task 7: Todo repository (depends: 2, 3, 4) [unspecified-high]
└── Task 8: Error detection with caching (depends: 2, 3, 4) [unspecified-high]

Wave 3 (Business Logic + TUI Shell - parallel, mixed deps):
├── Task 9: Attention classifier (depends: 2) [deep]
├── Task 10: Filter/search engine (depends: 2) [unspecified-high]
├── Task 11: Data aggregator - view model (depends: 5, 6, 7, 8) [unspecified-high]
├── Task 12: Bubbletea app skeleton (depends: 1) [quick]
└── Task 13: Left pane - project tree component (depends: 2, 12) [visual-engineering]

Wave 4 (TUI Components + Integration - depends on Wave 3):
├── Task 14: Right pane - session detail view (depends: 2, 12) [visual-engineering]
├── Task 15: Split layout + pane navigation (depends: 13, 14) [visual-engineering]
├── Task 16: Wire data layer into TUI (depends: 11, 15) [deep]
├── Task 17: Filter bar UI (depends: 10, 16) [visual-engineering]
└── Task 18: Auto-refresh + session launcher (depends: 16) [unspecified-high]

Wave FINAL (After ALL tasks - 4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: T1 -> T4 -> T5 -> T11 -> T16 -> T18 -> F1-F4 -> user okay
Parallel Speedup: ~65% faster than sequential
Max Concurrent: 4 (Waves 2 and 3)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | - | 4, 12 | 1 |
| 2 | - | 5, 6, 7, 8, 9, 10, 13, 14 | 1 |
| 3 | - | 5, 6, 7, 8 | 1 |
| 4 | 1 | 5, 6, 7, 8 | 1 |
| 5 | 2, 3, 4 | 11 | 2 |
| 6 | 2, 3, 4 | 11 | 2 |
| 7 | 2, 3, 4 | 11 | 2 |
| 8 | 2, 3, 4 | 11 | 2 |
| 9 | 2 | 11 | 3 |
| 10 | 2 | 17 | 3 |
| 11 | 5, 6, 7, 8, 9 | 16 | 3 |
| 12 | 1 | 13, 14 | 3 |
| 13 | 2, 12 | 15 | 3 |
| 14 | 2, 12 | 15 | 4 |
| 15 | 13, 14 | 16 | 4 |
| 16 | 11, 15 | 17, 18 | 4 |
| 17 | 10, 16 | - | 4 |
| 18 | 16 | - | 4 |

### Agent Dispatch Summary

- **Wave 1**: **4 tasks** - T1 `quick`, T2 `quick`, T3 `quick`, T4 `quick`
- **Wave 2**: **4 tasks** - T5-T8 `unspecified-high`
- **Wave 3**: **5 tasks** - T9 `deep`, T10 `unspecified-high`, T11 `unspecified-high`, T12 `quick`, T13 `visual-engineering`
- **Wave 4**: **5 tasks** - T14 `visual-engineering`, T15 `visual-engineering`, T16 `deep`, T17 `visual-engineering`, T18 `unspecified-high`
- **FINAL**: **4 tasks** - F1 `oracle`, F2 `unspecified-high`, F3 `unspecified-high`, F4 `deep`

---

## TODOs

- [x] 1. Project Scaffold + Go Module Init

  **What to do**:
  - Initialize git repository: `git init` (workspace has no .git/ - required for atomic commit strategy and F4 verification)
  - Add `.gitignore` with: binary outputs, `.sisyphus/evidence/`, OS files (.DS_Store)
  - Initialize Go module: `go mod init github.com/joeyparis/opencode-dashboard`
  - Create directory structure:
    ```
    cmd/opencode-dashboard/main.go
    internal/domain/
    internal/store/
    internal/attention/
    internal/filter/
    internal/app/
    internal/ui/
    internal/launcher/
    internal/testutil/
    ```
  - `main.go` should parse an optional `--db-path` flag (default: `~/.local/share/opencode/opencode.db`), print "opencode-dashboard" and exit cleanly
  - Add initial dependencies: `modernc.org/sqlite`, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles`, `github.com/stretchr/testify`
  - Verify `go build ./cmd/opencode-dashboard` succeeds

  **Must NOT do**:
  - No cobra/viper CLI framework - use `flag` package only
  - No application logic - just scaffold

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
    - Simple scaffolding task, no domain-specific skills needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: Tasks 4, 12
  - **Blocked By**: None

  **References**:
  **Pattern References**:
  - `/Users/joey/Sites/pcopy-server/` - `cmd/` directory structure pattern to follow
  - `/Users/joey/Sites/whatsapp-mcp/whatsapp-bridge/` - Go module init and `go-sqlite3` dependency pattern (but use `modernc.org/sqlite` instead)

  **External References**:
  - `https://github.com/nicholasgasior/gopher-bubbletea` - Example Bubbletea project structure
  - `https://pkg.go.dev/modernc.org/sqlite` - Pure Go SQLite driver

  **WHY Each Reference Matters**:
  - pcopy-server shows Joey's preferred `cmd/` layout for Go binaries
  - whatsapp-bridge shows how DB driver dependencies are structured in go.mod

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/opencode-dashboard` exits 0, produces binary
  - [ ] `./opencode-dashboard` prints "opencode-dashboard" and exits
  - [ ] `./opencode-dashboard --db-path /tmp/test.db` accepts the flag without error
  - [ ] All directories under `internal/` exist

  **QA Scenarios**:
  ```
  Scenario: Binary builds and runs
    Tool: Bash
    Preconditions: go.mod exists, dependencies downloaded
    Steps:
      1. Run `go build -o /tmp/ocd-test ./cmd/opencode-dashboard`
      2. Run `/tmp/ocd-test` and capture stdout
      3. Assert stdout contains "opencode-dashboard"
      4. Assert exit code is 0
    Expected Result: Binary builds cleanly, prints name, exits 0
    Failure Indicators: Build errors, non-zero exit, no output
    Evidence: .sisyphus/evidence/task-1-build-and-run.txt

  Scenario: Flag parsing works
    Tool: Bash
    Preconditions: Binary built
    Steps:
      1. Run `/tmp/ocd-test --db-path /tmp/fake.db`
      2. Assert exit code is 0
      3. Run `/tmp/ocd-test --help` and capture stdout
      4. Assert stdout contains "db-path"
    Expected Result: Flag accepted, help text shows flag
    Failure Indicators: Unknown flag error, missing from help
    Evidence: .sisyphus/evidence/task-1-flag-parsing.txt
  ```

  **Commit**: YES
  - Message: `feat(scaffold): project init with git, go.mod, and directory structure`
  - Files: `.gitignore, go.mod, go.sum, cmd/opencode-dashboard/main.go, internal/*/`
  - Pre-commit: `go build ./...`

- [x] 2. Domain Types + Store Interfaces

  **What to do**:
  - Create `internal/domain/types.go` with all domain types:
    - `Project` struct: ID, Name (nullable - see DisplayName below), Worktree, VCS, TimeCreated, TimeUpdated
    - `Project.DisplayName() string` method: returns Name if non-empty; otherwise returns basename of Worktree (e.g., `/Users/joey/Sites/alpha` -> `alpha`); special case: if Worktree is `/` or ID is `global`, returns `"Global"`. NOTE: In the real OpenCode DB, project.name is NULL for ALL 24 projects - this fallback is NOT optional, it is the primary display path.
    - `Session` struct: ID, ProjectID (FK only - no project name/worktree here), ParentID (nullable), Slug, Directory, Title, Version, SummaryAdditions, SummaryDeletions, SummaryFiles, TimeCreated, TimeUpdated, TimeArchived (nullable). NOTE: project display fields (Name, Worktree) live ONLY on SessionView and ProjectGroup - populated by the aggregator from ProjectStore.
    - `Todo` struct: SessionID, Content, Status, Priority, Position
    - `MessageMeta` struct: Role, Agent, ModelID, ProviderID, TimeCreated (metadata only - no content)
    - `AttentionSignal` enum/type: NeedsResponse, ActiveNow, HasErrors, StaleWork, PendingTodos, None
    - `AttentionPriority` - ordering: HasErrors > ActiveNow > NeedsResponse > StaleWork > PendingTodos > None
    - `FilterPreset` enum: NeedsAttention, AllActive, Archived
    - `SessionView` struct: combines Session + Project + attention signal + todo stats + todos list + last message meta + error count + child count + message count. Fields:
      - Session (embedded)
      - ProjectName string, ProjectWorktree string
      - AttentionSignal
      - Todos []Todo (full list for detail pane)
      - PendingTodoCount int, TotalTodoCount int
      - LastMessage MessageMeta
      - MessageCount int
      - ErrorCount int
      - ChildCount int
    - `ProjectGroup` struct (ORDERED container for grouped display):
      - Project domain.Project
      - Sessions []SessionView (sorted by attention priority DESC, then time_updated DESC)
      - AttentionCount int (sessions with signal != None)
  - Create `internal/domain/interfaces.go` with store interfaces:
    - `SessionStore`: ListRootSessions, GetSessionByID, GetChildCount
    - `MessageStore`: GetLastMessageMeta, GetMessageCount
    - `TodoStore`: GetTodosBySession, GetPendingTodoCount
    - `ErrorStore`: GetErrorCount, RefreshErrorCache
    - `ProjectStore`: ListProjects
  - All timestamps should be `time.Time` in Go (convert from Unix ms at the store layer)
  - NOTE: `LoadAll` returns `[]ProjectGroup` (ordered slice, NOT a map) - "Global" project is always last

  **Must NOT do**:
  - No implementation - interfaces and types only
  - No database imports
  - No Bubbletea imports

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4)
  - **Blocks**: Tasks 5, 6, 7, 8, 9, 10, 13, 14
  - **Blocked By**: None

  **References**:
  **API/Type References**:
  - OpenCode DB schema (from Metis analysis):
    - `session` table: id TEXT, project_id TEXT, parent_id TEXT, slug TEXT, directory TEXT, title TEXT, version TEXT, summary_additions INT, summary_deletions INT, summary_files INT, time_created INT (unix ms), time_updated INT (unix ms), time_archived INT (unix ms, nullable)
    - `project` table: id TEXT, worktree TEXT, name TEXT, vcs TEXT
    - `todo` table: session_id TEXT, content TEXT, status TEXT, priority TEXT, position INT
    - `message.data` JSON: `{"role":"user|assistant", "agent":"...", "modelID":"...", "providerID":"...", "time":{"created":...}}`
    - `part.data` JSON: `{"type":"tool-invocation", "state":{"status":"error|completed|running"}}`

  **WHY Each Reference Matters**:
  - DB schema maps directly to Go struct fields - executor must match field names and types exactly
  - Message JSON shape determines what MessageMeta can expose without loading full blobs

  **Acceptance Criteria**:
  - [ ] `go build ./internal/domain/...` succeeds
  - [ ] All 5 AttentionSignal values defined with correct priority ordering
  - [ ] SessionView struct has ALL fields: Todos list, MessageCount, ErrorCount, ChildCount, PendingTodoCount, TotalTodoCount, LastMessage, AttentionSignal, ProjectName, ProjectWorktree
  - [ ] ProjectGroup struct preserves ordering (slice, not map)
  - [ ] Store interfaces define all methods needed by aggregator (including ProjectStore)

  **QA Scenarios**:
  ```
  Scenario: Types compile and are usable
    Tool: Bash
    Preconditions: types.go and interfaces.go exist
    Steps:
      1. Run `go build ./internal/domain/...`
      2. Run `go vet ./internal/domain/...`
    Expected Result: Clean build, no vet warnings
    Failure Indicators: Compile error, unused types, vet warnings
    Evidence: .sisyphus/evidence/task-2-types-build.txt

  Scenario: Attention priority ordering is correct
    Tool: Bash
    Preconditions: types.go has AttentionSignal with numeric values
    Steps:
      1. Write a quick test in /tmp that imports domain and asserts HasErrors > ActiveNow > NeedsResponse > StaleWork > PendingTodos > None
      2. Run the test
    Expected Result: All comparisons pass
    Failure Indicators: Any ordering assertion fails
    Evidence: .sisyphus/evidence/task-2-attention-ordering.txt

  Scenario: Project DisplayName fallback works
    Tool: Bash
    Preconditions: types.go has Project.DisplayName() method
    Steps:
      1. Write a quick test that verifies:
         - Project{Name: "MyProject", Worktree: "/foo/bar"}.DisplayName() == "MyProject"
         - Project{Name: "", Worktree: "/Users/joey/Sites/alpha"}.DisplayName() == "alpha"
         - Project{Name: "", Worktree: "/", ID: "global"}.DisplayName() == "Global"
         - Project{Name: "", Worktree: "/"}.DisplayName() == "Global"
      2. Run the test
    Expected Result: All fallback cases correct
    Failure Indicators: Empty string, wrong basename, "Global" not detected
    Evidence: .sisyphus/evidence/task-2-display-name.txt
  ```

  **Commit**: YES
  - Message: `feat(types): domain types and store interfaces`
  - Files: `internal/domain/types.go, internal/domain/interfaces.go`
  - Pre-commit: `go build ./...`

- [x] 3. Test Fixture DB Builder

  **What to do**:
  - Create `internal/testutil/fixture.go` with a `NewTestDB(t *testing.T) *sql.DB` function that:
    - Creates an in-memory SQLite database (or temp file)
    - Creates all required tables matching OpenCode's schema exactly: session, message, part, todo, project
    - Creates all required indexes matching OpenCode's schema
    - Seeds known test data:
      - 3 projects with NULL name (matching real DB behavior): project-alpha (name=NULL, worktree=/Users/joey/Sites/alpha), project-beta (name=NULL, worktree=/Users/joey/Sites/beta), global (name=NULL, worktree=/). DisplayName() will derive: "alpha", "beta", "Global"
      - 8 root sessions across projects (varying attention states):
        - Session with last_role='assistant' updated 1h ago (needs response)
        - Session updated 2min ago (active now)
        - Session with error parts (has errors)
        - Session with pending todos updated 10 days ago (stale)
        - Session with pending todos updated 1 day ago (pending todos)
        - Session with all todos completed (no attention)
        - Archived session (time_archived set)
        - Session in "global" project
      - 4 child sessions (parent_id set) to verify filtering
      - Appropriate message, part, and todo records for each session
    - Returns the *sql.DB connection
  - Create `internal/testutil/fixture_test.go` that verifies the seeded data is correct
  - All timestamps should be set relative to `time.Now()` so tests don't break over time

  **Must NOT do**:
  - No real DB access - in-memory or temp file only
  - No Bubbletea imports

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4)
  - **Blocks**: Tasks 5, 6, 7, 8
  - **Blocked By**: None

  **References**:
  **API/Type References**:
  - Full OpenCode DB schema with CREATE TABLE statements (from Metis research):
    - session: id TEXT PK, project_id TEXT, parent_id TEXT, slug TEXT, directory TEXT, title TEXT, version TEXT, summary_additions INT, summary_deletions INT, summary_files INT, time_created INT, time_updated INT, time_archived INT, workspace_id TEXT, share_url TEXT, revert TEXT, permission TEXT, time_compacting INT, summary_diffs TEXT
    - message: id TEXT PK, session_id TEXT, time_created INT, time_updated INT, data TEXT
    - part: id TEXT PK, message_id TEXT, session_id TEXT, time_created INT, time_updated INT, data TEXT
    - todo: session_id TEXT, content TEXT, status TEXT, priority TEXT, position INT, time_created INT, time_updated INT, PK(session_id, position)
    - project: id TEXT PK, worktree TEXT, vcs TEXT, name TEXT, icon_url TEXT, icon_color TEXT, time_created INT, time_updated INT, time_initialized INT, sandboxes TEXT, commands TEXT
  - Index: message_session_time_created_id_idx ON message(session_id, time_created, id)
  - Index: part_session_idx ON part(session_id)
  - Index: session_project_idx ON session(project_id)
  - Index: todo_session_idx ON todo(session_id)

  **WHY Each Reference Matters**:
  - Schema must be replicated EXACTLY for test queries to match production behavior
  - Indexes affect query plans - include them to catch performance issues in tests

  **Acceptance Criteria**:
  - [ ] `go test ./internal/testutil/...` passes
  - [ ] Test DB has exactly 3 projects, 8 root sessions, 4 child sessions
  - [ ] Each attention signal scenario has at least 1 matching session
  - [ ] Timestamps are relative to time.Now() (not hardcoded)

  **QA Scenarios**:
  ```
  Scenario: Fixture creates valid seeded DB
    Tool: Bash
    Preconditions: fixture.go exists
    Steps:
      1. Run `go test -v ./internal/testutil/...`
      2. Assert all tests pass
      3. Verify test output mentions expected row counts
    Expected Result: All fixture verification tests pass
    Failure Indicators: SQL errors, wrong row counts, missing tables
    Evidence: .sisyphus/evidence/task-3-fixture-tests.txt

  Scenario: Fixture handles concurrent test usage
    Tool: Bash
    Preconditions: fixture.go exists
    Steps:
      1. Run `go test -count=3 -parallel=3 ./internal/testutil/...`
      2. Assert no race conditions or shared state issues
    Expected Result: All parallel runs pass
    Failure Indicators: Race detector warnings, DB locked errors
    Evidence: .sisyphus/evidence/task-3-parallel-safety.txt
  ```

  **Commit**: YES
  - Message: `feat(testdata): test fixture DB builder with seeded scenarios`
  - Files: `internal/testutil/fixture.go, internal/testutil/fixture_test.go`
  - Pre-commit: `go test ./internal/testutil/...`

- [x] 4. SQLite Connection Layer

  **What to do**:
  - TDD: Write tests FIRST in `internal/store/db_test.go`:
    - Test: opens real DB path successfully in read-only mode
    - Test: returns meaningful error for missing DB file
    - Test: returns meaningful error for invalid path
    - Test: connection pool is configured (max open conns, max idle)
    - Test: Close() works cleanly
    - Test: health check (ping) succeeds on valid connection
  - Implement `internal/store/db.go`:
    - `NewDB(dbPath string) (*sql.DB, error)` - opens SQLite with `?mode=ro` (read-only)
    - Configure connection pool: MaxOpenConns=1 (SQLite is single-writer), MaxIdleConns=1
    - Set PRAGMA: `journal_mode=wal` (allows concurrent reads while OpenCode writes)
    - Set PRAGMA: `query_only=on` (extra safety for read-only)
    - Validate the DB has expected tables on open (session, message, part, todo, project)
    - Return clear error messages for: file not found, permission denied, schema mismatch
  - Use `modernc.org/sqlite` driver (pure Go, no CGO required)
  - Register the driver as `"sqlite"` and use DSN format: `file:{path}?mode=ro`

  **Must NOT do**:
  - No write operations - read-only always
  - No query functions - just connection management
  - No Bubbletea imports

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3)
  - **Blocks**: Tasks 5, 6, 7, 8
  - **Blocked By**: Task 1 (needs go.mod with modernc.org/sqlite dependency)

  **References**:
  **Pattern References**:
  - `/Users/joey/Sites/whatsapp-mcp/whatsapp-bridge/` - DB connection pattern (uses go-sqlite3 CGO, adapt to modernc.org/sqlite)

  **External References**:
  - `https://pkg.go.dev/modernc.org/sqlite` - Driver registration and DSN format
  - `https://www.sqlite.org/pragma.html` - PRAGMA reference for journal_mode and query_only

  **WHY Each Reference Matters**:
  - whatsapp-bridge shows DB init pattern to adapt (connection pool config, error handling)
  - modernc.org/sqlite has slightly different DSN format than go-sqlite3 - must check docs

  **Acceptance Criteria**:
  - [ ] Tests written FIRST (RED), then implementation (GREEN)
  - [ ] `go test ./internal/store/...` passes all DB connection tests
  - [ ] Opens real OpenCode DB at `~/.local/share/opencode/opencode.db` successfully
  - [ ] Returns clear error for `/nonexistent/path.db`
  - [ ] Connection is truly read-only (attempt INSERT fails)

  **QA Scenarios**:
  ```
  Scenario: Opens real OpenCode database
    Tool: Bash
    Preconditions: OpenCode DB exists at default path
    Steps:
      1. Run `go test -v -run TestOpenRealDB ./internal/store/...`
      2. Assert test passes
    Expected Result: Successfully opens and pings the real DB
    Failure Indicators: File not found, driver error, permission denied
    Evidence: .sisyphus/evidence/task-4-real-db-open.txt

  Scenario: Rejects missing database gracefully
    Tool: Bash
    Preconditions: No file at /tmp/nonexistent.db
    Steps:
      1. Run `go test -v -run TestMissingDB ./internal/store/...`
      2. Assert test passes with meaningful error message
    Expected Result: Error contains "not found" or "no such file"
    Failure Indicators: Panic, generic error, creates the file
    Evidence: .sisyphus/evidence/task-4-missing-db.txt
  ```

  **Commit**: YES
  - Message: `feat(store): SQLite connection layer with read-only mode`
  - Files: `internal/store/db.go, internal/store/db_test.go`
  - Pre-commit: `go test ./internal/store/...`

- [x] 5. Session Repository + Project Repository (TDD)

  **What to do**:
  - **Part A - ProjectRepo**: TDD in `internal/store/project_test.go`:
    - Test: ListProjects returns all projects
    - Test: ListProjects includes the "global" project (worktree="/")
    - Test: Handles NULL name field gracefully (uses sql.NullString)
    - Test: Returns empty slice when no projects exist
  - Implement `internal/store/project.go`:
    - `type ProjectRepo struct` implementing `domain.ProjectStore`
    - `ListProjects(ctx context.Context) ([]domain.Project, error)` - `SELECT * FROM project ORDER BY worktree` (sort by worktree, NOT name - name is NULL in real DB)
  - **Part B - SessionRepo**: TDD in `internal/store/session_test.go` using test fixture DB:
    - Test: ListRootSessions returns only sessions where parent_id IS NULL
    - Test: ListRootSessions excludes child sessions (parent_id set)
    - Test: ListRootSessions returns Session with project_id FK only (no project name/worktree)
    - Test: GetSessionByID returns full session data
    - Test: GetSessionByID returns error for non-existent ID
    - Test: GetChildCount returns correct count per parent session
    - Test: Results are ordered by time_updated DESC
  - Implement `internal/store/session.go`:
    - `type SessionRepo struct` implementing `domain.SessionStore`
    - `ListRootSessions(ctx context.Context) ([]domain.Session, error)` - SELECT from session WHERE parent_id IS NULL, ORDER BY time_updated DESC. Returns Session structs with project_id FK only (NO JOIN on project table - project display data comes from ProjectStore during aggregation).
    - `GetSessionByID(ctx context.Context, id string) (domain.Session, error)`
    - `GetChildCount(ctx context.Context, parentID string) (int, error)` - COUNT(*) WHERE parent_id = ?
    - Convert Unix ms timestamps to time.Time at scan time
    - Handle NULL fields (parent_id, time_archived) with sql.NullString/sql.NullInt64

  **Must NOT do**:
  - No Bubbletea imports
  - No attention classification logic - just data retrieval
  - No message or todo queries - separate repos

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 7, 8)
  - **Blocks**: Task 11
  - **Blocked By**: Tasks 2 (types), 3 (fixture), 4 (DB connection)

  **References**:
  **API/Type References**:
  - `internal/domain/types.go:Session` - Target struct to populate
  - `internal/domain/interfaces.go:SessionStore` - Interface to implement
  - `internal/testutil/fixture.go:NewTestDB` - Test fixture to use

  **Pattern References**:
  - SQL query for root sessions (no project JOIN - project data comes from ProjectStore separately):
    ```sql
    SELECT id, project_id, parent_id, slug, directory, title, version,
           summary_additions, summary_deletions, summary_files,
           time_created, time_updated, time_archived
    FROM session
    WHERE parent_id IS NULL
    ORDER BY time_updated DESC
    ```

  **WHY Each Reference Matters**:
  - Domain types define exact struct shape to scan into
  - Test fixture provides seeded data with known counts for assertions
  - No JOIN needed here - aggregator (Task 11) handles project lookup via ProjectStore

  **Acceptance Criteria**:
  - [ ] Tests written FIRST (RED), then implementation (GREEN)
  - [ ] `go test ./internal/store/...` passes all session AND project tests
  - [ ] ListProjects returns all 3 projects from fixture (including "global")
  - [ ] ListRootSessions returns exactly 8 root sessions from fixture (not 12 total)
  - [ ] Child sessions (4) are excluded
  - [ ] Each returned Session has project_id FK set (project display fields NOT expected here - they come from ProjectStore via aggregator)

  **QA Scenarios**:
  ```
  Scenario: Root sessions only - no children
    Tool: Bash
    Preconditions: Test fixture DB seeded with 8 root + 4 child sessions
    Steps:
      1. Run `go test -v -run TestListRootSessions ./internal/store/...`
      2. Assert test passes
      3. Verify test output shows exactly 8 sessions returned
    Expected Result: Only root sessions returned, child sessions excluded
    Failure Indicators: Returns 12 (includes children), or 0 (query broken)
    Evidence: .sisyphus/evidence/task-5-root-sessions.txt

  Scenario: Session by ID returns full data
    Tool: Bash
    Preconditions: Test fixture DB seeded
    Steps:
      1. Run `go test -v -run TestGetSessionByID ./internal/store/...`
      2. Assert test verifies all Session fields populated (title, slug, project_id FK, timestamps)
    Expected Result: All Session fields non-zero, project_id is a valid FK string
    Failure Indicators: Zero timestamps, empty project_id, missing fields
    Evidence: .sisyphus/evidence/task-5-session-by-id.txt
  ```

  **Commit**: YES
  - Message: `feat(store): session and project repositories`
  - Files: `internal/store/session.go, internal/store/session_test.go, internal/store/project.go, internal/store/project_test.go`
  - Pre-commit: `go test ./internal/store/...`

- [x] 6. Message Metadata Repository (TDD)

  **What to do**:
  - TDD: Write tests FIRST in `internal/store/message_test.go` using test fixture DB:
    - Test: GetLastMessageMeta returns the most recent message's role and agent for a session
    - Test: GetLastMessageMeta uses json_extract on message.data for role and agent fields
    - Test: GetMessageCount returns total message count per session
    - Test: Returns zero/empty for session with no messages
  - Implement `internal/store/message.go`:
    - `type MessageRepo struct` implementing `domain.MessageStore`
    - `GetLastMessageMeta(ctx context.Context, sessionID string) (domain.MessageMeta, error)` - query: `SELECT json_extract(data, '$.role') as role, json_extract(data, '$.agent') as agent, json_extract(data, '$.modelID') as model_id, json_extract(data, '$.providerID') as provider_id, time_created FROM message WHERE session_id = ? ORDER BY time_created DESC LIMIT 1`
    - `GetMessageCount(ctx context.Context, sessionID string) (int, error)`
    - NEVER load full `data` blob - only json_extract specific fields

  **Must NOT do**:
  - No loading of full message.data blobs into Go
  - No message content/text extraction
  - No Bubbletea imports

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 7, 8)
  - **Blocks**: Task 11
  - **Blocked By**: Tasks 2 (types), 3 (fixture), 4 (DB connection)

  **References**:
  **API/Type References**:
  - Message JSON shape: `{"role":"user|assistant", "agent":"Sisyphus (Ultraworker)", "modelID":"claude-haiku-4-5", "providerID":"anthropic", "time":{"created":1775670988679}}`
  - Note: user messages have role at top level. Assistant messages also have role at top level.
  - `json_extract(data, '$.role')` confirmed working on this DB (Metis tested it)

  **WHY Each Reference Matters**:
  - JSON shape determines exact json_extract paths - wrong path = NULL results
  - Role field location confirmed at `$.role` for both user and assistant messages

  **Acceptance Criteria**:
  - [ ] Tests written FIRST (RED), then implementation (GREEN)
  - [ ] `go test ./internal/store/...` passes all message tests
  - [ ] GetLastMessageMeta returns correct role ('user' or 'assistant')
  - [ ] Never loads full data blob (verify query uses json_extract)

  **QA Scenarios**:
  ```
  Scenario: Last message role detected correctly
    Tool: Bash
    Preconditions: Fixture DB has sessions with known last message roles
    Steps:
      1. Run `go test -v -run TestGetLastMessageMeta ./internal/store/...`
      2. Assert test passes for both user-last and assistant-last sessions
    Expected Result: Correct role returned for each session
    Failure Indicators: NULL role, wrong role, SQL error on json_extract
    Evidence: .sisyphus/evidence/task-6-last-message.txt
  ```

  **Commit**: YES
  - Message: `feat(store): message metadata repository`
  - Files: `internal/store/message.go, internal/store/message_test.go`
  - Pre-commit: `go test ./internal/store/...`

- [x] 7. Todo Repository (TDD)

  **What to do**:
  - TDD: Write tests FIRST in `internal/store/todo_test.go`:
    - Test: GetTodosBySession returns all todos for a session ordered by position
    - Test: GetPendingTodoCount returns count of todos where status != 'completed'
    - Test: Returns empty list / 0 for session with no todos
    - Test: Handles sessions with mixed todo statuses
  - Implement `internal/store/todo.go`:
    - `type TodoRepo struct` implementing `domain.TodoStore`
    - `GetTodosBySession(ctx context.Context, sessionID string) ([]domain.Todo, error)` - ORDER BY position ASC
    - `GetPendingTodoCount(ctx context.Context, sessionID string) (int, error)` - COUNT WHERE status != 'completed'

  **Must NOT do**:
  - No todo modification - read-only
  - No Bubbletea imports

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 8)
  - **Blocks**: Task 11
  - **Blocked By**: Tasks 2 (types), 3 (fixture), 4 (DB connection)

  **References**:
  **API/Type References**:
  - Todo table: session_id TEXT, content TEXT, status TEXT ("completed", "pending", "in_progress", "cancelled"), priority TEXT ("high", "medium", "low"), position INT
  - PK is (session_id, position)

  **Acceptance Criteria**:
  - [ ] Tests written FIRST (RED), then implementation (GREEN)
  - [ ] `go test ./internal/store/...` passes all todo tests
  - [ ] Pending count correctly excludes "completed" todos
  - [ ] Todos returned in position order

  **QA Scenarios**:
  ```
  Scenario: Todo counts match expectations
    Tool: Bash
    Preconditions: Fixture DB has sessions with known todo counts
    Steps:
      1. Run `go test -v -run TestGetPendingTodoCount ./internal/store/...`
      2. Assert: session with 3 pending + 2 completed returns count=3
      3. Assert: session with 0 todos returns count=0
    Expected Result: Counts match seeded data exactly
    Failure Indicators: Wrong counts, includes completed in pending
    Evidence: .sisyphus/evidence/task-7-todo-counts.txt
  ```

  **Commit**: YES
  - Message: `feat(store): todo repository with pending counts`
  - Files: `internal/store/todo.go, internal/store/todo_test.go`
  - Pre-commit: `go test ./internal/store/...`

- [x] 8. Error Detection with Caching (TDD)

  **What to do**:
  - TDD: Write tests FIRST in `internal/store/errors_test.go`:
    - Test: GetErrorCount returns count of parts where json_extract(data, '$.state.status') = 'error' for a session
    - Test: RefreshErrorCache updates cached counts only for sessions updated since last check
    - Test: Returns 0 for session with no error parts
    - Test: Cache invalidation works (session gets new error, cache updates on refresh)
  - Implement `internal/store/errors.go`:
    - `type ErrorRepo struct` with internal cache (map[sessionID]errorCount + lastRefreshTime)
    - `GetErrorCount(ctx context.Context, sessionID string) (int, error)` - returns cached value
    - `RefreshErrorCache(ctx context.Context, since time.Time) error` - query: `SELECT session_id, COUNT(*) FROM part WHERE json_extract(data, '$.state.status') = 'error' AND session_id IN (SELECT id FROM session WHERE time_updated > ?) GROUP BY session_id`
    - `RefreshAll(ctx context.Context) error` - full scan for initial load
    - Initial load scans all sessions; subsequent refreshes only re-scan sessions updated since last check
    - Thread-safe: use sync.RWMutex for cache access

  **Must NOT do**:
  - No loading full part.data blobs
  - No Bubbletea imports
  - No real-time streaming - batch cache refresh only

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 7)
  - **Blocks**: Task 11
  - **Blocked By**: Tasks 2 (types), 3 (fixture), 4 (DB connection)

  **References**:
  **API/Type References**:
  - Part data JSON for errors: `{"type":"tool-invocation", "state":{"status":"error"}}` 
  - Part data JSON for success: `{"type":"tool-invocation", "state":{"status":"completed"}}`
  - From Metis: 1,348 error parts across 257K total, full scan ~1.5s
  - Incremental scan (since last refresh) should be much faster

  **WHY Each Reference Matters**:
  - json_extract path `$.state.status` confirmed working on real data
  - 1.5s full scan is acceptable for startup but too slow for 30s refresh - hence caching

  **Acceptance Criteria**:
  - [ ] Tests written FIRST (RED), then implementation (GREEN)
  - [ ] `go test -race ./internal/store/...` passes (thread safety)
  - [ ] Cache correctly returns stale values between refreshes
  - [ ] Incremental refresh only queries sessions updated since last check

  **QA Scenarios**:
  ```
  Scenario: Error count from cache
    Tool: Bash
    Preconditions: Fixture DB has session with 3 error parts
    Steps:
      1. Run `go test -v -run TestErrorCount ./internal/store/...`
      2. Assert RefreshAll populates cache
      3. Assert GetErrorCount returns 3 for the error session
      4. Assert GetErrorCount returns 0 for clean session
    Expected Result: Cache populated and queried correctly
    Failure Indicators: Wrong counts, cache miss panic, SQL error
    Evidence: .sisyphus/evidence/task-8-error-cache.txt

  Scenario: Thread-safe concurrent access
    Tool: Bash
    Preconditions: Error repo with populated cache
    Steps:
      1. Run `go test -race -run TestErrorConcurrency ./internal/store/...`
      2. Multiple goroutines read cache while one refreshes
    Expected Result: No race conditions detected
    Failure Indicators: Race detector warning
    Evidence: .sisyphus/evidence/task-8-race-test.txt
  ```

  **Commit**: YES
  - Message: `feat(store): error detection with caching`
  - Files: `internal/store/errors.go, internal/store/errors_test.go`
  - Pre-commit: `go test -race ./internal/store/...`

- [ ] 9. Attention Signal Classifier (TDD)

  **What to do**:
  - TDD: Write tests FIRST in `internal/attention/classifier_test.go`:
    - Test: Session with last_role='assistant' AND updated <24h ago -> NeedsResponse
    - Test: Session updated <5min ago -> ActiveNow (overrides NeedsResponse)
    - Test: Session with error_count > 0 -> HasErrors (highest priority)
    - Test: Session with pending_todos > 0 AND updated >7 days ago -> StaleWork
    - Test: Session with pending_todos > 0 AND updated <7 days ago -> PendingTodos
    - Test: Session with no signals -> None
    - Test: Priority ordering when multiple signals apply (HasErrors wins)
    - Test: Edge case: session with no messages -> None (not NeedsResponse)
    - Test: Edge case: archived session -> None regardless of other signals
    - Test: Edge case: NULL last_role -> None
  - Implement `internal/attention/classifier.go`:
    - `Classify(view domain.SessionView, now time.Time) domain.AttentionSignal` - pure function
    - Constants: `NeedsResponseWindow = 24 * time.Hour`, `ActiveNowWindow = 5 * time.Minute`, `StaleThreshold = 7 * 24 * time.Hour`
    - Priority chain: HasErrors > ActiveNow > NeedsResponse > StaleWork > PendingTodos > None
    - Accept `now time.Time` parameter for testability (no `time.Now()` inside)

  **Must NOT do**:
  - No database access - pure function only, takes SessionView as input
  - No Bubbletea imports
  - No configurable thresholds yet - hardcoded constants

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Pure logic with many edge cases requiring careful reasoning
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 10, 11, 12, 13)
  - **Blocks**: Task 11
  - **Blocked By**: Task 2 (types)

  **References**:
  **API/Type References**:
  - `internal/domain/types.go:SessionView` - input struct (has LastMessageRole, ErrorCount, PendingTodoCount, TimeUpdated, TimeArchived)
  - `internal/domain/types.go:AttentionSignal` - output enum

  **Pattern References**:
  - Attention signal definitions (from Metis):
    | Signal | Condition |
    |--------|-----------|
    | HasErrors | error_count > 0 |
    | ActiveNow | time_updated > (now - 5min) |
    | NeedsResponse | last_role = 'assistant' AND time_updated > (now - 24h) |
    | StaleWork | pending_todos > 0 AND time_updated < (now - 7d) |
    | PendingTodos | pending_todos > 0 AND time_updated >= (now - 7d) |

  **WHY Each Reference Matters**:
  - SessionView is the classifier's input contract - must match available fields
  - Signal definitions are the spec - each test case maps to one row in this table

  **Acceptance Criteria**:
  - [ ] Tests written FIRST (RED), then implementation (GREEN)
  - [ ] `go test ./internal/attention/...` passes all classifier tests
  - [ ] Each of the 5 signals has at least 1 dedicated test
  - [ ] Priority ordering verified: HasErrors beats all others
  - [ ] Pure function - no side effects, no DB, no time.Now()

  **QA Scenarios**:
  ```
  Scenario: All 5 signals classified correctly
    Tool: Bash
    Preconditions: classifier_test.go covers all 5 signals
    Steps:
      1. Run `go test -v ./internal/attention/...`
      2. Assert all tests pass
      3. Verify test names include: NeedsResponse, ActiveNow, HasErrors, StaleWork, PendingTodos
    Expected Result: Each signal independently triggered and verified
    Failure Indicators: Wrong signal for known input, priority ordering wrong
    Evidence: .sisyphus/evidence/task-9-classifier.txt

  Scenario: Edge cases handled
    Tool: Bash
    Preconditions: Tests include no-messages, archived, NULL role cases
    Steps:
      1. Run `go test -v -run "Edge" ./internal/attention/...`
      2. Assert all edge case tests pass
    Expected Result: No panics, graceful handling of empty/null data
    Failure Indicators: Panic on nil, wrong classification for edge case
    Evidence: .sisyphus/evidence/task-9-edge-cases.txt
  ```

  **Commit**: YES
  - Message: `feat(attention): attention signal classifier with priority ordering`
  - Files: `internal/attention/classifier.go, internal/attention/classifier_test.go`
  - Pre-commit: `go test ./internal/attention/...`

- [ ] 10. Filter/Search Engine (TDD)

  **What to do**:
  - TDD: Write tests FIRST in `internal/filter/filter_test.go`:
    - Test: NeedsAttention preset returns only sessions with AttentionSignal != None
    - Test: AllActive preset returns sessions where time_archived IS NULL
    - Test: Archived preset returns sessions where time_archived IS NOT NULL
    - Test: Free-text search matches session title (case-insensitive)
    - Test: Free-text search matches session slug (case-insensitive)
    - Test: Empty search returns all sessions (no filter)
    - Test: Combined preset + search works (AND logic)
    - Test: Empty result set handled gracefully
  - Implement `internal/filter/filter.go`:
    - `type Filter struct { Preset domain.FilterPreset; SearchText string }`
    - `Apply(sessions []domain.SessionView, filter Filter) []domain.SessionView` - pure function
    - Presets filter on attention signal or archive status
    - Search uses strings.Contains on Title and Slug (case-insensitive with strings.ToLower)
    - Returns a new slice (does not mutate input)

  **Must NOT do**:
  - No database queries - operates on in-memory slice
  - No regex or fuzzy matching - simple case-insensitive contains
  - No Bubbletea imports

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 9, 11, 12, 13)
  - **Blocks**: Task 17
  - **Blocked By**: Task 2 (types)

  **References**:
  **API/Type References**:
  - `internal/domain/types.go:FilterPreset` - NeedsAttention, AllActive, Archived
  - `internal/domain/types.go:SessionView` - has AttentionSignal, TimeArchived, Title, Slug

  **Acceptance Criteria**:
  - [ ] Tests written FIRST (RED), then implementation (GREEN)
  - [ ] `go test ./internal/filter/...` passes all filter tests
  - [ ] Each preset tested independently
  - [ ] Search is case-insensitive
  - [ ] Does not mutate input slice

  **QA Scenarios**:
  ```
  Scenario: Presets filter correctly
    Tool: Bash
    Preconditions: filter_test.go with known session views
    Steps:
      1. Run `go test -v ./internal/filter/...`
      2. Assert NeedsAttention returns only sessions with signals
      3. Assert AllActive excludes archived
      4. Assert Archived returns only archived
    Expected Result: Each preset returns correct subset
    Failure Indicators: Wrong count, includes sessions that should be filtered
    Evidence: .sisyphus/evidence/task-10-filter-presets.txt
  ```

  **Commit**: YES
  - Message: `feat(filter): filter and search engine with presets`
  - Files: `internal/filter/filter.go, internal/filter/filter_test.go`
  - Pre-commit: `go test ./internal/filter/...`

- [ ] 11. Data Aggregator - View Model (TDD)

  **What to do**:
  - TDD: Write tests FIRST in `internal/app/aggregator_test.go`:
    - Test: LoadAll fetches sessions, messages, todos, errors and builds []ProjectGroup
    - Test: Result is []ProjectGroup (ordered slice, NOT map) - preserves insertion order
    - Test: Projects sorted alphabetically, "Global" always last
    - Test: Each SessionView has correct attention signal
    - Test: Each SessionView has full Todos list AND MessageCount AND ErrorCount
    - Test: Child counts populated per parent session
    - Test: Refresh only updates error cache incrementally
  - Implement `internal/app/aggregator.go`:
    - `type Aggregator struct` - holds all store interfaces + attention classifier
    - `LoadAll(ctx context.Context) ([]domain.ProjectGroup, error)` - returns ORDERED slice of ProjectGroups
      1. Fetch all projects from ProjectStore
      2. Fetch all root sessions from SessionRepo
      3. For each session: fetch last message meta, FULL todo list (not just count), message count, child count, error count
      4. Build SessionView with all fields populated (PendingTodoCount derived from Todos list)
      5. Classify attention signal for each
      6. Group into ProjectGroups: sort projects alphabetically by Project.DisplayName(), "Global" ALWAYS last
      7. Within each ProjectGroup: sort sessions by attention priority DESC, then time_updated DESC
      8. Set ProjectGroup.AttentionCount = count of sessions where AttentionSignal != None
    - `Refresh(ctx context.Context) ([]domain.ProjectGroup, error)` - same as LoadAll but calls RefreshErrorCache(since) first
    - Accept store interfaces via constructor for testability (mock in tests)

  **Must NOT do**:
  - No direct SQL - use store interfaces only
  - No Bubbletea imports
  - No caching at this layer (stores handle their own caching)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 9, 10, 12, 13)
  - **Blocks**: Task 16
  - **Blocked By**: Tasks 5, 6, 7, 8 (all repos), 9 (classifier)

  **References**:
  **API/Type References**:
  - `internal/domain/interfaces.go` - all store interfaces to compose
  - `internal/domain/types.go:SessionView` - output type with all aggregated data
  - `internal/attention/classifier.go:Classify` - attention classification function

  **WHY Each Reference Matters**:
  - Aggregator is the bridge between data layer and UI - must correctly compose all repos
  - SessionView is the UI's input contract - must be fully populated

  **Acceptance Criteria**:
  - [ ] Tests written FIRST (RED), then implementation (GREEN)
  - [ ] `go test ./internal/app/...` passes all aggregator tests
  - [ ] "Global" project appears last in sorted output
  - [ ] Sessions within a project sorted by attention priority, then recency
  - [ ] Uses mocked store interfaces (not real DB)

  **QA Scenarios**:
  ```
  Scenario: Full aggregation pipeline
    Tool: Bash
    Preconditions: Mock stores with known return values
    Steps:
      1. Run `go test -v -run TestLoadAll ./internal/app/...`
      2. Assert result is []ProjectGroup (ordered slice)
      3. Assert projects sorted alphabetically, "Global" last
      4. Assert each SessionView has Todos (full list), MessageCount, ErrorCount populated
      5. Assert attention signals assigned correctly
    Expected Result: Complete view model built from store data
    Failure Indicators: Missing project groups, wrong sorting, nil pointers
    Evidence: .sisyphus/evidence/task-11-aggregator.txt
  ```

  **Commit**: YES
  - Message: `feat(app): data aggregator composing stores into view model`
  - Files: `internal/app/aggregator.go, internal/app/aggregator_test.go`
  - Pre-commit: `go test ./internal/app/...`

- [ ] 12. Bubbletea App Skeleton

  **What to do**:
  - Create `internal/ui/app.go`:
    - `type AppModel struct` - top-level Bubbletea model
    - `Init() tea.Cmd` - returns nil (no initial command yet)
    - `Update(msg tea.Msg) (tea.Model, tea.Cmd)` - handle tea.KeyMsg: 'q' or ctrl+c quits
    - `View() string` - render a placeholder: header bar with "OpenCode Dashboard" + "(press q to quit)"
    - `func New() *tea.Program` - creates and returns the Bubbletea program
  - Update `cmd/opencode-dashboard/main.go` to launch the Bubbletea program instead of printing and exiting
  - Use Lipgloss for the header styling (bold, colored background)

  **Must NOT do**:
  - No data loading yet - just the shell
  - No panes or layout - single view placeholder
  - No complex key handling beyond quit

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 9, 10, 11, 13)
  - **Blocks**: Tasks 13, 14
  - **Blocked By**: Task 1 (scaffold + dependencies)

  **References**:
  **External References**:
  - `https://github.com/charmbracelet/bubbletea` - Program lifecycle, Model interface
  - `https://github.com/charmbracelet/lipgloss` - Style API for header

  **WHY Each Reference Matters**:
  - Bubbletea Model interface (Init/Update/View) is the skeleton contract
  - Lipgloss styling needed for header bar from day 1

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/opencode-dashboard` succeeds
  - [ ] Binary launches a TUI showing "OpenCode Dashboard"
  - [ ] Pressing 'q' exits cleanly (exit code 0)
  - [ ] Ctrl+C also exits cleanly

  **QA Scenarios**:
  ```
  Scenario: TUI launches and shows header
    Tool: interactive_bash (tmux)
    Preconditions: Binary built
    Steps:
      1. Create tmux session: `new-session -d -s ocd-test`
      2. Send: `send-keys -t ocd-test "./opencode-dashboard" Enter`
      3. Wait 2s
      4. Capture pane: `capture-pane -t ocd-test -p`
      5. Assert pane contains "OpenCode Dashboard"
    Expected Result: Header text visible in terminal
    Failure Indicators: Blank screen, crash, no output
    Evidence: .sisyphus/evidence/task-12-tui-launch.png

  Scenario: Quit with q
    Tool: interactive_bash (tmux)
    Preconditions: TUI running in tmux
    Steps:
      1. Send: `send-keys -t ocd-test "q"`
      2. Wait 1s
      3. Capture pane and check for shell prompt (TUI exited)
    Expected Result: TUI exits, shell prompt visible
    Failure Indicators: TUI still running, error message on exit
    Evidence: .sisyphus/evidence/task-12-quit.txt
  ```

  **Commit**: YES
  - Message: `feat(ui): bubbletea app skeleton with header`
  - Files: `internal/ui/app.go, cmd/opencode-dashboard/main.go`
  - Pre-commit: `go build ./...`

- [ ] 13. Left Pane - Project Tree Component

  **What to do**:
  - Create `internal/ui/sessionlist.go`:
    - `type SessionListModel struct` - Bubbletea sub-model for the left pane
    - Renders project-grouped session list:
      - Project header row: `[icon] ProjectName (N sessions, M need attention)` - collapsible
      - Session row (indented under project): `[attention-icon] session-slug  2h ago  3/7`
        - Attention icon: color-coded based on signal (red=error, green=active, yellow=needs response, orange=stale, blue=pending todos, dim=none)
        - Relative time: "2h ago", "3d ago", etc.
        - Todo progress: "3/7" (completed/total), hidden if no todos
    - Keyboard: j/k moves cursor, Enter toggles project collapse
    - Scrolling: viewport scrolls when cursor moves beyond visible area
    - Highlighted row has distinct background color AND a text-based `>` prefix marker for QA verifiability via `capture-pane -p`
    - Accept `[]domain.ProjectGroup` as input data (ordered slice from aggregator)
  - Use Lipgloss for styling: attention colors, selected row highlight, project header bold

  **Must NOT do**:
  - No data fetching - receives data as input
  - No right pane interaction - left pane only
  - No filter logic - receives pre-filtered data

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: TUI component with styling, layout, scroll behavior
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 9, 10, 11, 12)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 2 (types for SessionView), 12 (app skeleton for tea.Model pattern)

  **References**:
  **External References**:
  - `https://github.com/charmbracelet/bubbles` - viewport component for scrolling
  - `https://github.com/charmbracelet/lipgloss` - Color, Bold, Background styles

  **API/Type References**:
  - `internal/domain/types.go:SessionView` - data shape for each row
  - `internal/domain/types.go:AttentionSignal` - determines icon/color per row

  **WHY Each Reference Matters**:
  - Bubbles viewport handles scroll math - don't reimplement
  - SessionView determines what data is available for rendering

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] Project headers show name and session count
  - [ ] Session rows show attention icon, slug, relative time, todo progress
  - [ ] j/k moves selection, Enter collapses/expands projects
  - [ ] Scrolling works when list exceeds viewport height

  **QA Scenarios**:
  NOTE: Task 13 is tested STANDALONE (not inside split layout from Task 15). Write a temporary `cmd/test-sessionlist/main.go` that renders only the SessionListModel full-screen with hardcoded stub ProjectGroup data: 2 projects (Project with DisplayName "alpha" / worktree="/Users/joey/Sites/alpha" with 3 sessions, "Global" / worktree="/" with 1 session), sessions with varied attention signals. Delete the test binary after QA.

  IMPORTANT: In real data, project.name is NULL. Stub data should use DisplayName() which derives from worktree basename. QA assertions match on DisplayName output, not raw name field.

  ```
  Scenario: Project tree renders with stub data
    Tool: interactive_bash (tmux)
    Preconditions: Build `cmd/test-sessionlist` with hardcoded ProjectGroup data using DisplayName-derived labels
    Steps:
      1. Build: `go build -o /tmp/test-sessionlist ./cmd/test-sessionlist`
      2. Launch in tmux: `send-keys -t ocd-test "/tmp/test-sessionlist" Enter`
      3. Wait 2s, capture pane with `capture-pane -t ocd-test -p`
      4. Assert pane text contains "alpha" (project display name from worktree basename)
      5. Assert pane text contains "Global" (special-case display name)
      6. Assert at least 1 session slug visible
      7. Assert selected row has `>` prefix marker (text-based selection indicator)
      8. Send j key 3 times, capture pane
      9. Assert `>` marker moved to a different row (text-based verification)
    Expected Result: Project groups and sessions render, navigation works via text markers
    Failure Indicators: Empty pane, no project labels, `>` marker doesn't move
    Evidence: .sisyphus/evidence/task-13-project-tree.txt

  Scenario: Project collapse/expand with stub data
    Tool: interactive_bash (tmux)
    Preconditions: test-sessionlist running in tmux
    Steps:
      1. Navigate to "alpha" project header row (first item, has `>` marker)
      2. Press Enter
      3. Capture pane text - sessions under "alpha" should no longer be visible
      4. Press Enter again
      5. Capture pane text - sessions should reappear
    Expected Result: Toggle works, session count persists on header
    Failure Indicators: Crash on Enter, sessions don't hide/show
    Evidence: .sisyphus/evidence/task-13-collapse.txt
  ```

  Post-QA cleanup: remove `cmd/test-sessionlist/` directory

  **Commit**: YES
  - Message: `feat(ui): project tree left pane with attention icons`
  - Files: `internal/ui/sessionlist.go`
  - Pre-commit: `go build ./...`

- [ ] 14. Right Pane - Session Detail View

  **What to do**:
  - Create `internal/ui/detail.go`:
    - `type DetailModel struct` - Bubbletea sub-model for the right pane
    - Renders full session detail for the currently selected session:
      - Header: Session title (full, not truncated) + attention badge
      - Metadata section: Project name, directory path, slug, version, last updated (relative + absolute)
      - Stats section: Messages count, code changes (+N -N, M files), child session count
      - Todos section: list of all todos with status icon (checkmark/pending/cancelled), priority badge, content text
      - Last activity: agent name, model used, relative timestamp
    - Scrollable if content exceeds viewport (use bubbles viewport)
    - Shows "Select a session" placeholder when no session is selected
  - Use Lipgloss for section headers, status colors, layout

  **Must NOT do**:
  - No message content display - metadata only
  - No data fetching - receives SessionView as input
  - No editing or management actions

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: TUI layout with multiple styled sections
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 15, 16, 17, 18)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 2 (types), 12 (app skeleton)

  **References**:
  **API/Type References**:
  - `internal/domain/types.go:SessionView` - all fields available for display
  - `internal/domain/types.go:Todo` - todo list items with status/priority/content

  **External References**:
  - `https://github.com/charmbracelet/bubbles` - viewport for scrollable detail
  - `https://github.com/charmbracelet/lipgloss` - Section headers, badges, colors

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] Detail view shows all sections: header, metadata, stats, todos, last activity
  - [ ] "Select a session" shown when no session selected
  - [ ] Todos show status icons and content
  - [ ] Content scrolls when exceeding viewport

  **QA Scenarios**:
  NOTE: Task 14 is tested STANDALONE (not inside split layout from Task 15). Write a temporary `cmd/test-detail/main.go` that renders only the DetailModel full-screen with hardcoded stub SessionView data. Delete the test binary after QA.

  ```
  Scenario: Detail renders all sections with stub data
    Tool: interactive_bash (tmux)
    Preconditions: Build test-detail binary with hardcoded SessionView containing: title="Test Session", 3 todos (2 pending, 1 completed), 42 messages, +120 -45 code changes, 2 child sessions, 1 error
    Steps:
      1. Build: `go build -o /tmp/test-detail ./cmd/test-detail`
      2. Launch in tmux: `send-keys -t ocd-test "/tmp/test-detail" Enter`
      3. Wait 2s, capture pane
      4. Assert pane contains "Test Session" (title)
      5. Assert pane contains "42" (message count)
      6. Assert pane contains "+120" and "-45" (code changes)
      7. Assert pane contains todo content text with status icons
      8. Assert pane contains "2 children" or similar child count
      9. Press q to exit
    Expected Result: All sections render with correct data from stub SessionView
    Failure Indicators: Missing sections, wrong data, layout overflow
    Evidence: .sisyphus/evidence/task-14-detail-standalone.png

  Scenario: Empty state when nil session
    Tool: interactive_bash (tmux)
    Preconditions: Build test-detail binary that passes nil/zero SessionView
    Steps:
      1. Launch test-detail with no session data
      2. Wait 1s, capture pane
      3. Assert pane contains "Select a session"
    Expected Result: Placeholder text visible, no crash
    Failure Indicators: Blank pane, panic on nil data
    Evidence: .sisyphus/evidence/task-14-empty-state.png
  ```

  **Commit**: YES
  - Message: `feat(ui): session detail right pane`
  - Files: `internal/ui/detail.go`
  - Pre-commit: `go build ./...`
  - Post-QA cleanup: remove `cmd/test-detail/` directory

- [ ] 15. Split Layout + Pane Navigation

  **What to do**:
  - Create `internal/ui/layout.go`:
    - `type LayoutModel struct` - composes SessionListModel (left) and DetailModel (right)
    - Split-pane layout: left pane ~40% width, right pane ~60% width, vertical divider between them
    - Active pane tracking: `activePane` field (left/right)
    - Key handling:
      - `h` or `Left arrow`: switch to left pane
      - `l` or `Right arrow`: switch to right pane
      - Active pane has highlighted border AND a text-based indicator in its header (e.g., `[*]` prefix on active pane title vs `[ ]` on inactive) for QA verifiability via `capture-pane -p`
      - Key events forwarded to active pane only
    - Handle terminal resize (tea.WindowSizeMsg): recalculate pane widths proportionally
    - Selection sync: when cursor moves in left pane, update right pane's displayed session
  - Integrate into AppModel: replace placeholder View with LayoutModel

  **Must NOT do**:
  - No filter bar yet - just the two panes
  - No data loading - receives pre-built data

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Layout composition, border styling, resize handling
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4
  - **Blocks**: Task 16
  - **Blocked By**: Tasks 13 (left pane), 14 (right pane)

  **References**:
  **Pattern References**:
  - `internal/ui/sessionlist.go` - left pane component to compose
  - `internal/ui/detail.go` - right pane component to compose
  - `internal/ui/app.go` - parent model to integrate into

  **External References**:
  - `https://github.com/charmbracelet/lipgloss` - `lipgloss.JoinHorizontal` for side-by-side panes
  - `https://github.com/charmbracelet/bubbletea` - tea.WindowSizeMsg for resize handling

  **WHY Each Reference Matters**:
  - JoinHorizontal is the idiomatic way to lay out side-by-side panes in Lipgloss
  - WindowSizeMsg must be handled or panes will overflow on terminal resize

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] Two panes visible side by side with divider
  - [ ] h/l switches active pane (border highlight + `[*]`/`[ ]` text indicator changes)
  - [ ] Arrow keys also switch panes
  - [ ] Selecting session in left pane updates right pane
  - [ ] Terminal resize doesn't break layout

  **QA Scenarios**:
  NOTE: Task 15 is tested STANDALONE (before Task 16 wires real data). Write a temporary `cmd/test-layout/main.go` that renders the LayoutModel full-screen with hardcoded stub ProjectGroup data (same as Task 13/14 stubs) populating both panes. Delete the test binary after QA.

  ```
  Scenario: Split pane layout with stub data
    Tool: interactive_bash (tmux)
    Preconditions: Build `cmd/test-layout` with hardcoded ProjectGroup data
    Steps:
      1. Build: `go build -o /tmp/test-layout ./cmd/test-layout`
      2. Create tmux: `new-session -d -s ocd-test -x 120 -y 40`
      3. Launch: `send-keys -t ocd-test "/tmp/test-layout" Enter`
      4. Wait 2s, capture pane text via `capture-pane -t ocd-test -p`
      5. Assert two distinct sections visible (left has project/session text, right has detail text)
      6. Assert left pane header shows `[*]` marker (active) and right shows `[ ]` (inactive)
      7. Send "l" key - capture pane text, assert right pane header now shows `[*]` and left shows `[ ]`
      8. Send "h" key - capture pane text, assert left pane header shows `[*]` again
      9. Send "j" 2 times in left pane - capture, assert right pane content changes (different session title in detail)
    Expected Result: Two-pane layout rendered, pane switching verified via text markers, selection syncs detail
    Failure Indicators: Single pane, `[*]` marker doesn't switch, right pane title doesn't change
    Evidence: .sisyphus/evidence/task-15-split-layout.png

  Scenario: Terminal resize handled
    Tool: interactive_bash (tmux)
    Preconditions: test-layout running in tmux
    Steps:
      1. Resize tmux pane: `resize-pane -t ocd-test -x 80 -y 24` - wait 1s
      2. Capture pane - assert both panes still visible (no overflow or crash)
      3. Resize back: `resize-pane -t ocd-test -x 120 -y 40` - wait 1s
      4. Capture pane - assert layout readjusted to larger size
    Expected Result: Panes resize proportionally without crash
    Failure Indicators: Overflow, crash, pane disappears, text garbled
    Evidence: .sisyphus/evidence/task-15-resize.png
  ```

  Post-QA cleanup: remove `cmd/test-layout/` directory

  **Commit**: YES
  - Message: `feat(ui): split layout with pane navigation`
  - Files: `internal/ui/layout.go`
  - Pre-commit: `go build ./...`

- [ ] 16. Wire Data Layer into TUI

  **What to do**:
  - Update `internal/ui/app.go`:
    - Add `*app.Aggregator` field to AppModel
    - On Init(): trigger initial data load via tea.Cmd (async, non-blocking)
    - Handle data load result message: populate LayoutModel with real SessionView data
    - Show loading indicator while data is being fetched
    - Handle data load errors: show error message in a styled banner
  - Update `cmd/opencode-dashboard/main.go`:
    - Open DB connection with store.NewDB(dbPath)
    - Create all repos (ProjectRepo, SessionRepo, MessageRepo, TodoRepo, ErrorRepo)
    - Create Aggregator with all repos + classifier
    - Pass Aggregator to UI
    - Handle startup errors gracefully (DB not found, schema mismatch)
  - First time the app shows REAL data from the OpenCode database

  **Must NOT do**:
  - No auto-refresh yet (Task 18)
  - No filter bar yet (Task 17)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Integration point between data and UI layers, async loading, error handling
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (sequential after T15)
  - **Blocks**: Tasks 17, 18
  - **Blocked By**: Tasks 11 (aggregator), 15 (layout)

  **References**:
  **Pattern References**:
  - `internal/app/aggregator.go:LoadAll` - data loading function
  - `internal/store/db.go:NewDB` - DB connection factory
  - `internal/ui/layout.go:LayoutModel` - UI target for data

  **External References**:
  - `https://github.com/charmbracelet/bubbletea` - tea.Cmd for async operations, custom Msg types

  **WHY Each Reference Matters**:
  - Bubbletea's async pattern (Cmd -> Msg) is critical for non-blocking data loads
  - Aggregator.LoadAll is the single entry point for all data

  **Acceptance Criteria**:
  - [ ] `go build ./... && go test ./...` both pass
  - [ ] Binary launches and shows REAL sessions from ~/.local/share/opencode/opencode.db
  - [ ] Loading state visible briefly on startup
  - [ ] Error state shown when DB path is invalid (no crash)
  - [ ] Sessions grouped by project with correct attention icons

  **QA Scenarios**:
  ```
  Scenario: Real data loads and displays
    Tool: interactive_bash (tmux)
    Preconditions: OpenCode DB exists at default path
    Steps:
      1. Build and launch: `go build ./cmd/opencode-dashboard && ./opencode-dashboard`
      2. Wait 3s for data load
      3. Capture pane
      4. Assert at least 1 project name visible (from real DB)
      5. Assert at least 1 session slug visible
      6. Navigate with j/k, assert right pane updates with real data
    Expected Result: Real session data from OpenCode DB displayed
    Failure Indicators: Empty panes, "no sessions", crash on data load
    Evidence: .sisyphus/evidence/task-16-real-data.png

  Scenario: Graceful error on missing DB
    Tool: interactive_bash (tmux)
    Preconditions: No DB at /tmp/fake-path.db
    Steps:
      1. Launch: `./opencode-dashboard --db-path /tmp/fake-path.db`
      2. Wait 2s, capture pane or stdout
      3. Assert error message contains "not found" or "does not exist"
      4. Assert exit is clean (no panic)
    Expected Result: Clear error message, clean exit
    Failure Indicators: Panic, stack trace, no error message
    Evidence: .sisyphus/evidence/task-16-missing-db.txt
  ```

  **Commit**: YES
  - Message: `feat(ui): wire data layer into TUI with real session data`
  - Files: `internal/ui/app.go, cmd/opencode-dashboard/main.go`
  - Pre-commit: `go build ./... && go test ./...`

- [ ] 17. Filter Bar UI

  **What to do**:
  - Create `internal/ui/filterbar.go`:
    - `type FilterBarModel struct` - Bubbletea sub-model for the top filter bar
    - Renders: `[Filter: Needs Attention | All Active | Archived]  Search: ___________  (Tab: cycle, /: search, Esc: clear)`
    - Active preset is highlighted/bold
    - Key handling:
      - `Tab`: cycle through presets (Needs Attention -> All Active -> Archived -> Needs Attention)
      - `/`: enter search mode (focus text input)
      - `Esc` in search mode: clear search text, exit search mode
      - `Enter` in search mode: apply search, exit search mode
    - Use bubbles `textinput` component for search field
  - Integrate into LayoutModel:
    - Filter bar sits above the split panes
    - Filter changes trigger re-filtering of session data via filter.Apply()
    - Session count updates in filter bar: "Showing N of M sessions"

  **Must NOT do**:
  - No additional filter types beyond the 3 presets + text search
  - No saved/custom filters

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI component with text input, styling, state management
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (after T16)
  - **Parallel Group**: Wave 4 (with Task 18)
  - **Blocks**: None
  - **Blocked By**: Tasks 10 (filter engine), 16 (wired data)

  **References**:
  **Pattern References**:
  - `internal/filter/filter.go:Apply` - filter function to invoke on preset/search change
  - `internal/ui/layout.go` - parent component to integrate into

  **External References**:
  - `https://github.com/charmbracelet/bubbles` - textinput component for search field

  **WHY Each Reference Matters**:
  - Bubbles textinput handles cursor, focus, key events for text entry
  - Filter.Apply is the pure function that actually filters the data

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] Filter bar visible at top with preset labels
  - [ ] Tab cycles through presets, session list updates
  - [ ] / activates search, typing filters by title, Esc clears
  - [ ] "Showing N of M sessions" counter updates

  **QA Scenarios**:
  ```
  Scenario: Preset cycling filters sessions
    Tool: interactive_bash (tmux)
    Preconditions: TUI running with real data
    Steps:
      1. Capture initial pane - note session count
      2. Press Tab - assert filter label changes to "All Active"
      3. Capture pane - assert session count may differ
      4. Press Tab - assert filter label changes to "Archived"
      5. Press Tab - assert filter label returns to "Needs Attention"
    Expected Result: Preset cycles, session list updates each time
    Failure Indicators: Label doesn't change, session list static
    Evidence: .sisyphus/evidence/task-17-filter-presets.png

  Scenario: Text search filters by title
    Tool: interactive_bash (tmux)
    Preconditions: TUI running with real data
    Steps:
      1. Press / to enter search mode
      2. Type a known session slug fragment (e.g., first 3 chars of a visible slug)
      3. Press Enter
      4. Capture pane - assert only matching sessions visible
      5. Press Esc - assert search cleared, all sessions return
    Expected Result: Search narrows list, Esc clears
    Failure Indicators: Search doesn't filter, Esc doesn't clear, crash on /
    Evidence: .sisyphus/evidence/task-17-text-search.png
  ```

  **Commit**: YES
  - Message: `feat(ui): filter bar with presets and search`
  - Files: `internal/ui/filterbar.go, internal/ui/layout.go (updated)`
  - Pre-commit: `go build ./...`

- [ ] 18. Auto-Refresh + Session Launcher

  **What to do**:
  - **Auto-refresh**: Update `internal/ui/app.go`:
    - Add 30s ticker using `tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return refreshMsg{} })`
    - On refreshMsg: call Aggregator.Refresh() via tea.Cmd
    - On refresh result: update session data without losing cursor position or scroll state
    - Store current selected session ID before refresh, restore after
    - Show subtle "Refreshing..." indicator during refresh (small text in status bar)
  - **Session launcher**: Create `internal/launcher/launch.go`:
    - `LaunchCmd(sessionID string, directory string) (*exec.Cmd, error)` - constructs the exec.Cmd (does NOT run it)
    - Command: `opencode --session <sessionID>` with Dir set to session's directory
    - Verify `opencode` binary exists in PATH via `exec.LookPath` before constructing
    - Return the *exec.Cmd so Bubbletea can use `tea.ExecProcess` to suspend/resume
  - **Integration**: In LayoutModel, Enter key on a session row:
    - If in left pane on a project header: toggle collapse
    - If in left pane on a session row: use `tea.ExecProcess(cmd, func(err error) tea.Msg)` to:
      1. Suspend dashboard (Bubbletea releases terminal)
      2. Run opencode as child process
      3. When opencode exits, dashboard resumes where it left off
      4. Trigger a data refresh on resume (session state may have changed)
  - Write tests for launcher: `internal/launcher/launch_test.go`
    - Test: command construction is correct (args, dir)
    - Test: returns error if opencode not in PATH
    - Do NOT actually exec in tests - verify command would be correct

  **Must NOT do**:
  - No "open in editor" or "open terminal" - only `opencode --session`
  - No `syscall.Exec` - use `tea.ExecProcess` for suspend/resume behavior
  - No configurable refresh interval - hardcoded 30s
  - No background refresh indicator beyond subtle text

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 17)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 16 (wired data layer)

  **References**:
  **External References**:
  - `https://github.com/charmbracelet/bubbletea` - tea.Tick for periodic commands, tea.ExecProcess for suspend/resume
  - Go stdlib `os/exec` - exec.Cmd construction, exec.LookPath for PATH validation

  **Pattern References**:
  - Session launch command: `opencode --session <session_id>` (confirmed from Metis research)
  - Bubbletea ExecProcess pattern: `tea.ExecProcess(cmd, func(err error) tea.Msg { return sessionResumeMsg{err} })`

  **WHY Each Reference Matters**:
  - tea.Tick is the idiomatic Bubbletea way to do periodic tasks
  - tea.ExecProcess suspends Bubbletea, runs external command, resumes on exit - exactly the suspend/resume UX needed
  - LookPath validates opencode exists before constructing command

  **Acceptance Criteria**:
  - [ ] `go test ./internal/launcher/...` passes
  - [ ] Auto-refresh runs every 30s without losing cursor position
  - [ ] Enter on session row suspends dashboard and launches opencode
  - [ ] When opencode exits, dashboard resumes with data refresh
  - [ ] Enter on project header toggles collapse (no launch)
  - [ ] "Refreshing..." indicator appears briefly during refresh

  **QA Scenarios**:
  ```
  Scenario: Auto-refresh updates data
    Tool: interactive_bash (tmux)
    Preconditions: TUI running with real data
    Steps:
      1. Capture pane text, note current session list
      2. Navigate to a specific row (j/k 3 times)
      3. Capture pane text, note which session has `>` selection marker and its slug text
      4. Wait 35 seconds (for at least 1 auto-refresh cycle)
      5. Capture pane text again
      6. Assert the same session slug still has `>` marker (cursor position preserved)
      7. Assert data is still showing (not blank during refresh)
    Expected Result: Data refreshed, selection marker on same session slug
    Failure Indicators: `>` marker jumps to first row, blank pane during refresh
    Evidence: .sisyphus/evidence/task-18-auto-refresh.txt

  Scenario: Session launch suspends and resumes
    Tool: interactive_bash (tmux)
    Preconditions: TUI running, opencode in PATH
    Steps:
      1. Navigate to a session row
      2. Press Enter
      3. Wait 2s
      4. Capture pane - assert opencode is now visible (dashboard suspended)
      5. Exit opencode (Ctrl+C or quit command)
      6. Wait 2s
      7. Capture pane - assert dashboard has resumed, data refreshed
    Expected Result: Dashboard suspends for opencode, resumes on exit
    Failure Indicators: Dashboard doesn't suspend, doesn't resume, crashes on return
    Evidence: .sisyphus/evidence/task-18-session-launch.png

  Scenario: Launch fails gracefully without opencode
    Tool: Bash
    Preconditions: launcher_test.go
    Steps:
      1. Run `go test -v -run TestLaunchMissingBinary ./internal/launcher/...`
      2. Assert test passes with "opencode not found" error
    Expected Result: Clear error, no panic
    Failure Indicators: Panic, generic OS error
    Evidence: .sisyphus/evidence/task-18-missing-binary.txt
  ```

  **Commit**: YES
  - Message: `feat(app): auto-refresh every 30s and session launcher`
  - Files: `internal/ui/app.go (updated), internal/launcher/launch.go, internal/launcher/launch_test.go`
  - Pre-commit: `go build ./... && go test ./...`

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** - `oracle`

  **What to do**:
  Read `.sisyphus/plans/opencode-dashboard.md` end-to-end. Systematically verify every Must Have and Must NOT Have.

  **QA Scenarios**:
  ```
  Scenario: Must Have verification
    Tool: Bash + Read
    Steps:
      1. Read the plan's "Must Have" section
      2. For each item, verify via file read or command:
         - "Root session filtering": grep for `parent_id IS NULL` in internal/store/session.go
         - "All 5 attention signals": grep for each signal constant in internal/attention/classifier.go
         - "Read-only SQLite": grep for `mode=ro` in internal/store/db.go
         - "Split-pane layout": read internal/ui/layout.go, verify left+right pane composition
         - "Session launch": read internal/launcher/launch.go, verify opencode --session command
         - "Vim + arrow navigation": grep for j/k/h/l key handling in internal/ui/
         - "Filter presets": grep for NeedsAttention/AllActive/Archived in internal/filter/
         - "TDD": verify _test.go exists for every data layer file
         - "Auto-refresh": grep for tea.Tick or ticker in internal/ui/app.go
      3. Run `go build ./cmd/opencode-dashboard` - must exit 0
      4. Run `go test ./...` - must pass all tests
    Expected Result: Every Must Have maps to verified code
    Evidence: .sisyphus/evidence/f1-must-have-audit.txt

  Scenario: Must NOT Have enforcement
    Tool: Bash (grep)
    Steps:
      1. Search for write operations: grep -r "INSERT\|UPDATE\|DELETE\|CREATE TABLE" internal/store/ (excluding test files)
      2. Search for cobra/viper: grep -r "cobra\|viper" go.mod
      3. Search for message content display: grep -r "message.*content\|part.*text" internal/ui/
      4. Search for Bubbletea imports in data layer: grep -r "bubbletea\|lipgloss" internal/store/ internal/attention/ internal/filter/
      5. Search for configurable thresholds: grep -r "config\|viper\|flag.*threshold" internal/attention/
      6. Count evidence files: ls .sisyphus/evidence/ | wc -l (should be >= 18)
    Expected Result: All forbidden patterns absent, evidence files present
    Evidence: .sisyphus/evidence/f1-must-not-have-audit.txt
  ```
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** - `unspecified-high`

  **What to do**:
  Run all static analysis and review code quality across every file.

  **QA Scenarios**:
  ```
  Scenario: Build and test pipeline
    Tool: Bash
    Steps:
      1. Run `go build ./...` - assert exit 0
      2. Run `go vet ./...` - assert exit 0, no warnings
      3. Run `go test ./...` - capture output, assert all pass
      4. Run `go test -race ./...` - assert no race conditions
    Expected Result: All 4 commands pass cleanly
    Evidence: .sisyphus/evidence/f2-build-test.txt

  Scenario: Code quality scan
    Tool: Bash + Read
    Steps:
      1. Search for `as any` or type assertion without ok: grep -r "\.(" --include="*.go" | grep -v "_test.go" | grep -v ", ok"
      2. Search for empty error handling: ast_grep for `if err != nil { }` pattern
      3. Search for fmt.Println in non-test, non-main files: grep -rn "fmt.Print" internal/ --include="*.go" | grep -v "_test.go"
      4. Search for commented-out code blocks: grep -n "^[[:space:]]*//" internal/ --include="*.go" | wc -l (flag if > 20% of lines)
      5. Verify driver: grep "modernc.org/sqlite" go.mod (must be present), grep "go-sqlite3" go.mod (must be absent)
      6. Verify layer isolation: grep -r "bubbletea" internal/store/ internal/attention/ internal/filter/ internal/app/ (must be empty)
    Expected Result: No quality issues found, correct driver, clean layer boundaries
    Evidence: .sisyphus/evidence/f2-quality-scan.txt
  ```
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Race [PASS/FAIL] | Quality [N issues] | VERDICT`

- [ ] F3. **Real Manual QA** - `unspecified-high`

  **What to do**:
  Build from clean state and execute every major user flow via tmux.

  **QA Scenarios**:
  ```
  Scenario: Full user journey
    Tool: interactive_bash (tmux)
    Steps:
      1. Build: `go build -o /tmp/ocd ./cmd/opencode-dashboard`
      2. Create tmux: `new-session -d -s final-qa -x 120 -y 40`
      3. Launch: `send-keys -t final-qa "/tmp/ocd" Enter`
      4. Wait 3s, capture pane - assert project names visible, sessions listed
      5. Send j 3 times - assert cursor moves, right pane updates with session detail
      6. Send l - assert right pane border highlighted (active)
      7. Send h - assert left pane border highlighted
      8. Send Tab - assert filter preset changes (check label text)
      9. Send Tab again - assert next preset
      10. Send / then type "test" then Enter - assert filtered list
      11. Send Esc - assert filter cleared, full list returns
      12. Wait 35s - assert data refreshes (capture before/after, check no cursor jump)
      13. Navigate to a session, send Enter - assert dashboard suspends, opencode starts
      14. Exit opencode - assert dashboard resumes
    Expected Result: Every interaction works as specified
    Evidence: .sisyphus/evidence/final-qa/full-journey.png (multiple screenshots)

  Scenario: Edge cases
    Tool: interactive_bash (tmux)
    Steps:
      1. Resize terminal: `resize-pane -t final-qa -x 60 -y 20` - assert no crash, panes adjust
      2. Resize back: `resize-pane -t final-qa -x 120 -y 40`
      3. Send rapid keystrokes: j j j j j k k k k - assert no lag or crash
      4. Navigate to session with no todos - assert detail pane shows no todo section (no empty box)
      5. Navigate to session with errors - assert red attention icon in left pane AND error info in detail
      6. Check "Global" project is at bottom of list
    Expected Result: All edge cases handled gracefully
    Evidence: .sisyphus/evidence/final-qa/edge-cases.png

  Scenario: Error startup
    Tool: Bash
    Steps:
      1. Run `/tmp/ocd --db-path /nonexistent` - assert error message, no panic, clean exit
      2. Run `/tmp/ocd --help` - assert help text with db-path flag
    Expected Result: Graceful errors, useful help
    Evidence: .sisyphus/evidence/final-qa/error-startup.txt
  ```
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** - `deep`

  **What to do**:
  Verify 1:1 correspondence between plan spec and actual implementation.

  **QA Scenarios**:
  ```
  Scenario: Task-to-code mapping
    Tool: Bash + Read
    Steps:
      1. For each task 1-18, read "What to do" section from plan
      2. Read the actual files listed in the task's Commit section
      3. Verify every specified function/struct exists in the actual file
      4. Verify no functions/structs exist that aren't specified (scope creep)
      5. Check "Must NOT do" per task - search for forbidden patterns in task's files
    Expected Result: Every task's spec matches actual code 1:1
    Evidence: .sisyphus/evidence/f4-task-mapping.txt

  Scenario: Cross-task contamination check
    Tool: Bash
    Steps:
      1. Run `git log --oneline` to get commit list
      2. For each commit, run `git diff --name-only` to get changed files
      3. Cross-reference: each commit should only touch files listed in its task's Commit section
      4. Flag any file modified by multiple unrelated tasks
      5. Run `git diff --stat HEAD~18..HEAD` to find any files not mentioned in any task
    Expected Result: Clean task isolation, no unaccounted files
    Evidence: .sisyphus/evidence/f4-contamination.txt
  ```
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| # | Message | Files | Pre-commit |
|---|---------|-------|------------|
| 1 | `feat(scaffold): project init with git, go.mod, and directory structure` | .gitignore, go.mod, go.sum, cmd/opencode-dashboard/main.go, internal/ dirs | `git init && go build ./...` |
| 2 | `feat(types): domain types and store interfaces` | internal/domain/*.go | `go build ./...` |
| 3 | `feat(testdata): test fixture DB builder` | internal/testutil/*.go, internal/testutil/testdata/ | `go test ./internal/testutil/...` |
| 4 | `feat(store): SQLite connection layer with read-only mode` | internal/store/db.go, internal/store/db_test.go | `go test ./internal/store/...` |
| 5 | `feat(store): session and project repositories` | internal/store/session.go, session_test.go, project.go, project_test.go | `go test ./internal/store/...` |
| 6 | `feat(store): message metadata repository` | internal/store/message.go, internal/store/message_test.go | `go test ./internal/store/...` |
| 7 | `feat(store): todo repository with pending counts` | internal/store/todo.go, internal/store/todo_test.go | `go test ./internal/store/...` |
| 8 | `feat(store): error detection with caching` | internal/store/errors.go, internal/store/errors_test.go | `go test ./internal/store/...` |
| 9 | `feat(attention): attention signal classifier` | internal/attention/classifier.go, internal/attention/classifier_test.go | `go test ./internal/attention/...` |
| 10 | `feat(filter): filter and search engine` | internal/filter/filter.go, internal/filter/filter_test.go | `go test ./internal/filter/...` |
| 11 | `feat(app): data aggregator view model` | internal/app/aggregator.go, internal/app/aggregator_test.go | `go test ./internal/app/...` |
| 12 | `feat(ui): bubbletea app skeleton` | internal/ui/app.go | `go build ./...` |
| 13 | `feat(ui): project tree left pane` | internal/ui/sessionlist.go | `go build ./...` |
| 14 | `feat(ui): session detail right pane` | internal/ui/detail.go | `go build ./...` |
| 15 | `feat(ui): split layout with pane navigation` | internal/ui/layout.go | `go build ./...` |
| 16 | `feat(ui): wire data layer into TUI` | internal/ui/app.go (updated), cmd/opencode-dashboard/main.go (updated) | `go build ./... && go test ./...` |
| 17 | `feat(ui): filter bar with presets and search` | internal/ui/filterbar.go | `go build ./...` |
| 18 | `feat(app): auto-refresh and session launcher` | internal/ui/app.go (updated), internal/launcher/launch.go | `go build ./... && go test ./...` |

---

## Success Criteria

### Verification Commands
```bash
go build ./cmd/opencode-dashboard    # Expected: binary produced, exit 0
go test ./...                         # Expected: all tests pass
go vet ./...                          # Expected: no issues
./opencode-dashboard                  # Expected: TUI launches, shows sessions
./opencode-dashboard --db-path /nonexistent  # Expected: graceful error message
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Binary builds with no CGO dependency (pure Go)
- [ ] TUI launches and displays real session data
- [ ] All 5 attention signals correctly classified
- [ ] Filter presets work (Needs Attention, All Active, Archived)
- [ ] Session launch works (Enter key -> opencode opens)
- [ ] Auto-refresh runs without losing scroll position
