package testutil

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// NewTestDB creates an in-memory SQLite database seeded with test data.
// It returns a *sql.DB ready for use in tests.
func NewTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// Create schema
	if err := createSchema(db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Seed data
	if err := seedData(db); err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	return db
}

func createSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS project (
    id TEXT PRIMARY KEY,
    worktree TEXT,
    vcs TEXT,
    name TEXT,
    icon_url TEXT,
    icon_color TEXT,
    time_created INTEGER,
    time_updated INTEGER,
    time_initialized INTEGER,
    sandboxes TEXT,
    commands TEXT
);

CREATE TABLE IF NOT EXISTS session (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    parent_id TEXT,
    slug TEXT,
    directory TEXT,
    title TEXT,
    version TEXT,
    summary_additions INTEGER,
    summary_deletions INTEGER,
    summary_files INTEGER,
    time_created INTEGER,
    time_updated INTEGER,
    time_archived INTEGER,
    workspace_id TEXT,
    share_url TEXT,
    revert TEXT,
    permission TEXT,
    time_compacting INTEGER,
    summary_diffs TEXT
);

CREATE TABLE IF NOT EXISTS message (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    time_created INTEGER,
    time_updated INTEGER,
    data TEXT
);

CREATE TABLE IF NOT EXISTS part (
    id TEXT PRIMARY KEY,
    message_id TEXT,
    session_id TEXT,
    time_created INTEGER,
    time_updated INTEGER,
    data TEXT
);

CREATE TABLE IF NOT EXISTS todo (
    session_id TEXT,
    content TEXT,
    status TEXT,
    priority TEXT,
    position INTEGER,
    time_created INTEGER,
    time_updated INTEGER,
    PRIMARY KEY (session_id, position)
);

CREATE INDEX IF NOT EXISTS message_session_time_created_id_idx ON message(session_id, time_created, id);
CREATE INDEX IF NOT EXISTS part_session_idx ON part(session_id);
CREATE INDEX IF NOT EXISTS session_project_idx ON session(project_id);
CREATE INDEX IF NOT EXISTS todo_session_idx ON todo(session_id);
`

	_, err := db.Exec(schema)
	return err
}

func seedData(db *sql.DB) error {
	now := time.Now()

	// Seed projects
	projects := []struct {
		id       string
		worktree string
		vcs      string
	}{
		{"project-alpha", "/Users/joey/Sites/alpha", "git"},
		{"project-beta", "/Users/joey/Sites/beta", "git"},
		{"global", "/", "git"},
	}

	for _, p := range projects {
		_, err := db.Exec(`
INSERT INTO project (id, worktree, vcs, name, time_created, time_updated)
VALUES (?, ?, ?, NULL, ?, ?)
`, p.id, p.worktree, p.vcs, now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return fmt.Errorf("failed to insert project %s: %w", p.id, err)
		}
	}

	// Seed root sessions
	rootSessions := []struct {
		id        string
		projectID string
		updated   time.Time
		archived  *int64
	}{
		{"ses-needs-response", "project-alpha", now.Add(-1 * time.Hour), nil},
		{"ses-active-now", "project-alpha", now.Add(-2 * time.Minute), nil},
		{"ses-has-errors", "project-alpha", now.Add(-30 * time.Minute), nil},
		{"ses-stale", "project-beta", now.Add(-10 * 24 * time.Hour), nil},
		{"ses-pending-todos", "project-beta", now.Add(-24 * time.Hour), nil},
		{"ses-no-signal", "project-beta", now.Add(-5 * 24 * time.Hour), nil},
		{"ses-archived", "project-alpha", now.Add(-30 * 24 * time.Hour), ptrInt64(now.Add(-30 * 24 * time.Hour).UnixMilli())},
		{"ses-global", "global", now.Add(-48 * time.Hour), nil},
	}

	for _, s := range rootSessions {
		archivedVal := sql.NullInt64{}
		if s.archived != nil {
			archivedVal = sql.NullInt64{Int64: *s.archived, Valid: true}
		}

		_, err := db.Exec(`
INSERT INTO session (id, project_id, parent_id, slug, directory, title, version, 
                     time_created, time_updated, time_archived)
VALUES (?, ?, NULL, ?, ?, ?, ?, ?, ?, ?)
`, s.id, s.projectID, s.id, "/tmp/"+s.id, s.id, "v1", now.UnixMilli(), s.updated.UnixMilli(), archivedVal)
		if err != nil {
			return fmt.Errorf("failed to insert root session %s: %w", s.id, err)
		}
	}

	// Seed child sessions
	childSessions := []struct {
		id       string
		parentID string
	}{
		{"ses-child-1", "ses-needs-response"},
		{"ses-child-2", "ses-needs-response"},
		{"ses-child-3", "ses-has-errors"},
		{"ses-child-4", "ses-stale"},
	}

	for _, s := range childSessions {
		_, err := db.Exec(`
