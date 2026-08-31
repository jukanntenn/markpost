package infra

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
)

// seedOldPost creates a post and backdates created_at past the given age
// (GORM's autoCreateTime would otherwise stamp now).
func seedOldPost(t *testing.T, db *gorm.DB, repo *PostRepository, uid int, title string, ageDays int) string {
	t.Helper()
	p, err := repo.Create(context.Background(), title, "Body", uid)
	if err != nil {
		t.Fatalf("create post %q: %v", title, err)
	}
	if err := db.Model(p).Update("created_at", time.Now().AddDate(0, 0, -ageDays)).Error; err != nil {
		t.Fatalf("backdate post %q: %v", title, err)
	}
	return p.QID
}

// setRetention writes a per-user retention policy; nil means inherit.
func setRetention(t *testing.T, db *gorm.DB, userID int, days *int) {
	t.Helper()
	if err := db.Exec("UPDATE users SET retention_days = ? WHERE id = ?", days, userID).Error; err != nil {
		t.Fatalf("set retention for user %d: %v", userID, err)
	}
}

func intPtr(v int) *int { return &v }

// TestPostRepository_PruneExpiredPerUserPolicy covers the per-row retention
// predicate (MRFC 2026-08-31-per-user-history-retention-policy): explicit
// forever (0) and explicit N-day windows override the global fallback, inherit
// (NULL) rows use the global window, and a global 0 (never expire) excludes
// every inheriting row.
func TestPostRepository_PruneExpiredPerUserPolicy(t *testing.T) {
	db := SetupTestDB(t)
	repo, ok := NewPostRepository(db).(*PostRepository)
	if !ok {
		t.Fatalf("NewPostRepository did not return *PostRepository")
	}
	ctx := context.Background()

	inherit := createTestUser(t, db, 1)
	forever := createTestUser(t, db, 2)
	thirty := createTestUser(t, db, 3)
	setRetention(t, db, forever, intPtr(0))
	setRetention(t, db, thirty, intPtr(30))

	// All posts are 10 days old; the global window below is 7 days.
	inheritQID := seedOldPost(t, db, repo, inherit, "inherit old", 10)
	seedOldPost(t, db, repo, forever, "forever old", 10)
	thirtyQID := seedOldPost(t, db, repo, thirty, "thirty old", 10)

	count, err := repo.CountExpired(ctx, 7)
	if err != nil {
		t.Fatalf("CountExpired: %v", err)
	}
	if count != 1 {
		t.Errorf("CountExpired = %d, want 1 (only the inherit row)", count)
	}

	pruned, err := repo.PruneExpired(ctx, 7, 100)
	if err != nil {
		t.Fatalf("PruneExpired: %v", err)
	}
	if len(pruned) != 1 || pruned[0] != inheritQID {
		t.Errorf("pruned = %v, want [%s]", pruned, inheritQID)
	}

	for uid, want := range map[int]int{inherit: 0, forever: 1, thirty: 1} {
		var got int64
		if err := db.Table("posts").Where("user_id = ?", uid).Count(&got).Error; err != nil {
			t.Fatalf("count posts for user %d: %v", uid, err)
		}
		if got != int64(want) {
			t.Errorf("user %d posts = %d, want %d", uid, got, want)
		}
	}

	// Shortening an explicit window applies at the next sweep: a 10-day-old
	// post under a policy shortened to 5 days goes.
	setRetention(t, db, thirty, intPtr(5))
	pruned, err = repo.PruneExpired(ctx, 7, 100)
	if err != nil {
		t.Fatalf("PruneExpired after shorten: %v", err)
	}
	if len(pruned) != 1 || pruned[0] != thirtyQID {
		t.Errorf("pruned after shorten = %v, want [%s]", pruned, thirtyQID)
	}

	// Global 0 = never expire: inherit rows are excluded too.
	keptQID := seedOldPost(t, db, repo, inherit, "inherit ancient", 3650)
	pruned, err = repo.PruneExpired(ctx, 0, 100)
	if err != nil {
		t.Fatalf("PruneExpired global 0: %v", err)
	}
	if len(pruned) != 0 {
		t.Errorf("pruned under global 0 = %v, want none", pruned)
	}
	count, err = repo.CountExpired(ctx, 0)
	if err != nil {
		t.Fatalf("CountExpired global 0: %v", err)
	}
	if count != 0 {
		t.Errorf("CountExpired under global 0 = %d, want 0", count)
	}
	var kept int64
	if err := db.Table("posts").Where("qid = ?", keptQID).Count(&kept).Error; err != nil {
		t.Fatalf("count kept post: %v", err)
	}
	if kept != 1 {
		t.Errorf("kept post rows = %d, want 1", kept)
	}
}
