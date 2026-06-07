// Package main verifies that SQLite schema migrations correctly handle table renames.
package main

import (
	"fmt"
	"log"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	dbPath := "file:/tmp/test_migrate.db?_foreign_keys=on&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Create old schema tables (simulating "old schemas")
	db.Exec("DROP TABLE IF EXISTS posts")
	db.Exec("DROP TABLE IF EXISTS delivery_channels")
	db.Exec("DROP TABLE IF EXISTS channels")
	db.Exec("DROP TABLE IF EXISTS users")

	// Old schema: no TableName override for Channel -> GORM uses "channels"
	type OldUser struct {
		ID       int    `gorm:"primaryKey;autoIncrement"`
		Email    string `gorm:"unique;not null"`
		Username string `gorm:"unique;not null"`
		Name     string
		Password string `gorm:"column:password_hash"`
		PostKey  string `gorm:"unique;not null"`
		Role     string `gorm:"not null;default:'user'"`
		IsActive bool   `gorm:"default:true"`
	}
	type OldChannel struct {
		ID         int    `gorm:"primaryKey;autoIncrement"`
		UserID     int    `gorm:"index;not null;column:user_id"`
		Kind       string `gorm:"not null;size:32"`
		Name       string `gorm:"not null;default:''"`
		Enabled    bool   `gorm:"not null;default:true"`
		WebhookURL string `gorm:"not null;type:text;column:webhook_url"`
		Keywords   string `gorm:"not null;type:text;default:''"`
	}
	type OldPost struct {
		ID     int    `gorm:"primaryKey;autoIncrement"`
		QID    string `gorm:"unique;not null;column:qid"`
		Title  string `gorm:"not null"`
		Body   string `gorm:"not null;type:text"`
		UserID int    `gorm:"index;not null;column:user_id"`
	}

	if err := db.AutoMigrate(&OldUser{}, &OldPost{}, &OldChannel{}); err != nil {
		log.Fatal("old migrate:", err)
	}

	// Insert test data
	db.Exec("INSERT INTO users (email, username, password_hash, post_key, role, is_active) VALUES ('test@test.com', 'testuser', 'hash', 'key123', 'admin', 1)")
	db.Exec("INSERT INTO posts (qid, title, body, user_id) VALUES ('abc123', 'Test Post 1', 'Body 1', 1)")
	db.Exec("INSERT INTO posts (qid, title, body, user_id) VALUES ('def456', 'Test Post 2', 'Body 2', 1)")
	db.Exec("INSERT INTO channels (user_id, kind, name, webhook_url) VALUES (1, 'feishu', 'Channel 1', 'https://hook.example.com')")

	var postCount, channelCount, userCount int64
	db.Raw("SELECT count(*) FROM users").Scan(&userCount)
	db.Raw("SELECT count(*) FROM posts").Scan(&postCount)
	db.Raw("SELECT count(*) FROM channels").Scan(&channelCount)
	fmt.Printf("BEFORE new migration: users=%d, posts=%d, channels=%d\n", userCount, postCount, channelCount)

	// Now run NEW code's AutoMigrate (Channel now has TableName() -> delivery_channels)
	fmt.Println("\n--- Running new AutoMigrate ---")
	if err := db.AutoMigrate(&user.User{}, &post.Post{}, &delivery.Channel{}); err != nil {
		log.Fatal("new migrate:", err)
	}

	db.Raw("SELECT count(*) FROM users").Scan(&userCount)
	db.Raw("SELECT count(*) FROM posts").Scan(&postCount)
	var oldChannels, newChannels int64
	db.Raw("SELECT count(*) FROM channels").Scan(&oldChannels)
	db.Raw("SELECT count(*) FROM delivery_channels").Scan(&newChannels)
	fmt.Printf("AFTER AutoMigrate: users=%d, posts=%d, channels=%d, delivery_channels=%d\n", userCount, postCount, oldChannels, newChannels)

	// Simulate dropStaleChannelsTable
	fmt.Println("\n--- dropStaleChannelsTable ---")
	var hasOldTable int
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='channels'").Scan(&hasOldTable)
	if hasOldTable > 0 {
		db.Exec("DROP TABLE channels")
		fmt.Println("Dropped 'channels' table")
	}

	db.Raw("SELECT count(*) FROM users").Scan(&userCount)
	db.Raw("SELECT count(*) FROM posts").Scan(&postCount)
	db.Raw("SELECT count(*) FROM delivery_channels").Scan(&newChannels)
	fmt.Printf("FINAL: users=%d, posts=%d, delivery_channels=%d\n", userCount, postCount, newChannels)

	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		log.Fatal(err)
	}
}
