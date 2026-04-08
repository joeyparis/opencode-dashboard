package filter

import (
	"strings"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

// Filter defines the criteria for filtering sessions.
type Filter struct {
	Preset     domain.FilterPreset
	SearchText string
}

// Apply returns a new slice of sessions matching the filter criteria.
// It does not mutate the input slice.
// Returns an empty (non-nil) slice when no sessions match.
func Apply(sessions []domain.SessionView, f Filter) []domain.SessionView {
	result := make([]domain.SessionView, 0)
	for _, s := range sessions {
		if !matchesPreset(s, f.Preset) {
			continue
		}
		if !matchesSearch(s, f.SearchText) {
			continue
		}
		result = append(result, s)
	}
	return result
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
