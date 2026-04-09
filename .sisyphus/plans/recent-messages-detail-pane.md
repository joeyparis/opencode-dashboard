# feat: Recent Messages in Session Detail Pane

## TL;DR

> **Quick Summary**: Add a "Recent Messages" section to the detail pane showing the last 5 messages with truncated content previews (~100 chars). Replaces the current single-message "Last Activity" section with richer context.
> 
> **Deliverables**:
> - `MessagePreview` domain type with role, agent, content snippet, timestamp
> - `GetRecentMessages()` store method querying message + part tables
> - Updated aggregator pre-fetching recent messages per session
> - `renderRecentMessages()` replacing `renderLastActivity()` in detail pane
> - Test fixtures with multi-message/multi-part data
> - Store-level tests for the new method
> 
> **Estimated Effort**: Short (5 focused tasks)
> **Parallel Execution**: YES - 2 waves
> **Critical Path**: Task 1 (verify part JSON + types) -> Task 2 (store impl) -> Task 4 (UI render)
> **GitHub Issue**: #2
> **Branch**: `feat/2-recent-messages-detail-pane`

---

## Context

### Original Request
Add the last 2-5 messages with content previews to the session detail pane.

### Interview Summary
**Key Discussions**:
- **Content display**: Truncated content previews (~80-120 chars), not just metadata
- **Loading strategy**: User initially chose lazy-load, but Metis review identified that pre-fetch in the aggregator is architecturally simpler and consistent with every other field in `SessionView`. Pre-fetch chosen - see rationale below.

**Loading Strategy Decision (Pre-fetch over Lazy Load)**:
The current architecture has ZERO precedent for async loading triggered by selection change. Every piece of data in the detail pane comes pre-populated in `SessionView` by the aggregator. Introducing lazy loading would require:
- A brand-new upward command flow pattern (detail -> layout -> app)
- Race condition handling for rapid j/k navigation
- Session ID matching to prevent stale data display
- Manual re-fetch after the 30s auto-refresh cycle
- Loading/empty states in the detail pane

Pre-fetch adds one `LIMIT 5` JOIN query per session to the aggregator - negligible on local SQLite - and automatically refreshes every 30s via the existing cycle. Zero new patterns needed.

### Metis Review
**Identified Gaps** (addressed):
- Part table JSON structure for text content is UNVERIFIED - added verification step as Task 1
- Content truncation must flatten newlines to spaces
- Messages with no text parts need graceful fallback
- Display width vs char count for truncation (resolved: use rune count, not display width - keep it simple)
- "Last Activity" section should be REPLACED, not supplemented

---

## Work Objectives

### Core Objective
Show the last 5 messages with truncated content previews in the session detail pane, replacing the current single-message "Last Activity" section.

### Concrete Deliverables
- `internal/domain/types.go` - `MessagePreview` struct
- `internal/domain/interfaces.go` - `GetRecentMessages` on `MessageStore`
- `internal/store/message.go` - SQL implementation
- `internal/store/message_test.go` - Tests for new method
- `internal/testutil/fixture.go` - Multi-message + text part fixtures
- `internal/app/aggregator.go` - Pre-fetch recent messages
- `internal/ui/detail.go` - `renderRecentMessages()` replacing `renderLastActivity()`

### Definition of Done
- [ ] `go test ./internal/... -v` passes with zero failures
- [ ] `go vet ./internal/...` reports zero warnings
- [ ] Detail pane shows up to 5 recent messages with truncated content
- [ ] Zero-message sessions render gracefully (section hidden)
- [ ] Messages with no text parts show `[tool use]` placeholder

### Must Have
- Content previews truncated to ~100 chars with `...` suffix
- Newlines in content replaced with spaces before truncation
- Leading/trailing whitespace trimmed
- Role label per message (`user` / `assistant`)
- Relative timestamps per message (existing `relativeTime()` pattern)
- Graceful handling of 0, 1, and 5+ message counts

