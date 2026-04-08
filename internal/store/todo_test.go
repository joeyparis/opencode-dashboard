package store

import (
	"context"
	"testing"

	"github.com/joeyparis/opencode-dashboard/internal/testutil"
)

func TestGetTodosBySession_WithTodos(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewTodoRepo(db)

	todos, err := repo.GetTodosBySession(context.Background(), "ses-stale")
	if err != nil {
		t.Fatalf("GetTodosBySession failed: %v", err)
	}

	if len(todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(todos))
	}

	if todos[0].Position != 1 {
		t.Errorf("expected first todo position=1, got %d", todos[0].Position)
	}
	if todos[1].Position != 2 {
		t.Errorf("expected second todo position=2, got %d", todos[1].Position)
	}

	for _, todo := range todos {
		if todo.SessionID != "ses-stale" {
			t.Errorf("expected session_id=ses-stale, got %s", todo.SessionID)
		}
	}
}

func TestGetTodosBySession_Mixed(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewTodoRepo(db)

	todos, err := repo.GetTodosBySession(context.Background(), "ses-pending-todos")
	if err != nil {
		t.Fatalf("GetTodosBySession failed: %v", err)
	}

	if len(todos) != 3 {
		t.Fatalf("expected 3 todos, got %d", len(todos))
	}

	statuses := map[string]int{}
	for _, todo := range todos {
		statuses[todo.Status]++
	}

	if statuses["pending"] != 2 {
		t.Errorf("expected 2 pending todos, got %d", statuses["pending"])
	}
	if statuses["completed"] != 1 {
		t.Errorf("expected 1 completed todo, got %d", statuses["completed"])
	}
}

func TestGetTodosBySession_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewTodoRepo(db)

	t.Run("no todos at all returns empty slice not nil", func(t *testing.T) {
		todos, err := repo.GetTodosBySession(context.Background(), "ses-active-now")
		if err != nil {
			t.Fatalf("GetTodosBySession failed: %v", err)
		}
		if todos == nil {
			t.Error("expected empty slice, got nil")
		}
		if len(todos) != 0 {
			t.Errorf("expected 0 todos, got %d", len(todos))
		}
	})

	t.Run("session with only completed todos still returns them", func(t *testing.T) {
		todos, err := repo.GetTodosBySession(context.Background(), "ses-no-signal")
		if err != nil {
			t.Fatalf("GetTodosBySession failed: %v", err)
		}
		if len(todos) != 2 {
			t.Errorf("expected 2 todos (all completed), got %d", len(todos))
		}
		for _, todo := range todos {
			if todo.Status != "completed" {
				t.Errorf("expected all todos completed, got status=%s", todo.Status)
			}
		}
	})
}

func TestGetPendingTodoCount_Pending(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewTodoRepo(db)

	count, err := repo.GetPendingTodoCount(context.Background(), "ses-stale")
	if err != nil {
		t.Fatalf("GetPendingTodoCount failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected pending count=2, got %d", count)
	}
}

func TestGetPendingTodoCount_Mixed(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewTodoRepo(db)

	count, err := repo.GetPendingTodoCount(context.Background(), "ses-pending-todos")
	if err != nil {
		t.Fatalf("GetPendingTodoCount failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected pending count=2 (excludes completed), got %d", count)
	}
}

func TestGetPendingTodoCount_AllCompleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewTodoRepo(db)

	count, err := repo.GetPendingTodoCount(context.Background(), "ses-no-signal")
	if err != nil {
		t.Fatalf("GetPendingTodoCount failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected pending count=0 (all completed), got %d", count)
	}
}

func TestGetPendingTodoCount_NoTodos(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewTodoRepo(db)

	count, err := repo.GetPendingTodoCount(context.Background(), "ses-active-now")
	if err != nil {
		t.Fatalf("GetPendingTodoCount failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected pending count=0 (no todos), got %d", count)
	}
}
