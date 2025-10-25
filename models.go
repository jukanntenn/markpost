package main

import (
	"time"
)

type User struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"column:password"`
	PostKey   string    `json:"post_key" gorm:"not null"`
	GitHubID  *int64    `json:"github_id" gorm:"column:github_id;uniqueIndex"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type Post struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title" gorm:"not null"`
	Body      string    `json:"body" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UserID    *int      `json:"user_id" gorm:"index;foreignKey:ID;references:users"`
}