package infra

import (
	"context"
	"testing"
	"time"

	"markpost/internal/domain/user"
)

func TestTokenRepository_StoreAndGetRefreshToken(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	expiresAt := time.Now().Add(24 * time.Hour)
	err := repo.StoreRefreshToken(ctx, 1, "hash123", expiresAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	token, err := repo.GetRefreshToken(ctx, "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.UserID != 1 {
		t.Errorf("user_id = %d, want 1", token.UserID)
	}
	if token.TokenHash != "hash123" {
		t.Errorf("token_hash = %q, want %q", token.TokenHash, "hash123")
	}
}

func TestTokenRepository_GetRefreshToken_NotFound(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	_, err := repo.GetRefreshToken(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTokenRepository_DeleteRefreshToken(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	_ = repo.StoreRefreshToken(ctx, 1, "hash123", time.Now().Add(time.Hour))

	err := repo.DeleteRefreshToken(ctx, "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repo.GetRefreshToken(ctx, "hash123")
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestTokenRepository_DeleteRefreshTokensByUserID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	_ = repo.StoreRefreshToken(ctx, 1, "hash1", time.Now().Add(time.Hour))
	_ = repo.StoreRefreshToken(ctx, 1, "hash2", time.Now().Add(time.Hour))
	_ = repo.StoreRefreshToken(ctx, 2, "hash3", time.Now().Add(time.Hour))

	err := repo.DeleteRefreshTokensByUserID(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repo.GetRefreshToken(ctx, "hash1")
	if err == nil {
		t.Error("expected hash1 to be deleted")
	}
	_, err = repo.GetRefreshToken(ctx, "hash3")
	if err != nil {
		t.Errorf("expected hash3 to still exist: %v", err)
	}
}

func TestTokenRepository_Blacklist(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	t.Run("stores and checks blacklisted token", func(t *testing.T) {
		expiresAt := time.Now().Add(time.Hour)
		err := repo.StoreBlacklistedToken(ctx, "bl-hash", expiresAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		blacklisted, err := repo.IsTokenBlacklisted(ctx, "bl-hash")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !blacklisted {
			t.Error("expected token to be blacklisted")
		}
	})

	t.Run("returns false for non-blacklisted token", func(t *testing.T) {
		blacklisted, err := repo.IsTokenBlacklisted(ctx, "unknown-hash")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if blacklisted {
			t.Error("expected token to not be blacklisted")
		}
	})

	t.Run("expired blacklist entry returns false", func(t *testing.T) {
		expiresAt := time.Now().Add(-time.Hour)
		_ = repo.StoreBlacklistedToken(ctx, "expired-hash", expiresAt)

		blacklisted, err := repo.IsTokenBlacklisted(ctx, "expired-hash")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if blacklisted {
			t.Error("expected expired blacklisted token to return false")
		}
	})
}

func TestTokenRepository_CleanupExpiredTokens(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	// Store expired refresh token
	_ = repo.StoreRefreshToken(ctx, 1, "expired-refresh", time.Now().Add(-time.Hour))
	// Store valid refresh token
	_ = repo.StoreRefreshToken(ctx, 2, "valid-refresh", time.Now().Add(time.Hour))
	// Store expired blacklist entry
	_ = repo.StoreBlacklistedToken(ctx, "expired-bl", time.Now().Add(-time.Hour))

	err := repo.CleanupExpiredTokens(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expired refresh token should be gone
	_, err = repo.GetRefreshToken(ctx, "expired-refresh")
	if err == nil {
		t.Error("expected expired refresh token to be cleaned up")
	}

	// Valid refresh token should remain
	_, err = repo.GetRefreshToken(ctx, "valid-refresh")
	if err != nil {
		t.Errorf("expected valid refresh token to remain: %v", err)
	}
}

func TestRefreshToken_IsExpired(t *testing.T) {
	t.Run("expired token", func(t *testing.T) {
		token := user.RefreshToken{ExpiresAt: time.Now().Add(-time.Hour)}
		if !token.IsExpired() {
			t.Error("expected token to be expired")
		}
	})

	t.Run("valid token", func(t *testing.T) {
		token := user.RefreshToken{ExpiresAt: time.Now().Add(time.Hour)}
		if token.IsExpired() {
			t.Error("expected token to not be expired")
		}
	})
}
