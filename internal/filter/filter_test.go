package filter_test

import (
	"testing"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/joeyparis/opencode-dashboard/internal/filter"
	"github.com/stretchr/testify/assert"
)

func fixedNow() time.Time {
	return time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
}

func makeTestSessions() []domain.SessionView {
	now := time.Now()
	return []domain.SessionView{
		{
			Session:         domain.Session{ID: "1", Title: "Fix Auth Bug", Slug: "fix-auth"},
			AttentionSignal: domain.HasErrors,
		},
		{
			Session:         domain.Session{ID: "2", Title: "Add Login Page", Slug: "add-login", TimeArchived: now.Add(-1 * time.Hour)},
			AttentionSignal: domain.None,
		},
		{
			Session:         domain.Session{ID: "3", Title: "Setup CI", Slug: "setup-ci"},
			AttentionSignal: domain.NeedsResponse,
		},
		{
			Session:         domain.Session{ID: "4", Title: "No Signal Session", Slug: "no-signal"},
			AttentionSignal: domain.None,
		},
	}
}

func sessionIDs(sessions []domain.SessionView) []string {
	ids := make([]string, len(sessions))
	for i, s := range sessions {
		ids[i] = s.ID
	}
	return ids
}

// Test 1: FilterNeedsAttention preset returns only sessions with non-None signal
func TestApply_NeedsAttentionPreset(t *testing.T) {
	sessions := makeTestSessions()
	f := filter.Filter{Preset: domain.FilterNeedsAttention}

	result := filter.Apply(sessions, f, time.Now())

	assert.Len(t, result, 2)
	ids := sessionIDs(result)
	assert.Contains(t, ids, "1") // HasErrors
	assert.Contains(t, ids, "3") // NeedsResponse
}

// Test 2: FilterAllActive preset excludes archived sessions
func TestApply_AllActivePreset(t *testing.T) {
	sessions := makeTestSessions()
	f := filter.Filter{Preset: domain.FilterAllActive}

	result := filter.Apply(sessions, f, time.Now())

	assert.Len(t, result, 3)
	ids := sessionIDs(result)
	assert.Contains(t, ids, "1")
	assert.Contains(t, ids, "3")
	assert.Contains(t, ids, "4")
	assert.NotContains(t, ids, "2") // archived
}

// Test 3: FilterArchived preset returns only archived sessions
func TestApply_ArchivedPreset(t *testing.T) {
	sessions := makeTestSessions()
	f := filter.Filter{Preset: domain.FilterArchived}

	result := filter.Apply(sessions, f, time.Now())

	assert.Len(t, result, 1)
	assert.Equal(t, "2", result[0].ID)
}

// Test 4: Search "auth" matches title of session 1
func TestApply_SearchByTitle(t *testing.T) {
	sessions := makeTestSessions()

	// Use FilterAllActive to keep only active sessions, "auth" appears in #1
	f := filter.Filter{
		Preset:     domain.FilterAllActive,
		SearchText: "auth",
	}

	result := filter.Apply(sessions, f, time.Now())

	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)
}

// Test 5: Search "CI" is case-insensitive and matches slug "setup-ci"
func TestApply_SearchCaseInsensitive(t *testing.T) {
	sessions := makeTestSessions()
	f := filter.Filter{
		Preset:     domain.FilterAllActive,
		SearchText: "CI",
	}

	result := filter.Apply(sessions, f, time.Now())

	assert.Len(t, result, 1)
	assert.Equal(t, "3", result[0].ID)
}

// Test 6: Empty search text returns all sessions matching preset
func TestApply_EmptySearchReturnsAll(t *testing.T) {
	sessions := makeTestSessions()
	f := filter.Filter{
		Preset:     domain.FilterAllActive,
		SearchText: "",
	}

	result := filter.Apply(sessions, f, time.Now())

	// All active: #1, #3, #4
	assert.Len(t, result, 3)
}

// Test 7: Combined preset + search uses AND logic
func TestApply_CombinedPresetAndSearch(t *testing.T) {
	sessions := makeTestSessions()
	f := filter.Filter{
		Preset:     domain.FilterNeedsAttention,
		SearchText: "auth",
	}

	result := filter.Apply(sessions, f, time.Now())

	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)
}

// Test 8: No matches returns empty slice (not nil)
func TestApply_NoMatchesReturnsEmptySlice(t *testing.T) {
	sessions := makeTestSessions()
	f := filter.Filter{
		Preset:     domain.FilterNeedsAttention,
		SearchText: "zzznomatch",
	}

	result := filter.Apply(sessions, f, time.Now())

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// Test 9: Apply does not mutate the input slice
func TestApply_DoesNotMutateInput(t *testing.T) {
	sessions := makeTestSessions()
	original := make([]domain.SessionView, len(sessions))
	copy(original, sessions)

	f := filter.Filter{Preset: domain.FilterNeedsAttention}
	_ = filter.Apply(sessions, f, time.Now())

	assert.Equal(t, original, sessions)
}

// Test 10: Slug search match
func TestApply_SearchBySlug(t *testing.T) {
	sessions := makeTestSessions()
	f := filter.Filter{
		Preset:     domain.FilterAllActive,
		SearchText: "no-signal",
	}

	result := filter.Apply(sessions, f, time.Now())

	assert.Len(t, result, 1)
	assert.Equal(t, "4", result[0].ID)
}

func TestApply_TimeWindow_All(t *testing.T) {
	now := fixedNow()
	sessions := []domain.SessionView{
		{Session: domain.Session{ID: "recent", TimeUpdated: now.Add(-10 * time.Hour)}},
		{Session: domain.Session{ID: "old", TimeUpdated: now.Add(-30 * 24 * time.Hour)}},
	}
	f := filter.Filter{
		Preset: domain.FilterAllActive,
		Window: 0,
	}

	result := filter.Apply(sessions, f, now)

	assert.Len(t, result, 2)
	assert.Equal(t, []string{"recent", "old"}, sessionIDs(result))
}

func TestApply_TimeWindow_1Day(t *testing.T) {
	now := fixedNow()
	sessions := []domain.SessionView{
		{Session: domain.Session{ID: "inside", TimeUpdated: now.Add(-10 * time.Hour)}},
		{Session: domain.Session{ID: "outside", TimeUpdated: now.Add(-30 * time.Hour)}},
	}
	f := filter.Filter{
		Preset: domain.FilterAllActive,
		Window: 24 * time.Hour,
	}

	result := filter.Apply(sessions, f, now)

	assert.Len(t, result, 1)
	assert.Equal(t, "inside", result[0].ID)
}

func TestApply_TimeWindow_Boundary(t *testing.T) {
	now := fixedNow()
	sessions := []domain.SessionView{
		{Session: domain.Session{ID: "boundary", TimeUpdated: now.Add(-24 * time.Hour)}},
		{Session: domain.Session{ID: "inside", TimeUpdated: now.Add(-23 * time.Hour)}},
	}
	f := filter.Filter{
		Preset: domain.FilterAllActive,
		Window: 24 * time.Hour,
	}

	result := filter.Apply(sessions, f, now)

	assert.Len(t, result, 1)
	assert.Equal(t, "inside", result[0].ID)
}
