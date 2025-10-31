package main

import (
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var database *Database

type Database struct {
	db *gorm.DB
}

func NewDatabase(url string) (*Database, error) {
    var gdb *gorm.DB
    var err error

    if config.Database.Type == "postgresql" {
        gdb, err = gorm.Open(postgres.Open(url), &gorm.Config{})
        if err != nil {
            return nil, err
        }
    } else {
        gdb, err = gorm.Open(sqlite.Open(url), &gorm.Config{})
        if err != nil {
            return nil, err
        }
        if sqlDB, err2 := gdb.DB(); err2 == nil {
            _, _ = sqlDB.Exec("PRAGMA journal_mode=WAL;")
            _, _ = sqlDB.Exec("PRAGMA foreign_keys = ON;")
        }
    }

	if err = gdb.AutoMigrate(&User{}, &Post{}); err != nil {
		return nil, err
	}

	var count int64
	initialUsername := config.InitialUser.Username
	if initialUsername == "" {
		initialUsername = "markpost"
	}
	initialPassword := config.InitialUser.Password
	if initialPassword == "" {
		initialPassword = "markpost"
	}

	gdb.Model(&User{}).Where("username = ?", initialUsername).Count(&count)
	if count == 0 {
		var hashedPassword string
		if initialPassword != "" {
			hp, err := HashPassword(initialPassword)
			if err != nil {
				return nil, err
			}
			hashedPassword = hp
		}
		postKey, err := GeneratePostKey(16)
		if err != nil {
			return nil, err
		}
		user := User{
			Username:  initialUsername,
			Password:  hashedPassword,
			PostKey:   postKey,
			CreatedAt: time.Now().UTC(),
		}
		if err = gdb.Create(&user).Error; err != nil {
			return nil, err
		}
		log.Printf("created initial user '%s' with post_key: %s", initialUsername, postKey)
	}

	return &Database{db: gdb}, nil
}

func (d *Database) GetDB() *gorm.DB {
	return d.db
}

func (d *Database) GetUserRepository() UserRepository {
	return &userRepository{db: d.db}
}

func (d *Database) GetPostRepository() PostRepository {
	return &postRepository{db: d.db}
}

func NewTestDatabase() (*Database, error) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if sqlDB, err2 := gdb.DB(); err2 == nil {
		_, _ = sqlDB.Exec("PRAGMA journal_mode=WAL;")
		_, _ = sqlDB.Exec("PRAGMA foreign_keys = ON;")
	}

	if err = gdb.AutoMigrate(&User{}, &Post{}); err != nil {
		return nil, err
	}

	return &Database{db: gdb}, nil
}
