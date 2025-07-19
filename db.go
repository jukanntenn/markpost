package main

import (
	"database/sql"
	"log"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("./data/db.sqlite3"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %v", err)
	}

	if _, err = sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		log.Fatalf("failed to set WAL mode: %v", err)
	}

	if err = db.AutoMigrate(&User{}, &Post{}); err != nil {
		log.Fatalf("failed to auto migrate: %v", err)
	}

	var count int64
	db.Model(&User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	postKey, err := GenerateShortKey(8)
	if err != nil {
		log.Fatalf("failed to generate post key: %v", err)
	}

	if err = db.Create(&User{Username: "admin", PostKey: postKey}).Error; err != nil {
		log.Fatalf("failed to create admin user: %v", err)
	}

	log.Printf("created admin user with post_key: %s", postKey)
}

// GetUserByPostKey 根据 post_key 获取用户
func GetUserByPostKey(postKey string) (*User, error) {
	var user User
	err := db.Where("post_key = ?", postKey).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &user, nil
}

// CreatePost 创建新的内容记录
func CreatePost(title, body string) (*Post, error) {
	// 生成 nano id
	id, err := gonanoid.New()
	if err != nil {
		return nil, err
	}

	post := Post{
		ID:        id,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}

	err = db.Create(&post).Error
	if err != nil {
		return nil, err
	}

	return &post, nil
}

// GetPostByID 根据 ID 获取内容记录
func GetPostByID(id string) (*Post, error) {
	var post Post
	err := db.Where("id = ?", id).First(&post).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &post, nil
}

// CloseDB 关闭数据库连接
func CloseDB() {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	if sqlDB != nil {
		sqlDB.Close()
	}
}
