package infra

import (
	"context"
	"testing"
	"time"

	"markpost/internal/domain/delivery"
)

// TestAttemptRepository_PruneHistoryPerUserPolicy covers the per-row retention
// predicate on delivery_history (MRFC 2026-08-31-per-user-history-retention-
// policy): an explicit user retention_days (0 = forever) overrides the global
// window, and rows orphaned by user deletion (user_id NULL via ON DELETE SET
// NULL) fall back to the global window.
func TestAttemptRepository_PruneHistoryPerUserPolicy(t *testing.T) {
	db := SetupTestDB(t)
	repo, ok := NewAttemptRepository(db).(*AttemptRepository)
	if !ok {
		t.Fatalf("NewAttemptRepository did not return *AttemptRepository")
	}
	ctx := context.Background()

	inherit := createTestUser(t, db, 1)
	forever := createTestUser(t, db, 2)
	orphan := createTestUser(t, db, 3)
	setRetention(t, db, forever, intPtr(0))

	seedHistory := func(userID *int, when time.Time) {
		t.Helper()
		if err := db.Exec(
			"INSERT INTO delivery_history (user_id, status, last_error, created_at) VALUES (?, ?, '', ?)",
			userID, delivery.StatusDelivered, when,
		).Error; err != nil {
			t.Fatalf("seed history: %v", err)
		}
	}

	old := time.Now().Add(-48 * time.Hour)
	seedHistory(&inherit, old)        // inherit → global 24h window deletes it
	seedHistory(&forever, old)        // explicit forever → kept
	seedHistory(&orphan, old)         // becomes anonymous below → global window
	seedHistory(nil, old)             // born anonymous → global window
	seedHistory(&inherit, time.Now()) // fresh → kept by any window

	// Deleting the user nulls the history row's user_id (ON DELETE SET NULL).
	if err := db.Exec("DELETE FROM users WHERE id = ?", orphan).Error; err != nil {
		t.Fatalf("delete orphan user: %v", err)
	}

	count, err := repo.CountHistoryExpired(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("CountHistoryExpired: %v", err)
	}
	if count != 3 {
		t.Errorf("CountHistoryExpired = %d, want 3 (inherit + two anonymous)", count)
	}

	deleted, err := repo.PruneHistory(ctx, 24*time.Hour, 1000)
	if err != nil {
		t.Fatalf("PruneHistory: %v", err)
	}
	if deleted != 3 {
		t.Errorf("deleted = %d, want 3", deleted)
	}

	var remaining []delivery.History
	if err := db.Find(&remaining).Error; err != nil {
		t.Fatalf("load remaining: %v", err)
	}
	if len(remaining) != 2 {
		t.Errorf("remaining rows = %d, want 2 (forever-old + fresh)", len(remaining))
	}
	var foreverOld int64
	if err := db.Model(&delivery.History{}).
		Where("user_id = ? AND created_at < ?", forever, time.Now().Add(-24*time.Hour)).
		Count(&foreverOld).Error; err != nil {
		t.Fatalf("count forever rows: %v", err)
	}
	if foreverOld != 1 {
		t.Errorf("old forever rows = %d, want 1 (explicit 0 must survive the sweep)", foreverOld)
	}

	// After the sweep the dry-run count drops to zero.
	count, err = repo.CountHistoryExpired(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("CountHistoryExpired after prune: %v", err)
	}
	if count != 0 {
		t.Errorf("CountHistoryExpired after prune = %d, want 0", count)
	}
}
