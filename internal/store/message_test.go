package store

import (
	"context"
	"testing"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
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

func TestGetRecentMessages_MultipleMessages(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	previews, err := repo.GetRecentMessages(context.Background(), "ses-needs-response", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(previews) != 5 {
		t.Errorf("expected 5 messages, got %d", len(previews))
	}

	if previews[0].Role != "assistant" {
		t.Errorf("expected first message role='assistant', got %q", previews[0].Role)
	}

	if previews[4].Role != "assistant" {
		t.Errorf("expected last message role='assistant', got %q", previews[4].Role)
	}
}

func TestGetRecentMessages_NoMessages(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	previews, err := repo.GetRecentMessages(context.Background(), "ses-archived", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if previews == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(previews) != 0 {
		t.Errorf("expected 0 messages, got %d", len(previews))
	}
}

func TestGetRecentMessages_SingleMessage(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	previews, err := repo.GetRecentMessages(context.Background(), "ses-global", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(previews) != 1 {
		t.Errorf("expected 1 message, got %d", len(previews))
	}

	if previews[0].Role != "assistant" {
		t.Errorf("expected role='assistant', got %q", previews[0].Role)
	}
}

func TestGetRecentMessages_ContentTruncation(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	previews, err := repo.GetRecentMessages(context.Background(), "ses-needs-response", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var msg4 *domain.MessagePreview
	for i := range previews {
		if previews[i].Content == "I've reviewed the session handling code. Here are my findings and recommendations for improving the implementation and test coverage."[:100]+"..." {
			msg4 = &previews[i]
			break
		}
	}

	if msg4 == nil {
		t.Fatal("could not find msg-4 with truncated content")
	}

	expectedContent := "I've reviewed the session handling code. Here are my findings and recommendations for improving the implementation and test coverage."
	expectedTruncated := string([]rune(expectedContent)[:100]) + "..."

	if msg4.Content != expectedTruncated {
		t.Errorf("expected truncated content %q, got %q", expectedTruncated, msg4.Content)
	}
}

func TestGetRecentMessages_NewlineFlattening(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	previews, err := repo.GetRecentMessages(context.Background(), "ses-needs-response", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var msg3 *domain.MessagePreview
	for i := range previews {
		if previews[i].Content == "Please also check the session handling.  And fix the tests." {
			msg3 = &previews[i]
			break
		}
	}

	if msg3 == nil {
		t.Fatal("could not find msg-3 with flattened newlines")
	}

	expectedContent := "Please also check the session handling.  And fix the tests."
	if msg3.Content != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, msg3.Content)
	}
}

func TestGetRecentMessages_NoTextParts(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewMessageRepo(db)
	previews, err := repo.GetRecentMessages(context.Background(), "ses-has-errors", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(previews) != 2 {
		t.Errorf("expected 2 messages, got %d", len(previews))
	}

	var msg2 *domain.MessagePreview
	for i := range previews {
		if previews[i].Content == "[tool use]" {
			msg2 = &previews[i]
			break
		}
	}

	if msg2 == nil {
		t.Fatal("could not find message with [tool use] content")
	}

	if msg2.Content != "[tool use]" {
		t.Errorf("expected content '[tool use]', got %q", msg2.Content)
	}
}
