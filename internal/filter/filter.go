package filter

import (
	"strings"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

type Filter struct {
	Preset     domain.FilterPreset
	SearchText string
	Window     time.Duration
}

func Apply(sessions []domain.SessionView, f Filter, now time.Time) []domain.SessionView {
	filtered := make([]domain.SessionView, 0)
	for _, s := range sessions {
		if !matchesPreset(s, f.Preset) {
			continue
		}
		if !matchesSearch(s, f.SearchText) {
			continue
		}
		if !matchesWindow(s, f.Window, now) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

func matchesPreset(s domain.SessionView, preset domain.FilterPreset) bool {
	switch preset {
	case domain.FilterNeedsAttention:
		return s.AttentionSignal != domain.None
	case domain.FilterAllActive:
		return !s.IsArchived()
	case domain.FilterArchived:
		return s.IsArchived()
	default:
		return true
	}
}

func matchesSearch(s domain.SessionView, text string) bool {
	if text == "" {
		return true
	}
	lower := strings.ToLower(text)
	return strings.Contains(strings.ToLower(s.Title), lower) ||
		strings.Contains(strings.ToLower(s.Slug), lower)
}

func matchesWindow(s domain.SessionView, d time.Duration, now time.Time) bool {
	if d == 0 {
		return true
	}
	return s.TimeUpdated.After(now.Add(-d))
}
