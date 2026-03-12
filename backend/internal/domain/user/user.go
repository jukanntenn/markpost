package user

import (
	"errors"
	"time"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

type User struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	PostKey   string    `json:"post_key" gorm:"unique;not null"`
	GitHubID  *int64    `json:"github_id" gorm:"unique;column:github_id"`
	Role      Role      `json:"role" gorm:"not null;default:'user'"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsRegularUser() bool {
	return u.Role == RoleUser
}

var ErrNotFound = errors.New("record not found")
