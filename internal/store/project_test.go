package store

import (
	"context"
	"testing"

	"github.com/joeyparis/opencode-dashboard/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewProjectRepo(db)
	projects, err := repo.ListProjects(context.Background())
	require.NoError(t, err)
	assert.Len(t, projects, 3, "expected 3 projects")

	for _, p := range projects {
		assert.Empty(t, p.Name, "project.Name should be empty (NULL in DB)")
		assert.NotEmpty(t, p.ID, "project.ID should not be empty")
		assert.NotEmpty(t, p.Worktree, "project.Worktree should not be empty")
		assert.False(t, p.TimeCreated.IsZero(), "TimeCreated should be set")
		assert.False(t, p.TimeUpdated.IsZero(), "TimeUpdated should be set")
	}

	ids := make([]string, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
	}
	assert.Contains(t, ids, "project-alpha")
	assert.Contains(t, ids, "project-beta")
	assert.Contains(t, ids, "global")
}

func TestListProjects_OrderByWorktree(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewProjectRepo(db)
	projects, err := repo.ListProjects(context.Background())
	require.NoError(t, err)
	require.Len(t, projects, 3)

	assert.Equal(t, "global", projects[0].ID, "first project (worktree '/') should be global")
	assert.Equal(t, "project-alpha", projects[1].ID, "second project should be alpha")
	assert.Equal(t, "project-beta", projects[2].ID, "third project should be beta")
}
