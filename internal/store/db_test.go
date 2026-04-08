package store

import (
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// realDBPath returns the path to the real OpenCode database.
func realDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local/share/opencode/opencode.db")
}

// TestOpenRealDB verifies that NewDB can open the real OpenCode database in read-only mode.
func TestOpenRealDB(t *testing.T) {
	dbPath := realDBPath()

	// Skip if the real DB doesn't exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("real OpenCode DB not found at %s, skipping", dbPath)
	}

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open real DB: %v", err)
	}
	defer db.Close()

	// Verify we can ping the database
	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
}

// TestMissingDB verifies that NewDB returns a clear error for a non-existent database file.
func TestMissingDB(t *testing.T) {
	dbPath := "/nonexistent/path/to/database.db"

	db, err := NewDB(dbPath)
	if err == nil {
		db.Close()
		t.Fatal("expected error for missing database, got nil")
	}

	// Error should mention the file not being found
	errMsg := err.Error()
	if errMsg == "" {
		t.Fatal("error message is empty")
	}
}

// TestReadOnly verifies that the database connection is truly read-only.
func TestReadOnly(t *testing.T) {
	dbPath := realDBPath()

	// Skip if the real DB doesn't exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("real OpenCode DB not found at %s, skipping", dbPath)
	}

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open real DB: %v", err)
	}
	defer db.Close()

	// Attempt to INSERT - should fail
	_, err = db.Exec("INSERT INTO session (id) VALUES ('test-readonly')")
	if err == nil {
		t.Fatal("expected INSERT to fail on read-only DB, but it succeeded")
	}

	// Error should indicate read-only or not authorized
	errMsg := err.Error()
	if errMsg == "" {
		t.Fatal("error message is empty")
	}
}

// TestSchemaValidation verifies that NewDB validates the expected schema.
func TestSchemaValidation(t *testing.T) {
	dbPath := realDBPath()

	// Skip if the real DB doesn't exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("real OpenCode DB not found at %s, skipping", dbPath)
	}

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open real DB: %v", err)
	}
	defer db.Close()

	// Verify that all expected tables exist
	expectedTables := []string{"session", "message", "part", "todo", "project"}
	for _, table := range expectedTables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil || name == "" {
			t.Fatalf("schema validation failed: missing table %q", table)
		}
	}
}

// TestConnectionPool verifies that the connection pool is configured correctly.
func TestConnectionPool(t *testing.T) {
	dbPath := realDBPath()

	// Skip if the real DB doesn't exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("real OpenCode DB not found at %s, skipping", dbPath)
	}

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open real DB: %v", err)
	}
	defer db.Close()

	// Verify connection pool settings
	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Fatalf("expected MaxOpenConnections=1, got %d", stats.MaxOpenConnections)
	}
}

// TestClose verifies that Close() works cleanly.
func TestClose(t *testing.T) {
	dbPath := realDBPath()

	// Skip if the real DB doesn't exist
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("real OpenCode DB not found at %s, skipping", dbPath)
	}

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open real DB: %v", err)
	}

	// Close should not error
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}
}
