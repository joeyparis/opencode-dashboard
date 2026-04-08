package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/joeyparis/opencode-dashboard/internal/testutil"
)

func TestRefreshAll(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewErrorRepo(db)
	ctx := context.Background()

	if err := repo.RefreshAll(ctx); err != nil {
		t.Fatalf("RefreshAll failed: %v", err)
	}

	count, err := repo.GetErrorCount(ctx, "ses-has-errors")
	if err != nil {
		t.Fatalf("GetErrorCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 error parts for ses-has-errors, got %d", count)
	}
}

func TestRefreshAll_ZeroCount(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewErrorRepo(db)
	ctx := context.Background()

	if err := repo.RefreshAll(ctx); err != nil {
		t.Fatalf("RefreshAll failed: %v", err)
	}

	count, err := repo.GetErrorCount(ctx, "ses-active-now")
	if err != nil {
		t.Fatalf("GetErrorCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 error parts for ses-active-now, got %d", count)
	}
}

func TestRefreshErrorCache(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewErrorRepo(db)
	ctx := context.Background()

	if err := repo.RefreshAll(ctx); err != nil {
		t.Fatalf("RefreshAll failed: %v", err)
	}

	since := time.Now()
	if err := repo.RefreshErrorCache(ctx, since); err != nil {
		t.Fatalf("RefreshErrorCache failed: %v", err)
	}

	count, err := repo.GetErrorCount(ctx, "ses-has-errors")
	if err != nil {
		t.Fatalf("GetErrorCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 error parts (unchanged from RefreshAll), got %d", count)
	}
}

func TestGetErrorCount_CacheOnly(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewErrorRepo(db)
	ctx := context.Background()

	count, err := repo.GetErrorCount(ctx, "ses-has-errors")
	if err != nil {
		t.Fatalf("GetErrorCount failed before refresh: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 before refresh (cache empty), got %d", count)
	}

	if err := repo.RefreshAll(ctx); err != nil {
		t.Fatalf("RefreshAll failed: %v", err)
	}

	count, err = repo.GetErrorCount(ctx, "ses-has-errors")
	if err != nil {
		t.Fatalf("GetErrorCount failed after refresh: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 after refresh, got %d", count)
	}
}

func TestConcurrentAccess(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	repo := NewErrorRepo(db)
	ctx := context.Background()

	if err := repo.RefreshAll(ctx); err != nil {
		t.Fatalf("RefreshAll failed: %v", err)
	}

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.GetErrorCount(ctx, "ses-has-errors")
			if err != nil {
				t.Errorf("GetErrorCount failed in goroutine: %v", err)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := repo.RefreshAll(ctx); err != nil {
			t.Errorf("RefreshAll failed in goroutine: %v", err)
		}
	}()

	wg.Wait()
}
