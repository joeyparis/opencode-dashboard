package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type WaitingRepo struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[string]bool
}

func NewWaitingRepo(db *sql.DB) *WaitingRepo {
	return &WaitingRepo{
		db:    db,
		cache: make(map[string]bool),
	}
}

func (r *WaitingRepo) GetWaiting(ctx context.Context, sessionID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cache[sessionID], nil
}

func (r *WaitingRepo) RefreshAll(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT session_id
		FROM part
		WHERE json_extract(data, '$.type') = 'tool'
		  AND json_extract(data, '$.tool') = 'question'
		  AND json_extract(data, '$.state.status') = 'running'
	`)
	if err != nil {
		return fmt.Errorf("WaitingRepo.RefreshAll: %w", err)
	}
	defer rows.Close()

	newCache := make(map[string]bool)
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return fmt.Errorf("WaitingRepo.RefreshAll scan: %w", err)
		}
		newCache[sessionID] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("WaitingRepo.RefreshAll rows: %w", err)
	}

	r.mu.Lock()
	r.cache = newCache
	r.mu.Unlock()
	return nil
}

func (r *WaitingRepo) RefreshWaitingCache(ctx context.Context, since time.Time) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT p.session_id
		FROM part p
		WHERE json_extract(p.data, '$.type') = 'tool'
		  AND json_extract(p.data, '$.tool') = 'question'
		  AND json_extract(p.data, '$.state.status') = 'running'
		  AND p.session_id IN (
		      SELECT id FROM session WHERE time_updated > ?
		  )
	`, since.UnixMilli())
	if err != nil {
		return fmt.Errorf("WaitingRepo.RefreshWaitingCache: %w", err)
	}
	defer rows.Close()

	updates := make(map[string]bool)
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return fmt.Errorf("WaitingRepo.RefreshWaitingCache scan: %w", err)
		}
		updates[sessionID] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("WaitingRepo.RefreshWaitingCache rows: %w", err)
	}

	if len(updates) > 0 {
		r.mu.Lock()
		for sessionID := range updates {
			r.cache[sessionID] = true
		}
		r.mu.Unlock()
	}
	return nil
}