### Must NOT Have (Guardrails)
- No loading spinner or loading states (pre-fetch, data is always present)
- No caching layer or mutex for messages (follow aggregator's existing pattern)
- No width-responsive truncation (use fixed 100-char constant)
- No different rendering per part type (text parts only, skip tool-invocation/error parts)
- No "show more messages" link or pagination
- No syntax highlighting or markdown rendering in previews
- No collapsible/expandable message previews
- No click-to-view-full-message interaction
- No message count badge in section header

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (Go test with `testutil.NewTestDB`)
- **Automated tests**: YES (tests-after, following existing pattern)
- **Framework**: `go test` with `testing` stdlib
- **Pattern**: Follow `store/message_test.go` style - `testutil.NewTestDB(t)`, direct assertions

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Store/Domain**: Use Bash (`go test`) - run tests, verify pass/fail counts
- **UI rendering**: Use Bash (`go build ./...`) - verify compilation, then interactive_bash (tmux) to visually verify TUI against a real OpenCode DB if available

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - types + fixtures):
├── Task 1: Verify part JSON structure + add domain types          [quick]
├── Task 2: Seed multi-message test fixtures with text parts       [quick]
└── Task 3: Add GetRecentMessages to MessageStore interface        [quick]

Wave 2 (After Wave 1 - implementation + UI):
├── Task 4: Implement GetRecentMessages store method + tests       [quick]
│           (depends: 1, 2, 3)
├── Task 5: Wire aggregator pre-fetch + update detail rendering    [unspecified-high]
│           (depends: 4)

Wave FINAL (After ALL tasks - 4 parallel reviews):
├── F1: Plan compliance audit                                      [oracle]
├── F2: Code quality review                                        [unspecified-high]
├── F3: Real manual QA                                             [unspecified-high]
└── F4: Scope fidelity check                                       [deep]
-> Present results -> Get explicit user okay
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1    | -         | 4      | 1    |
| 2    | -         | 4      | 1    |
| 3    | -         | 4      | 1    |
| 4    | 1, 2, 3   | 5      | 2    |
| 5    | 4         | F1-F4  | 2    |

### Agent Dispatch Summary

- **Wave 1**: 3 tasks - T1 `quick`, T2 `quick`, T3 `quick`
- **Wave 2**: 2 tasks - T4 `quick`, T5 `unspecified-high`
- **FINAL**: 4 tasks - F1 `oracle`, F2 `unspecified-high`, F3 `unspecified-high`, F4 `deep`

---

## TODOs

