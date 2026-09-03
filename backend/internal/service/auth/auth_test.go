package auth

import (
	"context"
	"testing"
	"time"

	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/service"
	"markpost/pkg/utils"

	"gorm.io/gorm"
)

func setupAuthService(t *testing.T) (*Service, user.Repository, user.TokenRepository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	tokenRepo := infra.NewTokenRepository(db)
	jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
	svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost", "testpassword")
	return svc, userRepo, tokenRepo
}

func TestService_LoginWithEmail(t *testing.T) {
	t.Run("returns tokens for valid credentials", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "correctpassword")

		u, tokens, err := svc.LoginWithEmail(ctx, "testuser", "correctpassword")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if u == nil {
			t.Fatal("expected user, got nil")
		}
		if tokens == nil {
			t.Fatal("expected tokens, got nil")
		}
		if tokens.AccessToken == "" {
			t.Error("expected access token, got empty")
		}
		if tokens.RefreshToken == "" {
			t.Error("expected refresh token, got empty")
		}
	})

	t.Run("returns error for invalid username", func(t *testing.T) {
		svc, _, _ := setupAuthService(t)
		ctx := context.Background()

		_, _, err := svc.LoginWithEmail(ctx, "nonexistent", "password")
		if err == nil {
			t.Fatal("expected error for invalid username")
		}
		se, ok := service.AsError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != ErrInvalidCredentials {
			t.Errorf("expected code %q, got %q", ErrInvalidCredentials.Value, se.Code.Value)
		}
	})

	t.Run("returns error for invalid password", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "correctpassword")

		_, _, err := svc.LoginWithEmail(ctx, "testuser", "wrongpassword")
		if err == nil {
			t.Fatal("expected error for invalid password")
		}
	})

	t.Run("returns error for disabled user", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)
		tokenRepo := infra.NewTokenRepository(db)
		jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
		svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost", "testpassword")
		ctx := context.Background()

		u, _ := userRepo.Create(ctx, "test@example.com", "testuser", "password")
		db.Model(&user.User{}).Where("id = ?", u.ID).Update("is_active", false)

		_, _, err := svc.LoginWithEmail(ctx, "testuser", "password")
		if err == nil {
			t.Fatal("expected error for disabled user")
		}
		se, ok := service.AsError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != ErrUserDisabled {
			t.Errorf("expected code %q, got %q", ErrUserDisabled.Value, se.Code.Value)
		}
	})
}

func TestService_QueryPostKey(t *testing.T) {
	t.Run("returns post key for valid user", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "password")

		postKey, _, err := svc.QueryPostKey(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if postKey == "" {
			t.Error("expected post key, got empty")
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		svc, _, _ := setupAuthService(t)
		ctx := context.Background()

		_, _, err := svc.QueryPostKey(ctx, 999)
		if err == nil {
			t.Fatal("expected error for non-existent user")
		}
	})
}

