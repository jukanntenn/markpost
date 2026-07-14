// Package user provides domain models for users.
package user

import (
	"context"
	"time"
)

// Repository defines the interface for user data access.
type Repository interface {
	GetByPostKey(ctx context.Context, postKey string) (*User, error)
	GetByID(ctx context.Context, id int) (*User, error)
	GetByGitHubID(ctx context.Context, githubID int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, email, username, password string) (*User, error)
	CreateFromGitHub(ctx context.Context, githubUser *GitHubUser) (*User, error)
	GetOrCreateFromGitHub(ctx context.Context, githubUser *GitHubUser) (*User, error)
	ValidatePassword(ctx context.Context, username, password string) (*User, error)
	SetPassword(ctx context.Context, userID int, password string) error
	SetRole(ctx context.Context, userID int, role Role) error
	DeleteByID(ctx context.Context, userID int) (int64, error)
	GetAll(ctx context.Context, offset, limit int) ([]User, error)
	Count(ctx context.Context) (int64, error)
	UpdateLastLoginAt(ctx context.Context, userID int, lastLoginAt time.Time) error
}

// TokenRepository defines the interface for token data access.
//
// Refresh tokens are soft-revoked (Revoked=true) rather than physically deleted
// so a reused token (revoked but resubmitted) can be detected as a theft
// signal. GetRefreshToken returns only non-revoked tokens; IsRefreshTokenRevoked
// checks the revoked set for reuse detection. See auth.md §2.2-2.4.
type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	// GetRefreshToken returns the active (non-revoked) refresh token for the
	// hash, or domain.ErrNotFound when absent or already revoked.
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	// IsRefreshTokenRevoked reports whether a refresh token with the hash has
	// been revoked (Revoked=true). Used by the reuse-detection path: a revoked
	// token resubmitted means theft.
	IsRefreshTokenRevoked(ctx context.Context, tokenHash string) (bool, error)
	// GetRevokedRefreshToken returns the revoked refresh token row for the hash
	// (for the reuse-detection path to read its UserID before revoking all of
	// the user's tokens), or domain.ErrNotFound when absent.
	GetRevokedRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	// RevokeRefreshToken soft-revokes a single refresh token (sets Revoked=true).
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	// RevokeAllByUserID soft-revokes every active refresh token for the user.
	// Called on logout and on detected token theft.
	RevokeAllByUserID(ctx context.Context, userID int) error

	StoreBlacklistedToken(ctx context.Context, tokenHash string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)
	CleanupExpiredTokens(ctx context.Context) error
}
