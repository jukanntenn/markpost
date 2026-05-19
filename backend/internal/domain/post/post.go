// Package post provides domain models for posts.
package post

import (
	"time"

	"markpost/internal/domain/user"
)

// Post represents a user post.
type Post struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	QID       string    `json:"qid" gorm:"unique;not null;column:qid"`
	Title     string    `json:"title" gorm:"not null"`
	Body      string    `json:"body" gorm:"not null;type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	UserID    int       `json:"user_id" gorm:"index;not null;column:user_id"`
	User      user.User `json:"user" gorm:"constraint:OnDelete:CASCADE"`
}