INSERT INTO session (id, project_id, parent_id, slug, directory, title, version,
                     time_created, time_updated)
VALUES (?, 'project-alpha', ?, ?, ?, ?, ?, ?, ?)
`, s.id, s.parentID, s.id, "/tmp/"+s.id, s.id, "v1", now.UnixMilli(), now.UnixMilli())
		if err != nil {
			return fmt.Errorf("failed to insert child session %s: %w", s.id, err)
		}
	}

	// Seed messages
	messageData := map[string]struct {
		role string
		time time.Time
	}{
		"ses-needs-response": {"assistant", now.Add(-1 * time.Hour)},
		"ses-active-now":     {"user", now.Add(-2 * time.Minute)},
		"ses-has-errors":     {"user", now.Add(-30 * time.Minute)},
		"ses-stale":          {"user", now.Add(-10 * 24 * time.Hour)},
		"ses-pending-todos":  {"user", now.Add(-24 * time.Hour)},
		"ses-no-signal":      {"user", now.Add(-5 * 24 * time.Hour)},
		"ses-global":         {"assistant", now.Add(-48 * time.Hour)},
	}

	for sessionID, msgInfo := range messageData {
		msgJSON := map[string]interface{}{
			"role":       msgInfo.role,
			"agent":      "Sisyphus",
			"modelID":    "claude-haiku-4-5",
			"providerID": "anthropic",
			"time": map[string]int64{
				"created": msgInfo.time.UnixMilli(),
			},
		}
		msgData, _ := json.Marshal(msgJSON)

		_, err := db.Exec(`
INSERT INTO message (id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?)
`, sessionID+"-msg-1", sessionID, msgInfo.time.UnixMilli(), msgInfo.time.UnixMilli(), string(msgData))
		if err != nil {
			return fmt.Errorf("failed to insert message for %s: %w", sessionID, err)
		}
	}

	// Seed additional messages for ses-needs-response with text parts
	sesNeedsResponseMessages := []struct {
		msgID string
		role  string
		time  time.Time
		text  string
	}{
		{"ses-needs-response-msg-2", "assistant", now.Add(-63 * time.Minute), "I'll analyze the auth module and provide recommendations for refactoring. This will involve reviewing the current implementation and identifying areas for improvement."},
		{"ses-needs-response-msg-3", "user", now.Add(-61 * time.Minute), "  Please also check the session handling.\n\nAnd fix the tests.  "},
		{"ses-needs-response-msg-4", "assistant", now.Add(-59 * time.Minute), "I've reviewed the session handling code. Here are my findings and recommendations for improving the implementation and test coverage."},
		{"ses-needs-response-msg-5", "assistant", now.Add(-57 * time.Minute), "Done. All changes committed."},
	}

	for _, msg := range sesNeedsResponseMessages {
		msgJSON := map[string]interface{}{
			"role":       msg.role,
			"agent":      "Sisyphus",
			"modelID":    "claude-haiku-4-5",
			"providerID": "anthropic",
			"time": map[string]int64{
				"created": msg.time.UnixMilli(),
			},
		}
		msgData, _ := json.Marshal(msgJSON)

		_, err := db.Exec(`
INSERT INTO message (id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?)
`, msg.msgID, "ses-needs-response", msg.time.UnixMilli(), msg.time.UnixMilli(), string(msgData))
		if err != nil {
			return fmt.Errorf("failed to insert message %s: %w", msg.msgID, err)
		}

		// Insert text part for each message
		partJSON := map[string]interface{}{
			"type": "text",
			"text": msg.text,
		}
		partData, _ := json.Marshal(partJSON)

		_, err = db.Exec(`
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?, ?)
`, msg.msgID+"-part-1", msg.msgID, "ses-needs-response", msg.time.UnixMilli(), msg.time.UnixMilli(), string(partData))
		if err != nil {
			return fmt.Errorf("failed to insert text part for %s: %w", msg.msgID, err)
		}
	}

	// Seed additional message for ses-active-now with text part
	sesActiveNowMsg := struct {
		msgID string
		role  string
		time  time.Time
		text  string
	}{
		"ses-active-now-msg-2",
		"assistant",
		now.Add(-3 * time.Minute),
		"Working on it now.",
	}

	msgJSON := map[string]interface{}{
		"role":       sesActiveNowMsg.role,
		"agent":      "Sisyphus",
		"modelID":    "claude-haiku-4-5",
		"providerID": "anthropic",
		"time": map[string]int64{
			"created": sesActiveNowMsg.time.UnixMilli(),
		},
	}
	msgData, _ := json.Marshal(msgJSON)

	_, err := db.Exec(`
