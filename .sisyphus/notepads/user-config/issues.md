# Issues - user-config

## Known Gotchas

### Task 5 is ATOMIC
Task 5 touches domain/types.go, filter/filter.go, filter/filter_test.go, filterbar.go, app.go simultaneously. All must change in one commit or the build breaks. The executor must make all changes before running `go build ./...`.

### KeysConfig vs KeyMap
The `config.KeysConfig` struct (TOML tags for loading) and `config.KeyMap` struct (used by UI for matching) may be the same struct or related. The plan specifies KeyMap as the UI type with `Matches()` as a standalone function. The Keys field on Config is `KeysConfig` (with TOML tags). Whether these are unified is up to the executor - just ensure the UI components receive a KeyMap-compatible type.

### ctrl+c Always Quits
In `app.go`, ctrl+c must ALWAYS quit even during search mode. The simplest implementation: check Quit bindings normally, but keep a separate direct check for ctrl+c that bypasses search mode. Alternatively: check if keyStr is ctrl+c specifically from the Quit bindings.

### headerStyle is a Package-Level Var
Currently `var headerStyle = lipgloss.NewStyle()...` at package level. To make it config-aware, compute it in NewAppWithConfig() constructor and store on AppModel, then use the stored style in View(). Don't recreate per-render.

### filterbar.go is Modified by Tasks 5, 7, AND 11
Task 5 changes the window-related fields/constructor/cycling.
Task 7 adds keys/display fields and replaces key checks.
Task 11 replaces legend colors (may overlap with Task 7).
These are sequential (7 depends on 5; 11 depends on 5,6), so no conflict in execution order.

### Task 5 Interim Layout Default
Until T9 expands `NewLayout()` config inputs, `internal/ui/layout.go` hardcodes the default window options and passes default index `1` to `NewFilterBar()` so the default remains `3d`.