func TestService_RefreshToken(t *testing.T) {
	t.Run("returns new tokens for valid refresh token", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "password")

		_, tokens, _ := svc.LoginWithEmail(ctx, "testuser", "password")

		u, newTokens, err := svc.RefreshToken(ctx, tokens.RefreshToken)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if u == nil {
			t.Fatal("expected user, got nil")
		}
		if newTokens.AccessToken == "" {
			t.Error("expected new access token")
		}
		if newTokens.RefreshToken == "" {
			t.Error("expected new refresh token")
		}
	})

	t.Run("returns error for invalid refresh token", func(t *testing.T) {
		svc, _, _ := setupAuthService(t)
		ctx := context.Background()

		_, _, err := svc.RefreshToken(ctx, "invalid-token")
		if err == nil {
			t.Fatal("expected error for invalid refresh token")
		}
		se, ok := service.AsError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != ErrInvalidToken {
			t.Errorf("expected code %q, got %q", ErrInvalidToken.Value, se.Code.Value)
		}
	})

	t.Run("replay within the grace window rejects but keeps the family", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "password")
		_, pair1, _ := svc.LoginWithEmail(ctx, "testuser", "password")

		// The crash scenario: a tab rotated the pair server-side but died
		// before persisting the successor, so localStorage (and sibling tabs)
		// still hold the already-consumed token (auth.md §2.5).
		_, pair2, err := svc.RefreshToken(ctx, pair1.RefreshToken)
		if err != nil {
			t.Fatalf("first refresh should rotate: %v", err)
		}

		_, _, err = svc.RefreshToken(ctx, pair1.RefreshToken)
		if err == nil {
			t.Fatal("expected the stale replay to be rejected")
		}
		se, ok := service.AsError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != ErrInvalidToken {
			t.Errorf("expected code %q, got %q", ErrInvalidToken.Value, se.Code.Value)
		}

		// The family survives: the successor pair from the legitimate
		// rotation still refreshes.
		if _, _, err := svc.RefreshToken(ctx, pair2.RefreshToken); err != nil {
			t.Errorf("expected the successor pair to survive the in-window replay: %v", err)
		}
	})

	t.Run("replay just inside the window keeps the family, just outside revokes it", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)
		tokenRepo := infra.NewTokenRepository(db)
		jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
		svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost", "testpassword")
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "inside@example.com", "inside", "password")
		_, _ = userRepo.Create(ctx, "outside@example.com", "outside", "password")
		_, insideStale, _ := svc.LoginWithEmail(ctx, "inside", "password")
		_, outsideStale, _ := svc.LoginWithEmail(ctx, "outside", "password")
		_, insideLive, _ := svc.RefreshToken(ctx, insideStale.RefreshToken)
		_, outsideLive, _ := svc.RefreshToken(ctx, outsideStale.RefreshToken)

		// Backdate the two revocations around the 30 s boundary.
		if err := db.Model(&user.RefreshToken{}).Where("token_hash = ?", utils.HashToken(insideStale.RefreshToken)).
			Update("revoked_at", time.Now().Add(-29*time.Second)).Error; err != nil {
			t.Fatalf("backdate inside: %v", err)
		}
		if err := db.Model(&user.RefreshToken{}).Where("token_hash = ?", utils.HashToken(outsideStale.RefreshToken)).
			Update("revoked_at", time.Now().Add(-31*time.Second)).Error; err != nil {
			t.Fatalf("backdate outside: %v", err)
		}

		if _, _, err := svc.RefreshToken(ctx, insideStale.RefreshToken); err == nil {
			t.Fatal("expected the inside-window replay to be rejected")
		}
		if _, _, err := svc.RefreshToken(ctx, insideLive.RefreshToken); err != nil {
			t.Errorf("expected the inside-window family to survive: %v", err)
		}

		if _, _, err := svc.RefreshToken(ctx, outsideStale.RefreshToken); err == nil {
			t.Fatal("expected the outside-window replay to be rejected")
		}
		if _, _, err := svc.RefreshToken(ctx, outsideLive.RefreshToken); err == nil {
			t.Error("expected the outside-window replay to revoke the family")
		}
	})

	t.Run("replay of a legacy revoked row without a timestamp revokes the family", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)
		tokenRepo := infra.NewTokenRepository(db)
		jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
		svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost", "testpassword")
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "password")
		_, stale, _ := svc.LoginWithEmail(ctx, "testuser", "password")
		_, live, _ := svc.RefreshToken(ctx, stale.RefreshToken)

		// A row revoked before the revoked_at column existed carries NULL.
		if err := db.Model(&user.RefreshToken{}).Where("token_hash = ?", utils.HashToken(stale.RefreshToken)).
			Update("revoked_at", gorm.Expr("NULL")).Error; err != nil {
			t.Fatalf("clear revoked_at: %v", err)
		}

		if _, _, err := svc.RefreshToken(ctx, stale.RefreshToken); err == nil {
			t.Fatal("expected the legacy replay to be rejected")
		}
		if _, _, err := svc.RefreshToken(ctx, live.RefreshToken); err == nil {
			t.Error("expected the legacy replay to revoke the family")
		}
	})

	t.Run("returns error for expired refresh token", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)
		tokenRepo := infra.NewTokenRepository(db)
		jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, -time.Hour)
		svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost", "testpassword")
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "password")
		_, tokens, _ := svc.LoginWithEmail(ctx, "testuser", "password")

		_, _, err := svc.RefreshToken(ctx, tokens.RefreshToken)
		if err == nil {
			t.Fatal("expected error for expired refresh token")
		}
	})
}

func TestService_Logout(t *testing.T) {
	t.Run("blacklists access token", func(t *testing.T) {
		svc, userRepo, tokenRepo := setupAuthService(t)
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "password")
		_, tokens, _ := svc.LoginWithEmail(ctx, "testuser", "password")

		err := svc.Logout(ctx, tokens.AccessToken)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		blacklisted, _ := tokenRepo.IsTokenBlacklisted(ctx, utils.HashToken(tokens.AccessToken))
		if !blacklisted {
			t.Error("expected token to be blacklisted after logout")
		}
	})

	t.Run("handles empty token gracefully", func(t *testing.T) {
		svc, _, _ := setupAuthService(t)
		ctx := context.Background()

		err := svc.Logout(ctx, "")
		if err != nil {
			t.Fatalf("expected no error for empty token, got: %v", err)
		}
	})
}

