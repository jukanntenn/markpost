package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
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

// FindUserByGitHubID 根据 GitHub ID 查找用户
func FindUserByGitHubID(githubID int64) (*User, error) {
	var user User
	err := db.Where("github_id = ?", githubID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &user, nil
}

// CreateUserFromGitHub 从 GitHub 用户信息创建新用户
func CreateUserFromGitHub(githubUser *GitHubUser) (*User, error) {
	postKey, err := GenerateShortKey(8)
	if err != nil {
		return nil, err
	}

	user := User{
		Username: githubUser.Login,
		PostKey:  postKey,
		GitHubID: &githubUser.ID,
	}

	err = db.Create(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// FindOrCreateUser 查找或创建用户
func FindOrCreateUser(githubUser *GitHubUser) (*User, error) {
	// 首先尝试根据 GitHub ID 查找用户
	user, err := FindUserByGitHubID(githubUser.ID)
	if err == nil {
		// 用户存在，直接返回
		return user, nil
	}

	if err != sql.ErrNoRows {
		// 发生其他错误
		return nil, err
	}

	// 用户不存在，创建新用户
	return CreateUserFromGitHub(githubUser)
}

// CreatePost 创建新的内容记录
func CreatePost(title, body string, userID ...int) (*Post, error) {
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

	// 如果提供了userID参数，则设置UserID字段
	if len(userID) > 0 {
		post.UserID = &userID[0]
	}

	err = db.Create(&post).Error
	if err != nil {
		// 检查是否为外键约束违反错误
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
		return nil, err
	}

	return &post, nil
}

// CreatePostWithUser 创建关联用户的post
func CreatePostWithUser(title, body string, userID int) (*Post, error) {
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
		UserID:    &userID,
	}

	err = db.Create(&post).Error
	if err != nil {
		// 检查是否为外键约束违反错误
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
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

// GetPostsByUserID 获取指定用户创建的所有post
func GetPostsByUserID(userID int) ([]Post, error) {
	var posts []Post
	err := db.Where("user_id = ?", userID).Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
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
