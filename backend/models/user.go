package models

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
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
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
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
