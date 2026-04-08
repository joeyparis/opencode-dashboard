package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/app"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Stores ---

type mockProjectStore struct {
	projects []domain.Project
}

func (m *mockProjectStore) ListProjects(_ context.Context) ([]domain.Project, error) {
	return m.projects, nil
}

type mockSessionStore struct {
	sessions    []domain.Session
	childCounts map[string]int
}

func (m *mockSessionStore) ListRootSessions(_ context.Context) ([]domain.Session, error) {
	return m.sessions, nil
}

func (m *mockSessionStore) GetSessionByID(_ context.Context, id string) (domain.Session, error) {
	for _, s := range m.sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return domain.Session{}, nil
}

func (m *mockSessionStore) GetChildCount(_ context.Context, parentID string) (int, error) {
	if m.childCounts == nil {
		return 0, nil
	}
	return m.childCounts[parentID], nil
}

type mockMessageStore struct {
	lastMessages  map[string]domain.MessageMeta
	messageCounts map[string]int
}

func (m *mockMessageStore) GetLastMessageMeta(_ context.Context, sessionID string) (domain.MessageMeta, error) {
	if m.lastMessages == nil {
		return domain.MessageMeta{}, nil
	}
	return m.lastMessages[sessionID], nil
}

func (m *mockMessageStore) GetMessageCount(_ context.Context, sessionID string) (int, error) {
	if m.messageCounts == nil {
		return 0, nil
	}
	return m.messageCounts[sessionID], nil
}

type mockTodoStore struct {
	todos map[string][]domain.Todo
}

func (m *mockTodoStore) GetTodosBySession(_ context.Context, sessionID string) ([]domain.Todo, error) {
	if m.todos == nil {
		return nil, nil
	}
	return m.todos[sessionID], nil
}

func (m *mockTodoStore) GetPendingTodoCount(_ context.Context, sessionID string) (int, error) {
	count := 0
	for _, t := range m.todos[sessionID] {
		if t.Status != "completed" {
			count++
		}
	}
	return count, nil
}

type mockErrorStore struct {
	errorCounts   map[string]int
	refreshCalled bool
	refreshSince  time.Time
}

func (m *mockErrorStore) GetErrorCount(_ context.Context, sessionID string) (int, error) {
	if m.errorCounts == nil {
		return 0, nil
	}
	return m.errorCounts[sessionID], nil
}

func (m *mockErrorStore) RefreshErrorCache(_ context.Context, since time.Time) error {
	m.refreshCalled = true
	m.refreshSince = since
	return nil
}

func (m *mockErrorStore) RefreshAll(_ context.Context) error {
	return nil
}

// --- Test helper ---

func newTestAggregator(
	projects []domain.Project,
	sessions []domain.Session,
	msgs *mockMessageStore,
	todos *mockTodoStore,
	errors *mockErrorStore,
	childCounts map[string]int,
) *app.Aggregator {
	return app.NewAggregator(
		&mockProjectStore{projects: projects},
		&mockSessionStore{sessions: sessions, childCounts: childCounts},
		msgs,
		todos,
		errors,
	)
}

// --- Tests ---

func TestLoadAll_GroupsByProject(t *testing.T) {
	projects := []domain.Project{
		{ID: "proj1", Worktree: "/work/alpha"},
		{ID: "proj2", Worktree: "/work/beta"},
	}
	now := time.Now()
	sessions := []domain.Session{
		{ID: "s1", ProjectID: "proj1", TimeUpdated: now.Add(-2 * time.Hour)},
		{ID: "s2", ProjectID: "proj1", TimeUpdated: now.Add(-3 * time.Hour)},
		{ID: "s3", ProjectID: "proj2", TimeUpdated: now.Add(-1 * time.Hour)},
	}

	agg := newTestAggregator(projects, sessions, &mockMessageStore{}, &mockTodoStore{}, &mockErrorStore{}, nil)
	groups, err := agg.LoadAll(context.Background())

	require.NoError(t, err)
	require.Len(t, groups, 2)

	var alphaSessions, betaSessions int
	for _, g := range groups {
		switch g.Project.ID {
		case "proj1":
			alphaSessions = len(g.Sessions)
		case "proj2":
			betaSessions = len(g.Sessions)
		}
	}
	assert.Equal(t, 2, alphaSessions, "proj1 should have 2 sessions")
	assert.Equal(t, 1, betaSessions, "proj2 should have 1 session")
}

func TestLoadAll_GlobalLast(t *testing.T) {
	projects := []domain.Project{
		{ID: "proj1", Worktree: "/work/alpha"},
		{ID: "proj2", Worktree: "/work/beta"},
		{ID: "global", Worktree: "/"},
	}

	agg := newTestAggregator(projects, nil, &mockMessageStore{}, &mockTodoStore{}, &mockErrorStore{}, nil)
	groups, err := agg.LoadAll(context.Background())

	require.NoError(t, err)
	require.Len(t, groups, 3)
	assert.Equal(t, "global", groups[len(groups)-1].Project.ID, "Global project must be last")
}

