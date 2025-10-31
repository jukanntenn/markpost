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

	if url != "" && (len(url) >= 10 && (url[:10] == "postgres://" || (len(url) >= 12 && url[:12] == "postgresql://"))) {
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
		}
	}

	if err = gdb.AutoMigrate(&User{}, &Post{}); err != nil {
		return nil, err
	}

	migrateUserCreatedAtWithDB(gdb)
	migrateUserPasswordWithDB(gdb)

	var count int64
	gdb.Model(&User{}).Where("username = ?", "markpost").Count(&count)
	if count == 0 {
		postKey, err := GeneratePostKey(8)
		if err != nil {
			return nil, err
		}
		hashedPassword, err := HashPassword("markpost")
		if err != nil {
			return nil, err
		}
		user := User{
			Username:  "markpost",
			Password:  hashedPassword,
			PostKey:   postKey,
			CreatedAt: time.Now().UTC(),
		}
		if err = gdb.Create(&user).Error; err != nil {
			return nil, err
		}
		log.Printf("created markpost user with post_key: %s", postKey)
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
	}

	if err = gdb.AutoMigrate(&User{}, &Post{}); err != nil {
		return nil, err
	}

	return &Database{db: gdb}, nil
}

func migrateUserCreatedAtWithDB(gdb *gorm.DB) {
	if !gdb.Migrator().HasColumn(&User{}, "created_at") {
		if err := gdb.Migrator().AddColumn(&User{}, "created_at"); err != nil {
			return
		}
	}
	now := time.Now().UTC()
	_ = gdb.Model(&User{}).Where("created_at IS NULL OR created_at = '0001-01-01 00:00:00'").Update("created_at", now).Error
}

func migrateUserPasswordWithDB(gdb *gorm.DB) {
	if !gdb.Migrator().HasColumn(&User{}, "password") {
		if err := gdb.Migrator().AddColumn(&User{}, "password"); err != nil {
			return
		}
	}
}
