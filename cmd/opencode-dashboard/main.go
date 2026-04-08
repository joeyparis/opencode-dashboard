package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joeyparis/opencode-dashboard/internal/app"
	"github.com/joeyparis/opencode-dashboard/internal/store"
	"github.com/joeyparis/opencode-dashboard/internal/ui"
)

func main() {
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".local", "share", "opencode", "opencode.db")

	dbPath := flag.String("db-path", defaultDB, "path to OpenCode SQLite database")
	flag.Parse()

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

	agg := app.NewAggregator(projectRepo, sessionRepo, messageRepo, todoRepo, errorRepo, waitingRepo)
	p := tea.NewProgram(ui.NewAppWithAggregator(agg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