func TestLoadAll_AlphabeticalSort(t *testing.T) {
	projects := []domain.Project{
		{ID: "p3", Worktree: "/work/zeta"},
		{ID: "p1", Worktree: "/work/alpha"},
		{ID: "p2", Worktree: "/work/beta"},
	}

	agg := newTestAggregator(projects, nil, &mockMessageStore{}, &mockTodoStore{}, &mockErrorStore{}, nil)
	groups, err := agg.LoadAll(context.Background())

	require.NoError(t, err)
	require.Len(t, groups, 3)
	assert.Equal(t, "alpha", groups[0].Project.DisplayName())
	assert.Equal(t, "beta", groups[1].Project.DisplayName())
	assert.Equal(t, "zeta", groups[2].Project.DisplayName())
}

func TestLoadAll_GlobalLastAmongAlpha(t *testing.T) {
	// Global should be last even when alphabetically it would sort before others
	projects := []domain.Project{
		{ID: "proj1", Worktree: "/work/zeta"},
		{ID: "global", Worktree: "/"}, // "Global" alphabetically before "zeta" but must be last
		{ID: "proj2", Worktree: "/work/alpha"},
	}

	agg := newTestAggregator(projects, nil, &mockMessageStore{}, &mockTodoStore{}, &mockErrorStore{}, nil)
	groups, err := agg.LoadAll(context.Background())

	require.NoError(t, err)
	require.Len(t, groups, 3)
	assert.Equal(t, "global", groups[2].Project.ID, "Global must be last regardless of alphabetical order")
	assert.Equal(t, "alpha", groups[0].Project.DisplayName())
	assert.Equal(t, "zeta", groups[1].Project.DisplayName())
}

func TestLoadAll_AttentionClassified(t *testing.T) {
	proj := domain.Project{ID: "proj1", Worktree: "/work/proj"}
	now := time.Now()

	// Session updated < 5 minutes ago -> ActiveNow
	sessions := []domain.Session{
		{ID: "s1", ProjectID: "proj1", TimeUpdated: now.Add(-1 * time.Minute)},
	}

	agg := newTestAggregator(
		[]domain.Project{proj},
		sessions,
		&mockMessageStore{},
		&mockTodoStore{},
		&mockErrorStore{},
		nil,
	)

	groups, err := agg.LoadAll(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Sessions, 1)

	assert.Equal(t, domain.ActiveNow, groups[0].Sessions[0].AttentionSignal)
}

func TestLoadAll_SessionsSortedByAttention(t *testing.T) {
	proj := domain.Project{ID: "proj1", Worktree: "/work/proj"}
	now := time.Now()

	sessions := []domain.Session{
		// s1: NeedsResponse (3) - last msg was assistant within 24h
		{ID: "s1", ProjectID: "proj1", TimeUpdated: now.Add(-12 * time.Hour)},
		// s2: ActiveNow (4) - updated < 5 min ago
		{ID: "s2", ProjectID: "proj1", TimeUpdated: now.Add(-1 * time.Minute)},
	}

	agg := newTestAggregator(
		[]domain.Project{proj},
		sessions,
		&mockMessageStore{
			lastMessages: map[string]domain.MessageMeta{
				"s1": {Role: "assistant", TimeCreated: now.Add(-12 * time.Hour)},
			},
		},
		&mockTodoStore{},
		&mockErrorStore{},
		nil,
	)

	groups, err := agg.LoadAll(context.Background())
	require.NoError(t, err)
	require.Len(t, groups[0].Sessions, 2)

	// s2 (ActiveNow=4) must come before s1 (NeedsResponse=3)
	assert.Equal(t, "s2", groups[0].Sessions[0].ID, "Higher attention signal must come first")
	assert.Equal(t, "s1", groups[0].Sessions[1].ID)
}

func TestLoadAll_SessionsSortedByTimeThenAttention(t *testing.T) {
	// When attention signals are equal, more recently updated sessions come first
	proj := domain.Project{ID: "proj1", Worktree: "/work/proj"}
	now := time.Now()

	sessions := []domain.Session{
		// Both have pending todos (PendingTodos=1), sort by TimeUpdated
		{ID: "older", ProjectID: "proj1", TimeUpdated: now.Add(-5 * time.Hour)},
		{ID: "newer", ProjectID: "proj1", TimeUpdated: now.Add(-2 * time.Hour)},
	}

	agg := newTestAggregator(
		[]domain.Project{proj},
		sessions,
		&mockMessageStore{},
		&mockTodoStore{
			todos: map[string][]domain.Todo{
				"older": {{SessionID: "older", Content: "task", Status: "pending"}},
				"newer": {{SessionID: "newer", Content: "task", Status: "pending"}},
			},
		},
		&mockErrorStore{},
		nil,
	)

	groups, err := agg.LoadAll(context.Background())
	require.NoError(t, err)
	require.Len(t, groups[0].Sessions, 2)

	assert.Equal(t, "newer", groups[0].Sessions[0].ID, "More recently updated must come first when signals are equal")
	assert.Equal(t, "older", groups[0].Sessions[1].ID)
}

