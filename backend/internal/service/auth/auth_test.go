package auth

import (
	"context"
	"testing"
	"time"

	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/service"
	"markpost/pkg/utils"
)

func setupAuthService(t *testing.T) (*Service, user.Repository, user.TokenRepository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	tokenRepo := infra.NewTokenRepository(db)
	jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
	svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost")
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
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInvalidCredentials {
			t.Errorf("expected code %q, got %q", service.ErrInvalidCredentials, se.Code)
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
		svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost")
		ctx := context.Background()

		u, _ := userRepo.Create(ctx, "test@example.com", "testuser", "password")
		db.Model(&user.User{}).Where("id = ?", u.ID).Update("is_active", false)

		_, _, err := svc.LoginWithEmail(ctx, "testuser", "password")
		if err == nil {
			t.Fatal("expected error for disabled user")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrUserDisabled {
			t.Errorf("expected code %q, got %q", service.ErrUserDisabled, se.Code)
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
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInvalidToken {
			t.Errorf("expected code %q, got %q", service.ErrInvalidToken, se.Code)
		}
	})

	t.Run("returns error for expired refresh token", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)
		tokenRepo := infra.NewTokenRepository(db)
		jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, -time.Hour)
		svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost")
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
	t.Run("changes password successfully", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "oldpassword")

		err := svc.ChangePassword(ctx, created.ID, "oldpassword", "newpassword")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		_, _, err = svc.LoginWithEmail(ctx, "testuser", "newpassword")
		if err != nil {
			t.Fatalf("expected login with new password to work, got: %v", err)
		}
	})

	t.Run("returns error for wrong current password", func(t *testing.T) {
		svc, userRepo, _ := setupAuthService(t)
		ctx := context.Background()

		created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "oldpassword")

		err := svc.ChangePassword(ctx, created.ID, "wrongpassword", "newpassword")
		if err == nil {
			t.Fatal("expected error for wrong current password")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInvalidPassword {
			t.Errorf("expected code %q, got %q", service.ErrInvalidPassword, se.Code)
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		svc, _, _ := setupAuthService(t)
		ctx := context.Background()

		err := svc.ChangePassword(ctx, 999, "old", "new")
		if err == nil {
			t.Fatal("expected error for non-existent user")
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

	t.Run("returns error for non-existent user", func(t *testing.T) {
		svc, _, _ := setupAuthService(t)
		ctx := context.Background()

		err := svc.InitializeFirstAdmin(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent user")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrNotFound {
			t.Errorf("expected code %q, got %q", service.ErrNotFound, se.Code)
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
