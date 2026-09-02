package admin

import (
	"context"
	"testing"
	"time"

	"markpost/internal/domain/settings"
	"markpost/internal/infra"
)

// TestSetUserVIPMaterializesClassDefault covers grant-time materialization
// (MRFC 2026-08-31-per-user-history-retention-policy): a VIP still inheriting
// takes the class default in the grant; an explicit policy survives both grant
// and revoke.
func TestSetUserVIPMaterializesClassDefault(t *testing.T) {
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	settingsRepo := infra.NewSettingsRepository(db)
	svc := NewService(userRepo, nil, nil, nil, nil, nil)
	svc.SetUserMutator(userRepo)
	svc.SetSettingsStore(settingsRepo)

	forever := 0
	if err := settingsRepo.Set(t.Context(), settings.KeyVIPRetention, settings.SettingValue{Days: &forever}, 1); err != nil {
		t.Fatalf("seed class default: %v", err)
	}

	inheritor, _ := userRepo.Create(t.Context(), "mat1@ret.com", "mat1", "pass")
	explicit, _ := userRepo.Create(t.Context(), "mat2@ret.com", "mat2", "pass")
	thirty := 30
	if err := userRepo.SetUserRetention(t.Context(), explicit.ID, &thirty); err != nil {
		t.Fatalf("seed explicit policy: %v", err)
	}

	// Grant: the inheritor takes the class default (0 = forever); the
	// explicit 30-day policy is untouched.
	if err := svc.SetUserVIP(t.Context(), inheritor.ID, true); err != nil {
		t.Fatalf("grant inheritor: %v", err)
	}
	if err := svc.SetUserVIP(t.Context(), explicit.ID, true); err != nil {
		t.Fatalf("grant explicit: %v", err)
	}
	got, _ := userRepo.GetByID(t.Context(), inheritor.ID)
	if got.RetentionDays == nil || *got.RetentionDays != 0 {
		t.Errorf("inheritor retention = %v, want 0 (class default)", got.RetentionDays)
	}
	got, _ = userRepo.GetByID(t.Context(), explicit.ID)
	if got.RetentionDays == nil || *got.RetentionDays != 30 {
		t.Errorf("explicit retention = %v, want 30 (survives grant)", got.RetentionDays)
	}

	// Revoke: the materialized value stays — an honorific demotion must never
	// re-expose data to the global sweep.
	if err := svc.SetUserVIP(t.Context(), inheritor.ID, false); err != nil {
		t.Fatalf("revoke inheritor: %v", err)
	}
	got, _ = userRepo.GetByID(t.Context(), inheritor.ID)
	if got.RetentionDays == nil || *got.RetentionDays != 0 {
		t.Errorf("inheritor retention after revoke = %v, want 0 (kept)", got.RetentionDays)
	}
}

// TestRetentionImpactCounts seeds old posts/history under an inheriting and a
// forever user and verifies the preview counts for an explicit shortening.
func TestRetentionImpactCounts(t *testing.T) {
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	postRepo := infra.NewPostRepository(db)
	attemptRepo := infra.NewAttemptRepository(db)

	svc := NewService(userRepo, nil, nil, nil, nil, nil)
	svc.SetUserMutator(userRepo)
	svc.SetRetentionCounters(postRepo, historyCounter{attemptRepo}, 7, 168*time.Hour)

	inheritor, _ := userRepo.Create(t.Context(), "imp1@ret.com", "imp1", "pass")
	foreverUser, _ := userRepo.Create(t.Context(), "imp2@ret.com", "imp2", "pass")
	forever := 0
	if err := userRepo.SetUserRetention(t.Context(), foreverUser.ID, &forever); err != nil {
		t.Fatalf("seed forever: %v", err)
	}

	old := time.Now().AddDate(0, 0, -10)
	for _, uid := range []int{inheritor.ID, foreverUser.ID} {
		if _, err := postRepo.Create(t.Context(), "old post", "body", uid); err != nil {
			t.Fatalf("seed post: %v", err)
		}
	}
	// Backdate every post (autoCreateTime stamped now).
	if err := db.Exec("UPDATE posts SET created_at = ?", old).Error; err != nil {
		t.Fatalf("backdate posts: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO delivery_history (user_id, status, last_error, created_at) VALUES (?, 1, '', ?), (?, 1, '', ?)",
		inheritor.ID, old, foreverUser.ID, old,
	).Error; err != nil {
		t.Fatalf("seed history: %v", err)
	}

	// Candidate 5 days for the inheritor: one post + one history row would go.
	// (Impact counts rows past the candidate cutoff for the targeted users —
	// their CURRENT policy is irrelevant to the preview, which is exactly what
	// the confirm dialog needs: what the candidate would delete.)
	five := 5
	impact, err := svc.RetentionImpact(t.Context(), []int{inheritor.ID}, "", &five)
	if err != nil {
		t.Fatalf("RetentionImpact: %v", err)
	}
	if impact.PostsToDelete != 1 || impact.HistoryToDelete != 1 || impact.UsersAffected != 1 {
		t.Errorf("impact = %+v, want 1/1/1", impact)
	}

	// Candidate 3 days for both users: both rows are 10 days old.
	three := 3
	impact, err = svc.RetentionImpact(t.Context(), []int{inheritor.ID, foreverUser.ID}, "", &three)
	if err != nil {
		t.Fatalf("RetentionImpact both: %v", err)
	}
	if impact.PostsToDelete != 2 || impact.HistoryToDelete != 2 || impact.UsersAffected != 2 {
		t.Errorf("impact = %+v, want 2/2/2", impact)
	}

	// Candidate forever (0): nothing matches.
	zero := 0
	impact, err = svc.RetentionImpact(t.Context(), []int{inheritor.ID, foreverUser.ID}, "", &zero)
	if err != nil {
		t.Fatalf("RetentionImpact zero: %v", err)
	}
	if impact.PostsToDelete != 0 || impact.HistoryToDelete != 0 {
		t.Errorf("impact = %+v, want zero deletions", impact)
	}
}

// historyCounter adapts the attempt repository to the RetentionCounter port
// (mirrors main.go's historyExpiryCounter).
type historyCounter struct {
	repo interface {
		CountHistoryExpiringForUsers(ctx context.Context, userIDs []int, vipOnly bool, cutoff *time.Time) (int64, error)
	}
}

func (h historyCounter) CountExpiringForUsers(ctx context.Context, userIDs []int, vipOnly bool, cutoff *time.Time) (int64, error) {
	return h.repo.CountHistoryExpiringForUsers(ctx, userIDs, vipOnly, cutoff)
}
