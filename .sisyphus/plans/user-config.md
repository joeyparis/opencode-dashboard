# User Config File System (Issue #1)

## TL;DR

> **Quick Summary**: Add an optional TOML config file (`~/.config/opencode-dashboard/config.toml`) so users can customize keybindings, default filters/time windows, attention thresholds, and display colors/icons without recompiling. All current behavior is preserved when no config file exists.
> 
> **Deliverables**:
> - New `internal/config/` package (Config struct, TOML loading, KeyMap, validation)
> - Configurable attention thresholds in `internal/attention/`
> - TimeWindow enum replaced with config-driven `[]TimeWindowOption` slice
> - All UI components wired to accept config (keys, icons, colors, defaults, split ratio)
> - `--config` flag in main.go
> - Full TDD test coverage for config loading, key matching, threshold application
> 
> **Estimated Effort**: Medium-Large
> **Parallel Execution**: YES - 4 waves
> **Critical Path**: Task 1 -> Task 2 -> Task 5 -> Tasks 7-11 -> Task 12 -> F1-F4

---

## Context

### Original Request
GitHub issue #1 on joeyparis/opencode-dashboard: Add a TOML config file for customizing the Go/Bubbletea TUI dashboard. The dashboard reads OpenCode's SQLite database and shows session status with attention signals. Currently all keybindings, thresholds, colors, and defaults are hardcoded.

### Interview Summary
**Key Discussions**:
- TimeWindow enum replacement: User confirmed full replacement with config-driven `[]TimeWindowOption{Label, Duration}` slice, dropping the iota enum entirely
- TOML library: `github.com/BurntSushi/toml` (user's choice from issue)
- Config flows as a struct through constructors - no globals
- TDD approach with existing test infrastructure (go test + testify/assert)

**Research Findings**:
- Codebase has 11 existing test files, all using testify/assert with table-driven patterns
- Aggregator already stores classify as `func(SessionView, time.Time) AttentionSignal` - supports closure injection for configurable thresholds with zero struct changes
- All UI constructors are minimal-arg value-returning types - clean extension points
- Two parallel icon functions exist (`attentionIcon` colored + `attentionIconPlain` uncolored) - both need DisplayConfig
- `matchesWindow()` in filter.go calls `time.Now()` directly (untestable) - will fix as part of TimeWindow replacement
- No existing UI tests - constructor signature changes are safe from test breakage

### Metis Review
**Identified Gaps** (addressed):
- `filter.Filter.Window` type after enum removal: resolved as `time.Duration` (simplest, UI resolves option to duration before building Filter)
- `matchesWindow()` testability: will accept `now time.Time` parameter (trivial scope addition, improves test quality)
- XDG path resolution: follow issue spec (`~/.config/` default), not `os.UserConfigDir()` which returns `~/Library/Application Support` on macOS
- Explicit `--config path` with missing file = error (different from silent fallback when no path specified)
- Space key TOML representation: accept `"space"` alias, normalize to `" "` internally
- TOML slice replacement semantics: BurntSushi/toml replaces slices entirely (user-defined windows replace all defaults, not merge). Document and test.
- Unknown TOML keys: use `MetaData.Undecoded()` to warn on stderr, not error
- `attentionIconPlain()` still needed for selected-row rendering - DisplayConfig provides raw icon string, styled variant wraps with color

---

## Work Objectives

### Core Objective
Add a user-facing TOML configuration file that controls keybindings, attention thresholds, display icons/colors, default filter/time settings, refresh interval, and pane layout - while preserving identical behavior when no config file is present.

### Concrete Deliverables
- `internal/config/config.go` - Config struct with TOML tags, Default(), Load(), ResolvePath(), Validate()
- `internal/config/keys.go` - KeyMap struct with Matches() helper
- `internal/config/config_test.go` - Full loading/merging/validation tests
- `internal/config/keys_test.go` - KeyMap matching tests
- Modified `internal/attention/classifier.go` with Thresholds struct + ClassifyWith()
- Modified `internal/domain/types.go` - TimeWindowOption replaces TimeWindow enum
- Modified `internal/filter/filter.go` - Duration-based window matching with `now` parameter
- Modified UI files (app.go, layout.go, sessionlist.go, filterbar.go, styles.go) - config-driven keys/icons/colors/defaults
- Modified `cmd/opencode-dashboard/main.go` - `--config` flag, config loading, type bridging
- Updated `go.mod` with `github.com/BurntSushi/toml`

### Definition of Done
- [ ] `go test ./...` passes (all existing + new tests)
- [ ] `go build ./cmd/opencode-dashboard` succeeds
- [ ] `go vet ./...` clean
- [ ] Running without config file produces identical behavior to current
- [ ] Running with partial config merges onto defaults correctly
- [ ] Parse errors produce clear messages with filename and exit non-zero

### Must Have
- Config loaded once at startup, immutable after Load() returns
- Missing config file = silent fallback to defaults (no error)
- Parse error = clear error message to stderr and os.Exit(1)
- `--config` flag with explicit path that doesn't exist = error (different from silent fallback)
- All `[defaults]` fields functional (filter, time_window, refresh_interval, list_width_ratio)
- All `[thresholds]` fields functional (active_now_minutes, needs_response_hours, stale_work_days)
- `[time_windows]` customization works, cycle order matches list order, "All" always appended last
- All `[keys]` rebindings work; multiple keys per action supported via `[]string`
- All `[display]` icons and colors work for the 6 attention signals + header color
- Backward-compatible `Classify()` wrapper alongside new `ClassifyWith()`
- `"space"` alias normalized to `" "` in key config
- Validation: refresh >= 10s, ratio 0.2-0.8, no empty required bindings (quit)

### Must NOT Have (Guardrails)
- No config hot-reload - restart required for config changes
- No config subcommands (`--dump-config`, `--validate-config`, `--generate-config`)
- No environment variable overrides (only `--config` flag and TOML file)
- No per-project config - one global config file
- FilterPreset is NOT configurable (only its default selection via `[defaults].filter`)
- No new dependencies beyond `github.com/BurntSushi/toml`
- Store layer (`internal/store/`) has ZERO file modifications
- `internal/config` must NOT import any package from this module (`internal/domain`, `internal/ui`, `internal/store`, `internal/app`) - it defines its own types (e.g., `config.TimeWindowOption{Label, Hours}`) and conversion to domain types happens in `internal/ui/app.go` (which imports both `config` and `domain`). Attention threshold conversion happens in `main.go` (which imports `config` and `attention`).
- Aggregator struct and `NewAggregator()` 6-interface signature remain UNCHANGED. Add a separate `NewAggregatorWithClassifier()` for config-driven classification.. Add a separate `NewAggregatorWithClassifier(classify, repos...)` constructor for config-driven classification. Do NOT modify the existing `NewAggregator()` signature.
- Do NOT make todo priority colors or diff colors configurable. The `ColorHeader` accent color (`#7D56F4`) IS configurable and applies everywhere it's currently used: app header bar, active preset highlight, pane border accent, and `sectionHeaderStyle` in styles.go. These all share one color via `ColorHeader`, not separate config fields.
- No hex color validator - let lipgloss handle invalid colors (falls back to terminal default)
- Do NOT change the `attention.Classify()` function signature - add ClassifyWith() alongside it

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (11 test files, testify/assert, go test)
- **Automated tests**: TDD - write tests first for config package, tests alongside for UI wiring
- **Framework**: `go test` with `github.com/stretchr/testify/assert`
- **Pattern**: Follow existing style - `fixedNow()` helper for deterministic time, table-driven tests, testify/assert

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Config package**: Use Bash (`go test -run TestName ./internal/config/`) for TDD red-green cycles
- **Attention/Filter**: Use Bash (`go test ./internal/attention/ ./internal/filter/`)
- **UI wiring**: Use Bash (`go build ./...` + `go test ./...`) for compilation and regression checks
- **Integration**: Use interactive_bash (tmux) to launch the TUI with and without config files

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - foundation):
├── Task 1: Add BurntSushi/toml dependency [quick]
├── Task 2: Config package - struct, defaults, loading, validation [deep]
├── Task 3: Config package - KeyMap struct and Matches helper [quick]
└── Task 4: Attention - Thresholds struct + ClassifyWith [quick]

Wave 2 (After Wave 1 - type transitions + wiring prep):
├── Task 5: TimeWindow enum replacement (domain + filter + filterbar) [deep]
├── Task 6: Display config helpers (styles.go) [quick]
└── (Task 4 has no wave-2 dependents - just needs to be done before Task 12)

Wave 3 (After Wave 2 - UI wiring, MAX PARALLEL):
├── Task 7: Wire config into FilterBarModel [unspecified-high]
├── Task 8: Wire config into SessionListModel [quick]
├── Task 9: Wire config into LayoutModel [quick]
├── Task 10: Wire config into AppModel (keys, refresh, footer) [unspecified-high]
└── Task 11: Wire display config into filter bar legend [quick]

Wave 4 (After Wave 3 - integration):
└── Task 12: main.go - flag, load, bridge types, pass through [deep]

