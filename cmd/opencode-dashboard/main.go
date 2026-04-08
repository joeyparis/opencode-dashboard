package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	home, _ := os.UserHomeDir()
	defaultDB := filepath.Join(home, ".local", "share", "opencode", "opencode.db")

	dbPath := flag.String("db-path", defaultDB, "path to OpenCode SQLite database")
	flag.Parse()

	_ = dbPath // will be used in future tasks
	fmt.Println("opencode-dashboard")
}
