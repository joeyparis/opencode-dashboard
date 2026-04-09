package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefault_AllFieldsPopulated(t *testing.T) {
	cfg := Default()

	assert.NotEmpty(t, cfg.Defaults.Filter)
	assert.NotEmpty(t, cfg.Defaults.TimeWindow)
	assert.NotZero(t, cfg.Defaults.RefreshInterval)
	assert.NotZero(t, cfg.Defaults.ListWidthRatio)

	assert.NotZero(t, cfg.Thresholds.ActiveNowMinutes)
	assert.NotZero(t, cfg.Thresholds.NeedsResponseHours)
	assert.NotZero(t, cfg.Thresholds.StaleWorkDays)

	assert.NotEmpty(t, cfg.TimeWindows.Options)
	for _, option := range cfg.TimeWindows.Options {
		assert.NotEmpty(t, option.Label)
		assert.NotZero(t, option.Hours)
	}

	assert.NotEmpty(t, cfg.Keys.Up)
	assert.NotEmpty(t, cfg.Keys.Down)
	assert.NotEmpty(t, cfg.Keys.PaneLeft)
	assert.NotEmpty(t, cfg.Keys.PaneRight)
	assert.NotEmpty(t, cfg.Keys.TreeNav)
	assert.NotEmpty(t, cfg.Keys.Collapse)
	assert.NotEmpty(t, cfg.Keys.CycleFilter)
	assert.NotEmpty(t, cfg.Keys.CycleTime)
	assert.NotEmpty(t, cfg.Keys.Search)
	assert.NotEmpty(t, cfg.Keys.Refresh)
	assert.NotEmpty(t, cfg.Keys.Launch)
	assert.NotEmpty(t, cfg.Keys.Quit)

	assert.NotEmpty(t, cfg.Display.IconError)
	assert.NotEmpty(t, cfg.Display.IconWaiting)
	assert.NotEmpty(t, cfg.Display.IconActive)
	assert.NotEmpty(t, cfg.Display.IconReply)
	assert.NotEmpty(t, cfg.Display.IconStale)
	assert.NotEmpty(t, cfg.Display.IconTodos)
	assert.NotEmpty(t, cfg.Display.ColorError)
	assert.NotEmpty(t, cfg.Display.ColorWaiting)
	assert.NotEmpty(t, cfg.Display.ColorActive)
	assert.NotEmpty(t, cfg.Display.ColorReply)
	assert.NotEmpty(t, cfg.Display.ColorStale)
	assert.NotEmpty(t, cfg.Display.ColorTodos)
	assert.NotEmpty(t, cfg.Display.ColorHeader)
}

func TestDefault_GoldenValues(t *testing.T) {
	cfg := Default()

	assert.Equal(t, 30, cfg.Defaults.RefreshInterval)
	assert.Equal(t, 5, cfg.Thresholds.ActiveNowMinutes)
	assert.Equal(t, 24, cfg.Thresholds.NeedsResponseHours)
	assert.Equal(t, 7, cfg.Thresholds.StaleWorkDays)
	assert.Equal(t, 0.5, cfg.Defaults.ListWidthRatio)
	assert.Equal(t, "!", cfg.Display.IconError)
	assert.Equal(t, "#FF0000", cfg.Display.ColorError)
	assert.Equal(t, "#7D56F4", cfg.Display.ColorHeader)
}

func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := Load("")

	assert.NoError(t, err)
	assert.Equal(t, Default(), cfg)
}

func TestLoad_ExplicitPathMissing_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")

	_, err := Load(path)

	assert.Error(t, err)
	assert.ErrorContains(t, err, path)
}

func TestLoad_EmptyFile_ReturnsDefaults(t *testing.T) {
	path := writeTempConfig(t, "")

	cfg, err := Load(path)

	assert.NoError(t, err)
	assert.Equal(t, Default(), cfg)
}

func TestLoad_PartialConfig_MergesDefaults(t *testing.T) {
	path := writeTempConfig(t, `
[thresholds]
active_now_minutes = 15
needs_response_hours = 48
stale_work_days = 14
`)

	cfg, err := Load(path)

	assert.NoError(t, err)
	assert.Equal(t, Default().Defaults, cfg.Defaults)
	assert.Equal(t, ThresholdsConfig{
		ActiveNowMinutes:   15,
		NeedsResponseHours: 48,
		StaleWorkDays:      14,
	}, cfg.Thresholds)
	assert.Equal(t, Default().TimeWindows, cfg.TimeWindows)
	assert.Equal(t, Default().Keys, cfg.Keys)
	assert.Equal(t, Default().Display, cfg.Display)
}

