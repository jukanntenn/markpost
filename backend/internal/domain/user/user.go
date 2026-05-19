// Package user defines the user domain model and repository interface.
package user

import (
	"time"
)

// Role represents a user role.
type Role string

const (
	// RoleAdmin represents an admin user role.
	RoleAdmin Role = "admin"
	// RoleUser represents a regular user role.
	RoleUser Role = "user"
)

// GitHubUser represents a GitHub user.
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
}

// User represents a user entity.
type User struct {
	ID              int        `json:"id" gorm:"primaryKey;autoIncrement"`
	Email           string     `json:"email" gorm:"unique;not null"`
	Username        string     `json:"username" gorm:"unique;not null"`
	Name            string     `json:"name"`
	Password        string     `json:"-" gorm:"column:password_hash"`
	AvatarURL       *string    `json:"avatar_url"`
	PostKey         string     `json:"post_key" gorm:"unique;not null"`
	GitHubID        *int64     `json:"github_id" gorm:"unique;column:github_id"`
	Role            Role       `json:"role" gorm:"not null;default:'user'"`
	IsActive        bool       `json:"is_active" gorm:"default:true"`
	IsEmailVerified bool       `json:"is_email_verified" gorm:"default:false"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// IsAdmin returns true if the user has the admin role.
func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
