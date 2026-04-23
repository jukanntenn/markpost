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
type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	DeleteRefreshTokensByUserID(ctx context.Context, userID int) error

	StoreBlacklistedToken(ctx context.Context, tokenHash string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)
	CleanupExpiredTokens(ctx context.Context) error
}