Wave FINAL (After ALL tasks - 4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Task 1 -> Task 2 -> Task 5 -> Tasks 7-11 -> Task 12 -> F1-F4 -> user okay
Parallel Speedup: ~55% faster than sequential
Max Concurrent: 5 (Wave 3)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | - | 2, 3 | 1 |
| 2 | 1 | 5, 6, 7, 8, 9, 10, 11, 12 | 1 |
| 3 | 1 | 7, 8, 9, 10, 12 | 1 |
| 4 | - | 12 | 1 |
| 5 | 2 | 7, 10, 11, 12 | 2 |
| 6 | 2 | 7, 8, 11, 12 | 2 |
| 7 | 2, 3, 5, 6 | 12 | 3 |
| 8 | 2, 3, 6 | 12 | 3 |
| 9 | 2, 3, 5 | 12 | 3 |
| 10 | 2, 3, 5 | 12 | 3 |
| 11 | 2, 5, 6 | 12 | 3 |
| 12 | ALL 1-11 | F1-F4 | 4 |
| F1-F4 | 12 | user okay | FINAL |

### Agent Dispatch Summary

- **Wave 1**: 4 tasks - T1 `quick`, T2 `deep`, T3 `quick`, T4 `quick`
- **Wave 2**: 2 tasks - T5 `deep`, T6 `quick`
- **Wave 3**: 5 tasks - T7 `unspecified-high`, T8 `quick`, T9 `quick`, T10 `unspecified-high`, T11 `quick`
- **Wave 4**: 1 task - T12 `deep`
- **FINAL**: 4 tasks - F1 `oracle`, F2 `unspecified-high`, F3 `unspecified-high`, F4 `deep`

---

## TODOs

- [x] 1. Add BurntSushi/toml dependency

  **What to do**:
  - Run `go get github.com/BurntSushi/toml` in the project root
  - Verify `go.mod` and `go.sum` are updated
  - Verify `go build ./...` still compiles

  **Must NOT do**:
  - Do not add any other dependencies
  - Do not modify any source files

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single command, trivial verification
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: Tasks 2, 3
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `go.mod` - current module declaration and dependencies

  **External References**:
  - https://pkg.go.dev/github.com/BurntSushi/toml - TOML library for Go

  **Acceptance Criteria**:

  ```
  Scenario: BurntSushi/toml dependency added successfully
    Tool: Bash
    Preconditions: Working directory is /Users/joey/Sites/opencode-dashboard
    Steps:
      1. Run: go get github.com/BurntSushi/toml
      2. Run: grep 'BurntSushi/toml' go.mod
      3. Run: go build ./...
    Expected Result: grep finds "github.com/BurntSushi/toml" in go.mod; build succeeds with exit code 0
    Failure Indicators: grep returns empty; build fails
    Evidence: .sisyphus/evidence/task-1-toml-dep.txt
  ```

  **Commit**: YES
  - Message: `chore: add BurntSushi/toml dependency`
  - Files: `go.mod`, `go.sum`
  - Pre-commit: `go build ./...`

- [x] 2. Config package - struct, defaults, loading, validation (TDD)

  **What to do**:
  - Create `internal/config/config.go` with the full Config struct hierarchy:
    ```go
    type Config struct {
        Defaults    DefaultsConfig    `toml:"defaults"`
        Thresholds  ThresholdsConfig  `toml:"thresholds"`
        TimeWindows TimeWindowsConfig `toml:"time_windows"`
        Keys        KeysConfig        `toml:"keys"`
        Display     DisplayConfig     `toml:"display"`
    }
    ```
  - **DefaultsConfig**: `Filter string` (default `"needs_attention"`), `TimeWindow string` (default `"3d"`), `RefreshInterval int` (default `30`, minimum 10), `ListWidthRatio float64` (default `0.5`, range 0.2-0.8)
  - **ThresholdsConfig**: `ActiveNowMinutes int` (default `5`), `NeedsResponseHours int` (default `24`), `StaleWorkDays int` (default `7`)
  - **TimeWindowsConfig**: `Options []TimeWindowOption` where `TimeWindowOption{Label string, Hours int}`. Default: `[{1d,24}, {3d,72}, {7d,168}]`
  - **KeysConfig**: One `[]string` field per action with TOML tags. Defaults must EXACTLY match current hardcoded strings:
    - `Up: ["k", "up"]`, `Down: ["j", "down"]`, `PaneLeft: ["h"]`, `PaneRight: ["l", "right"]`
    - `TreeNav: ["left"]`, `Collapse: [" "]`, `CycleFilter: ["tab"]`, `CycleTime: ["t"]`
    - `Search: ["/"]`, `Refresh: ["r"]`, `Launch: ["enter"]`, `Quit: ["q", "ctrl+c"]`
  - **DisplayConfig**: Icon fields (`IconError string` default `"!"`, `IconWaiting` `"@"`, `IconActive` `"*"`, `IconReply` `"?"`, `IconStale` `"~"`, `IconTodos` `"."`) and color fields (`ColorError` `"#FF0000"`, `ColorWaiting` `"#FF44FF"`, `ColorActive` `"#00FF00"`, `ColorReply` `"#FFFF00"`, `ColorStale` `"#FFA500"`, `ColorTodos` `"#0088FF"`, `ColorSelected` `""`, `ColorHeader` `"#7D56F4"`)
  - Implement `Default() Config` returning all built-in defaults
  - Implement `Load(explicitPath string) (Config, error)`:
    - If `explicitPath != ""`: try to open that path. If not found, return error (NOT silent fallback). If found, decode.
    - If `explicitPath == ""`: resolve path via `ResolvePath()`. If not found, return `Default(), nil` (silent fallback).
    - Decode TOML onto a pre-filled `Default()` struct (missing fields keep defaults).
    - After decode, check `MetaData.Undecoded()` - if any unknown keys, print warning to stderr (NOT error).
    - Call `Validate()` on the result. Return error if invalid.
  - Implement `ResolvePath() string`:
    - Check `$XDG_CONFIG_HOME/opencode-dashboard/config.toml`
    - Fall back to `~/.config/opencode-dashboard/config.toml`
    - Return empty string if neither exists
  - Implement `Validate(cfg Config) error`:
    - `RefreshInterval >= 10`
    - `ListWidthRatio >= 0.2 && <= 0.8`
    - `ActiveNowMinutes > 0`, `NeedsResponseHours > 0`, `StaleWorkDays > 0`
    - Quit binding must not be empty
    - Return descriptive error with field name
  - Create `internal/config/config_test.go` with TDD tests (RED first, then GREEN):
    - `TestDefault_AllFieldsPopulated` - verify every field matches current hardcoded values
    - `TestDefault_GoldenValues` - exact comparison: RefreshInterval==30, ActiveNowMinutes==5, etc.
    - `TestLoad_MissingFile_ReturnsDefaults` - empty path, no file on disk = Default(), nil
    - `TestLoad_ExplicitPathMissing_ReturnsError` - explicit path that doesn't exist = non-nil error
    - `TestLoad_EmptyFile_ReturnsDefaults` - file exists but empty = all defaults preserved
    - `TestLoad_PartialConfig_MergesDefaults` - only `[keys]` section, everything else defaults
    - `TestLoad_FullConfig` - all sections populated, verify each field
    - `TestLoad_ParseError_ReturnsError` - malformed TOML = non-nil error
    - `TestLoad_SliceReplacement` - user-defined time windows REPLACE defaults (not merge)
    - `TestValidate_RefreshTooLow` - value 5 rejected
    - `TestValidate_RatioOutOfRange` - 0.0 and 1.0 rejected
    - `TestValidate_NegativeThresholds` - 0 or negative rejected
    - `TestValidate_EmptyQuitBinding` - empty quit = rejected
    - `TestResolvePath_XDGOverride` - respects $XDG_CONFIG_HOME
    - Use `t.TempDir()` for test config files (cleaned up automatically)

  **Must NOT do**:
  - Do not import any UI, store, or app package (`internal/ui`, `internal/store`, `internal/app`). Importing `internal/domain` is NOT allowed either - config defines its own `TimeWindowOption` struct (with `Label string` and `Hours int`). Conversion to `domain.TimeWindowOption` (with `Duration`) happens in `main.go`, not in the config package.
  - Only import stdlib (`time`, `os`, `path/filepath`, `fmt`, `strings`, `errors`) + `github.com/BurntSushi/toml` + `github.com/stretchr/testify/assert` (test only)
  - Do not add config hot-reload, file watching, or environment variable overrides
  - Do not add a `--dump-config` or `--validate-config` subcommand
  - Do not use `os.UserConfigDir()` - use `$XDG_CONFIG_HOME` with `~/.config` fallback per the issue spec

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Core package with TDD, many test cases, validation logic, TOML integration - needs careful design
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 3, 4 after Task 1 completes)
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 5, 6, 7, 8, 9, 10, 11, 12
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - `internal/attention/classifier.go:9-12` - Current hardcoded threshold values that become Config defaults: `ActiveNowWindow = 5 * time.Minute`, `NeedsResponseWindow = 24 * time.Hour`, `StaleThreshold = 7 * 24 * time.Hour`
  - `internal/ui/filterbar.go:38-40` - Current default filter preset (`FilterNeedsAttention`) and time window (`TimeWindow3Days`) that become Config.Defaults
  - `internal/ui/app.go:76` - Current hardcoded refresh interval `30*time.Second` that becomes Config.Defaults.RefreshInterval
  - `internal/ui/layout.go:227-233` - Current hardcoded 50/50 pane split (`m.width / 2`) that becomes Config.Defaults.ListWidthRatio
  - `internal/ui/styles.go:16-52` - Current hardcoded icons and colors for all 6 attention signals that become Config.Display fields
  - `internal/ui/app.go:18-21` - Header color `#7D56F4` that becomes Config.Display.ColorHeader
  - `internal/ui/layout.go:96,102,116,123,135-148` - All hardcoded key strings in layout that become Config.Keys defaults
  - `internal/ui/sessionlist.go:108-138` - All hardcoded key strings in session list
  - `internal/ui/app.go:172-181` - All hardcoded key strings in app (quit, refresh)
  - `internal/ui/filterbar.go:89-99` - All hardcoded key strings in filter bar (tab, t, /)
  - `internal/attention/classifier_test.go:149-151` - Test pattern using `fixedNow()` helper and `testify/assert`

  **External References**:
  - https://pkg.go.dev/github.com/BurntSushi/toml - TOML decoding API, especially `toml.Decode()`, `toml.DecodeFile()`, `MetaData.Undecoded()`

  **WHY Each Reference Matters**:
  - The threshold/key/icon/color values from the source files are the EXACT defaults that `Default()` must return
  - The classifier test pattern shows the project's testing conventions to follow
  - The TOML docs explain decode-onto-struct semantics and MetaData.Undecoded() for unknown key warnings

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Golden defaults match current hardcoded values
    Tool: Bash
    Preconditions: internal/config/ package created with config.go and config_test.go
    Steps:
      1. Run: go test -v -run TestDefault_GoldenValues ./internal/config/
    Expected Result: Test passes. Default().Thresholds.ActiveNowMinutes == 5, NeedsResponseHours == 24, StaleWorkDays == 7. Default().Defaults.RefreshInterval == 30. Default().Display.IconError == "!". All fields verified.
    Failure Indicators: Any assertion failure showing default != current hardcoded value
    Evidence: .sisyphus/evidence/task-2-golden-defaults.txt

  Scenario: Missing config file returns defaults silently
    Tool: Bash
    Preconditions: No config file at resolved path
    Steps:
      1. Run: go test -v -run TestLoad_MissingFile_ReturnsDefaults ./internal/config/
    Expected Result: Load("") returns (Default(), nil) - no error, all defaults
    Failure Indicators: Non-nil error returned, or config differs from Default()
    Evidence: .sisyphus/evidence/task-2-missing-file.txt

  Scenario: Explicit path missing returns error
    Tool: Bash
    Preconditions: None
    Steps:
      1. Run: go test -v -run TestLoad_ExplicitPathMissing_ReturnsError ./internal/config/
    Expected Result: Load("/nonexistent/path.toml") returns non-nil error
    Failure Indicators: Error is nil (silent fallback when explicit path given)
    Evidence: .sisyphus/evidence/task-2-explicit-missing.txt

  Scenario: Partial config merges onto defaults
    Tool: Bash
    Preconditions: None
    Steps:
      1. Run: go test -v -run TestLoad_PartialConfig_MergesDefaults ./internal/config/
    Expected Result: Only specified fields changed, all others equal Default()
    Failure Indicators: Unspecified fields don't match Default()
    Evidence: .sisyphus/evidence/task-2-partial-config.txt

  Scenario: Invalid TOML returns parse error
    Tool: Bash
    Preconditions: None
    Steps:
      1. Run: go test -v -run TestLoad_ParseError_ReturnsError ./internal/config/
    Expected Result: Non-nil error containing TOML parse information
    Failure Indicators: Error is nil or doesn't mention parse failure
    Evidence: .sisyphus/evidence/task-2-parse-error.txt

  Scenario: Validation catches invalid values
    Tool: Bash
    Preconditions: None
    Steps:
      1. Run: go test -v -run TestValidate ./internal/config/
    Expected Result: All validation tests pass (refresh too low, ratio out of range, negative thresholds, empty quit)
    Failure Indicators: Any validation test fails
    Evidence: .sisyphus/evidence/task-2-validation.txt

  Scenario: Full test suite passes
    Tool: Bash
    Preconditions: All config tests written
    Steps:
      1. Run: go test -v ./internal/config/
    Expected Result: All tests pass, zero failures
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-2-full-suite.txt
  ```

  **Commit**: YES
  - Message: `feat(config): add config package with TOML loading and validation`
  - Files: `internal/config/config.go`, `internal/config/config_test.go`
  - Pre-commit: `go test ./internal/config/`

- [x] 3. Config package - KeyMap struct and Matches helper (TDD)

  **What to do**:
  - Create `internal/config/keys.go` with:
    ```go
    // KeyMap maps actions to key binding strings.
    // Key strings match bubbletea's tea.KeyMsg.String() output.
    // Examples: "ctrl+c", "tab", "enter", "q", " " (space).
    // See: https://pkg.go.dev/github.com/charmbracelet/bubbletea#KeyMsg
    type KeyMap struct {
        Up          []string
        Down        []string
        PaneLeft    []string
        PaneRight   []string
        TreeNav     []string
        Collapse    []string
        CycleFilter []string
        CycleTime   []string
        Search      []string
        Refresh     []string
        Launch      []string
        Quit        []string
    }
    ```
  - Implement `Matches(keyStr string, bindings []string) bool` as a **standalone function** (not method on KeyMap). Does exact string comparison with one normalization: if `bindings` contains `"space"`, it also matches `" "`.
  - Implement `NormalizeKey(key string) string` that converts `"space"` -> `" "` (used during config loading to normalize all binding values)
  - Implement `DefaultKeyMap() KeyMap` returning the exact current hardcoded bindings
  - Create `internal/config/keys_test.go`:
    - `TestMatches_SingleKey` - `Matches("q", []string{"q"})` == true
    - `TestMatches_MultipleKeys` - `Matches("ctrl+c", []string{"q", "ctrl+c"})` == true
    - `TestMatches_NoMatch` - `Matches("x", []string{"q", "ctrl+c"})` == false
    - `TestMatches_EmptyBindings` - `Matches("q", []string{})` == false
    - `TestMatches_SpaceAlias` - `Matches(" ", []string{"space"})` == true
    - `TestMatches_SpaceAliasReverse` - `Matches(" ", []string{" "})` == true
    - `TestNormalizeKey` - `NormalizeKey("space")` == `" "`
    - `TestDefaultKeyMap_MatchesHardcoded` - verify every default binding matches the current hardcoded strings from the UI files

  **Must NOT do**:
  - Do not import bubbletea or any UI package
  - Do not add complex normalization beyond the "space" alias
  - Keep Matches as a standalone function, not a method

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small focused file with straightforward matching logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 2, 4 after Task 1)
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 7, 8, 9, 10, 12
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - `internal/ui/layout.go:96` - `msg.String() == "tab"` - CycleFilter default
  - `internal/ui/layout.go:102` - `msg.String() == "t"` - CycleTime default
  - `internal/ui/layout.go:116` - `msg.String() == "/"` - Search default
  - `internal/ui/layout.go:123` - `msg.String() == "left"` - TreeNav default
  - `internal/ui/layout.go:136-141` - `"h"`, `"l"`, `"right"` - Pane switch defaults
  - `internal/ui/layout.go:142` - `"enter"` - Launch default
  - `internal/ui/sessionlist.go:109` - `"j", "down"` - Down defaults
  - `internal/ui/sessionlist.go:113` - `"k", "up"` - Up defaults
  - `internal/ui/sessionlist.go:117` - `" "` (space) - Collapse default
  - `internal/ui/sessionlist.go:124` - `"left"` - TreeNav in list context
  - `internal/ui/app.go:172` - `"ctrl+c"` - Quit default
  - `internal/ui/app.go:175` - `"q"` - Quit default
  - `internal/ui/app.go:178` - `"r"` - Refresh default
  - `internal/ui/filterbar.go:90-93` - `"tab"`, `"t"` - CycleFilter/CycleTime in filterbar

  **WHY Each Reference Matters**:
  - These are the EXACT strings that DefaultKeyMap() must produce. Getting any one wrong breaks backward compatibility.

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Key matching works correctly
    Tool: Bash
    Preconditions: internal/config/keys.go and keys_test.go created
    Steps:
      1. Run: go test -v -run TestMatches ./internal/config/
    Expected Result: All Matches tests pass - single key, multi-key, no match, empty, space alias
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-3-matches.txt

  Scenario: Default KeyMap matches hardcoded strings
    Tool: Bash
    Preconditions: keys.go with DefaultKeyMap() implemented
    Steps:
      1. Run: go test -v -run TestDefaultKeyMap_MatchesHardcoded ./internal/config/
    Expected Result: Every default binding verified against current hardcoded values
    Failure Indicators: Any binding mismatch
    Evidence: .sisyphus/evidence/task-3-default-keymap.txt

  Scenario: Space normalization works
    Tool: Bash
    Preconditions: None
    Steps:
      1. Run: go test -v -run TestNormalizeKey ./internal/config/
      2. Run: go test -v -run TestMatches_SpaceAlias ./internal/config/
    Expected Result: "space" normalizes to " "; Matches(" ", ["space"]) == true
    Failure Indicators: Normalization fails
    Evidence: .sisyphus/evidence/task-3-space-normalize.txt
  ```

  **Commit**: YES
  - Message: `feat(config): add KeyMap struct with key matching`
  - Files: `internal/config/keys.go`, `internal/config/keys_test.go`
  - Pre-commit: `go test ./internal/config/`

