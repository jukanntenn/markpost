package infra

import (
	"context"
	"fmt"
	"time"

	"markpost/internal/domain"
	"markpost/internal/domain/user"

	"gorm.io/gorm"
)

// TokenRepository provides token data access operations.
type TokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository creates a new TokenRepository instance.
func NewTokenRepository(db *gorm.DB) user.TokenRepository {
	return &TokenRepository{db: db}
}

// StoreRefreshToken stores a refresh token for a user.
func (r *TokenRepository) StoreRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	token := user.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&token).Error; err != nil {
		return fmt.Errorf("StoreRefreshToken: %w", err)
	}
	return nil
}

// GetRefreshToken retrieves the active (non-revoked) refresh token by its
// hash, or domain.ErrNotFound when absent or already revoked. Filtering on
// revoked=false is what lets the reuse-detection path distinguish "never
// existed" (ErrNotFound here) from "revoked then resubmitted" (IsRefreshTokenRevoked).
func (r *TokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*user.RefreshToken, error) {
	t, err := findFirst[user.RefreshToken](
		ctx,
		r.db.Where("token_hash = ? AND revoked = ?", tokenHash, false),
		domain.ErrNotFound,
	)
	if err != nil {
		return nil, fmt.Errorf("GetRefreshToken: %w", err)
	}
	return t, nil
}

// IsRefreshTokenRevoked reports whether the hash matches a revoked refresh
// token — the token-theft signal when a refresh request resubmits a revoked
// token.
func (r *TokenRepository) IsRefreshTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	count, err := countQuery(ctx,
		r.db.Model(&user.RefreshToken{}).Where("token_hash = ? AND revoked = ?", tokenHash, true),
		"IsRefreshTokenRevoked",
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetRevokedRefreshToken returns the revoked refresh token row for the hash
// (for the reuse-detection path to read its UserID), or domain.ErrNotFound.
func (r *TokenRepository) GetRevokedRefreshToken(ctx context.Context, tokenHash string) (*user.RefreshToken, error) {
	t, err := findFirst[user.RefreshToken](
		ctx,
		r.db.Where("token_hash = ? AND revoked = ?", tokenHash, true),
		domain.ErrNotFound,
	)
	if err != nil {
		return nil, fmt.Errorf("GetRevokedRefreshToken: %w", err)
	}
	return t, nil
}

// RevokeRefreshToken soft-revokes a single refresh token (sets revoked=true),
// preserving the row for reuse detection instead of deleting it.
func (r *TokenRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	result := r.db.WithContext(ctx).
		Model(&user.RefreshToken{}).
		Where("token_hash = ? AND revoked = ?", tokenHash, false).
		Update("revoked", true)
	if result.Error != nil {
		return fmt.Errorf("RevokeRefreshToken: %w", result.Error)
	}
	return nil
}

// RevokeAllByUserID soft-revokes every active refresh token for a user — used
// on logout and on detected token theft (auth.md §2.3, §5).
func (r *TokenRepository) RevokeAllByUserID(ctx context.Context, userID int) error {
	result := r.db.WithContext(ctx).
		Model(&user.RefreshToken{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Update("revoked", true)
	if result.Error != nil {
		return fmt.Errorf("RevokeAllByUserID: %w", result.Error)
	}
	return nil
}

// StoreBlacklistedToken adds a token to the blacklist.
func (r *TokenRepository) StoreBlacklistedToken(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	blacklist := user.TokenBlacklist{
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&blacklist).Error; err != nil {
		return fmt.Errorf("StoreBlacklistedToken: %w", err)
	}
	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted.
func (r *TokenRepository) IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	count, err := countQuery(ctx,
		r.db.Model(&user.TokenBlacklist{}).
			Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()),
		"IsTokenBlacklisted",
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CleanupExpiredTokens removes expired tokens from the database.
func (r *TokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at < ?", now).Delete(&user.RefreshToken{}).Error; err != nil {
			return fmt.Errorf("CleanupExpiredTokens/refresh: %w", err)
		}
		if err := tx.Where("expires_at < ?", now).Delete(&user.TokenBlacklist{}).Error; err != nil {
			return fmt.Errorf("CleanupExpiredTokens/blacklist: %w", err)
		}
		return nil
	})
}
