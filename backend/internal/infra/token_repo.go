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

// GetRefreshToken retrieves a refresh token by its hash.
func (r *TokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*user.RefreshToken, error) {
	t, err := findFirst[user.RefreshToken](ctx, r.db.Where("token_hash = ?", tokenHash), domain.ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("GetRefreshToken: %w", err)
	}
	return t, nil
}

// DeleteRefreshToken deletes a refresh token by its hash.
func (r *TokenRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := deleteWhere[user.RefreshToken](ctx, r.db.Where("token_hash = ?", tokenHash))
	return err
}

// DeleteRefreshTokensByUserID deletes all refresh tokens for a user.
func (r *TokenRepository) DeleteRefreshTokensByUserID(ctx context.Context, userID int) error {
	_, err := deleteWhere[user.RefreshToken](ctx, r.db.Where("user_id = ?", userID))
	return err
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
