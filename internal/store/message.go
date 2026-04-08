package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

// compile-time interface check
var _ domain.MessageStore = (*MessageRepo)(nil)

const maxPreviewLen = 100

type MessageRepo struct {
	db *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) GetLastMessageMeta(ctx context.Context, sessionID string) (domain.MessageMeta, error) {
	const q = `
SELECT
    COALESCE(json_extract(data, '$.role'),       '') AS role,
    COALESCE(json_extract(data, '$.agent'),      '') AS agent,
    COALESCE(json_extract(data, '$.modelID'),    '') AS model_id,
    COALESCE(json_extract(data, '$.providerID'), '') AS provider_id,
    time_created
FROM message
WHERE session_id = ?
ORDER BY time_created DESC
LIMIT 1`

	var (
		role          string
		agent         string
		modelID       string
		providerID    string
		timeCreatedMs int64
	)

	err := r.db.QueryRowContext(ctx, q, sessionID).Scan(
		&role, &agent, &modelID, &providerID, &timeCreatedMs,
	)
	if err == sql.ErrNoRows {
		return domain.MessageMeta{}, nil
	}
	if err != nil {
		return domain.MessageMeta{}, err
	}

	return domain.MessageMeta{
		Role:        role,
		Agent:       agent,
		ModelID:     modelID,
		ProviderID:  providerID,
		TimeCreated: time.UnixMilli(timeCreatedMs).UTC(),
	}, nil
}

// GetMessageCount returns the total number of messages for a session.
func (r *MessageRepo) GetMessageCount(ctx context.Context, sessionID string) (int, error) {
	const q = `SELECT COUNT(*) FROM message WHERE session_id = ?`

	var count int
	if err := r.db.QueryRowContext(ctx, q, sessionID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetRecentMessages returns the N most recent messages for a session with content preview.
func (r *MessageRepo) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]domain.MessagePreview, error) {
	const q = `
SELECT
    COALESCE(json_extract(m.data, '$.role'),  '') AS role,
    COALESCE(json_extract(m.data, '$.agent'), '') AS agent,
    COALESCE(
        (SELECT json_extract(p.data, '$.text')
         FROM part p
         WHERE p.message_id = m.id
           AND json_extract(p.data, '$.type') = 'text'
         LIMIT 1),
        ''
    ) AS content,
    m.time_created
FROM message m
WHERE m.session_id = ?
ORDER BY m.time_created DESC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, q, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var previews []domain.MessagePreview
	for rows.Next() {
		var (
			role          string
			agent         string
			content       string
			timeCreatedMs int64
		)

		if err := rows.Scan(&role, &agent, &content, &timeCreatedMs); err != nil {
			return nil, err
		}

		previews = append(previews, domain.MessagePreview{
			Role:        role,
			Agent:       agent,
			Content:     prepareContent(content),
			TimeCreated: time.UnixMilli(timeCreatedMs).UTC(),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil
	if previews == nil {
		previews = []domain.MessagePreview{}
	}

	return previews, nil
}

// prepareContent processes raw message content for preview display.
func prepareContent(raw string) string {
	if raw == "" {
		return "[tool use]"
	}
	// Replace all newline variants with single space
	s := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(raw)
	// Trim whitespace
	s = strings.TrimSpace(s)
	if s == "" {
		return "[tool use]"
	}
	// Truncate by rune count
	runes := []rune(s)
	if len(runes) > maxPreviewLen {
		return string(runes[:maxPreviewLen]) + "..."
	}
	return s
}
