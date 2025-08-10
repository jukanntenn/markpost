package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB() {
	var err error

	// 根据配置选择数据库驱动
	switch config.Database.Type {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(config.Database.URL), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to open SQLite database: %v", err)
		}

		// 为 SQLite 设置 WAL 模式
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("failed to get sql.DB: %v", err)
		}

		if _, err = sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
			log.Fatalf("failed to set WAL mode: %v", err)
		}

	case "postgresql":
		db, err = gorm.Open(postgres.Open(config.Database.URL), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to open PostgreSQL database: %v", err)
		}

	default:
		log.Fatalf("unsupported database type: %s (supported types: sqlite, postgresql)", config.Database.Type)
	}

	if err = db.AutoMigrate(&User{}, &Post{}); err != nil {
		log.Fatalf("failed to auto migrate: %v", err)
	}

	// 迁移 User 表的 created_at 字段
	migrateUserCreatedAt()

	// 迁移 User 表的 password 字段
	migrateUserPassword()

	var count int64
	db.Model(&User{}).Where("username = ?", "markpost").Count(&count)
	if count > 0 {
		return
	}

	postKey, err := GenerateShortKey(8)
	if err != nil {
		log.Fatalf("failed to generate post key: %v", err)
	}

	// 哈希密码
	hashedPassword, err := HashPassword("markpost")
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	user := User{
		Username:  "markpost",
		Password:  hashedPassword,
		PostKey:   postKey,
		CreatedAt: time.Now().UTC(),
	}

	if err = db.Create(&user).Error; err != nil {
		log.Fatalf("failed to create markpost user: %v", err)
	}

	log.Printf("created markpost user with post_key: %s", postKey)
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

// GetUserByID 根据 ID 获取用户
func GetUserByID(id int) (*User, error) {
	var user User
	err := db.Where("id = ?", id).First(&user).Error
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
		Username:  githubUser.Login,
		PostKey:   postKey,
		GitHubID:  &githubUser.ID,
		CreatedAt: time.Now().UTC(), // 使用 UTC 时间
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

// FindUserByUsername 根据用户名查找用户
func FindUserByUsername(username string) (*User, error) {
	var user User
	err := db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &user, nil
}

// CreateUserWithPassword 创建带密码的用户
func CreateUserWithPassword(username, password string) (*User, error) {
	// 检查用户名是否已存在
	var existingUser User
	err := db.Where("username = ?", username).First(&existingUser).Error
	if err == nil {
		return nil, fmt.Errorf("username already exists")
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 哈希密码
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// 生成post key
	postKey, err := GenerateShortKey(8)
	if err != nil {
		return nil, err
	}

	user := User{
		Username:  username,
		Password:  hashedPassword,
		PostKey:   postKey,
		CreatedAt: time.Now().UTC(),
	}

	err = db.Create(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ValidateUserPassword 验证用户密码
func ValidateUserPassword(username, password string) (*User, error) {
	user, err := FindUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// 如果用户没有设置密码，则不允许密码登录
	if user.Password == "" {
		return nil, fmt.Errorf("user does not have password set")
	}

	// 验证密码
	if err := CheckPassword(password, user.Password); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
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
		CreatedAt: time.Now().UTC(), // 使用 UTC 时间
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
		CreatedAt: time.Now().UTC(), // 使用 UTC 时间
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

// migrateUserCreatedAt 迁移 User 表的 created_at 字段
func migrateUserCreatedAt() {
	// 检查列是否存在
	if !db.Migrator().HasColumn(&User{}, "created_at") {
		log.Println("created_at column not found in users table, adding it...")

		// 添加列
		if err := db.Migrator().AddColumn(&User{}, "created_at"); err != nil {
			log.Printf("failed to add created_at column: %v", err)
			return
		}
	}

	// 为已有记录设置默认值（使用当前UTC时间）
	now := time.Now().UTC()
	result := db.Model(&User{}).Where("created_at IS NULL OR created_at = '0001-01-01 00:00:00'").Update("created_at", now)
	if result.Error != nil {
		log.Printf("failed to update existing users with created_at: %v", result.Error)
	} else {
		log.Printf("updated %d existing users with created_at", result.RowsAffected)
	}
}

// migrateUserPassword 迁移 User 表的 password 字段
func migrateUserPassword() {
	// 检查列是否存在
	if !db.Migrator().HasColumn(&User{}, "password") {
		log.Println("password column not found in users table, adding it...")

		// 添加列
		if err := db.Migrator().AddColumn(&User{}, "password"); err != nil {
			log.Printf("failed to add password column: %v", err)
			return
		}
		log.Println("password column added successfully")
	}
}
