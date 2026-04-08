package app

import (
	"context"
	"sort"
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
	classify func(domain.SessionView, time.Time) domain.AttentionSignal
}

// NewAggregator constructs an Aggregator wired to the real attention classifier.
func NewAggregator(
	projects domain.ProjectStore,
	sessions domain.SessionStore,
	messages domain.MessageStore,
	todos domain.TodoStore,
	errors domain.ErrorStore,
) *Aggregator {
	return &Aggregator{
		projects: projects,
		sessions: sessions,
		messages: messages,
		todos:    todos,
		errors:   errors,
		classify: attention.Classify,
	}
}

// LoadAll fetches all projects and sessions and returns an ordered []ProjectGroup.
// Groups are sorted alphabetically by DisplayName; Global is always last.
// Within each group sessions are sorted by AttentionSignal DESC, then TimeUpdated DESC.
func (a *Aggregator) LoadAll(ctx context.Context) ([]domain.ProjectGroup, error) {
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
		todos, _ := a.todos.GetTodosBySession(ctx, s.ID)
		lastMsg, _ := a.messages.GetLastMessageMeta(ctx, s.ID)
		msgCount, _ := a.messages.GetMessageCount(ctx, s.ID)
		childCount, _ := a.sessions.GetChildCount(ctx, s.ID)
		errCount, _ := a.errors.GetErrorCount(ctx, s.ID)

		pendingCount := 0
		for _, t := range todos {
			if t.Status != "completed" {
				pendingCount++
			}
		}

		proj := projectMap[s.ProjectID]
		view := domain.SessionView{
			Session:          s,
			ProjectName:      proj.DisplayName(),
			ProjectWorktree:  proj.Worktree,
			Todos:            todos,
			PendingTodoCount: pendingCount,
			TotalTodoCount:   len(todos),
			LastMessage:      lastMsg,
			MessageCount:     msgCount,
			ErrorCount:       errCount,
			ChildCount:       childCount,
		}
		view.AttentionSignal = a.classify(view, now)

		viewsByProject[s.ProjectID] = append(viewsByProject[s.ProjectID], view)
	}

	// Build ProjectGroups
	groups := make([]domain.ProjectGroup, 0, len(projects))
	for _, p := range projects {
		views := viewsByProject[p.ID]

		// Sort sessions: AttentionSignal DESC, then TimeUpdated DESC
		sort.SliceStable(views, func(i, j int) bool {
			if views[i].AttentionSignal != views[j].AttentionSignal {
				return views[i].AttentionSignal > views[j].AttentionSignal
			}
			return views[i].TimeUpdated.After(views[j].TimeUpdated)
		})

		attentionCount := 0
		for _, v := range views {
			if v.AttentionSignal != domain.None {
				attentionCount++
			}
		}

		groups = append(groups, domain.ProjectGroup{
			Project:        p,
			Sessions:       views,
			AttentionCount: attentionCount,
		})
	}

	// Sort groups: alphabetical by DisplayName, Global always last
	sort.SliceStable(groups, func(i, j int) bool {
		iGlobal := groups[i].Project.ID == "global" || groups[i].Project.Worktree == "/"
		jGlobal := groups[j].Project.ID == "global" || groups[j].Project.Worktree == "/"
		if iGlobal != jGlobal {
			return !iGlobal // non-global comes before global
		}
		return groups[i].Project.DisplayName() < groups[j].Project.DisplayName()
	})

	return groups, nil
}

// Refresh refreshes the error cache then returns the same result as LoadAll.
func (a *Aggregator) Refresh(ctx context.Context) ([]domain.ProjectGroup, error) {
	if err := a.errors.RefreshErrorCache(ctx, time.Time{}); err != nil {
		return nil, err
	}
	return a.LoadAll(ctx)
}