- [x] 4. Attention - Thresholds struct + ClassifyWith (TDD)

  **What to do**:
  - In `internal/attention/classifier.go`, add:
    ```go
    type Thresholds struct {
        ActiveNowWindow     time.Duration
        NeedsResponseWindow time.Duration
        StaleThreshold      time.Duration
    }

    func DefaultThresholds() Thresholds {
        return Thresholds{
            ActiveNowWindow:     5 * time.Minute,
            NeedsResponseWindow: 24 * time.Hour,
            StaleThreshold:      7 * 24 * time.Hour,
        }
    }

    func ClassifyWith(view domain.SessionView, now time.Time, t Thresholds) domain.AttentionSignal {
        // Same logic as Classify but using t.ActiveNowWindow, t.NeedsResponseWindow, t.StaleThreshold
    }
    ```
  - Refactor existing `Classify()` to call `ClassifyWith(view, now, DefaultThresholds())` - this is a one-line wrapper
  - Keep the existing package-level `const` values (they're still referenced by existing tests via `ActiveNowWindow`, `NeedsResponseWindow`, `StaleThreshold`)
  - In `internal/attention/classifier_test.go`, add new tests (DO NOT modify existing tests):
    - `TestClassifyWith_CustomActiveWindow` - 10-minute active window: session at 7 minutes is ActiveNow (would be None with default 5min)
    - `TestClassifyWith_CustomNeedsResponseWindow` - 48-hour response window: session at 30 hours with assistant last message is NeedsResponse (would be None with default 24h)
    - `TestClassifyWith_CustomStaleThreshold` - 3-day stale: session at 4 days with pending todos is StaleWork (would be PendingTodos with default 7d)
    - `TestClassifyWith_DefaultThresholdsMatchClassify` - verify ClassifyWith with DefaultThresholds() produces identical results to Classify() for all existing test cases

  **Must NOT do**:
  - Do NOT change the signature of `Classify(view domain.SessionView, now time.Time) domain.AttentionSignal`
  - Do NOT modify ANY existing test function
  - Do NOT remove the package-level const values (existing tests reference them)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small addition to existing file with clear pattern, straightforward tests
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 2, 3 - no dependency on Task 1)
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 12
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/attention/classifier.go:9-12` - Current const values that become DefaultThresholds() return values
  - `internal/attention/classifier.go:15-47` - Current Classify() logic that ClassifyWith() replicates with parameterized thresholds
  - `internal/attention/classifier_test.go:11-151` - All 14 existing tests showing the testing pattern, fixedNow() helper, testify/assert usage

  **WHY Each Reference Matters**:
  - classifier.go contains the exact logic to parameterize - the three const references (lines 30, 34, 39) become threshold struct field references
  - classifier_test.go shows the exact test style to follow and the exact values that must remain unchanged for backward compatibility

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: All existing classifier tests still pass unchanged
    Tool: Bash
    Preconditions: ClassifyWith() added, Classify() refactored as wrapper
    Steps:
      1. Run: go test -v -run "TestClassify_" ./internal/attention/
    Expected Result: All 14 existing TestClassify_* tests pass with zero modifications
    Failure Indicators: Any existing test fails
    Evidence: .sisyphus/evidence/task-4-existing-tests.txt

  Scenario: Custom thresholds produce different classifications
    Tool: Bash
    Preconditions: ClassifyWith tests written
    Steps:
      1. Run: go test -v -run "TestClassifyWith_" ./internal/attention/
    Expected Result: All ClassifyWith tests pass - custom windows/thresholds change classification outcomes
    Failure Indicators: Custom thresholds produce same result as defaults (not testing correctly)
    Evidence: .sisyphus/evidence/task-4-custom-thresholds.txt

  Scenario: DefaultThresholds matches Classify
    Tool: Bash
    Preconditions: None
    Steps:
      1. Run: go test -v -run TestClassifyWith_DefaultThresholdsMatchClassify ./internal/attention/
    Expected Result: ClassifyWith(view, now, DefaultThresholds()) == Classify(view, now) for all test cases
    Failure Indicators: Any divergence between Classify and ClassifyWith+DefaultThresholds
    Evidence: .sisyphus/evidence/task-4-default-match.txt
  ```

  **Commit**: YES
  - Message: `feat(attention): add configurable thresholds via ClassifyWith`
  - Files: `internal/attention/classifier.go`, `internal/attention/classifier_test.go`
  - Pre-commit: `go test ./internal/attention/`

