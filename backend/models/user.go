package models

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
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

func (model *User) Create(database *Database) error {
	db := database.DB()

	if err := db.Create(model).Error; err != nil {
		return fmt.Errorf("User.Create: %w", err)
	}

	return nil
}

func (model *User) Update(database *Database) error {
	db := database.DB()

	if err := db.Model(&model).Updates(model).Error; err != nil {
		return fmt.Errorf("User.Update: %w", err)
	}

	return nil
}

func GetUser(database *Database, query map[string]any) (*User, error) {
	db := database.DB()

	var user User
	err := db.Take(&user, query).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetUser: %w", err)
	}

	return &user, nil
}

func UserExists(database *Database, query map[string]any) (bool, error) {
	db := database.DB()

	var count int64
	if err := db.Model(&User{}).Where(query).Count(&count).Error; err != nil {
		return false, fmt.Errorf("UserExists: %w", err)
	}

	return count > 0, nil
}

func GetUsers(database *Database, offset, limit int) ([]User, error) {
	db := database.DB()

	var users []User
	err := db.Order("id asc").Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("GetUsers: %w", err)
	}

	return users, nil
}

func CountUsers(database *Database) (int64, error) {
	db := database.DB()

	var count int64
	if err := db.Model(&User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("CountUsers: %w", err)
	}

	return count, nil
}
