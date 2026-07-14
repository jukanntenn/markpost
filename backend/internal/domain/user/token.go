// Package user defines the user domain model and repository interface.
package user

import "time"

// RefreshToken represents a refresh token entity. Records are created and
// soft-revoked (Revoked=true) rather than deleted, so a reused (already-revoked)
// token can be distinguished from a never-existed one — the signal for
// token-theft detection. See auth.md §2.2-2.3. Expired+revoked rows are pruned
// by periodic cleanup.
type RefreshToken struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int       `json:"user_id" gorm:"not null;index"`
	TokenHash string    `json:"-" gorm:"unique;not null"`
	Revoked   bool      `json:"-" gorm:"not null;default:false"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns the table name for RefreshToken.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired reports whether the refresh token has passed its expiration time.
func (t RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// TokenBlacklist represents a blacklisted token entity.
type TokenBlacklist struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TokenHash string    `json:"-" gorm:"unique;not null;index"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns the table name for TokenBlacklist.
func (TokenBlacklist) TableName() string {
	return "token_blacklist"
}
