package auth

import (
	"context"
	"testing"
	"time"

	"markpost/internal/service"
	"markpost/pkg/utils"
)

// C2.1 层 B：账号级登录失败锁定。
func TestService_LoginAttemptLockout(t *testing.T) {
	svc, userRepo, tokenRepo := setupAuthService(t)
	ctx := context.Background()

	// 使用足量密码满足策略
	_, _ = userRepo.Create(ctx, "lock@example.com", "lockuser", "correctpass")

	// 5 次连续失败 → 第 5 次即锁定
	for i := 1; i <= 5; i++ {
		_, _, err := svc.LoginWithEmail(ctx, "lockuser", "wrongpass")
		if err == nil {
			t.Fatalf("attempt %d: expected error", i)
		}
		if i < 5 {
			se, _ := service.AsError(err)
			if se == nil || se.Code != ErrInvalidCredentials {
				t.Fatalf("attempt %d: expected invalid_credentials, got %v", i, se)
			}
		}
	}

	// 第 5 次与锁定期间均返回 account_locked
	_, _, err := svc.LoginWithEmail(ctx, "lockuser", "correctpass")
	if err == nil {
		t.Fatal("expected account_locked for correct password during lock")
	}
	se, ok := service.AsError(err)
	if !ok {
		t.Fatalf("expected service error, got %v", err)
	}
	if se.Code != ErrAccountLocked {
		t.Fatalf("expected code %q, got %q", ErrAccountLocked.Value, se.Code.Value)
	}
	if se.Data == nil {
		t.Fatal("expected lock error to carry data (Minutes/RetryAfter)")
	}
	if minutes, ok := se.Data["Minutes"].(int); !ok || minutes <= 0 {
		t.Errorf("expected positive Minutes in error data, got %v", se.Data["Minutes"])
	}
	if retry, ok := se.Data["RetryAfter"].(int); !ok || retry <= 0 {
		t.Errorf("expected positive RetryAfter in error data, got %v", se.Data["RetryAfter"])
	}

	// 锁定期间计数不继续加（K.7 C2-1）：再多错 10 次后，剩余时间不延长。
	before, _ := se.Data["RetryAfter"].(int)
	for i := 0; i < 10; i++ {
		_, _, err = svc.LoginWithEmail(ctx, "lockuser", "wrongpass")
		se2, _ := service.AsError(err)
		if se2 == nil || se2.Code != ErrAccountLocked {
			t.Fatalf("expected account_locked, got %v", err)
		}
	}
	after, _ := se.Data["RetryAfter"].(int)
	if after-before > 0 {
		t.Errorf("lock duration should not extend during lock (before=%d after=%d)", before, after)
	}

	// 解锁后（直接改内部状态模拟到期）登录成功清零计数。
	svc.attempts.Reset("lockuser")
	_, pair, err := svc.LoginWithEmail(ctx, "lockuser", "correctpass")
	if err != nil {
		t.Fatalf("expected successful login after lock expiry, got %v", err)
	}
	if pair == nil {
		t.Fatal("expected token pair")
	}

	// 成功后再失败不立即锁定（计数已清零）。
	_, _, err = svc.LoginWithEmail(ctx, "lockuser", "wrongpass")
	if se, _ := service.AsError(err); se == nil || se.Code != ErrInvalidCredentials {
		t.Fatalf("expected invalid_credentials after counter reset, got %v", err)
	}

	_ = tokenRepo
}

// C2.5：post key 轮换 —— 旧 key 立即失效，新 key 可反查。
func TestService_RotatePostKey(t *testing.T) {
	svc, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	created, _ := userRepo.Create(ctx, "rotate@example.com", "rotateuser", "correctpass")
	oldKey, createdAt, err := svc.QueryPostKey(ctx, created.ID)
	if err != nil {
		t.Fatalf("query post key: %v", err)
	}
	if createdAt.IsZero() {
		t.Error("expected createdAt")
	}

	newKey, err := svc.RotatePostKey(ctx, created.ID)
	if err != nil {
		t.Fatalf("rotate post key: %v", err)
	}
	if newKey == "" || newKey == oldKey {
		t.Errorf("expected a new key, old=%q new=%q", oldKey, newKey)
	}

	u, err := userRepo.GetByPostKey(ctx, newKey)
	if err != nil {
		t.Fatalf("new key should resolve to the user: %v", err)
	}
	if u.ID != created.ID {
		t.Errorf("expected user %d, got %d", created.ID, u.ID)
	}
	if _, err := userRepo.GetByPostKey(ctx, oldKey); err == nil {
		t.Error("old key should stop resolving immediately")
	}
}