func TestLoadAll_FullSessionView(t *testing.T) {
	proj := domain.Project{ID: "proj1", Worktree: "/work/myapp"}
	now := time.Now()
	sess := domain.Session{
		ID:          "s1",
		ProjectID:   "proj1",
		TimeUpdated: now.Add(-2 * time.Hour),
	}

	todos := []domain.Todo{
		{SessionID: "s1", Content: "do thing", Status: "pending"},
		{SessionID: "s1", Content: "done thing", Status: "completed"},
	}
	lastMsg := domain.MessageMeta{Role: "user", TimeCreated: now.Add(-2 * time.Hour)}

	agg := newTestAggregator(
		[]domain.Project{proj},
		[]domain.Session{sess},
		&mockMessageStore{
			lastMessages:  map[string]domain.MessageMeta{"s1": lastMsg},
			messageCounts: map[string]int{"s1": 10},
		},
		&mockTodoStore{
			todos: map[string][]domain.Todo{"s1": todos},
		},
		&mockErrorStore{
			errorCounts: map[string]int{"s1": 3},
		},
		map[string]int{"s1": 2},
	)

	groups, err := agg.LoadAll(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Sessions, 1)

	sv := groups[0].Sessions[0]
	assert.Equal(t, "myapp", sv.ProjectName)
	assert.Equal(t, "/work/myapp", sv.ProjectWorktree)
	assert.Equal(t, 2, len(sv.Todos), "Todos list must be populated")
	assert.Equal(t, 1, sv.PendingTodoCount, "One pending todo")
	assert.Equal(t, 2, sv.TotalTodoCount, "Two total todos")
	assert.Equal(t, 10, sv.MessageCount, "MessageCount must be populated")
	assert.Equal(t, 3, sv.ErrorCount, "ErrorCount must be populated")
	assert.Equal(t, 2, sv.ChildCount, "ChildCount must be populated")
	assert.Equal(t, lastMsg, sv.LastMessage, "LastMessage must be populated")
	// ErrorCount > 0 triggers HasErrors
	assert.Equal(t, domain.HasErrors, sv.AttentionSignal)
}

func TestLoadAll_AttentionCount(t *testing.T) {
	proj := domain.Project{ID: "proj1", Worktree: "/work/proj"}
	now := time.Now()

	sessions := []domain.Session{
		// s1: ActiveNow - attention != None
		{ID: "s1", ProjectID: "proj1", TimeUpdated: now.Add(-1 * time.Minute)},
		// s2: None - old, no todos, no errors, last msg from user
		{ID: "s2", ProjectID: "proj1", TimeUpdated: now.Add(-30 * 24 * time.Hour)},
		// s3: HasErrors - attention != None
		{ID: "s3", ProjectID: "proj1", TimeUpdated: now.Add(-48 * time.Hour)},
	}

	agg := newTestAggregator(
		[]domain.Project{proj},
		sessions,
		&mockMessageStore{},
		&mockTodoStore{},
		&mockErrorStore{
			errorCounts: map[string]int{"s3": 2},
		},
		nil,
	)

	groups, err := agg.LoadAll(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 2, groups[0].AttentionCount, "AttentionCount should be 2 (s1 + s3)")
}

func TestRefresh_CallsErrorRefresh(t *testing.T) {
	errStore := &mockErrorStore{}

	agg := newTestAggregator(
		[]domain.Project{{ID: "proj1", Worktree: "/work/proj"}},
		nil,
		&mockMessageStore{},
		&mockTodoStore{},
		errStore,
		nil,
	)

	_, err := agg.Refresh(context.Background())
	require.NoError(t, err)
	assert.True(t, errStore.refreshCalled, "RefreshErrorCache must be called before loading")
}

func TestRefresh_ReturnsGroups(t *testing.T) {
	projects := []domain.Project{
		{ID: "proj1", Worktree: "/work/alpha"},
		{ID: "proj2", Worktree: "/work/beta"},
	}
	now := time.Now()
	sessions := []domain.Session{
		{ID: "s1", ProjectID: "proj1", TimeUpdated: now.Add(-1 * time.Hour)},
	}

	agg := newTestAggregator(projects, sessions, &mockMessageStore{}, &mockTodoStore{}, &mockErrorStore{}, nil)
	groups, err := agg.Refresh(context.Background())

	require.NoError(t, err)
	assert.Len(t, groups, 2)
}
