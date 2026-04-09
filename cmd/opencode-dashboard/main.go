package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joeyparis/opencode-dashboard/internal/app"
	"github.com/joeyparis/opencode-dashboard/internal/attention"
	"github.com/joeyparis/opencode-dashboard/internal/config"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/joeyparis/opencode-dashboard/internal/store"
	"github.com/joeyparis/opencode-dashboard/internal/ui"
)

func main() {
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".local", "share", "opencode", "opencode.db")

	dbPath := flag.String("db-path", defaultDB, "path to OpenCode SQLite database")
	configPath := flag.String("config", "", "path to config.toml (default: ~/.config/opencode-dashboard/config.toml)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	thresholds := attention.Thresholds{
		ActiveNowWindow:     time.Duration(cfg.Thresholds.ActiveNowMinutes) * time.Minute,
		NeedsResponseWindow: time.Duration(cfg.Thresholds.NeedsResponseHours) * time.Hour,
		StaleThreshold:      time.Duration(cfg.Thresholds.StaleWorkDays) * 24 * time.Hour,
	}
	classify := func(v domain.SessionView, now time.Time) domain.AttentionSignal {
		return attention.ClassifyWith(v, now, thresholds)
	}

	db, err := store.NewDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	projectRepo := store.NewProjectRepo(db)
	sessionRepo := store.NewSessionRepo(db)
	messageRepo := store.NewMessageRepo(db)
	todoRepo := store.NewTodoRepo(db)
	errorRepo := store.NewErrorRepo(db)
	waitingRepo := store.NewWaitingRepo(db)

	if err := errorRepo.RefreshAll(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading error cache: %v\n", err)
		os.Exit(1)
	}

	agg := app.NewAggregatorWithClassifier(classify, projectRepo, sessionRepo, messageRepo, todoRepo, errorRepo, waitingRepo)
	p := tea.NewProgram(ui.NewAppWithConfig(cfg, agg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
