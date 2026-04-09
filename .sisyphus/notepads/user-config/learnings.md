# Learnings - user-config

## Project Conventions
- Module: `github.com/joeyparis/opencode-dashboard`
- Go 1.25, all tests use `github.com/stretchr/testify/assert`
- Test helper pattern: `fixedNow()` for deterministic time (see classifier_test.go:149)
- UI models are value types (not pointers), stores/aggregator are pointer types
- No CGO - using `modernc.org/sqlite` pure-Go driver

## Package Boundaries (CRITICAL)
- `internal/config` MUST NOT import any internal package - defines its own types
- Conversion from `config.TimeWindowOption{Hours int}` to `domain.TimeWindowOption{Duration}` happens in `internal/ui/app.go`
- Attention threshold conversion happens in `cmd/opencode-dashboard/main.go`
- `internal/store/` must NOT be modified (config feature)

## Key Architecture Facts
- Aggregator.classify is `func(domain.SessionView, time.Time) domain.AttentionSignal` - inject via closure in main.go
- `NewAggregator()` signature MUST stay unchanged; add `NewAggregatorWithClassifier()` alongside
- `filter.Apply()` will gain a `now time.Time` parameter (testability fix, part of Task 5)
- TimeWindow enum (iota) is FULLY REMOVED and replaced with `domain.TimeWindowOption{Label, Duration}`
- "All" sentinel appended last: `{Label: "All", Duration: 0}`

## Default Values (EXACT)
- ActiveNowWindow: 5 minutes
- NeedsResponseWindow: 24 hours
- StaleThreshold: 7 days
- RefreshInterval: 30 seconds
- ListWidthRatio: 0.5
- Default filter: needs_attention
- Default time window: 3d
- ColorHeader: "#7D56F4"
- Hardcoded keys: Up=["k","up"], Down=["j","down"], PaneLeft=["h"], PaneRight=["l","right"], TreeNav=["left"], Collapse=[" "], CycleFilter=["tab"], CycleTime=["t"], Search=["/"], Refresh=["r"], Launch=["enter"], Quit=["q","ctrl+c"]
