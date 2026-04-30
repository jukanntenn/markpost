// Package infra provides infrastructure layer implementations.
package infra

import (
	"fmt"
	"log"

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

// Database wraps a GORM database connection.
type Database struct {
	db *gorm.DB
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
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("NewDatabase open sqlite: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DB.Driver)
	}

	if err = db.AutoMigrate(
		&user.User{},
		&user.RefreshToken{},
		&user.TokenBlacklist{},
		&post.Post{},
		&delivery.Channel{},
	); err != nil {
		return nil, fmt.Errorf("NewDatabase auto migrate: %w", err)
	}

	database := &Database{db: db}

	if err := database.migrateQIDPrefix(); err != nil {
		return nil, fmt.Errorf("NewDatabase migrate qid prefix: %w", err)
	}

	username := cfg.Admin.InitialUsername
	exists, err := database.userExists(username)
	if err != nil {
		return nil, err
	}
	if !exists {
		password, err := utils.HashPassword(cfg.Admin.InitialPassword)
		if err != nil {
			return nil, err
		}

		postKey, err := utils.GeneratePostKey(cfg.PostKeyLength)
		if err != nil {
			return nil, err
		}

		u := user.User{
			Email:    username + "@localhost",
			Username: username,
			Password: password,
			PostKey:  postKey,
			IsActive: true,
		}
		if err = database.createUser(&u); err != nil {
			return nil, err
		}
		log.Printf("initialized user: %s", username)
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

	if err = gdb.AutoMigrate(
		&user.User{},
		&user.RefreshToken{},
		&user.TokenBlacklist{},
		&post.Post{},
		&delivery.Channel{},
	); err != nil {
		return nil, fmt.Errorf("NewTestDatabase auto migrate: %w", err)
	}

	return &Database{db: gdb}, nil
}

// DB returns the underlying GORM database connection.
func (d *Database) DB() *gorm.DB {
	return d.db
}

func (d *Database) userExists(username string) (bool, error) {
	var count int64
	if err := d.db.Model(&user.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, fmt.Errorf("userExists: %w", err)
	}
	return count > 0, nil
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
