package domain

import (
	"context"
	"time"
)

// SessionStore provides read-only access to session data
type SessionStore interface {
	ListRootSessions(ctx context.Context) ([]Session, error)
	GetSessionByID(ctx context.Context, id string) (Session, error)
	GetChildCount(ctx context.Context, parentID string) (int, error)
}

// MessageStore provides read-only access to message metadata
type MessageStore interface {
	GetLastMessageMeta(ctx context.Context, sessionID string) (MessageMeta, error)
	GetMessageCount(ctx context.Context, sessionID string) (int, error)
}

// TodoStore provides read-only access to todo data
type TodoStore interface {
	GetTodosBySession(ctx context.Context, sessionID string) ([]Todo, error)
	GetPendingTodoCount(ctx context.Context, sessionID string) (int, error)
}

// ErrorStore provides access to error count cache
type ErrorStore interface {
	GetErrorCount(ctx context.Context, sessionID string) (int, error)
	RefreshErrorCache(ctx context.Context, since time.Time) error
	RefreshAll(ctx context.Context) error
}

// WaitingStore detects sessions with unanswered question tool prompts
type WaitingStore interface {
	GetWaiting(ctx context.Context, sessionID string) (bool, error)
	RefreshAll(ctx context.Context) error
	RefreshWaitingCache(ctx context.Context, since time.Time) error
}

// ProjectStore provides read-only access to project data
type ProjectStore interface {
	ListProjects(ctx context.Context) ([]Project, error)
}
