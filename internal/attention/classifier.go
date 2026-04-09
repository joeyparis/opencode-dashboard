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

// Thresholds configures attention classifier time windows.
type Thresholds struct {
	ActiveNowWindow     time.Duration
	NeedsResponseWindow time.Duration
	StaleThreshold      time.Duration
}

// DefaultThresholds returns the built-in default attention thresholds.
func DefaultThresholds() Thresholds {
	return Thresholds{
		ActiveNowWindow:     ActiveNowWindow,
		NeedsResponseWindow: NeedsResponseWindow,
		StaleThreshold:      StaleThreshold,
	}
}

// ClassifyWith classifies a session using configurable thresholds.
func ClassifyWith(view domain.SessionView, now time.Time, t Thresholds) domain.AttentionSignal {
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

	if age < t.ActiveNowWindow {
		return domain.ActiveNow
	}

	if view.LastMessage.Role == "assistant" && age < t.NeedsResponseWindow {
		return domain.NeedsResponse
	}

	if view.PendingTodoCount > 0 {
		if age >= t.StaleThreshold {
			return domain.StaleWork
		}

		return domain.PendingTodos
	}

	return domain.None
}

func Classify(view domain.SessionView, now time.Time) domain.AttentionSignal {
	return ClassifyWith(view, now, DefaultThresholds())
}
