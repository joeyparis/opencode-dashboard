package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

var _ domain.SessionStore = (*SessionRepo)(nil)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) ListRootSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, COALESCE(parent_id, ''), COALESCE(slug, ''),
		       COALESCE(directory, ''), COALESCE(title, ''), COALESCE(version, ''),
		       COALESCE(summary_additions, 0), COALESCE(summary_deletions, 0), COALESCE(summary_files, 0),
		       time_created, time_updated, COALESCE(time_archived, 0)
		FROM session
		WHERE parent_id IS NULL
		ORDER BY time_updated DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListRootSessions query: %w", err)
	}
	defer rows.Close()

	return scanSessions(rows)
}

func (r *SessionRepo) GetSessionByID(ctx context.Context, id string) (domain.Session, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, COALESCE(parent_id, ''), COALESCE(slug, ''),
		       COALESCE(directory, ''), COALESCE(title, ''), COALESCE(version, ''),
		       COALESCE(summary_additions, 0), COALESCE(summary_deletions, 0), COALESCE(summary_files, 0),
		       time_created, time_updated, COALESCE(time_archived, 0)
		FROM session
		WHERE id = ?
	`, id)

	var s domain.Session
	var timeCreatedMs, timeUpdatedMs, timeArchivedMs int64

	err := row.Scan(
		&s.ID, &s.ProjectID, &s.ParentID, &s.Slug,
		&s.Directory, &s.Title, &s.Version,
		&s.SummaryAdditions, &s.SummaryDeletions, &s.SummaryFiles,
		&timeCreatedMs, &timeUpdatedMs, &timeArchivedMs,
	)
	if err == sql.ErrNoRows {
		return domain.Session{}, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("GetSessionByID scan: %w", err)
	}

	s.TimeCreated = time.UnixMilli(timeCreatedMs).UTC()
	s.TimeUpdated = time.UnixMilli(timeUpdatedMs).UTC()
	if timeArchivedMs > 0 {
		s.TimeArchived = time.UnixMilli(timeArchivedMs).UTC()
	}

	return s, nil
}

func (r *SessionRepo) GetChildCount(ctx context.Context, parentID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM session WHERE parent_id = ?
	`, parentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GetChildCount query: %w", err)
	}
	return count, nil
}

func scanSessions(rows *sql.Rows) ([]domain.Session, error) {
	var sessions []domain.Session
	for rows.Next() {
		var s domain.Session
		var timeCreatedMs, timeUpdatedMs, timeArchivedMs int64

		if err := rows.Scan(
			&s.ID, &s.ProjectID, &s.ParentID, &s.Slug,
			&s.Directory, &s.Title, &s.Version,
			&s.SummaryAdditions, &s.SummaryDeletions, &s.SummaryFiles,
			&timeCreatedMs, &timeUpdatedMs, &timeArchivedMs,
		); err != nil {
			return nil, fmt.Errorf("scanSessions: %w", err)
		}

		s.TimeCreated = time.UnixMilli(timeCreatedMs).UTC()
		s.TimeUpdated = time.UnixMilli(timeUpdatedMs).UTC()
		if timeArchivedMs > 0 {
			s.TimeArchived = time.UnixMilli(timeArchivedMs).UTC()
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanSessions rows: %w", err)
	}
	return sessions, nil
}