- [x] 5. Replace TimeWindow enum with config-driven TimeWindowOption (ATOMIC)

  **What to do**:
  This is an ATOMIC change across 4 files. All changes must happen in a single commit because the type is shared.

  **Step A - `internal/domain/types.go`**:
  - REMOVE the `TimeWindow` type, its iota constants (`TimeWindowAll`, `TimeWindow1Day`, `TimeWindow3Days`, `TimeWindow7Days`), and its `String()` and `Duration()` methods (lines 130-163)
  - ADD:
    ```go
    type TimeWindowOption struct {
        Label    string
        Duration time.Duration // 0 means "all" (no time filtering)
    }
    ```

  **Step B - `internal/filter/filter.go`**:
  - Change `Filter` struct field from `Window domain.TimeWindow` to `Window time.Duration` (the resolved duration, not the option struct)
  - Update `matchesWindow(s domain.SessionView, w domain.TimeWindow)` to `matchesWindow(s domain.SessionView, d time.Duration, now time.Time)`:
    ```go
    func matchesWindow(s domain.SessionView, d time.Duration, now time.Time) bool {
        if d == 0 {
            return true // "all" - no filtering
        }
        return s.TimeUpdated.After(now.Add(-d))
    }
    ```
  - Update `Apply()` to pass `f.Window` and `time.Now()` (or accept `now` parameter for testability - prefer adding `now` parameter)
  - Actually, update `Apply` signature to `Apply(sessions []domain.SessionView, f Filter, now time.Time)` for testability. The caller (app.go `applyFilter`) passes `time.Now()`.

  **Step C - `internal/filter/filter_test.go`**:
  - Update all test calls to use `time.Duration` for the Window field instead of the old enum
  - Use `0` for "all", `24 * time.Hour` for 1-day, etc.
  - Add time window filter tests (currently missing):
    - `TestApply_TimeWindow_All` - duration 0 matches everything
    - `TestApply_TimeWindow_1Day` - filters sessions older than 24h
    - `TestApply_TimeWindow_Boundary` - session exactly at window boundary

  **Step D - `internal/ui/filterbar.go`**:
  - Add fields to FilterBarModel: `windowOptions []domain.TimeWindowOption`, `windowIdx int`
  - Replace `window domain.TimeWindow` field with the index-based approach
  - Update `NewFilterBar()` to accept `windowOptions []domain.TimeWindowOption` and `defaultWindowIdx int`
  - Replace `nextWindow()` function with index arithmetic: `m.windowIdx = (m.windowIdx + 1) % len(m.windowOptions)`
  - Update `CurrentFilter()` to return `filter.Filter{Window: m.windowOptions[m.windowIdx].Duration, ...}`
  - Update `View()` to display `m.windowOptions[m.windowIdx].Label` instead of calling `.String()` on enum

  **Step E - `internal/ui/app.go`**:
  - Update `applyFilter()` to pass `time.Now()` to `filter.Apply()`

  **Step F - Conversion location: `internal/ui/app.go` (NOT in config or main.go)**:
  - The config package defines its own `config.TimeWindowOption{Label string, Hours int}`. Conversion to `domain.TimeWindowOption{Label string, Duration time.Duration}` happens in `internal/ui/app.go` inside `NewAppWithConfig()` (Task 10), because app.go imports both `internal/config` and `internal/domain`.
  - `app.go` iterates `cfg.TimeWindows.Options`, converts each `Hours` to `time.Duration`, appends the "All" sentinel `{Label: "All", Duration: 0}`, and passes the result down to `NewLayout()` which passes to `NewFilterBar()`.
  - Default preset resolution (`cfg.Defaults.Filter` string -> `domain.FilterPreset`) and default window index resolution (`cfg.Defaults.TimeWindow` label -> index) also happen in `app.go`.

  **Must NOT do**:
  - Do not keep the old TimeWindow enum around "for compatibility" - clean removal
  - Do not make FilterPreset configurable
  - Do not change the Store layer

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Atomic 6-file change with type system migration, needs careful coordination
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (touches shared types, must land as one unit)
  - **Parallel Group**: Wave 2 (after Task 2 completes)
  - **Blocks**: Tasks 7, 10, 11, 12
  - **Blocked By**: Task 2

  **References**:

  **Pattern References**:
  - `internal/domain/types.go:130-163` - TimeWindow enum, String(), Duration() methods TO BE REMOVED
  - `internal/filter/filter.go:13-14` - `Filter.Window` field that changes type
  - `internal/filter/filter.go:55-60` - `matchesWindow()` that changes signature and adds `now` parameter
  - `internal/filter/filter.go:16` - `Apply()` call site that adds `now` parameter
  - `internal/ui/filterbar.go:22-28` - FilterBarModel fields that change (window -> windowOptions + windowIdx)
  - `internal/ui/filterbar.go:34-42` - `NewFilterBar()` constructor that gets new parameters
  - `internal/ui/filterbar.go:51-57` - `CurrentFilter()` that returns Duration instead of enum
  - `internal/ui/filterbar.go:93-94` - `nextWindow()` call that becomes index arithmetic
  - `internal/ui/filterbar.go:136` - `View()` window label display
  - `internal/ui/filterbar.go:182-193` - `nextWindow()` function TO BE REPLACED with index arithmetic
  - `internal/ui/app.go:193-206` - `applyFilter()` that needs to pass `time.Now()` to `filter.Apply()`
  - `internal/filter/filter_test.go` - All existing filter tests that need Window field type updates

  **External References**:
  - Issue TOML schema: `[time_windows] options = [{label = "1d", hours = 24}, ...]` - the cycle order follows list order; "all" always appended

  **WHY Each Reference Matters**:
  - domain/types.go is the type being removed - executor needs to see exactly what goes away
  - filter.go/filter_test.go show what changes type from enum to duration
  - filterbar.go shows all the places that used the enum for cycling/display
  - app.go shows the Apply() call site that needs the new `now` parameter

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Project compiles after atomic type change
    Tool: Bash
    Preconditions: All 4+ files modified atomically
    Steps:
      1. Run: go build ./...
    Expected Result: Compilation succeeds with zero errors
    Failure Indicators: Any compilation error (usually means a file was missed in the atomic change)
    Evidence: .sisyphus/evidence/task-5-compiles.txt

  Scenario: All existing filter tests pass (updated for new type)
    Tool: Bash
    Preconditions: filter_test.go updated with Duration instead of enum
    Steps:
      1. Run: go test -v ./internal/filter/
    Expected Result: All existing tests pass + new time window tests pass
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-5-filter-tests.txt

  Scenario: Time window cycling uses index arithmetic, not enum switch
    Tool: Bash
    Preconditions: FilterBarModel updated with windowOptions slice
    Steps:
      1. Run: go build ./...
      2. Run: grep -c 'nextWindow' internal/ui/filterbar.go (expect 0 - function removed)
      3. Run: grep -c 'windowIdx' internal/ui/filterbar.go (expect >= 2 - field used in cycling)
      4. Run: grep -c 'TimeWindowAll\|TimeWindow1Day\|TimeWindow3Days\|TimeWindow7Days' internal/ui/filterbar.go (expect 0 - old enum gone)
    Expected Result: Build passes; nextWindow function removed; windowIdx field present; no old enum references
    Failure Indicators: nextWindow function still exists; old enum constants still referenced
    Evidence: .sisyphus/evidence/task-5-cycling.txt

  Scenario: Full test suite passes after type migration
    Tool: Bash
    Preconditions: All changes landed
    Steps:
      1. Run: go test ./...
    Expected Result: ALL test files pass (existing + new)
    Failure Indicators: Any test failure in any package
    Evidence: .sisyphus/evidence/task-5-full-suite.txt
  ```

  **Commit**: YES
  - Message: `refactor: replace TimeWindow enum with config-driven TimeWindowOption`
  - Files: `internal/domain/types.go`, `internal/filter/filter.go`, `internal/filter/filter_test.go`, `internal/ui/filterbar.go`, `internal/ui/app.go`, `internal/config/config.go`
  - Pre-commit: `go test ./...`

- [x] 6. Display config helpers in styles.go

  **What to do**:
  - In `internal/ui/styles.go`, modify `attentionIcon()` and `attentionIconPlain()` to accept a `config.DisplayConfig` parameter:
    ```go
    func attentionIcon(signal domain.AttentionSignal, d config.DisplayConfig) string {
        icon, color := iconAndColor(signal, d)
        if icon == "" {
            return " "
        }
        return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(icon)
    }

    func attentionIconPlain(signal domain.AttentionSignal, d config.DisplayConfig) string {
        icon, _ := iconAndColor(signal, d)
        if icon == "" {
            return " "
        }
        return icon
    }
    ```
  - Add a private helper `iconAndColor(signal, d) (string, string)` that does the switch-case lookup against DisplayConfig fields
  - Add methods to `config.DisplayConfig` for clean lookup:
    ```go
    func (d DisplayConfig) IconFor(signal domain.AttentionSignal) string
    func (d DisplayConfig) ColorFor(signal domain.AttentionSignal) string
    ```
    Wait - DisplayConfig is in `internal/config` which must NOT import `internal/domain`. So these methods must live in the UI layer, not config. Use the private `iconAndColor()` helper in styles.go instead.
  - Update all call sites of `attentionIcon()` and `attentionIconPlain()` to pass the DisplayConfig. These are in:
    - `internal/ui/sessionlist.go:229,231` - renderSessionRow calls attentionIcon/attentionIconPlain
  - This means SessionListModel needs to store a DisplayConfig (or receive it). For now, add a `display config.DisplayConfig` field to SessionListModel and pass it in the updated constructor (prepared for Task 8).
  - Also update `sectionHeaderStyle` color to use `config.DisplayConfig.ColorHeader` (this is the shared accent color, not a separate section-specific config field)

  **Must NOT do**:
  - Do not add `domain` import to the `config` package
  - Do not create IconFor/ColorFor methods on DisplayConfig in the config package
  - Do not change todo priority colors or section styles beyond what's in DisplayConfig

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small refactor of two existing functions plus adding a helper, pattern is clear
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 5 in Wave 2)
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 7, 8, 11, 12
  - **Blocked By**: Task 2

  **References**:

  **Pattern References**:
  - `internal/ui/styles.go:16-52` - The two existing functions to modify (`attentionIcon`, `attentionIconPlain`) with their switch-case structure
  - `internal/ui/styles.go:12-14` - `sectionHeaderStyle` with hardcoded `#7D56F4` color
  - `internal/ui/sessionlist.go:229,231` - Call sites of attentionIcon/attentionIconPlain
  - `internal/config/config.go` - DisplayConfig struct with all icon/color fields

  **WHY Each Reference Matters**:
  - styles.go contains the exact functions being parameterized - executor sees current structure
  - sessionlist.go shows the call sites that need the extra parameter
  - config.go provides the DisplayConfig type being passed in

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Project compiles after display config wiring
    Tool: Bash
    Preconditions: styles.go updated, sessionlist.go call sites updated
    Steps:
      1. Run: go build ./...
    Expected Result: Compilation succeeds
    Failure Indicators: Missing parameter errors at call sites
    Evidence: .sisyphus/evidence/task-6-compiles.txt

  Scenario: All tests still pass
    Tool: Bash
    Preconditions: All changes made
    Steps:
      1. Run: go test ./...
    Expected Result: All tests pass (no UI tests to break, but ensure no import cycles)
    Failure Indicators: Import cycle errors or compilation failures
    Evidence: .sisyphus/evidence/task-6-tests.txt
  ```

  **Commit**: YES
  - Message: `refactor(ui): extract display config helpers in styles.go`
  - Files: `internal/ui/styles.go`, `internal/ui/sessionlist.go`
  - Pre-commit: `go build ./...`

- [ ] 7. Wire config into FilterBarModel

  **What to do**:
  - Update `NewFilterBar()` signature to accept config parameters:
    ```go
    func NewFilterBar(keys config.KeyMap, display config.DisplayConfig, windowOptions []domain.TimeWindowOption, defaultPreset domain.FilterPreset, defaultWindowIdx int) FilterBarModel
    ```
  - Store `keys`, `display` on the FilterBarModel struct
  - Replace all hardcoded key checks in `Update()` with `config.Matches(msg.String(), m.keys.XXX)`:
    - `"tab"` -> `config.Matches(keyStr, m.keys.CycleFilter)`
    - `"t"` -> `config.Matches(keyStr, m.keys.CycleTime)`
    - `"/"` -> `config.Matches(keyStr, m.keys.Search)`
    - `"esc"` and `"enter"` in search mode stay hardcoded (they're search-mode-specific, not configurable per issue schema)
  - Update `View()` legend section to use display config for icon/color rendering:
    ```go
    // Current hardcoded:
    color("#FF0000", "!") + " err"
    // Becomes:
    color(m.display.ColorError, m.display.IconError) + " err"
    ```
  - Default preset comes from `defaultPreset` parameter (parsed from `cfg.Defaults.Filter` string to `domain.FilterPreset` in the caller)
  - Window options + default index already wired in Task 5

  **Must NOT do**:
  - Do not make FilterPreset options configurable (only the default selection)
  - Do not make search-mode keys (esc, enter) configurable
  - Do not change the search text behavior or filter logic

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Multiple interrelated changes in one file, needs careful attention to not break filter bar behavior
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 8, 9, 10, 11)
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 2, 3, 5, 6

  **References**:

  **Pattern References**:
  - `internal/ui/filterbar.go:69-110` - Full Update() method with all key checks to replace
  - `internal/ui/filterbar.go:112-169` - Full View() method with legend colors to parameterize
  - `internal/ui/filterbar.go:34-42` - NewFilterBar() constructor to extend (already modified in Task 5 for windowOptions)
  - `internal/ui/filterbar.go:21-28` - FilterBarModel struct to add keys/display fields
  - `internal/config/keys.go` - KeyMap struct and Matches() function

  **WHY Each Reference Matters**:
  - filterbar.go Update() has the exact key checks to replace - executor sees current `msg.String() == "tab"` patterns
  - filterbar.go View() has the exact legend colors to parameterize
  - config/keys.go provides the Matches() function the new code calls

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Filter bar compiles with config wiring
    Tool: Bash
    Preconditions: FilterBarModel updated with keys and display fields
    Steps:
      1. Run: go build ./...
    Expected Result: Compilation succeeds
    Failure Indicators: Missing field errors, import issues
    Evidence: .sisyphus/evidence/task-7-compiles.txt

  Scenario: No hardcoded key strings remain in filterbar.go Update() outside search mode
    Tool: Bash
    Preconditions: All key replacements done
    Steps:
      1. Run: grep -n 'msg.String() ==' internal/ui/filterbar.go (expect only lines inside searchMode block for "esc"/"enter")
      2. Run: grep -c 'case "tab"\|== "tab"\|== "t"\|case "t"\|== "/"' internal/ui/filterbar.go (expect 0 - these should use config.Matches now)
      3. Run: grep -c 'config.Matches\|Matches(' internal/ui/filterbar.go (expect >= 3 - one per configurable action: CycleFilter, CycleTime, Search)
      4. Run: go build ./...
    Expected Result: msg.String()== appears only for esc/enter in search mode; config.Matches used for tab/t/slash; build succeeds
    Failure Indicators: "tab", "t", "/" appear in switch cases outside search mode
    Evidence: .sisyphus/evidence/task-7-no-hardcoded-keys.txt

  Scenario: Full test suite passes
    Tool: Bash
    Preconditions: All changes made
    Steps:
      1. Run: go test ./...
    Expected Result: All tests pass
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-7-tests.txt
  ```

  **Commit**: YES (groups with Tasks 8, 9, 10, 11 - one commit per file is fine, or group Wave 3)
  - Message: `feat(ui): wire config into FilterBarModel`
  - Files: `internal/ui/filterbar.go`
  - Pre-commit: `go build ./...`

