// Package infra provides infrastructure layer implementations.
package infra

import (
	"context"
	"fmt"
	"time"

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
	return r.db.WithContext(ctx).Create(&token).Error
}

// GetRefreshToken retrieves a refresh token by its hash.
func (r *TokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*user.RefreshToken, error) {
	t, err := findFirst[user.RefreshToken](ctx, r.db.Where("token_hash = ?", tokenHash), user.ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("GetRefreshToken: %w", err)
	}
	return t, nil
}

// DeleteRefreshToken deletes a refresh token by its hash.
func (r *TokenRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&user.RefreshToken{}).Error
}

// DeleteRefreshTokensByUserID deletes all refresh tokens for a user.
func (r *TokenRepository) DeleteRefreshTokensByUserID(ctx context.Context, userID int) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&user.RefreshToken{}).Error
}

// StoreBlacklistedToken adds a token to the blacklist.
func (r *TokenRepository) StoreBlacklistedToken(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	blacklist := user.TokenBlacklist{
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&blacklist).Error
}

// IsTokenBlacklisted checks if a token is blacklisted.
func (r *TokenRepository) IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&user.TokenBlacklist{}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("IsTokenBlacklisted: %w", err)
	}
	return count > 0, nil
}

// CleanupExpiredTokens removes expired tokens from the database.
func (r *TokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&user.RefreshToken{}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&user.TokenBlacklist{}).Error
}
