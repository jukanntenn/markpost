package main

import (
	"time"
)

type User struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	PostKey   string    `json:"post_key" gorm:"unique;not null"`
	GitHubID  *int64    `json:"github_id" gorm:"unique"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Post struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	QID       string    `json:"qid" gorm:"unique;not null;column:qid"`
	Title     string    `json:"title" gorm:"not null"`
	Body      string    `json:"body" gorm:"not null;type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	UserID    int       `json:"user_id" gorm:"index;not null"`
	User      User      `json:"user" gorm:"constraint:OnDelete:CASCADE"`
}