- [ ] 8. Wire config into SessionListModel

  **What to do**:
  - Update `NewSessionList()` to accept KeyMap and DisplayConfig:
    ```go
    func NewSessionList(groups []domain.ProjectGroup, keys config.KeyMap, display config.DisplayConfig) SessionListModel
    ```
  - Store `keys` and `display` on SessionListModel struct
  - Replace all hardcoded key checks in `Update()` with `config.Matches()`:
    - `"j", "down"` -> `config.Matches(keyStr, m.keys.Down)`
    - `"k", "up"` -> `config.Matches(keyStr, m.keys.Up)`
    - `" "` (space) -> `config.Matches(keyStr, m.keys.Collapse)`
    - `"left"` -> `config.Matches(keyStr, m.keys.TreeNav)`
  - The switch statement in Update() changes from string-matching cases to if/else-if with Matches() calls
  - Pass display config to `attentionIcon()` and `attentionIconPlain()` calls in `renderSessionRow()` (already prepared by Task 6)

  **Must NOT do**:
  - Do not change the scrolling, cursor, or collapse behavior
  - Do not modify buildVisibleRows() logic
  - Do not change rendering layout or formatting

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Straightforward key replacement, small number of call sites
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 2, 3, 6

  **References**:

  **Pattern References**:
  - `internal/ui/sessionlist.go:104-141` - Full Update() method with all key checks to replace
  - `internal/ui/sessionlist.go:37-43` - NewSessionList() constructor to extend
  - `internal/ui/sessionlist.go:27-34` - SessionListModel struct to add keys/display fields
  - `internal/ui/sessionlist.go:226-278` - renderSessionRow() that calls attentionIcon/attentionIconPlain with display param

  **WHY Each Reference Matters**:
  - sessionlist.go Update() has the exact switch cases to replace with config.Matches()
  - Constructor and struct show where new fields go
  - renderSessionRow shows where DisplayConfig flows to icon rendering

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Session list compiles with config wiring
    Tool: Bash
    Preconditions: SessionListModel updated
    Steps:
      1. Run: go build ./...
    Expected Result: Compilation succeeds
    Failure Indicators: Missing parameter errors
    Evidence: .sisyphus/evidence/task-8-compiles.txt

  Scenario: No hardcoded key strings remain in sessionlist.go Update()
    Tool: Bash
    Preconditions: All key replacements done
    Steps:
      1. Run: grep -n 'case "' internal/ui/sessionlist.go (expect 0 - no string-literal case clauses in Update)
      2. Run: grep -c 'config.Matches\|Matches(' internal/ui/sessionlist.go (expect >= 4 - down, up, collapse, tree-nav)
      3. Run: grep -c 'msg.String()' internal/ui/sessionlist.go (expect 0 or only inside Matches calls)
      4. Run: go build ./...
    Expected Result: Zero hardcoded key strings in switch cases; config.Matches used for all 4 actions; build succeeds
    Failure Indicators: String literal case clauses like case "j" still present
    Evidence: .sisyphus/evidence/task-8-no-hardcoded-keys.txt

  Scenario: Full test suite passes
    Tool: Bash
    Preconditions: All changes made
    Steps:
      1. Run: go test ./...
    Expected Result: All tests pass
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-8-tests.txt
  ```

  **Commit**: YES
  - Message: `feat(ui): wire config into SessionListModel`
  - Files: `internal/ui/sessionlist.go`
  - Pre-commit: `go build ./...`

- [ ] 9. Wire config into LayoutModel

  **What to do**:
  - Update `NewLayout()` to accept all config needed by itself and its children:
    ```go
    func NewLayout(groups []domain.ProjectGroup, keys config.KeyMap, display config.DisplayConfig, listWidthRatio float64, windowOptions []domain.TimeWindowOption, defaultPreset domain.FilterPreset, defaultWindowIdx int) LayoutModel
    ```
  - Store `keys`, `display`, `listWidthRatio` on LayoutModel struct
  - Pass through to child constructors:
    - `NewFilterBar(keys, display, windowOptions, defaultPreset, defaultWindowIdx)` (signature from Task 5/7)
    - `NewSessionList(groups, keys, display)` (signature from Task 8)
  - Replace all hardcoded key checks in `Update()` with `config.Matches()`:
    - `"tab"` -> `config.Matches(keyStr, m.keys.CycleFilter)`
    - `"t"` -> `config.Matches(keyStr, m.keys.CycleTime)`
    - `"/"` -> `config.Matches(keyStr, m.keys.Search)`
    - `"left"` -> `config.Matches(keyStr, m.keys.TreeNav)` (for pane context) and `config.Matches(keyStr, m.keys.PaneLeft)` (conceptually the same but used differently)
    - `"h"` -> `config.Matches(keyStr, m.keys.PaneLeft)`
    - `"l"`, `"right"` -> `config.Matches(keyStr, m.keys.PaneRight)`
    - `"enter"` -> `config.Matches(keyStr, m.keys.Launch)`
  - Update `paneWidths()` to use stored `listWidthRatio`:
    ```go
    func (m LayoutModel) paneWidths() (left, right int) {
        left = int(float64(m.width) * m.listWidthRatio)
        right = m.width - left - 1
        if right < 0 {
            right = 0
        }
        return left, right
    }
    ```
  - Update `View()` to use `m.display.ColorHeader` for `activeColor` instead of hardcoded `"#7D56F4"`

  **Must NOT do**:
  - Do not change pane focus/switching logic beyond key parameterization
  - Do not change the View rendering structure (border styles, title format)
  - Do not modify WindowSizeMsg handling logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Direct key replacement + one ratio change, follows same pattern as Tasks 7/8
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 2, 3, 5 (needs domain.TimeWindowOption type from Task 5)

  **References**:

  **Pattern References**:
  - `internal/ui/layout.go:27-33` - NewLayout() constructor to extend
  - `internal/ui/layout.go:18-25` - LayoutModel struct to add keys/display/ratio/windowOptions/preset/idx fields
  - `internal/ui/layout.go:94-165` - Full Update() method with all key checks to replace
  - `internal/ui/layout.go:227-233` - paneWidths() to parameterize with ratio
  - `internal/ui/layout.go:185-186` - activeColor/inactiveColor to use display config

  **WHY Each Reference Matters**:
  - layout.go Update() has the most complex key routing (tab routes to filterbar, keys route to different panes based on context) - executor needs to see the full routing logic
  - paneWidths() shows the exact calculation to parameterize
  - View() shows where the accent color is used

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Layout compiles with config wiring
    Tool: Bash
    Preconditions: LayoutModel updated
    Steps:
      1. Run: go build ./...
    Expected Result: Compilation succeeds
    Failure Indicators: Constructor mismatch or missing fields
    Evidence: .sisyphus/evidence/task-9-compiles.txt

  Scenario: Pane width ratio is configurable (no more hardcoded m.width / 2)
    Tool: Bash
    Preconditions: paneWidths() uses m.listWidthRatio
    Steps:
      1. Run: grep -c 'm.width / 2' internal/ui/layout.go (expect 0 - hardcoded division removed)
      2. Run: grep -c 'listWidthRatio' internal/ui/layout.go (expect >= 2 - field declaration + usage in paneWidths)
      3. Run: go build ./...
    Expected Result: Zero occurrences of 'm.width / 2'; listWidthRatio field present and used; build succeeds
    Failure Indicators: 'm.width / 2' still present in paneWidths()
    Evidence: .sisyphus/evidence/task-9-ratio.txt

  Scenario: Full test suite passes
    Tool: Bash
    Preconditions: All changes made
    Steps:
      1. Run: go test ./...
    Expected Result: All tests pass
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-9-tests.txt
  ```

  **Commit**: YES
  - Message: `feat(ui): wire config into LayoutModel`
  - Files: `internal/ui/layout.go`
  - Pre-commit: `go build ./...`

