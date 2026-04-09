package testutil

import (
	"database/sql"
	"testing"
)

func TestNewTestDB_ProjectCount(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM project").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query project count: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 projects, got %d", count)
	}
}

func TestNewTestDB_RootSessionCount(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM session WHERE parent_id IS NULL").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query root session count: %v", err)
	}

	if count != 8 {
		t.Errorf("expected 8 root sessions, got %d", count)
	}
}

func TestNewTestDB_ChildSessionCount(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM session WHERE parent_id IS NOT NULL").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query child session count: %v", err)
	}

	if count != 4 {
		t.Errorf("expected 4 child sessions, got %d", count)
	}
}

func TestNewTestDB_TotalSessionCount(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM session").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query total session count: %v", err)
	}

	if count != 12 {
		t.Errorf("expected 12 total sessions, got %d", count)
	}
}

func TestNewTestDB_AttentionSignalScenarios(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	tests := []struct {
		name      string
		sessionID string
		query     string
	}{
		{
			name:      "NeedsResponse scenario",
			sessionID: "ses-needs-response",
			query:     "SELECT COUNT(*) FROM message WHERE session_id = ? AND json_extract(data, '$.role') = 'assistant'",
		},
		{
			name:      "ActiveNow scenario",
			sessionID: "ses-active-now",
			query:     "SELECT COUNT(*) FROM session WHERE id = ?",
		},
		{
			name:      "HasErrors scenario",
			sessionID: "ses-has-errors",
			query:     "SELECT COUNT(*) FROM part WHERE session_id = ? AND json_extract(data, '$.state.status') = 'error'",
		},
		{
			name:      "StaleWork scenario",
			sessionID: "ses-stale",
			query:     "SELECT COUNT(*) FROM todo WHERE session_id = ? AND status = 'pending'",
		},
		{
			name:      "PendingTodos scenario",
			sessionID: "ses-pending-todos",
			query:     "SELECT COUNT(*) FROM todo WHERE session_id = ? AND status = 'pending'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int
			err := db.QueryRow(tt.query, tt.sessionID).Scan(&count)
			if err != nil {
				t.Fatalf("failed to query %s: %v", tt.name, err)
			}

			if count < 1 {
				t.Errorf("%s: expected at least 1 matching record, got %d", tt.name, count)
			}
		})
	}
}

func TestNewTestDB_MessageData(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM message").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query message count: %v", err)
	}

	if count < 1 {
		t.Errorf("expected at least 1 message, got %d", count)
	}
}

func TestNewTestDB_ErrorParts(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM part WHERE session_id = 'ses-has-errors'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query error parts: %v", err)
	}

	if count != 4 {
		t.Errorf("expected 4 error parts for ses-has-errors, got %d", count)
	}
}

func TestNewTestDB_TodoData(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	tests := []struct {
		name      string
		sessionID string
		status    string
		expected  int
	}{
		{"ses-stale pending", "ses-stale", "pending", 2},
		{"ses-pending-todos pending", "ses-pending-todos", "pending", 2},
		{"ses-pending-todos completed", "ses-pending-todos", "completed", 1},
		{"ses-no-signal completed", "ses-no-signal", "completed", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int
			err := db.QueryRow(
				"SELECT COUNT(*) FROM todo WHERE session_id = ? AND status = ?",
				tt.sessionID, tt.status,
			).Scan(&count)
			if err != nil {
				t.Fatalf("failed to query todos: %v", err)
			}

			if count != tt.expected {
				t.Errorf("expected %d %s todos for %s, got %d", tt.expected, tt.status, tt.sessionID, count)
			}
		})
	}
}

func TestNewTestDB_ArchivedSession(t *testing.T) {
	db := NewTestDB(t)
	defer db.Close()

	var archived sql.NullInt64
	err := db.QueryRow("SELECT time_archived FROM session WHERE id = 'ses-archived'").Scan(&archived)
	if err != nil {
		t.Fatalf("failed to query archived session: %v", err)
	}

	if !archived.Valid {
		t.Errorf("expected ses-archived to have time_archived set")
	}
}