func TestLoad_FullConfig(t *testing.T) {
	path := writeTempConfig(t, `
[defaults]
filter = "all"
time_window = "7d"
refresh_interval = 60
list_width_ratio = 0.33

[thresholds]
active_now_minutes = 10
needs_response_hours = 12
stale_work_days = 21

[[time_windows.options]]
label = "12h"
hours = 12

[[time_windows.options]]
label = "30d"
hours = 720

[keys]
up = ["w"]
down = ["s"]
pane_left = ["a"]
pane_right = ["d"]
tree_nav = ["backspace"]
collapse = ["space"]
cycle_filter = ["f"]
cycle_time = ["y"]
search = ["ctrl+f"]
refresh = ["F5"]
launch = ["o"]
quit = ["esc"]

[display]
icon_error = "E"
icon_waiting = "W"
icon_active = "A"
icon_reply = "R"
icon_stale = "S"
icon_todos = "T"
color_error = "#111111"
color_waiting = "#222222"
color_active = "#333333"
color_reply = "#444444"
color_stale = "#555555"
color_todos = "#666666"
color_selected = "#777777"
color_header = "#888888"
`)

	cfg, err := Load(path)

	assert.NoError(t, err)
	assert.Equal(t, Config{
		Defaults: DefaultsConfig{
			Filter:          "all",
			TimeWindow:      "7d",
			RefreshInterval: 60,
			ListWidthRatio:  0.33,
		},
		Thresholds: ThresholdsConfig{
			ActiveNowMinutes:   10,
			NeedsResponseHours: 12,
			StaleWorkDays:      21,
		},
		TimeWindows: TimeWindowsConfig{Options: []TimeWindowOption{{Label: "12h", Hours: 12}, {Label: "30d", Hours: 720}}},
		Keys: KeysConfig{
			Up:          []string{"w"},
			Down:        []string{"s"},
			PaneLeft:    []string{"a"},
			PaneRight:   []string{"d"},
			TreeNav:     []string{"backspace"},
			Collapse:    []string{"space"},
			CycleFilter: []string{"f"},
			CycleTime:   []string{"y"},
			Search:      []string{"ctrl+f"},
			Refresh:     []string{"F5"},
			Launch:      []string{"o"},
			Quit:        []string{"esc"},
		},
		Display: DisplayConfig{
			IconError:     "E",
			IconWaiting:   "W",
			IconActive:    "A",
			IconReply:     "R",
			IconStale:     "S",
			IconTodos:     "T",
			ColorError:    "#111111",
			ColorWaiting:  "#222222",
			ColorActive:   "#333333",
			ColorReply:    "#444444",
			ColorStale:    "#555555",
			ColorTodos:    "#666666",
			ColorSelected: "#777777",
			ColorHeader:   "#888888",
		},
	}, cfg)
}

func TestLoad_ParseError_ReturnsError(t *testing.T) {
	path := writeTempConfig(t, `
[defaults
filter = "broken"
`)

	_, err := Load(path)

	assert.Error(t, err)
}

func TestLoad_SliceReplacement(t *testing.T) {
	path := writeTempConfig(t, `
[[time_windows.options]]
label = "30m"
hours = 1
`)

	cfg, err := Load(path)

	assert.NoError(t, err)
	assert.Equal(t, []TimeWindowOption{{Label: "30m", Hours: 1}}, cfg.TimeWindows.Options)
}

func TestValidate_RefreshTooLow(t *testing.T) {
	cfg := Default()
	cfg.Defaults.RefreshInterval = 5

	err := Validate(cfg)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "defaults.refresh_interval")
}

func TestValidate_RatioOutOfRange(t *testing.T) {
	t.Run("too low", func(t *testing.T) {
		cfg := Default()
		cfg.Defaults.ListWidthRatio = 0.0

		err := Validate(cfg)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "defaults.list_width_ratio")
	})

	t.Run("too high", func(t *testing.T) {
		cfg := Default()
		cfg.Defaults.ListWidthRatio = 1.0

		err := Validate(cfg)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "defaults.list_width_ratio")
	})

	t.Run("lower bound ok", func(t *testing.T) {
		cfg := Default()
		cfg.Defaults.ListWidthRatio = 0.2

		assert.NoError(t, Validate(cfg))
	})

	t.Run("upper bound ok", func(t *testing.T) {
		cfg := Default()
		cfg.Defaults.ListWidthRatio = 0.8

		assert.NoError(t, Validate(cfg))
	})
}

func TestValidate_NegativeThresholds(t *testing.T) {
	t.Run("active now minutes", func(t *testing.T) {
		cfg := Default()
		cfg.Thresholds.ActiveNowMinutes = 0

		err := Validate(cfg)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "thresholds.active_now_minutes")
	})

	t.Run("needs response hours", func(t *testing.T) {
		cfg := Default()
		cfg.Thresholds.NeedsResponseHours = 0

		err := Validate(cfg)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "thresholds.needs_response_hours")
	})

	t.Run("stale work days", func(t *testing.T) {
		cfg := Default()
		cfg.Thresholds.StaleWorkDays = 0

		err := Validate(cfg)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "thresholds.stale_work_days")
	})
}

func TestValidate_EmptyQuitBinding(t *testing.T) {
	cfg := Default()
	cfg.Keys.Quit = nil

	err := Validate(cfg)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "keys.quit")
}

func TestResolvePath_XDGOverride(t *testing.T) {
	xdgHome := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)
	t.Setenv("HOME", home)

	expected := filepath.Join(xdgHome, "opencode-dashboard", "config.toml")
	assert.NoError(t, os.MkdirAll(filepath.Dir(expected), 0o755))
	assert.NoError(t, os.WriteFile(expected, []byte(""), 0o644))

	assert.Equal(t, expected, ResolvePath())
}

func TestResolvePath_HomeFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	expected := filepath.Join(home, ".config", "opencode-dashboard", "config.toml")
	assert.NoError(t, os.MkdirAll(filepath.Dir(expected), 0o755))
	assert.NoError(t, os.WriteFile(expected, []byte(""), 0o644))

	assert.Equal(t, expected, ResolvePath())
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	assert.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}