- [x] 1. Verify part table JSON structure + add MessagePreview domain type

  **What to do**:
  - Examine the real OpenCode database's `part` table to determine the exact JSON structure for text content parts. Run a query like:
    ```sql
    SELECT data FROM part WHERE json_extract(data, '$.type') = 'text' LIMIT 5;
    ```
    If no real DB is available, check the OpenCode source code (likely at `~/.opencode/` or check `cmd/opencode-dashboard/main.go` for the DB path pattern) to find the schema. The DB path pattern is `{project_worktree}/.opencode/state.db`.
  - Once the JSON structure is confirmed, add `MessagePreview` struct to `internal/domain/types.go`:
    ```go
    type MessagePreview struct {
        Role        string    // "user" or "assistant"
        Agent       string
        Content     string    // truncated content preview (~100 chars)
        TimeCreated time.Time
    }
    ```
  - Add `RecentMessages []MessagePreview` field to `SessionView` struct in `internal/domain/types.go`
  - Record the verified JSON path (e.g., `$.text`, `$.content`, `$.value`) in the commit message for downstream tasks

  **Must NOT do**:
  - Do not write the SQL query yet - just verify the JSON path and add types
  - Do not modify any store files

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small type additions to two files, plus a DB inspection query
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - None needed for simple type additions

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: Task 4
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/domain/types.go:66-73` - Existing `MessageMeta` struct pattern (role, agent, model, provider, time)
  - `internal/domain/types.go:165-180` - `SessionView` struct where `RecentMessages` field should be added (after `LastMessage`)

  **API/Type References**:
  - `internal/testutil/fixture.go:72-78` - Schema showing `part` table structure: `id, message_id, session_id, time_created, time_updated, data`
  - `internal/testutil/fixture.go:224-242` - Example of part JSON structure for tool-invocation type: `{"type": "tool-invocation", "state": {"status": "error"}}`

  **External References**:
  - OpenCode state DB location pattern: `{project_worktree}/.opencode/state.db` - look for any `.opencode/state.db` file on the filesystem

  **WHY Each Reference Matters**:
  - `MessageMeta` shows the existing field naming convention (Role, Agent, TimeCreated) - `MessagePreview` should match
  - `SessionView` is where the new `RecentMessages` field goes - add it near `LastMessage` for logical grouping
  - The fixture schema confirms `part.data` is a TEXT column with JSON - the exact JSON keys for text content need discovery
  - The part fixture shows the pattern for other part types - text parts likely follow `{"type": "text", ...}`

  **Acceptance Criteria**:
  - [ ] `MessagePreview` struct added to `internal/domain/types.go` with Role, Agent, Content, TimeCreated fields
  - [ ] `RecentMessages []MessagePreview` field added to `SessionView`
  - [ ] `go build ./...` compiles successfully
  - [ ] JSON path for text content parts documented (in code comment or commit message)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Types compile and are accessible
    Tool: Bash
    Preconditions: Working Go module
    Steps:
      1. Run `go build ./...`
      2. Run `go vet ./internal/domain/...`
    Expected Result: Exit code 0 for both commands, zero errors
    Failure Indicators: Compilation errors, undefined type references
    Evidence: .sisyphus/evidence/task-1-types-compile.txt

  Scenario: Part JSON structure verified
    Tool: Bash
    Preconditions: At least one `.opencode/state.db` exists on the filesystem
    Steps:
      1. Find a real state.db: `find ~/Sites -name "state.db" -path "*/.opencode/*" 2>/dev/null | head -1`
      2. Query text parts: `sqlite3 <db_path> "SELECT data FROM part WHERE json_extract(data, '$.type') = 'text' LIMIT 3;"`
      3. If no text parts found, try: `sqlite3 <db_path> "SELECT DISTINCT json_extract(data, '$.type') FROM part LIMIT 20;"`
    Expected Result: JSON structure revealed showing the field name for text content
    Failure Indicators: No state.db found, no text parts exist, unexpected JSON structure
    Evidence: .sisyphus/evidence/task-1-part-json-structure.txt
  ```

  **Commit**: YES (group 1)
  - Message: `feat(domain): add MessagePreview type and GetRecentMessages interface`
  - Files: `internal/domain/types.go`
  - Pre-commit: `go build ./...`

