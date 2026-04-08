package store

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func NewDB(dbPath string) (*sql.DB, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found: %s", dbPath)
	}

	dsn := fmt.Sprintf("file:%s?mode=ro", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=wal"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set journal_mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA query_only=on"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set query_only: %w", err)
	}

	if err := validateSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func validateSchema(db *sql.DB) error {
	tables := []string{"session", "message", "part", "todo", "project"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil || name == "" {
			return fmt.Errorf("schema validation failed: missing table %q", table)
		}
	}
	return nil
}
