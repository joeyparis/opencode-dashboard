package attention

import (
	"testing"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestClassify_HasErrors(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{ErrorCount: 3}
	view.TimeUpdated = now.Add(-1 * time.Hour)

	assert.Equal(t, domain.HasErrors, Classify(view, now))
}

func TestClassify_ActiveNow(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{}
	view.TimeUpdated = now.Add(-2 * time.Minute)

	assert.Equal(t, domain.ActiveNow, Classify(view, now))
}

func TestClassify_ActiveNowBoundary(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{}
	view.TimeUpdated = now.Add(-ActiveNowWindow)

	assert.Equal(t, domain.None, Classify(view, now))
}

func TestClassify_NeedsResponse(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{
		LastMessage: domain.MessageMeta{Role: "assistant"},
	}
	view.TimeUpdated = now.Add(-1 * time.Hour)

	assert.Equal(t, domain.NeedsResponse, Classify(view, now))
}

func TestClassify_NeedsResponseBoundary(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{
		LastMessage: domain.MessageMeta{Role: "assistant"},
	}
	view.TimeUpdated = now.Add(-NeedsResponseWindow)

	assert.Equal(t, domain.None, Classify(view, now))
}

func TestClassify_StaleWork(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{PendingTodoCount: 2}
	view.TimeUpdated = now.Add(-10 * 24 * time.Hour)

	assert.Equal(t, domain.StaleWork, Classify(view, now))
}

func TestClassify_StaleWorkBoundary(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{PendingTodoCount: 2}
	view.TimeUpdated = now.Add(-StaleThreshold)

	assert.Equal(t, domain.StaleWork, Classify(view, now))
}

func TestClassify_PendingTodos(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{PendingTodoCount: 2}
	view.TimeUpdated = now.Add(-24 * time.Hour)

	assert.Equal(t, domain.PendingTodos, Classify(view, now))
}

func TestClassify_None(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{}
	view.TimeUpdated = now.Add(-5 * 24 * time.Hour)

	assert.Equal(t, domain.None, Classify(view, now))
}

func TestClassify_ArchivedOverridesAllSignals(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{
		ErrorCount:       5,
		PendingTodoCount: 2,
		LastMessage:      domain.MessageMeta{Role: "assistant"},
	}
	view.TimeUpdated = now.Add(-1 * time.Minute)
	view.TimeArchived = now.Add(-30 * 24 * time.Hour)

	assert.Equal(t, domain.None, Classify(view, now))
}

func TestClassify_HasErrorsWinsOverActiveNowAndNeedsResponse(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{
		ErrorCount:  1,
		LastMessage: domain.MessageMeta{Role: "assistant"},
	}
	view.TimeUpdated = now.Add(-1 * time.Minute)

	assert.Equal(t, domain.HasErrors, Classify(view, now))
}

func TestClassify_ActiveNowWinsOverNeedsResponse(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{
		LastMessage: domain.MessageMeta{Role: "assistant"},
	}
	view.TimeUpdated = now.Add(-2 * time.Minute)

	assert.Equal(t, domain.ActiveNow, Classify(view, now))
}

func TestClassify_EdgeNoMessages(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{}
	view.TimeUpdated = now.Add(-1 * time.Hour)

	assert.Equal(t, domain.None, Classify(view, now))
}

func TestClassify_EdgeEmptyRole(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{
		LastMessage: domain.MessageMeta{Role: ""},
	}
	view.TimeUpdated = now.Add(-1 * time.Hour)

	assert.Equal(t, domain.None, Classify(view, now))
}

func TestClassify_UserMessageDoesNotNeedResponse(t *testing.T) {
	now := fixedNow()
	view := domain.SessionView{
		LastMessage: domain.MessageMeta{Role: "user"},
	}
	view.TimeUpdated = now.Add(-1 * time.Hour)

	assert.Equal(t, domain.None, Classify(view, now))
}

func fixedNow() time.Time {
	return time.Date(2026, time.April, 8, 12, 0, 0, 0, time.UTC)
}
