package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type ErrorRepo struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[string]int
}

func NewErrorRepo(db *sql.DB) *ErrorRepo {
	return &ErrorRepo{
		db:    db,
		cache: make(map[string]int),
	}
}

func (r *ErrorRepo) GetErrorCount(ctx context.Context, sessionID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cache[sessionID], nil
}

func (r *ErrorRepo) RefreshAll(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT session_id, COUNT(*) as error_count
		FROM part
		WHERE json_extract(data, '$.state.status') = 'error'
		GROUP BY session_id
	`)
	if err != nil {
		return fmt.Errorf("RefreshAll query: %w", err)
	}
	defer rows.Close()

	newCache := make(map[string]int)
	for rows.Next() {
		var sessionID string
		var count int
		if err := rows.Scan(&sessionID, &count); err != nil {
			return fmt.Errorf("RefreshAll scan: %w", err)
		}
		newCache[sessionID] = count
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("RefreshAll rows: %w", err)
	}

	r.mu.Lock()
	r.cache = newCache
	r.mu.Unlock()

	return nil
}

func (r *ErrorRepo) RefreshErrorCache(ctx context.Context, since time.Time) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.session_id, COUNT(*) as error_count
		FROM part p
		WHERE json_extract(p.data, '$.state.status') = 'error'
		  AND p.session_id IN (
		      SELECT id FROM session WHERE time_updated > ?
		  )
		GROUP BY p.session_id
	`, since.UnixMilli())
	if err != nil {
		return fmt.Errorf("RefreshErrorCache query: %w", err)
	}
	defer rows.Close()

	updates := make(map[string]int)
	for rows.Next() {
		var sessionID string
		var count int
		if err := rows.Scan(&sessionID, &count); err != nil {
			return fmt.Errorf("RefreshErrorCache scan: %w", err)
		}
		updates[sessionID] = count
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("RefreshErrorCache rows: %w", err)
	}

	if len(updates) > 0 {
		r.mu.Lock()
		for sessionID, count := range updates {
			r.cache[sessionID] = count
		}
		r.mu.Unlock()
	}

	return nil
}
