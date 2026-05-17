// Package infra provides infrastructure layer implementations.
package infra

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/pkg/utils"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var allModels = []any{
	&user.User{},
	&user.RefreshToken{},
	&user.TokenBlacklist{},
	&post.Post{},
	&delivery.Channel{},
}

// Database wraps a GORM database connection.
type Database struct {
	db *gorm.DB
}

func ensureSQLiteDir(dsn string) error {
	if dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:") {
		return nil
	}

	path := dsn
	if strings.HasPrefix(path, "file:") {
		path = strings.TrimPrefix(path, "file:")
	}
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create sqlite data directory: %w", err)
	}
	return nil
}

// New creates a new Database instance with the provided DSN.
func New(dsn string) (*Database, error) {
	cfg := config.Get()

	var db *gorm.DB
	var err error

	switch cfg.DB.Driver {
	case "postgresql":
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("NewDatabase open postgres: %w", err)
		}
	case "sqlite":
		if err = ensureSQLiteDir(dsn); err != nil {
			return nil, fmt.Errorf("NewDatabase prepare sqlite: %w", err)
		}
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("NewDatabase open sqlite: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DB.Driver)
	}

	if err = db.AutoMigrate(allModels...); err != nil {
		return nil, fmt.Errorf("NewDatabase auto migrate: %w", err)
	}

	database := &Database{db: db}

	if err := database.migrateQIDPrefix(); err != nil {
		return nil, fmt.Errorf("NewDatabase migrate qid prefix: %w", err)
	}

	if err := database.seedAdminUser(); err != nil {
		return nil, fmt.Errorf("NewDatabase seed admin: %w", err)
	}

	return database, nil
}

// NewTestDatabase creates a new in-memory database for testing.
func NewTestDatabase() (*Database, error) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("NewTestDatabase open sqlite: %w", err)
	}
	if sqlDB, err2 := gdb.DB(); err2 == nil {
		_, _ = sqlDB.Exec("PRAGMA journal_mode=WAL;")
		_, _ = sqlDB.Exec("PRAGMA foreign_keys = ON;")
	}

	if err = gdb.AutoMigrate(allModels...); err != nil {
		return nil, fmt.Errorf("NewTestDatabase auto migrate: %w", err)
	}

	return &Database{db: gdb}, nil
}

// DB returns the underlying GORM database connection.
func (d *Database) DB() *gorm.DB {
	return d.db
}

func (d *Database) userExists(username string) (bool, error) {
	return existsBy[user.User](context.Background(), d.db, "username", username, "userExists")
}

func (d *Database) createUser(u *user.User) error {
	return d.db.Create(u).Error
}

func (d *Database) migrateQIDPrefix() error {
	result := d.db.Model(&post.Post{}).
		Where("qid NOT LIKE ?", "p-%").
		Update("qid", d.db.Statement.DB.Raw("CONCAT('p-', qid)"))
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "CONCAT") {
			var posts []post.Post
			if err := d.db.Where("qid NOT LIKE ?", "p-%").Find(&posts).Error; err != nil {
				return err
			}
			for _, p := range posts {
				if err := d.db.Model(&post.Post{}).Where("id = ?", p.ID).Update("qid", "p-"+p.QID).Error; err != nil {
					return err
				}
			}
			return nil
		}
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("migrated %d post qids with p- prefix", result.RowsAffected)
	}
	return nil
}

func (d *Database) seedAdminUser() error {
	cfg := config.Get()
	username := cfg.Admin.InitialUsername

	exists, err := d.userExists(username)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	password, err := utils.HashPassword(cfg.Admin.InitialPassword)
	if err != nil {
		return err
	}

	postKey, err := utils.GeneratePostKey(cfg.PostKeyLength)
	if err != nil {
		return err
	}

	u := user.User{
		Email:    username + "@localhost",
		Username: username,
		Password: password,
		PostKey:  postKey,
		IsActive: true,
	}
	if err = d.createUser(&u); err != nil {
		return err
	}
	log.Printf("initialized user: %s", username)
	return nil
}