- [ ] 10. Wire config into AppModel (keys, refresh, footer)

  **What to do**:
  - Create new constructor `NewAppWithConfig(cfg config.Config, agg *appcore.Aggregator) AppModel` that replaces `NewAppWithAggregator`:
    ```go
    func NewAppWithConfig(cfg config.Config, agg *appcore.Aggregator) AppModel {
        // Convert config types to domain types HERE (app.go imports both config and domain)
        windowOptions := buildTimeWindowOptions(cfg) // local helper in app.go: converts config.TimeWindowOption{Hours} -> domain.TimeWindowOption{Duration}, appends "All" sentinel
        defaultPreset := resolveDefaultPreset(cfg.Defaults.Filter) // local helper: "needs_attention" -> domain.FilterNeedsAttention
        defaultWindowIdx := resolveDefaultWindowIdx(cfg.Defaults.TimeWindow, windowOptions) // local helper: matches label to index
        keys := cfg.Keys // KeysConfig fields match KeyMap layout - use directly or copy

        return AppModel{
            cfg:        cfg,
            layout:     NewLayout(nil, keys, cfg.Display, cfg.Defaults.ListWidthRatio, windowOptions, defaultPreset, defaultWindowIdx),
            aggregator: agg,
            loading:    agg != nil,
        }
    }
    }
    ```
  - Keep `NewAppWithAggregator` as a backward-compat wrapper using `config.Default()`
  - Store `cfg config.Config` on AppModel struct
  - Replace hardcoded key checks in `Update()`:
    - `"ctrl+c"` -> `config.Matches(keyStr, m.cfg.Keys.Quit)` (ctrl+c is just part of the Quit binding list)
    - `"q"` -> same Quit binding (but respecting `!m.layout.IsSearchActive()` guard)
    - `"r"` -> `config.Matches(keyStr, m.cfg.Keys.Refresh)` (with `!m.layout.IsSearchActive()` guard)
    - Note: `ctrl+c` should ALWAYS quit (even during search) - keep separate check for ctrl+c specifically, or make Quit binding check ignore search mode for ctrl+c only. Simplest: check quit bindings, and if in search mode, only allow `ctrl+c` from the quit list.
  - Update `tickCmd()` to use configurable refresh interval:
    ```go
    func tickCmd(interval time.Duration) tea.Cmd {
        return tea.Tick(interval, func(t time.Time) tea.Msg {
            return refreshMsg{}
        })
    }
    ```
    Called as `tickCmd(time.Duration(m.cfg.Defaults.RefreshInterval) * time.Second)`
  - Update footer legend in `View()` to be dynamically generated from KeyMap:
    ```go
    func (m AppModel) buildFooterLegend() string {
        // Format: "j/k up/down  h/l pane  ..."
        // Becomes dynamic based on actual key bindings
        return fmt.Sprintf("%s up/down  %s pane  ...", formatKeys(m.cfg.Keys.Down), formatKeys(m.cfg.Keys.PaneLeft))
    }
    ```
    Where `formatKeys(keys []string) string` joins keys with `/` (e.g., `["j", "down"]` -> `"j/down"`)
  - Update header color to use `m.cfg.Display.ColorHeader`:
    ```go
    headerStyle = lipgloss.NewStyle().Bold(true).
        Foreground(lipgloss.Color("#FAFAFA")).
        Background(lipgloss.Color(m.cfg.Display.ColorHeader)).
        Padding(0, 1)
    ```
    Note: headerStyle is currently a package-level var. It needs to be computed per-render using config, or set once in the constructor. Prefer constructor-time since config is immutable.

  **Must NOT do**:
  - Do not add config hot-reload or re-reading
  - Do not change data loading/refresh logic beyond the interval
  - Do not modify session launch behavior
  - Do not remove NewAppWithAggregator (keep as backward-compat wrapper)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Most complex UI wiring - touches keys, refresh timing, footer generation, header style, plus constructor chain management
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 2, 3, 5

  **References**:

  **Pattern References**:
  - `internal/ui/app.go:81-101` - AppModel struct and constructors to extend
  - `internal/ui/app.go:111-191` - Full Update() with key checks and refresh logic
  - `internal/ui/app.go:75-79` - tickCmd() with hardcoded 30s interval
  - `internal/ui/app.go:263-281` - View() with hardcoded header style and footer legend
  - `internal/ui/app.go:17-21` - Package-level headerStyle var to make config-aware
  - `internal/ui/app.go:277-279` - Hardcoded footer legend string to generate dynamically
  - `internal/ui/app.go:172-181` - Key handling for quit (ctrl+c always, q outside search, r outside search)

  **WHY Each Reference Matters**:
  - app.go Update() has nuanced key handling (ctrl+c vs q behavior differs during search mode) - executor needs to see this to avoid breaking the ctrl+c-always-quits guarantee
  - tickCmd shows the refresh interval to parameterize
  - View() footer/header show the exact rendering to change

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: App compiles with config wiring
    Tool: Bash
    Preconditions: AppModel updated with config
    Steps:
      1. Run: go build ./...
    Expected Result: Compilation succeeds
    Failure Indicators: Constructor chain mismatches
    Evidence: .sisyphus/evidence/task-10-compiles.txt

  Scenario: Backward-compat constructor still works
    Tool: Bash
    Preconditions: NewAppWithAggregator kept as wrapper
    Steps:
      1. Run: grep -A3 'func NewAppWithAggregator' internal/ui/app.go (expect function exists and calls NewAppWithConfig or config.Default)
      2. Run: grep -c 'NewAppWithAggregator' internal/ui/app.go (expect >= 1 - function definition present)
      3. Run: go build ./...
    Expected Result: NewAppWithAggregator function exists, uses config.Default(), build succeeds
    Failure Indicators: Function removed; grep returns 0 matches
    Evidence: .sisyphus/evidence/task-10-compat.txt

  Scenario: No hardcoded key strings in app.go Update()
    Tool: Bash
    Preconditions: All key replacements done
    Steps:
      1. Run: grep -n 'msg.String() ==' internal/ui/app.go
      2. Run: grep -c '"q"\|"r"' internal/ui/app.go (expect 0 in key-handling context - these should use config.Matches)
      3. Run: grep -c 'config.Matches\|Matches(' internal/ui/app.go (expect >= 2 - quit and refresh)
      4. Run: go build ./...
    Expected Result: msg.String()== not used for q/r; config.Matches used for quit and refresh; build succeeds
    Failure Indicators: "q" or "r" in direct string comparison outside of Matches calls
    Evidence: .sisyphus/evidence/task-10-no-hardcoded.txt

  Scenario: Full test suite passes
    Tool: Bash
    Preconditions: All changes made
    Steps:
      1. Run: go test ./...
    Expected Result: All tests pass
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-10-tests.txt
  ```

  **Commit**: YES
  - Message: `feat(ui): wire config into AppModel with configurable refresh and footer`
  - Files: `internal/ui/app.go`
  - Pre-commit: `go build ./...`

- [ ] 11. Wire display config into filter bar legend

  **What to do**:
  - This is a focused follow-up to Task 7 for the View() legend specifically
  - In `internal/ui/filterbar.go` View(), replace the hardcoded legend icon/color pairs with DisplayConfig lookups:
    ```go
    // Before:
    color("#FF0000", "!") + " err",
    color("#FF44FF", "@") + " waiting",
    // ...

    // After:
    color(m.display.ColorError, m.display.IconError) + " err",
    color(m.display.ColorWaiting, m.display.IconWaiting) + " waiting",
    color(m.display.ColorActive, m.display.IconActive) + " active",
    color(m.display.ColorReply, m.display.IconReply) + " reply",
    color(m.display.ColorStale, m.display.IconStale) + " stale",
    color(m.display.ColorTodos, m.display.IconTodos) + " todos",
    ```
  - Also update the active filter highlight color in View():
    ```go
    // Before:
    Background(lipgloss.Color("#7D56F4"))
    // After:
    Background(lipgloss.Color(m.display.ColorHeader))
    ```
  - Note: If Task 7 already did this, this task is a verification pass. If Task 7 focused on keys and left legend colors, this task completes the display wiring.

  **Must NOT do**:
  - Do not change the legend label text ("err", "waiting", "active", etc.)
  - Do not add new legend items

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Direct string replacements in one method
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 12
  - **Blocked By**: Tasks 2, 5, 6

  **References**:

  **Pattern References**:
  - `internal/ui/filterbar.go:150-160` - Legend section with hardcoded color/icon pairs
  - `internal/ui/filterbar.go:123-126` - Active preset highlight with `#7D56F4`

  **WHY Each Reference Matters**:
  - Lines 150-160 are the EXACT lines to change - 6 color/icon pairs
  - Line 126 is the accent color to parameterize

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Legend uses display config colors/icons (no hardcoded hex)
    Tool: Bash
    Preconditions: filterbar.go View() updated
    Steps:
      1. Run: grep -c '#FF0000\|#FF44FF\|#00FF00\|#FFFF00\|#FFA500\|#0088FF' internal/ui/filterbar.go (expect 0 - no hardcoded hex colors)
      2. Run: grep -c 'm.display\.\|display\.' internal/ui/filterbar.go (expect >= 6 - one per legend entry)
      3. Run: go build ./...
    Expected Result: Zero hardcoded color hex strings; display config references present; build succeeds
    Failure Indicators: Hardcoded hex colors like #FF0000 still in filterbar.go
    Evidence: .sisyphus/evidence/task-11-legend-config.txt

  Scenario: Full test suite passes
    Tool: Bash
    Preconditions: All changes made
    Steps:
      1. Run: go test ./...
    Expected Result: All tests pass
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-11-tests.txt
  ```

  **Commit**: YES (can group with Task 7 if done by same agent)
  - Message: `feat(ui): wire display config into filter bar legend`
  - Files: `internal/ui/filterbar.go`
  - Pre-commit: `go build ./...`

- [ ] 12. main.go - Add --config flag, load config, bridge types, wire through

  **What to do**:
  - In `cmd/opencode-dashboard/main.go`:
  - Add `--config` flag alongside existing `--db-path`:
    ```go
    configPath := flag.String("config", "", "path to config.toml (default: ~/.config/opencode-dashboard/config.toml)")
    ```
  - After `flag.Parse()`, load and validate config:
    ```go
    cfg, err := config.Load(*configPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
        os.Exit(1)
    }
    ```
  - Bridge config thresholds to attention package:
    ```go
    thresholds := attention.Thresholds{
        ActiveNowWindow:     time.Duration(cfg.Thresholds.ActiveNowMinutes) * time.Minute,
        NeedsResponseWindow: time.Duration(cfg.Thresholds.NeedsResponseHours) * time.Hour,
        StaleThreshold:      time.Duration(cfg.Thresholds.StaleWorkDays) * 24 * time.Hour,
    }
    ```
  - Create a custom classify closure and inject into aggregator. The existing `NewAggregator()` signature MUST NOT change. Instead, add a NEW constructor:
    ```go
    // NewAggregatorWithClassifier creates an Aggregator with a custom attention classifier.
    func NewAggregatorWithClassifier(
        classify func(domain.SessionView, time.Time) domain.AttentionSignal,
        projects domain.ProjectStore,
        sessions domain.SessionStore,
        messages domain.MessageStore,
        todos domain.TodoStore,
        errors domain.ErrorStore,
        waiting domain.WaitingStore,
    ) *Aggregator
    ```
    Usage in main.go:
    ```go
    classify := func(v domain.SessionView, now time.Time) domain.AttentionSignal {
        return attention.ClassifyWith(v, now, thresholds)
    }
    agg := app.NewAggregatorWithClassifier(classify, projectRepo, sessionRepo, messageRepo, todoRepo, errorRepo, waitingRepo)
    ```
  - In `internal/app/aggregator.go`: Add the new constructor. Refactor so both constructors share a common init path (e.g., NewAggregator calls NewAggregatorWithClassifier with `attention.Classify` as the default).
  - Existing `NewAggregator()` signature and all its call sites (including `aggregator_test.go`) remain COMPLETELY UNCHANGED.
  - Pass raw config to the UI (time window conversion happens inside `NewAppWithConfig`, NOT in main.go):
    ```go
    p := tea.NewProgram(ui.NewAppWithConfig(cfg, agg), tea.WithAltScreen())
    ```
  - main.go only converts attention thresholds (config -> attention.Thresholds). All other config-to-domain conversions happen in `ui/app.go` which imports both packages.
  - Add new imports: `config`, `attention`, `time`

  **Must NOT do**:
  - Do not add config subcommands (--dump-config, --validate-config)
  - Do not add environment variable overrides
  - Do not modify database, store, or repo code
  - Do not change error handling patterns for existing code

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Integration point - connects all packages, changes aggregator constructor, must verify everything works end-to-end
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on all other tasks)
  - **Parallel Group**: Wave 4 (final implementation)
  - **Blocks**: F1-F4
  - **Blocked By**: ALL Tasks 1-11

  **References**:

  **Pattern References**:
  - `cmd/opencode-dashboard/main.go:16-49` - Full current main.go showing flag parsing, DB opening, repo construction, aggregator creation, and TUI launch. This is the file being modified.
  - `internal/app/aggregator.go:26-43` - NewAggregator() constructor that needs the classify parameter added
  - `internal/app/aggregator.go:22` - `classify` field on Aggregator struct (already exists, currently set to `attention.Classify`)
  - `internal/app/aggregator_test.go` - Test file that calls NewAggregator() - needs updated constructor calls
  - `internal/attention/classifier.go` - ClassifyWith() function and Thresholds struct (from Task 4)
  - `internal/config/config.go` - Load() function and Config struct
  - `internal/ui/app.go` - NewAppWithConfig() constructor (from Task 10)

  **WHY Each Reference Matters**:
  - main.go is the file being modified - executor sees the exact integration point and current flow
  - aggregator.go shows the constructor to change and the classify field that receives the closure
  - aggregator_test.go shows test calls that need updating for the new constructor signature
  - attention classifier.go provides the ClassifyWith function being wrapped in the closure
  - config.go provides the Load function being called
  - ui/app.go provides the new constructor being called

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Binary builds successfully
    Tool: Bash
    Preconditions: All Tasks 1-11 completed
    Steps:
      1. Run: go build -o /tmp/opencode-dashboard-test ./cmd/opencode-dashboard
    Expected Result: Binary compiled successfully at /tmp/opencode-dashboard-test
    Failure Indicators: Compilation error
    Evidence: .sisyphus/evidence/task-12-build.txt

  Scenario: Help shows --config flag
    Tool: Bash
    Preconditions: Binary built
    Steps:
      1. Run: /tmp/opencode-dashboard-test --help 2>&1
    Expected Result: Output contains "--config" flag description mentioning config.toml
    Failure Indicators: --config not present in help output
    Evidence: .sisyphus/evidence/task-12-help-flag.txt

  Scenario: Launches successfully without config file
    Tool: interactive_bash (tmux)
    Preconditions: Binary built, no config file at default path
    Steps:
      1. Create tmux session: new-session -d -s config-test
      2. Run the dashboard: send-keys -t config-test "/tmp/opencode-dashboard-test --db-path /tmp/nonexistent.db" Enter
      3. Wait 2 seconds
      4. Capture output: capture-pane -t config-test -p
      5. Expected: Dashboard shows "Error opening database" (DB doesn't exist) but NOT a config error
      6. Kill session: kill-session -t config-test
    Expected Result: Error message is about DB, not config - proving config silently defaulted
    Failure Indicators: Config error message appears
    Evidence: .sisyphus/evidence/task-12-no-config-launch.txt

  Scenario: Invalid config file exits with clear error
    Tool: Bash
    Preconditions: Binary built
    Steps:
      1. Write invalid TOML to /tmp/bad-config.toml: echo "this is not [valid toml" > /tmp/bad-config.toml
      2. Run: /tmp/opencode-dashboard-test --config /tmp/bad-config.toml 2>&1; echo "EXIT: $?"
    Expected Result: stderr contains "Config error", exit code is 1
    Failure Indicators: No error message, or exit code 0
    Evidence: .sisyphus/evidence/task-12-bad-config.txt

  Scenario: Explicit --config with nonexistent path exits with error
    Tool: Bash
    Preconditions: Binary built
    Steps:
      1. Run: /tmp/opencode-dashboard-test --config /nonexistent/path.toml 2>&1; echo "EXIT: $?"
    Expected Result: stderr contains "Config error", exit code is 1 (not silent fallback)
    Failure Indicators: Silent fallback to defaults (exit code 0, no error)
    Evidence: .sisyphus/evidence/task-12-explicit-missing.txt

  Scenario: Full test suite passes
    Tool: Bash
    Preconditions: All changes made including aggregator constructor update
    Steps:
      1. Run: go test ./...
    Expected Result: ALL tests pass including updated aggregator tests
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-12-full-suite.txt

  Scenario: go vet clean
    Tool: Bash
    Preconditions: All changes made
    Steps:
      1. Run: go vet ./...
    Expected Result: No warnings or errors
    Failure Indicators: Any vet warning
    Evidence: .sisyphus/evidence/task-12-vet.txt
  ```

  **Commit**: YES
  - Message: `feat: wire config loading with --config flag in main.go`
  - Files: `cmd/opencode-dashboard/main.go`, `internal/app/aggregator.go`
  - Pre-commit: `go test ./...`

