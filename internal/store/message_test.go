package store

import (
	"context"
	"testing"

	"github.com/joeyparis/opencode-dashboard/internal/testutil"
)

func TestGetLastMessageMeta_AssistantRole(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	meta, err := repo.GetLastMessageMeta(context.Background(), "ses-needs-response")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Role != "assistant" {
		t.Errorf("expected role='assistant', got %q", meta.Role)
	}
	if meta.Agent != "Sisyphus" {
		t.Errorf("expected agent='Sisyphus', got %q", meta.Agent)
	}
	if meta.ModelID != "claude-haiku-4-5" {
		t.Errorf("expected modelID='claude-haiku-4-5', got %q", meta.ModelID)
	}
	if meta.ProviderID != "anthropic" {
		t.Errorf("expected providerID='anthropic', got %q", meta.ProviderID)
	}
	if meta.TimeCreated.IsZero() {
		t.Error("expected non-zero TimeCreated")
	}
}

func TestGetLastMessageMeta_UserRole(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	meta, err := repo.GetLastMessageMeta(context.Background(), "ses-active-now")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Role != "user" {
		t.Errorf("expected role='user', got %q", meta.Role)
	}
}

func TestGetLastMessageMeta_NoMessages(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	meta, err := repo.GetLastMessageMeta(context.Background(), "ses-archived")
	if err != nil {
		t.Fatalf("expected no error for session with no messages, got: %v", err)
	}

	if meta.Role != "" {
		t.Errorf("expected empty role for no-message session, got %q", meta.Role)
	}
	if !meta.TimeCreated.IsZero() {
		t.Errorf("expected zero TimeCreated for no-message session, got %v", meta.TimeCreated)
	}
}

func TestGetMessageCount(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	count, err := repo.GetMessageCount(context.Background(), "ses-needs-response")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 5 {
		t.Errorf("expected count=5, got %d", count)
	}
}

func TestGetMessageCount_NoMessages(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	count, err := repo.GetMessageCount(context.Background(), "ses-archived")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 0 {
		t.Errorf("expected count=0, got %d", count)
	}
}