INSERT INTO message (id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?)
`, sesActiveNowMsg.msgID, "ses-active-now", sesActiveNowMsg.time.UnixMilli(), sesActiveNowMsg.time.UnixMilli(), string(msgData))
	if err != nil {
		return fmt.Errorf("failed to insert message %s: %w", sesActiveNowMsg.msgID, err)
	}

	// Insert text part for ses-active-now-msg-2
	partJSON := map[string]interface{}{
		"type": "text",
		"text": sesActiveNowMsg.text,
	}
	partData, _ := json.Marshal(partJSON)

	_, err = db.Exec(`
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?, ?)
`, sesActiveNowMsg.msgID+"-part-1", sesActiveNowMsg.msgID, "ses-active-now", sesActiveNowMsg.time.UnixMilli(), sesActiveNowMsg.time.UnixMilli(), string(partData))
	if err != nil {
		return fmt.Errorf("failed to insert text part for %s: %w", sesActiveNowMsg.msgID, err)
	}

	// Seed additional message for ses-has-errors with tool-invocation part only (no text part)
	sesHasErrorsMsg := struct {
		msgID string
		role  string
		time  time.Time
	}{
		"ses-has-errors-msg-2",
		"assistant",
		now.Add(-31 * time.Minute),
	}

	msgJSON = map[string]interface{}{
		"role":       sesHasErrorsMsg.role,
		"agent":      "Sisyphus",
		"modelID":    "claude-haiku-4-5",
		"providerID": "anthropic",
		"time": map[string]int64{
			"created": sesHasErrorsMsg.time.UnixMilli(),
		},
	}
	msgData, _ = json.Marshal(msgJSON)

	_, err = db.Exec(`
INSERT INTO message (id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?)
`, sesHasErrorsMsg.msgID, "ses-has-errors", sesHasErrorsMsg.time.UnixMilli(), sesHasErrorsMsg.time.UnixMilli(), string(msgData))
	if err != nil {
		return fmt.Errorf("failed to insert message %s: %w", sesHasErrorsMsg.msgID, err)
	}

	// Insert tool-invocation part for ses-has-errors-msg-2 (no text part)
	partJSON = map[string]interface{}{
		"type": "tool-invocation",
		"state": map[string]string{
			"status": "error",
		},
	}
	partData, _ = json.Marshal(partJSON)

	_, err = db.Exec(`
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?, ?)
`, "ses-has-errors-msg-2-part-1", sesHasErrorsMsg.msgID, "ses-has-errors", sesHasErrorsMsg.time.UnixMilli(), sesHasErrorsMsg.time.UnixMilli(), string(partData))
	if err != nil {
		return fmt.Errorf("failed to insert tool-invocation part for %s: %w", sesHasErrorsMsg.msgID, err)
	}

	// Seed error parts for ses-has-errors
	for i := 1; i <= 3; i++ {
		partJSON := map[string]interface{}{
			"type": "tool-invocation",
			"state": map[string]string{
				"status": "error",
			},
		}
		partData, _ := json.Marshal(partJSON)

		_, err := db.Exec(`
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data)
VALUES (?, ?, ?, ?, ?, ?)
`, fmt.Sprintf("ses-has-errors-part-%d", i), "ses-has-errors-msg-1", "ses-has-errors",
			now.UnixMilli(), now.UnixMilli(), string(partData))
		if err != nil {
			return fmt.Errorf("failed to insert error part %d: %w", i, err)
		}
	}

	// Seed todos
	todoSessions := map[string][]struct {
		content  string
		status   string
		priority string
		position int
	}{
		"ses-stale": {
			{"Task 1", "pending", "high", 1},
			{"Task 2", "pending", "high", 2},
		},
		"ses-pending-todos": {
			{"Task 1", "pending", "high", 1},
			{"Task 2", "pending", "high", 2},
			{"Task 3", "completed", "high", 3},
		},
		"ses-no-signal": {
			{"Task 1", "completed", "high", 1},
			{"Task 2", "completed", "high", 2},
		},
	}

	for sessionID, todos := range todoSessions {
		for _, todo := range todos {
			_, err := db.Exec(`
INSERT INTO todo (session_id, content, status, priority, position, time_created, time_updated)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, sessionID, todo.content, todo.status, todo.priority, todo.position, now.UnixMilli(), now.UnixMilli())
			if err != nil {
				return fmt.Errorf("failed to insert todo for %s: %w", sessionID, err)
			}
		}
	}

	return nil
}

func ptrInt64(v int64) *int64 {
	return &v
}
