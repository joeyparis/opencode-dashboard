# Decisions - user-config

## filter.Filter.Window Type
Decision: `time.Duration` (resolved value, not the option struct or an index).
Reason: Simplest; UI layer resolves config option to duration before building Filter.

## Time Window "All" Position
Decision: Always APPENDED LAST (not first).
Reason: Issue spec says "All is always appended"; changes cycle order from "All->1d->3d->7d" to "1d->3d->7d->All".

## XDG Path Resolution
Decision: Check `$XDG_CONFIG_HOME/opencode-dashboard/config.toml`, fall back to `~/.config/opencode-dashboard/config.toml`.
Reason: Issue spec says `~/.config/...`; `os.UserConfigDir()` returns `~/Library/Application Support` on macOS which is wrong for this feature.

## Explicit --config with Missing File
Decision: Return error (NOT silent fallback).
Reason: If user explicitly specifies a path, they expect it to exist. Silent fallback would hide misconfiguration.

## TOML Slice Replacement Semantics
Decision: User-defined time windows REPLACE defaults entirely (not merged).
Reason: BurntSushi/toml naturally replaces slices; this is documented and tested.

## Unknown TOML Keys
Decision: Warn to stderr (NOT error).
Reason: Allows forward-compatibility; user has extra keys from a newer version, shouldn't break.

## Space Key Alias
Decision: "space" accepted as alias, normalized to " " internally.
Reason: `" "` in TOML is confusing/error-prone for users.

## sectionHeaderStyle
Decision: Uses `ColorHeader` (shared accent color), not a separate config field.
Reason: The same #7D56F4 accent appears everywhere; single configurable color is simpler.
