package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

var _ domain.ProjectStore = (*ProjectRepo)(nil)

type ProjectRepo struct {
	db *sql.DB
}

func NewProjectRepo(db *sql.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(name, ''), COALESCE(worktree, ''), COALESCE(vcs, ''),
		       time_created, time_updated
		FROM project
		ORDER BY worktree ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListProjects query: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		var timeCreatedMs, timeUpdatedMs int64

		if err := rows.Scan(&p.ID, &p.Name, &p.Worktree, &p.VCS, &timeCreatedMs, &timeUpdatedMs); err != nil {
			return nil, fmt.Errorf("ListProjects scan: %w", err)
		}

		p.TimeCreated = time.UnixMilli(timeCreatedMs).UTC()
		p.TimeUpdated = time.UnixMilli(timeUpdatedMs).UTC()
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListProjects rows: %w", err)
	}

	return projects, nil
}
