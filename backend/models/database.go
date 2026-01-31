package models

import (
	"fmt"
	"log"

	"markpost/conf"
	"markpost/utils"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

func (d *Database) DB() *gorm.DB {
	return d.db
}

func NewDatabase(dsn string) (*Database, error) {
	cfg := conf.Conf()

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

	if err = db.AutoMigrate(&User{}, &Post{}, &DeliveryChannel{}); err != nil {
		return nil, fmt.Errorf("NewDatabase auto migrate: %w", err)
	}

	database := Database{db: db}

	username := cfg.Admin.InitialUsername
	exists, err := UserExists(&database, map[string]any{"username": username})
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

		user := User{
			Username: username,
			Password: password,
			PostKey:  postKey,
		}
		if err = user.Create(&database); err != nil {
			return nil, err
		}
		log.Printf("initialized user: %s", username)
	}

	return &database, nil
}

func NewTestDatabase() (*Database, error) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("NewTestDatabase open sqlite: %w", err)
	}
	if sqlDB, err2 := gdb.DB(); err2 == nil {
		_, _ = sqlDB.Exec("PRAGMA journal_mode=WAL;")
		_, _ = sqlDB.Exec("PRAGMA foreign_keys = ON;")
	}

	if err = gdb.AutoMigrate(&User{}, &Post{}, &DeliveryChannel{}); err != nil {
		return nil, fmt.Errorf("NewTestDatabase auto migrate: %w", err)
	}

	return &Database{db: gdb}, nil
}
