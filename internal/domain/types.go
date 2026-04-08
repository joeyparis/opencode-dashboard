package domain

import (
	"path/filepath"
	"time"
)

// Project represents an OpenCode project directory
type Project struct {
	ID          string
	Name        string // NULL in real DB - use DisplayName() always
	Worktree    string // directory path
	VCS         string
	TimeCreated time.Time
	TimeUpdated time.Time
}

// DisplayName returns the human-readable project name.
// In the real OpenCode DB, project.name is NULL for ALL projects.
// Falls back to worktree basename, with special case for "global" project.
func (p Project) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	if p.Worktree == "/" || p.ID == "global" {
		return "Global"
	}
	base := filepath.Base(p.Worktree)
	if base == "." || base == "/" || base == "" {
		return "Global"
	}
	return base
}

// Session is raw session data from DB (no project display fields here)
type Session struct {
	ID               string
	ProjectID        string // FK only - project display comes from ProjectStore
	ParentID         string // empty string if root session
	Slug             string
	Directory        string
	Title            string
	Version          string
	SummaryAdditions int
	SummaryDeletions int
	SummaryFiles     int
	TimeCreated      time.Time
	TimeUpdated      time.Time
	TimeArchived     time.Time // zero value if not archived
}

// IsArchived returns true if the session has been archived
func (s Session) IsArchived() bool {
	return !s.TimeArchived.IsZero()
}

// Todo represents a task item in a session
type Todo struct {
	SessionID string
	Content   string
	Status    string // "completed", "pending", "in_progress", "cancelled"
	Priority  string // "high", "medium", "low"
	Position  int
}

// MessageMeta is metadata about a message (no content - just role/agent/model info)
type MessageMeta struct {
	Role        string // "user" or "assistant"
	Agent       string
	ModelID     string
	ProviderID  string
	TimeCreated time.Time
}

// AttentionSignal represents why a session needs user attention.
// Higher numeric value = higher priority.
type AttentionSignal int

const (
	None          AttentionSignal = 0
	PendingTodos  AttentionSignal = 1
	StaleWork     AttentionSignal = 2
	NeedsResponse AttentionSignal = 3
	ActiveNow     AttentionSignal = 4
	HasErrors     AttentionSignal = 5
)

// String returns a human-readable name for the attention signal
func (a AttentionSignal) String() string {
	switch a {
	case HasErrors:
		return "Has Errors"
	case ActiveNow:
		return "Active Now"
	case NeedsResponse:
		return "Needs Response"
	case StaleWork:
		return "Stale Work"
	case PendingTodos:
		return "Pending Todos"
	default:
		return ""
	}
}

// FilterPreset represents a named filter mode
type FilterPreset int

const (
	FilterNeedsAttention FilterPreset = iota
	FilterAllActive
	FilterArchived
)

// String returns the display label for a filter preset
func (f FilterPreset) String() string {
	switch f {
	case FilterAllActive:
		return "All Active"
	case FilterArchived:
		return "Archived"
	default:
		return "Needs Attention"
	}
}

// SessionView is the fully-populated view model for a session (used by UI)
type SessionView struct {
	Session          // embedded - all Session fields available directly
	ProjectName      string
	ProjectWorktree  string
	AttentionSignal  AttentionSignal
	Todos            []Todo // full todo list for detail pane
	PendingTodoCount int
	TotalTodoCount   int
	LastMessage      MessageMeta
	MessageCount     int
	ErrorCount       int
	ChildCount       int
}

// ProjectGroup is an ordered container of sessions grouped by project
type ProjectGroup struct {
	Project        Project
	Sessions       []SessionView // sorted: attention priority DESC, then time_updated DESC
	AttentionCount int           // sessions with AttentionSignal != None
}
