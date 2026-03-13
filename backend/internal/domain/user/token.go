package user

import "time"

type RefreshToken struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int       `json:"user_id" gorm:"not null;index"`
	TokenHash string    `json:"-" gorm:"unique;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

type TokenBlacklist struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TokenHash string    `json:"-" gorm:"unique;not null;index"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (TokenBlacklist) TableName() string {
	return "token_blacklist"
}
