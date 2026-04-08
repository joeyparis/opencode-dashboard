package store

import (
	"context"
	"database/sql"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

type TodoRepo struct {
	db *sql.DB
}

func NewTodoRepo(db *sql.DB) *TodoRepo {
	return &TodoRepo{db: db}
}

func (r *TodoRepo) GetTodosBySession(ctx context.Context, sessionID string) ([]domain.Todo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT session_id, content, status, priority, position
		FROM todo
		WHERE session_id = ?
		ORDER BY position ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := []domain.Todo{}
	for rows.Next() {
		var t domain.Todo
		if err := rows.Scan(&t.SessionID, &t.Content, &t.Status, &t.Priority, &t.Position); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return todos, nil
}

func (r *TodoRepo) GetPendingTodoCount(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM todo
		WHERE session_id = ? AND status != 'completed'
	`, sessionID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