- [x] 2. Seed multi-message test fixtures with text parts

  **What to do**:
  - Update `internal/testutil/fixture.go` `seedData()` to insert multiple messages per session, with text parts containing actual content. Target sessions:
    - `ses-needs-response`: 5+ messages (user/assistant alternating) with text parts - tests the "full list" case
    - `ses-active-now`: 2 messages with text parts - tests the "fewer than limit" case
    - `ses-has-errors`: Keep existing messages + add a message with ONLY tool-invocation parts (no text) - tests the "no text content" case
    - `ses-archived`: Keep at 0 messages - tests the "empty" case
  - Each text part should have realistic-ish content strings of varying lengths:
    - Some short (< 100 chars) - no truncation needed
    - Some long (> 100 chars) - truncation required
    - Some with newlines - flattening required
    - Some with leading/trailing whitespace - trimming required
  - Use the verified JSON path from Task 1 for text part data structure. If Task 1 found `$.text`, then: `{"type": "text", "text": "content here"}`
  - Update existing test assertions in `message_test.go` if message counts changed (e.g., `ses-needs-response` count changes from 1 to 5+)

  **Must NOT do**:
  - Do not add any new test functions yet (that's Task 4)
  - Do not modify the schema - only add seed data
  - Keep existing fixture data intact - only ADD to it

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Fixture data seeding following established patterns
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Task 4
  - **Blocked By**: None (but should use Task 1's verified JSON path if available; if not, use best guess `{"type": "text", "text": "..."}` and adjust later)

  **References**:

  **Pattern References**:
  - `internal/testutil/fixture.go:189-222` - Existing message seeding pattern: map of sessionID -> message info, JSON marshaling, INSERT with time_created as UnixMilli
  - `internal/testutil/fixture.go:224-242` - Existing part seeding pattern for error parts: JSON with `type` and nested fields

  **Test References**:
  - `internal/store/message_test.go:70-83` - `TestGetMessageCount` asserts count=1 for `ses-needs-response` - THIS WILL BREAK if we add more messages. Must update assertion.
  - `internal/store/message_test.go:10-35` - `TestGetLastMessageMeta_AssistantRole` checks role/agent/model of last message for `ses-needs-response` - ensure the LAST seeded message still has role=assistant, agent=Sisyphus, model=claude-haiku-4-5

  **WHY Each Reference Matters**:
  - Message seeding pattern shows the exact INSERT format and JSON structure to follow
  - Part seeding pattern shows how to create parts linked to messages via `message_id`
  - The existing test assertions are fragile to fixture changes - must preserve the last message's properties and update count assertions

  **Acceptance Criteria**:
  - [ ] `ses-needs-response` has 5+ messages with text parts
  - [ ] `ses-active-now` has 2+ messages with text parts
  - [ ] `ses-has-errors` has at least one message with only tool-invocation parts (no text)
  - [ ] `ses-archived` still has 0 messages
  - [ ] Text part content includes: short strings, long strings (>100 chars), strings with newlines, strings with whitespace
  - [ ] `go test ./internal/... -v` passes - all existing tests still work
  - [ ] `TestGetMessageCount` for `ses-needs-response` updated to expect new count

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: All existing tests pass with updated fixtures
    Tool: Bash
    Preconditions: Fixtures updated with new seed data
    Steps:
      1. Run `go test ./internal/... -v`
      2. Verify zero FAIL lines in output
      3. Verify `TestGetLastMessageMeta_AssistantRole` still passes (last message still assistant/Sisyphus)
    Expected Result: All tests PASS, zero failures
    Failure Indicators: Any FAIL in test output, especially message_test.go tests
    Evidence: .sisyphus/evidence/task-2-existing-tests-pass.txt

  Scenario: Fixture data is queryable
    Tool: Bash
    Preconditions: Test DB created with new fixtures
    Steps:
      1. Write a quick Go test that creates the test DB and counts messages per session:
         `SELECT session_id, COUNT(*) FROM message GROUP BY session_id`
      2. Verify ses-needs-response has 5+, ses-active-now has 2+, ses-archived has 0
      3. Verify text parts exist: `SELECT COUNT(*) FROM part WHERE json_extract(data, '$.type') = 'text'`
    Expected Result: Expected counts per session, text parts seeded
    Failure Indicators: Wrong counts, zero text parts
    Evidence: .sisyphus/evidence/task-2-fixture-counts.txt
  ```

  **Commit**: YES (group 2)
  - Message: `test(fixture): seed multi-message fixtures with text parts`
  - Files: `internal/testutil/fixture.go`, `internal/store/message_test.go`
  - Pre-commit: `go test ./internal/store/... -v`

- [x] 3. Add GetRecentMessages to MessageStore interface

  **What to do**:
  - Add `GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]domain.MessagePreview, error)` to the `MessageStore` interface in `internal/domain/interfaces.go`
  - This is interface-only - no implementation yet

  **Must NOT do**:
  - Do not implement the method on `MessageRepo` yet (Task 4)
  - Do not modify any other interfaces

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single line addition to an interface
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: Task 4
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/domain/interfaces.go:16-19` - `MessageStore` interface with existing methods `GetLastMessageMeta` and `GetMessageCount`

  **API/Type References**:
  - `internal/domain/types.go` - `MessagePreview` type added in Task 1 (the return type)

  **WHY Each Reference Matters**:
  - The interface shows the exact signature pattern: `ctx context.Context` first, session ID second, return `(type, error)`
  - Method should follow naming convention: `Get` prefix, descriptive name

  **Acceptance Criteria**:
  - [ ] `GetRecentMessages` method added to `MessageStore` interface
  - [ ] Signature matches: `GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]domain.MessagePreview, error)`
  - [ ] NOTE: `go build` will FAIL after this because `MessageRepo` doesn't implement the new method yet - this is expected and will be fixed in Task 4. Verify with `go vet ./internal/domain/...` instead.

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Interface is syntactically valid
    Tool: Bash
    Preconditions: Method added to interface
    Steps:
      1. Run `go vet ./internal/domain/...`
    Expected Result: Zero errors from domain package
    Failure Indicators: Syntax errors, invalid type references
    Evidence: .sisyphus/evidence/task-3-interface-valid.txt
  ```

  **Commit**: YES (group with Task 1)
  - Message: `feat(domain): add MessagePreview type and GetRecentMessages interface`
  - Files: `internal/domain/interfaces.go` (combined with Task 1's types.go)
  - Pre-commit: `go vet ./internal/domain/...`

- [x] 4. Implement GetRecentMessages store method + tests

  **What to do**:
  - Implement `GetRecentMessages` on `MessageRepo` in `internal/store/message.go`
  - SQL query should:
    1. Select the N most recent messages for a session (ORDER BY time_created DESC LIMIT ?)
    2. For each message, extract the first text part's content via a subquery on the `part` table
    3. Use `json_extract(m.data, '$.role')` for role and `json_extract(m.data, '$.agent')` for agent (following existing pattern at line 23-28)
    4. Use the verified JSON path from Task 1 for text content extraction from parts
    5. Truncate content in Go code (not SQL) - `SUBSTR` in SQLite doesn't handle rune boundaries well
  - Content processing in Go (after SQL fetch):
    1. Replace all newlines (`\n`, `\r\n`, `\r`) with single space
    2. Trim leading/trailing whitespace
    3. If content length > 100 runes, truncate to 100 runes and append `...`
    4. If no text parts found for a message, set Content to `[tool use]`
  - Define a constant `maxPreviewLen = 100` in `message.go`
  - Return messages in reverse chronological order (newest first) - same as the SQL ORDER BY
  - Add compile-time interface check: the existing `var _ domain.MessageStore = (*MessageRepo)(nil)` at line 12 will enforce this
  - Write tests in `internal/store/message_test.go`:
    - `TestGetRecentMessages_MultipleMessages` - ses-needs-response has 5+ messages, request limit 5, verify count and order
    - `TestGetRecentMessages_NoMessages` - ses-archived, expect empty slice (not nil)
    - `TestGetRecentMessages_SingleMessage` - use a session with exactly 1 message
    - `TestGetRecentMessages_ContentTruncation` - verify long content is truncated to 100 chars + `...`
    - `TestGetRecentMessages_NewlineFlattening` - verify newlines replaced with spaces
    - `TestGetRecentMessages_NoTextParts` - ses-has-errors tool-only message, expect `[tool use]` content

  **Must NOT do**:
  - Do not use SQLite `SUBSTR` for truncation - handle in Go for proper rune support
  - Do not add a cache or mutex
  - Do not modify the schema or add indexes (existing `message_session_time_created_id_idx` covers the query)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single store method with SQL query + Go processing, following established patterns exactly
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (sequential after Wave 1)
  - **Blocks**: Task 5
  - **Blocked By**: Tasks 1, 2, 3

  **References**:

  **Pattern References**:
  - `internal/store/message.go:22-60` - `GetLastMessageMeta` SQL query pattern: `json_extract` on `data` column, `COALESCE` for NULL safety, `time.UnixMilli().UTC()` for timestamp conversion, `sql.ErrNoRows` handling
  - `internal/store/message_test.go:10-35` - Test pattern: `testutil.NewTestDB(t)`, `defer db.Close()`, `NewMessageRepo(db)`, direct field assertions

  **API/Type References**:
  - `internal/domain/types.go` - `MessagePreview` struct (from Task 1) - the return type
  - `internal/domain/interfaces.go` - `GetRecentMessages` signature (from Task 3)

  **Test References**:
  - `internal/store/message_test.go` - All existing test patterns to follow
  - `internal/testutil/fixture.go` - Fixture data from Task 2 determines expected test values

  **External References**:
  - SQLite JSON functions: `json_extract(column, '$.path')` - https://www.sqlite.org/json1.html

  **WHY Each Reference Matters**:
  - `GetLastMessageMeta` is the exact pattern to extend - same table, same JSON extraction, similar query structure
  - Test patterns show the exact setup/teardown and assertion style to follow
  - The fixture data from Task 2 determines what values the tests should expect

  **Acceptance Criteria**:
  - [ ] `GetRecentMessages` implemented on `MessageRepo`
  - [ ] Compile-time interface check passes (existing `var _` line)
  - [ ] 6 new tests written and passing
  - [ ] `go test ./internal/store/... -v -run TestGetRecentMessages` shows 6 PASS
  - [ ] Content truncation at 100 runes + `...` verified by test
  - [ ] Newline flattening verified by test
  - [ ] `[tool use]` fallback verified by test
  - [ ] `go test ./internal/... -v` all pass (no regressions)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: All GetRecentMessages tests pass
    Tool: Bash
    Preconditions: Store method implemented, fixtures seeded (Tasks 1-3 complete)
    Steps:
      1. Run `go test ./internal/store/... -v -run TestGetRecentMessages`
      2. Count PASS lines - expect 6
      3. Count FAIL lines - expect 0
    Expected Result: 6 tests pass, 0 failures
    Failure Indicators: Any FAIL, unexpected test count, compilation errors
    Evidence: .sisyphus/evidence/task-4-store-tests.txt

  Scenario: No regressions in existing tests
    Tool: Bash
    Preconditions: All changes from Tasks 1-4 applied
    Steps:
      1. Run `go test ./internal/... -v`
      2. Verify zero FAIL lines
      3. Verify test count >= previous count (no tests removed)
    Expected Result: All existing + new tests PASS
    Failure Indicators: Any FAIL line, reduced test count
    Evidence: .sisyphus/evidence/task-4-no-regressions.txt

  Scenario: Empty session returns empty slice (not nil)
    Tool: Bash
    Preconditions: TestGetRecentMessages_NoMessages test exists
    Steps:
      1. Run `go test ./internal/store/... -v -run TestGetRecentMessages_NoMessages`
      2. Verify PASS
    Expected Result: Returns empty []MessagePreview{}, not nil
    Failure Indicators: nil pointer, FAIL
    Evidence: .sisyphus/evidence/task-4-empty-session.txt
  ```

  **Commit**: YES (group 3)
  - Message: `feat(store): implement GetRecentMessages with content preview`
  - Files: `internal/store/message.go`, `internal/store/message_test.go`
  - Pre-commit: `go test ./internal/store/... -v`