---

## Final Verification Wave (MANDATORY - after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
>
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**

- [ ] F1. **Plan Compliance Audit** - `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run `go test`, check config loading). For each "Must NOT Have": search codebase for forbidden patterns (hot-reload, env var overrides, config subcommands, store modifications). Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** - `unspecified-high`
  Run `go vet ./...` + `go build ./...` + `go test ./...`. Review all changed files for: unnecessary type assertions, empty error handling, unused imports, `fmt.Println` in production code. Check AI slop: excessive comments, over-abstraction, generic variable names (data/result/item/temp). Verify config package imports only stdlib + BurntSushi/toml.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** - `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task - follow exact steps, capture evidence. Test cross-task integration: launch TUI without config (defaults), launch with partial config, launch with full config, launch with invalid config. Test keybinding changes actually work in the TUI. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** - `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 - everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Verify store layer has ZERO changes. Verify FilterPreset is not configurable. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Order | Scope | Message | Files | Pre-commit |
|-------|-------|---------|-------|------------|
| 1 | deps | `chore: add BurntSushi/toml dependency` | go.mod, go.sum | `go build ./...` |
| 2 | config | `feat(config): add config package with TOML loading and validation` | internal/config/*.go | `go test ./internal/config/` |
| 3 | config | `feat(config): add KeyMap struct with key matching` | internal/config/keys*.go | `go test ./internal/config/` |
| 4 | attention | `feat(attention): add configurable thresholds via ClassifyWith` | internal/attention/classifier*.go | `go test ./internal/attention/` |
| 5 | domain+filter+ui | `refactor: replace TimeWindow enum with config-driven TimeWindowOption` | internal/domain/types.go, internal/filter/filter*.go, internal/ui/filterbar.go | `go test ./...` |
| 6 | ui | `refactor(ui): extract display config helpers in styles.go` | internal/ui/styles.go | `go build ./...` |
| 7-11 | ui | `feat(ui): wire config into {component}` | internal/ui/{component}.go | `go build ./...` |
| 12 | main | `feat: wire config loading with --config flag in main.go` | cmd/opencode-dashboard/main.go, internal/app/aggregator.go | `go test ./...` |

---

## Success Criteria

### Verification Commands
```bash
go vet ./...                    # Expected: clean, no warnings
go build ./cmd/opencode-dashboard  # Expected: compiles successfully
go test ./...                   # Expected: all tests pass (existing + new)
go test -v ./internal/config/   # Expected: config loading, merging, validation, KeyMap tests pass
go test -v ./internal/attention/ # Expected: all 14 existing + new ClassifyWith tests pass
go test -v ./internal/filter/   # Expected: all existing + updated window tests pass
```

### Final Checklist
- [ ] All "Must Have" present and verified via tests
- [ ] All "Must NOT Have" absent (no hot-reload, no env vars, no store changes, no preset config)
- [ ] All existing tests pass unmodified (backward compatibility)
- [ ] Config Default() values exactly match current hardcoded behavior
- [ ] `go test ./...` passes
- [ ] `go vet ./...` clean
- [ ] `go build ./cmd/opencode-dashboard` succeeds