func TestService_ChangePassword(t *testing.T) {
	t.Run("changes password successfully and returns a fresh token pair", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "oldpassword")

		pair, err := svc.ChangePassword(ctx, created.ID, "oldpassword", "newpassword")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Fatal("expected a fresh token pair (C2.2)")
		}

		// 新 token 对可用。
		if _, err := svc.jwt.ValidateAccess(pair.AccessToken); err != nil {
			t.Errorf("new access token invalid: %v", err)
		}

		_, _, err = svc.LoginWithEmail(ctx, "testuser", "newpassword")
		if err != nil {
			t.Fatalf("expected login with new password to work, got: %v", err)
		}
	})

	t.Run("bumps token_version and invalidates old tokens (C2.6)", func(t *testing.T) {
		svc, userRepo, tokenRepo := setupAuthService(t)
		ctx := context.Background()

		created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "oldpassword")
		_, pair, _ := svc.LoginWithEmail(ctx, "testuser", "oldpassword")

		u, _ := userRepo.GetByID(ctx, created.ID)
		if u.TokenVersion != 0 {
			t.Fatalf("expected initial token_version 0, got %d", u.TokenVersion)
		}
		claims, err := svc.jwt.ValidateAccess(pair.AccessToken)
		if err != nil || claims.TokenVersion != 0 {
			t.Fatalf("expected old token valid with tv=0, got err=%v", err)
		}

		newPair, err := svc.ChangePassword(ctx, created.ID, "oldpassword", "newpassword")
		if err != nil {
			t.Fatalf("change password failed: %v", err)
		}

		u, _ = userRepo.GetByID(ctx, created.ID)
		if u.TokenVersion != 1 {
			t.Errorf("expected token_version 1 after change, got %d", u.TokenVersion)
		}

		// 旧 access token 携带旧 tv —— 中间件层会拒绝。
		oldClaims, err := svc.jwt.ValidateAccess(pair.AccessToken)
		if err != nil {
			t.Fatalf("old token should still parse: %v", err)
		}
		if oldClaims.TokenVersion != 0 {
			t.Errorf("old token should carry tv=0, got %d", oldClaims.TokenVersion)
		}
		if claims2, _ := svc.jwt.ValidateAccess(newPair.AccessToken); claims2.TokenVersion != 1 {
			t.Errorf("new token should carry tv=1, got %d", claims2.TokenVersion)
		}

		// 旧 refresh token 已吊销（C2.2 revoke all）。
		oldHash := utils.HashToken(pair.RefreshToken)
		revoked, _ := tokenRepo.IsRefreshTokenRevoked(ctx, oldHash)
		if !revoked {
			t.Error("expected the pre-change refresh token to be revoked")
		}
	})

	t.Run("returns error for wrong current password", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "oldpassword")

		_, err := svc.ChangePassword(ctx, created.ID, "wrongpassword", "newpassword")
		if err == nil {
			t.Fatal("expected error for wrong current password")
		}
		se, ok := service.AsError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != ErrInvalidPassword {
			t.Errorf("expected code %q, got %q", ErrInvalidPassword.Value, se.Code.Value)
		}
	})

	t.Run("rejects passwords violating the policy (C2.3)", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "oldpassword")

		for _, short := range []string{"", "abc", "1234567"} {
			if _, err := svc.ChangePassword(ctx, created.ID, "oldpassword", short); err == nil {
				t.Errorf("expected policy error for %q", short)
			}
		}
		// 多字节场景：73 个中文字符（RuneCount 超 72）→ 拒绝。
		long := ""
		for range 73 {
			long += "界"
		}
		if _, err := svc.ChangePassword(ctx, created.ID, "oldpassword", long); err == nil {
			t.Error("expected policy error for 73 multi-byte runes")
		}
		// 字节超 72 但 RuneCount 未超（auth.md §4.3 双校验）。
		mixed := ""
		for range 25 { // 25 个 3 字节字符 = 75 字节
			mixed += "界"
		}
		if _, err := svc.ChangePassword(ctx, created.ID, "oldpassword", mixed); err == nil {
			t.Error("expected policy error for >72 bytes")
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		svc, _, _ := setupAuthService(t)
		ctx := context.Background()

		_, err := svc.ChangePassword(ctx, 999, "old", "new")
		if err == nil {
			t.Fatal("expected error for non-existent user")
		}
	})

	t.Run("sets password for OAuth user with empty current password", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		created, _ := userRepo.CreateFromGitHub(ctx, &user.GitHubUser{
			ID:    12345,
			Login: "ghuser",
			Email: "gh@example.com",
		})

		if _, err := svc.ChangePassword(ctx, created.ID, "", "newpassword"); err != nil {
			t.Fatalf("expected no error for passwordless user, got: %v", err)
		}

		_, _, err := svc.LoginWithEmail(ctx, "ghuser", "newpassword")
		if err != nil {
			t.Fatalf("expected login with new password to work, got: %v", err)
		}
	})
}

func TestService_InitializeFirstAdmin(t *testing.T) {
	t.Run("promotes user to admin", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "password")

		err := svc.InitializeFirstAdmin(ctx, "testuser")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		u, _ := userRepo.GetByUsername(ctx, "testuser")
		if !u.IsAdmin() {
			t.Error("expected user to be admin after InitializeFirstAdmin")
		}
	})

	t.Run("creates user if non-existent", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		err := svc.InitializeFirstAdmin(ctx, "newadmin")
		if err != nil {
			t.Fatalf("expected creation, got: %v", err)
		}
		u, _ := userRepo.GetByUsername(ctx, "newadmin")
		if u == nil || !u.IsAdmin() {
			t.Fatal("expected new admin created and promoted")
		}
	})

	t.Run("no-op if already admin", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		u, _ := userRepo.Create(ctx, "test@example.com", "testuser", "password")
		_ = userRepo.SetRole(ctx, u.ID, user.RoleAdmin)

		err := svc.InitializeFirstAdmin(ctx, "testuser")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})
}
