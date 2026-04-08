package attention

import (
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

const (
	NeedsResponseWindow = 24 * time.Hour
	ActiveNowWindow     = 5 * time.Minute
	StaleThreshold      = 7 * 24 * time.Hour
)

func Classify(view domain.SessionView, now time.Time) domain.AttentionSignal {
	if view.IsArchived() {
		return domain.None
	}

	age := now.Sub(view.TimeUpdated)

	if view.ErrorCount > 0 {
		return domain.HasErrors
	}

	if view.HasPendingQuestion {
		return domain.WaitingForInput
	}

	if age < ActiveNowWindow {
		return domain.ActiveNow
	}

	if view.LastMessage.Role == "assistant" && age < NeedsResponseWindow {
		return domain.NeedsResponse
	}

	if view.PendingTodoCount > 0 {
		if age >= StaleThreshold {
			return domain.StaleWork
		}

		return domain.PendingTodos
	}

	return domain.None
}
