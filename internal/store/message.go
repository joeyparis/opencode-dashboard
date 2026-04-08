package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

// compile-time interface check
var _ domain.MessageStore = (*MessageRepo)(nil)

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