- [x] 5. Wire aggregator pre-fetch + update detail pane rendering

  **What to do**:
  - **Aggregator** (`internal/app/aggregator.go`):
    - In the `LoadAll` loop (lines 73-108), add a call to fetch recent messages:
      ```go
      recentMsgs, _ := a.messages.GetRecentMessages(ctx, s.ID, 5)
      ```
    - Add `RecentMessages: recentMsgs` to the `SessionView` construction (around line 92-105)
    - The `_ =` error handling follows the existing pattern for non-critical data (same as todos, lastMsg, msgCount)
  - **Aggregator test** (`internal/app/aggregator_test.go`):
    - Update the mock `MessageStore` to implement `GetRecentMessages` (return empty slice)
    - Verify the aggregator passes through the recent messages to `SessionView`
  - **Detail pane** (`internal/ui/detail.go`):
    - Replace `renderLastActivity()` with `renderRecentMessages()` in the `View()` sections array (line 72)
    - Remove the `renderLastActivity()` method entirely (lines 182-203)
    - Implement `renderRecentMessages()`:
      - If `m.session.RecentMessages` is empty, return `""` (section hidden)
      - Section header: `sectionHeaderStyle.Render("Recent Messages")`
      - For each message, render a line like:
        ```
          user  3m ago  Can you help me fix the login bug?
          asst  2m ago  I'll look into the login handler...
        ```
      - Role should be left-aligned and abbreviated: `user` / `asst` (4 chars each for alignment)
      - Use `relativeTime()` for timestamps (existing helper)
      - Content on the same line, after timestamp
      - If the detail pane is narrow and content wraps, that's fine - lipgloss handles it

  **Must NOT do**:
  - Do not add loading states or spinners
  - Do not add lazy loading mechanism
  - Do not modify the `SetSession` method signature
  - Do not add width-responsive truncation
  - Do not add message count badges
  - Do not add collapsible sections

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Touches 3+ files across aggregator and UI layers, needs careful integration
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (after Task 4)
  - **Blocks**: F1-F4
  - **Blocked By**: Task 4

  **References**:

  **Pattern References**:
  - `internal/app/aggregator.go:73-108` - The per-session loop where all data is fetched. Add `GetRecentMessages` call alongside existing `GetLastMessageMeta`, `GetMessageCount`, etc.
  - `internal/ui/detail.go:182-203` - `renderLastActivity()` - the method being REPLACED. Shows section header + metadata line rendering pattern.
  - `internal/ui/detail.go:67-73` - `View()` sections array where `renderLastActivity` needs to become `renderRecentMessages`
  - `internal/ui/detail.go:165-179` - `renderTodos()` - similar list-rendering pattern with icons and formatted lines
  - `internal/ui/styles.go` - `sectionHeaderStyle` and other style definitions

  **API/Type References**:
  - `internal/domain/types.go` - `SessionView.RecentMessages []MessagePreview` (from Task 1)
  - `internal/domain/types.go` - `MessagePreview` struct fields: Role, Agent, Content, TimeCreated

  **Test References**:
  - `internal/app/aggregator_test.go` - Existing mock-based aggregator tests. The mock `MessageStore` needs to implement the new `GetRecentMessages` method.

  **WHY Each Reference Matters**:
  - The aggregator loop shows exactly WHERE to add the fetch call and HOW to handle errors (ignore with `_`)
  - `renderLastActivity` is the method being replaced - understand its structure before removing
  - `renderTodos` shows how to render a list of items with formatting - similar pattern for messages
  - The aggregator test mock must be updated or tests will fail to compile

  **Acceptance Criteria**:
  - [ ] Aggregator fetches recent messages per session in `LoadAll`
  - [ ] `SessionView.RecentMessages` populated with up to 5 messages
  - [ ] `renderLastActivity()` removed entirely from detail.go
  - [ ] `renderRecentMessages()` renders messages with role, relative time, and content preview
  - [ ] Empty `RecentMessages` results in section being hidden (not rendered)
  - [ ] Aggregator test mock updated, all aggregator tests pass
  - [ ] `go test ./internal/... -v` all pass
  - [ ] `go build ./cmd/...` compiles successfully

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Full build + test pass
    Tool: Bash
    Preconditions: All Tasks 1-5 complete
    Steps:
      1. Run `go build ./cmd/...`
      2. Run `go test ./internal/... -v`
      3. Run `go vet ./internal/...`
    Expected Result: All three commands exit 0, zero FAIL lines in test output
    Failure Indicators: Build errors, test failures, vet warnings
    Evidence: .sisyphus/evidence/task-5-full-build-test.txt

  Scenario: Detail pane renders messages correctly (visual inspection via build)
    Tool: Bash
    Preconditions: Binary builds successfully
    Steps:
      1. Run `go build -o /tmp/ocd-test ./cmd/opencode-dashboard/`
      2. If a real OpenCode state.db exists, run: `/tmp/ocd-test --db <path-to-state.db>` in tmux
      3. Navigate to a session with messages using j/k keys
      4. Verify the detail pane shows "Recent Messages" section with role labels and content
      5. Capture screenshot or terminal output
    Expected Result: "Recent Messages" section visible with formatted message previews
    Failure Indicators: Missing section, garbled output, panic
    Evidence: .sisyphus/evidence/task-5-detail-pane-render.png

  Scenario: Zero-message session hides Recent Messages section
    Tool: Bash
    Preconditions: Binary builds, navigating to archived session
    Steps:
      1. Navigate to an archived/empty session
      2. Verify "Recent Messages" section is NOT shown
    Expected Result: Section absent, no empty header or blank space
    Failure Indicators: Empty "Recent Messages" header with no content below
    Evidence: .sisyphus/evidence/task-5-zero-messages.png
  ```

  **Commit**: YES (group 4)
  - Message: `feat(ui): show recent messages in detail pane`
  - Files: `internal/app/aggregator.go`, `internal/app/aggregator_test.go`, `internal/ui/detail.go`
  - Pre-commit: `go test ./internal/... -v`

---

## Final Verification Wave (MANDATORY - after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [x] F1. **Plan Compliance Audit** - `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns - reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** - `unspecified-high`
  Run `go vet ./internal/...` + `go test ./internal/... -v`. Review all changed files for: type assertion errors, unhandled errors, empty string comparisons vs len checks, hardcoded magic numbers. Check AI slop: excessive comments, over-abstraction, generic variable names.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** - `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task - follow exact steps, capture evidence. Test cross-task integration (messages appearing in detail pane with correct content). Test edge cases: zero messages, single message, tool-only messages. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** - `deep`
  For each task: read "What to do", read actual diff (`git diff main`). Verify 1:1 - everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| # | Message | Files | Pre-commit |
|---|---------|-------|------------|
| 1 | `feat(domain): add MessagePreview type and GetRecentMessages interface` | types.go, interfaces.go | `go build ./...` |
| 2 | `test(store): seed multi-message fixtures with text parts` | fixture.go, fixture_test.go | `go test ./internal/testutil/...` |
| 3 | `feat(store): implement GetRecentMessages with content preview` | message.go, message_test.go | `go test ./internal/store/...` |
| 4 | `feat(ui): show recent messages in detail pane` | aggregator.go, detail.go | `go test ./internal/...` |

---

## Success Criteria

### Verification Commands
```bash
go test ./internal/... -v        # Expected: all PASS, 0 failures
go vet ./internal/...            # Expected: zero warnings  
go build ./cmd/...               # Expected: successful build
```

### Final Checklist
- [ ] All "Must Have" items present and verified
- [ ] All "Must NOT Have" items absent (no loading spinners, no caching, no responsive truncation)
- [ ] All existing tests still pass
- [ ] New tests cover: multi-message, zero-message, single-message, no-text-parts scenarios
- [ ] Detail pane renders recent messages section correctly
- [ ] GitHub issue #2 acceptance criteria met
