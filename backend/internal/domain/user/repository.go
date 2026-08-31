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
	// BumpTokenVersion increments token_version — the single primitive behind
	// instant invalidation of all of a user's tokens (C2.6).
	BumpTokenVersion(ctx context.Context, userID int) error
	// UpdatePostKey rotates a user's post key (C2.5).
	UpdatePostKey(ctx context.Context, userID int, postKey string) error
	// RotatePostKey generates a fresh unique post key and stores it (C2.5).
	RotatePostKey(ctx context.Context, userID int) (string, error)
	SetRole(ctx context.Context, userID int, role Role) error
	SetActive(ctx context.Context, userID int, active bool) error
	// SetUserVIP writes the durable VIP honorific (MRFC 2026-08-23-user-vip-flag).
	// retentionIfUnset materializes the VIP-class retention default onto the
	// user in the same statement, but only while the user still inherits
	// (NULL) — an explicit policy survives grant and revoke alike (MRFC
	// 2026-08-31-per-user-history-retention-policy); nil never writes.
	SetUserVIP(ctx context.Context, userID int, vip bool, retentionIfUnset *int) error
	// SetUserRetention writes one user's retention policy (nil = inherit).
	SetUserRetention(ctx context.Context, userID int, days *int) error
	// SetUserRetentionBatch writes the policy onto explicit user ids,
	// returning the affected row count.
	SetUserRetentionBatch(ctx context.Context, userIDs []int, days *int) (int64, error)
	// SetVIPUsersRetention writes the policy onto every VIP user (bulk
	// realignment), returning the affected row count.
	SetVIPUsersRetention(ctx context.Context, days *int) (int64, error)
	// CountVIP counts users carrying the VIP flag (bulk preview).
	CountVIP(ctx context.Context) (int64, error)
	DeleteByID(ctx context.Context, userID int) (int64, error)
	GetAll(ctx context.Context, offset, limit int) ([]User, error)
	// Search returns users whose username matches the LIKE pattern (admin user
	// list search, D3.1), ordered by id.
	Search(ctx context.Context, search string, offset, limit int) ([]User, error)
	// CountSearch returns the total users matching the search pattern.
	CountSearch(ctx context.Context, search string) (int64, error)
	Count(ctx context.Context) (int64, error)
	// CountByRole counts users with the given role (last-admin guard, K.7 D3-3).
	CountByRole(ctx context.Context, role Role) (int64, error)
	// CountBanned counts disabled users (admin 需要关注, D2.1).
	CountBanned(ctx context.Context) (int64, error)
	// CountSince counts users created at or after since (stats week delta, D2.4).
	CountSince(ctx context.Context, since time.Time) (int64, error)
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
	// RevokeRefreshTokenByID soft-revokes a single active refresh token, scoped
	// to the user when userID > 0 (I.12); userID == 0 revokes regardless of
	// owner (D3.2 admin). Returns domain.ErrNotFound when no active row matches.
	RevokeRefreshTokenByID(ctx context.Context, tokenID, userID int) error
	// GetRefreshTokenByID returns a refresh token row by primary key, or
	// domain.ErrNotFound (admin single-session revoke owner resolution).
	GetRefreshTokenByID(ctx context.Context, tokenID int) (*RefreshToken, error)
	// RevokeAllByUserID soft-revokes every active refresh token for the user.
	// Called on logout and on detected token theft.
	RevokeAllByUserID(ctx context.Context, userID int) error
	// ListByUserID returns all refresh tokens for a user (for session management).
	ListByUserID(ctx context.Context, userID int) ([]RefreshToken, error)

	StoreBlacklistedToken(ctx context.Context, tokenHash string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)
	CleanupExpiredTokens(ctx context.Context) error
}
