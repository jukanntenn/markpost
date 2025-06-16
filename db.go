package main

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// User 用户表结构
type User struct {
	ID      int    `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	PostKey  string `json:"post_key" db:"post_key"`
}

// Post 内容表结构（原文档中的 xxx 表）
type Post struct {
	ID        string    `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	Body      string    `json:"body" db:"body"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

var db *sql.DB

// InitDB 初始化数据库
func InitDB() {
	var err error
	db, err = sql.Open("sqlite", "./data/db.sqlite3")
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	// 开启 WAL 模式以提高并发能力
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Fatalf("设置 WAL 模式失败: %v", err)
	}

	// 创建用户表
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		post_key TEXT NOT NULL
	);`

	_, err = db.Exec(createUsersTable)
	if err != nil {
		log.Fatalf("创建用户表失败: %v", err)
	}

	// 创建内容表
	createPostsTable := `
	CREATE TABLE IF NOT EXISTS posts (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createPostsTable)
	if err != nil {
		log.Fatalf("创建内容表失败: %v", err)
	}

	// 检查是否存在 admin 用户，如果不存在则创建
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", "admin").Scan(&count)
	if err != nil {
		log.Fatalf("查询admin用户失败: %v", err)
	}

	if count == 0 {
		// 生成随机 post_key
		postKey, err := GenerateShortKey(8)
		if err != nil {
			log.Fatalf("生成 post_key 失败: %v", err)
		}

		_, err = db.Exec("INSERT INTO users (username, post_key) VALUES (?, ?)", "admin", postKey)
		if err != nil {
			log.Fatalf("创建admin用户失败: %v", err)
		}

		log.Printf("已创建 admin 用户，post_key: %s", postKey)
	}

	log.Println("数据库初始化完成")
}

// GetUserByPostKey 根据 post_key 获取用户
func GetUserByPostKey(postKey string) (*User, error) {
	user := &User{}
	err := db.QueryRow("SELECT id, username, post_key FROM users WHERE post_key = ?", postKey).
		Scan(&user.ID, &user.Username, &user.PostKey)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreatePost 创建新的内容记录
func CreatePost(title, body string) (*Post, error) {
	// 生成 nano id
	id, err := gonanoid.New()
	if err != nil {
		return nil, err
	}

	post := &Post{
		ID:        id,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}

	_, err = db.Exec("INSERT INTO posts (id, title, body, created_at) VALUES (?, ?, ?, ?)",
		post.ID, post.Title, post.Body, post.CreatedAt)
	if err != nil {
		return nil, err
	}

	return post, nil
}

// GetPostByID 根据 ID 获取内容记录
func GetPostByID(id string) (*Post, error) {
	post := &Post{}
	err := db.QueryRow("SELECT id, title, body, created_at FROM posts WHERE id = ?", id).
		Scan(&post.ID, &post.Title, &post.Body, &post.CreatedAt)
	if err != nil {
		return nil, err
	}
	return post, nil
}

// CloseDB 关闭数据库连接
func CloseDB() {
	if db != nil {
		db.Close()
	}
} 