// I.12：用户查看/吊销自己的会话。
func TestService_Sessions(t *testing.T) {
	svc, userRepo, tokenRepo := setupAuthService(t)
	ctx := context.Background()

	created, _ := userRepo.Create(ctx, "sess@example.com", "sessuser", "correctpass")

	// 登录两次 → 两个活跃 refresh token。
	_, pair1, _ := svc.LoginWithEmail(ctx, "sessuser", "correctpass")
	_, pair2, _ := svc.LoginWithEmail(ctx, "sessuser", "correctpass")

	sessions, err := svc.ListSessions(ctx, created.ID)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	for _, s := range sessions {
		if s.Revoked {
			t.Error("expected active sessions")
		}
	}

	// 吊销第一条。
	targetID := int(sessions[0].ID)
	if err := svc.RevokeSession(ctx, created.ID, targetID); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	sessions, _ = svc.ListSessions(ctx, created.ID)
	revokedCount := 0
	for _, s := range sessions {
		if s.Revoked {
			revokedCount++
		}
	}
	if revokedCount != 1 {
		t.Errorf("expected exactly 1 revoked session, got %d", revokedCount)
	}

	// 吊销他人会话 → not_found（I.4 对象级权限）。
	other, _ := userRepo.Create(ctx, "other@example.com", "otheruser", "correctpass")
	_, pairOther, _ := svc.LoginWithEmail(ctx, "otheruser", "correctpass")
	_ = pairOther
	if err := svc.RevokeSession(ctx, other.ID, targetID); err == nil {
		t.Error("expected error revoking another user's session")
	}

	// 全部下线（保留当前 access token 语义：仅吊销 refresh token）。
	if err := svc.RevokeAllSessions(ctx, created.ID); err != nil {
		t.Fatalf("revoke all sessions: %v", err)
	}
	sessions, _ = svc.ListSessions(ctx, created.ID)
	for _, s := range sessions {
		if !s.Revoked {
			t.Errorf("expected session %d revoked after revoke-all", s.ID)
		}
	}

	// 吊销后旧 refresh token 无法再换新 access（轮转已吊销）。
	hash := utils.HashToken(pair1.RefreshToken)
	revoked, _ := tokenRepo.IsRefreshTokenRevoked(ctx, hash)
	if !revoked {
		t.Error("expected pair1 refresh token revoked")
	}
	if _, _, err := svc.RefreshToken(ctx, pair2.RefreshToken); err == nil {
		t.Error("expected refresh to fail after revoke-all")
	}
}

// 锁定状态本身的时间逻辑：CheckLocked/RecordFailure 的纯单元行为。
func TestLoginAttemptTracker(t *testing.T) {
	tr := NewLoginAttemptTracker()
	username := "tracker-user"

	if locked, _ := tr.CheckLocked(username); locked {
		t.Fatal("expected not locked initially")
	}
	for i := 0; i < 4; i++ {
		if locked, _ := tr.RecordFailure(username); locked {
			t.Fatalf("attempt %d should not lock yet", i+1)
		}
	}
	locked, remaining := tr.RecordFailure(username)
	if !locked {
		t.Fatal("expected lock on 5th failure")
	}
	if remaining <= 0 || remaining > lockDuration {
		t.Fatalf("unexpected remaining %v", remaining)
	}

	// 锁定期间重复失败不延长。
	locked, r2 := tr.RecordFailure(username)
	if !locked {
		t.Fatal("expected still locked")
	}
	if r2 > remaining+time.Second {
		t.Fatalf("lock duration extended during lock: %v -> %v", remaining, r2)
	}

	tr.Reset(username)
	if locked, _ := tr.CheckLocked(username); locked {
		t.Fatal("expected unlocked after reset")
	}
}
