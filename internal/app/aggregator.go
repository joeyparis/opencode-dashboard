package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/attention"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

// Aggregator composes all store interfaces into []ProjectGroup view models.
type Aggregator struct {
	projects domain.ProjectStore
	sessions domain.SessionStore
	messages domain.MessageStore
	todos    domain.TodoStore
	errors   domain.ErrorStore
	waiting  domain.WaitingStore
	classify func(domain.SessionView, time.Time) domain.AttentionSignal
}

// NewAggregator constructs an Aggregator wired to the real attention classifier.
func NewAggregator(
	projects domain.ProjectStore,
	sessions domain.SessionStore,
	messages domain.MessageStore,
	todos domain.TodoStore,
	errors domain.ErrorStore,
	waiting domain.WaitingStore,
) *Aggregator {
	return NewAggregatorWithClassifier(attention.Classify, projects, sessions, messages, todos, errors, waiting)
}

// NewAggregatorWithClassifier creates an Aggregator with a custom attention classifier.
func NewAggregatorWithClassifier(
	classify func(domain.SessionView, time.Time) domain.AttentionSignal,
	projects domain.ProjectStore,
	sessions domain.SessionStore,
	messages domain.MessageStore,
	todos domain.TodoStore,
	errors domain.ErrorStore,
	waiting domain.WaitingStore,
) *Aggregator {
	return &Aggregator{
		projects: projects,
		sessions: sessions,
		messages: messages,
		todos:    todos,
		errors:   errors,
		waiting:  waiting,
		classify: classify,
	}
}

// LoadAll fetches all projects and sessions and returns an ordered []ProjectGroup.
// Groups are sorted alphabetically by DisplayName; Global is always last.
// Within each group sessions are sorted by AttentionSignal DESC, then TimeUpdated DESC.
func (a *Aggregator) LoadAll(ctx context.Context) ([]domain.ProjectGroup, error) {
	if err := a.waiting.RefreshAll(ctx); err != nil {
		return nil, fmt.Errorf("waiting refresh: %w", err)
	}

	projects, err := a.projects.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	sessions, err := a.sessions.ListRootSessions(ctx)
	if err != nil {
		return nil, err
	}

	// Build project lookup map
	projectMap := make(map[string]domain.Project, len(projects))
	for _, p := range projects {
		projectMap[p.ID] = p
	}

	// Build SessionViews grouped by project
	now := time.Now()
	viewsByProject := make(map[string][]domain.SessionView)

	for _, s := range sessions {
		// Best-effort enrichment: data fetch failures degrade gracefully to zero values
		// rather than aborting the session load entirely.
		todos, _ := a.todos.GetTodosBySession(ctx, s.ID)
		lastMsg, _ := a.messages.GetLastMessageMeta(ctx, s.ID)
		msgCount, _ := a.messages.GetMessageCount(ctx, s.ID)
		childCount, _ := a.sessions.GetChildCount(ctx, s.ID)
		errCount, _ := a.errors.GetErrorCount(ctx, s.ID)
		hasPendingQuestion, _ := a.waiting.GetWaiting(ctx, s.ID)

		pendingCount := 0
		completedCount := 0
		for _, t := range todos {
			if t.Status != "completed" {
				pendingCount++
			} else {
				completedCount++
			}
		}

		proj := projectMap[s.ProjectID]
		view := domain.SessionView{
			Session:            s,
			ProjectName:        proj.DisplayName(),
			ProjectWorktree:    proj.Worktree,
			Todos:              todos,
			PendingTodoCount:   pendingCount,
			TotalTodoCount:     len(todos),
			CompletedTodoCount: completedCount,
			LastMessage:        lastMsg,
			MessageCount:       msgCount,
			ErrorCount:         errCount,
			HasPendingQuestion: hasPendingQuestion,
			ChildCount:         childCount,
		}
		view.AttentionSignal = a.classify(view, now)

		viewsByProject[s.ProjectID] = append(viewsByProject[s.ProjectID], view)
	}

	// Build ProjectGroups
	grouped := make([]domain.ProjectGroup, 0, len(projects))
	for _, p := range projects {
		views := viewsByProject[p.ID]

		sort.SliceStable(views, func(i, j int) bool {
			return views[i].TimeUpdated.After(views[j].TimeUpdated)
		})

		attentionCount := 0
		for _, v := range views {
			if v.AttentionSignal != domain.None {
				attentionCount++
			}
		}

		grouped = append(grouped, domain.ProjectGroup{
			Project:        p,
			Sessions:       views,
			AttentionCount: attentionCount,
		})
	}

	// Sort groups: alphabetical by DisplayName, Global always last
	sort.SliceStable(grouped, func(i, j int) bool {
		iGlobal := grouped[i].Project.ID == "global" || grouped[i].Project.Worktree == "/"
		jGlobal := grouped[j].Project.ID == "global" || grouped[j].Project.Worktree == "/"
		if iGlobal != jGlobal {
			return !iGlobal // non-global comes before global
		}
		return strings.ToLower(grouped[i].Project.DisplayName()) < strings.ToLower(grouped[j].Project.DisplayName())
	})

	return grouped, nil
}

// Refresh refreshes the error and waiting caches then returns the same result as LoadAll.
func (a *Aggregator) Refresh(ctx context.Context) ([]domain.ProjectGroup, error) {
	if err := a.errors.RefreshErrorCache(ctx, time.Time{}); err != nil {
		return nil, err
	}
	if err := a.waiting.RefreshWaitingCache(ctx, time.Time{}); err != nil {
		return nil, fmt.Errorf("waiting cache refresh: %w", err)
	}
	return a.LoadAll(ctx)
}
