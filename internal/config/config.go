package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Defaults    DefaultsConfig    `toml:"defaults"`
	Thresholds  ThresholdsConfig  `toml:"thresholds"`
	TimeWindows TimeWindowsConfig `toml:"time_windows"`
	Keys        KeysConfig        `toml:"keys"`
	Display     DisplayConfig     `toml:"display"`
}

type DefaultsConfig struct {
	Filter          string  `toml:"filter"`
	TimeWindow      string  `toml:"time_window"`
	RefreshInterval int     `toml:"refresh_interval"`
	ListWidthRatio  float64 `toml:"list_width_ratio"`
}

type ThresholdsConfig struct {
	ActiveNowMinutes   int `toml:"active_now_minutes"`
	NeedsResponseHours int `toml:"needs_response_hours"`
	StaleWorkDays      int `toml:"stale_work_days"`
}

type TimeWindowsConfig struct {
	Options []TimeWindowOption `toml:"options"`
}

type TimeWindowOption struct {
	Label string `toml:"label"`
	Hours int    `toml:"hours"`
}

type KeysConfig struct {
	Up          []string `toml:"up"`
	Down        []string `toml:"down"`
	PaneLeft    []string `toml:"pane_left"`
	PaneRight   []string `toml:"pane_right"`
	TreeNav     []string `toml:"tree_nav"`
	Collapse    []string `toml:"collapse"`
	CycleFilter []string `toml:"cycle_filter"`
	CycleTime   []string `toml:"cycle_time"`
	Search      []string `toml:"search"`
	Refresh     []string `toml:"refresh"`
	Launch      []string `toml:"launch"`
	Quit        []string `toml:"quit"`
}

type DisplayConfig struct {
	IconError     string `toml:"icon_error"`
	IconWaiting   string `toml:"icon_waiting"`
	IconActive    string `toml:"icon_active"`
	IconReply     string `toml:"icon_reply"`
	IconStale     string `toml:"icon_stale"`
	IconTodos     string `toml:"icon_todos"`
	ColorError    string `toml:"color_error"`
	ColorWaiting  string `toml:"color_waiting"`
	ColorActive   string `toml:"color_active"`
	ColorReply    string `toml:"color_reply"`
	ColorStale    string `toml:"color_stale"`
	ColorTodos    string `toml:"color_todos"`
	ColorSelected string `toml:"color_selected"`
	ColorHeader   string `toml:"color_header"`
}

func Default() Config {
	return Config{
		Defaults: DefaultsConfig{
			Filter:          "needs_attention",
			TimeWindow:      "3d",
			RefreshInterval: 30,
			ListWidthRatio:  0.5,
		},
		Thresholds: ThresholdsConfig{
			ActiveNowMinutes:   5,
			NeedsResponseHours: 24,
			StaleWorkDays:      7,
		},
		TimeWindows: TimeWindowsConfig{
			Options: []TimeWindowOption{
				{Label: "1d", Hours: 24},
				{Label: "3d", Hours: 72},
				{Label: "7d", Hours: 168},
			},
		},
		Keys: KeysConfig{
			Up:          []string{"k", "up"},
			Down:        []string{"j", "down"},
			PaneLeft:    []string{"h"},
			PaneRight:   []string{"l", "right"},
			TreeNav:     []string{"left"},
			Collapse:    []string{" "},
			CycleFilter: []string{"tab"},
			CycleTime:   []string{"t"},
			Search:      []string{"/"},
			Refresh:     []string{"r"},
			Launch:      []string{"enter"},
			Quit:        []string{"q", "ctrl+c"},
		},
		Display: DisplayConfig{
			IconError:     "!",
			IconWaiting:   "@",
			IconActive:    "*",
			IconReply:     "?",
			IconStale:     "~",
			IconTodos:     ".",
			ColorError:    "#FF0000",
			ColorWaiting:  "#FF44FF",
			ColorActive:   "#00FF00",
			ColorReply:    "#FFFF00",
			ColorStale:    "#FFA500",
			ColorTodos:    "#0088FF",
			ColorSelected: "",
			ColorHeader:   "#7D56F4",
		},
	}
}

func Load(explicitPath string) (Config, error) {
	path := explicitPath
	if path == "" {
		path = ResolvePath()
		if path == "" {
			return Default(), nil
		}
	} else {
		if _, err := os.Stat(path); err != nil {
			return Config{}, err
		}
	}

	cfg := Default()
	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return Config{}, err
	}

	for _, key := range meta.Undecoded() {
		_, _ = fmt.Fprintf(os.Stderr, "warning: unknown config key: %s\n", key)
	}

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func ResolvePath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		path := filepath.Join(xdg, "opencode-dashboard", "config.toml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}

	path := filepath.Join(home, ".config", "opencode-dashboard", "config.toml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}

func Validate(cfg Config) error {
	if cfg.Defaults.RefreshInterval < 10 {
		return fmt.Errorf("defaults.refresh_interval must be >= 10")
	}

	if cfg.Defaults.ListWidthRatio < 0.2 || cfg.Defaults.ListWidthRatio > 0.8 {
		return fmt.Errorf("defaults.list_width_ratio must be between 0.2 and 0.8")
	}

	if cfg.Thresholds.ActiveNowMinutes <= 0 {
		return fmt.Errorf("thresholds.active_now_minutes must be > 0")
	}

	if cfg.Thresholds.NeedsResponseHours <= 0 {
		return fmt.Errorf("thresholds.needs_response_hours must be > 0")
	}

	if cfg.Thresholds.StaleWorkDays <= 0 {
		return fmt.Errorf("thresholds.stale_work_days must be > 0")
	}

	if len(cfg.Keys.Quit) == 0 {
		return fmt.Errorf("keys.quit must not be empty")
	}

	return nil
}
