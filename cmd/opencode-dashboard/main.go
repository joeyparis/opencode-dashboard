package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joeyparis/opencode-dashboard/internal/ui"
)

func main() {
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".local", "share", "opencode", "opencode.db")

	dbPath := flag.String("db-path", defaultDB, "path to OpenCode SQLite database")
	flag.Parse()

	_ = dbPath // will be used in future tasks

	p := tea.NewProgram(ui.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
