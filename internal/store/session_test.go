package store

import (
	"context"
	"testing"

	"github.com/joeyparis/opencode-dashboard/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRootSessions(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewSessionRepo(db)
	sessions, err := repo.ListRootSessions(context.Background())
	require.NoError(t, err)
	assert.Len(t, sessions, 8, "expected 8 root sessions")

	for _, s := range sessions {
		assert.Empty(t, s.ParentID, "root session should have empty ParentID")
	}
}

func TestListRootSessions_ExcludesChildren(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewSessionRepo(db)
	sessions, err := repo.ListRootSessions(context.Background())
	require.NoError(t, err)

	ids := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		ids[s.ID] = true
	}

	assert.False(t, ids["ses-child-1"], "child session must not appear in root list")
	assert.False(t, ids["ses-child-2"], "child session must not appear in root list")
	assert.False(t, ids["ses-child-3"], "child session must not appear in root list")
	assert.False(t, ids["ses-child-4"], "child session must not appear in root list")
}

func TestGetSessionByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewSessionRepo(db)
	s, err := repo.GetSessionByID(context.Background(), "ses-needs-response")
	require.NoError(t, err)
	assert.Equal(t, "ses-needs-response", s.ID)
	assert.Equal(t, "project-alpha", s.ProjectID)
	assert.Empty(t, s.ParentID)
	assert.False(t, s.IsArchived())
	assert.False(t, s.TimeCreated.IsZero())
	assert.False(t, s.TimeUpdated.IsZero())
}

func TestGetSessionByID_Archived(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewSessionRepo(db)
	s, err := repo.GetSessionByID(context.Background(), "ses-archived")
	require.NoError(t, err)
	assert.True(t, s.IsArchived())
}

func TestGetSessionByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewSessionRepo(db)
	_, err := repo.GetSessionByID(context.Background(), "non-existent-id")
	assert.Error(t, err)
}

func TestGetChildCount(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewSessionRepo(db)

	count, err := repo.GetChildCount(context.Background(), "ses-needs-response")
	require.NoError(t, err)
	assert.Equal(t, 2, count, "ses-needs-response should have 2 children")

	count, err = repo.GetChildCount(context.Background(), "ses-has-errors")
	require.NoError(t, err)
	assert.Equal(t, 1, count, "ses-has-errors should have 1 child")

	count, err = repo.GetChildCount(context.Background(), "ses-stale")
	require.NoError(t, err)
	assert.Equal(t, 1, count, "ses-stale should have 1 child")

	count, err = repo.GetChildCount(context.Background(), "ses-active-now")
	require.NoError(t, err)
	assert.Equal(t, 0, count, "ses-active-now should have 0 children")
}
