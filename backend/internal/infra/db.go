// Package infra provides infrastructure layer implementations.
package infra

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database wraps a GORM database connection.
type Database struct {
	db *gorm.DB
}

// New creates a new Database instance with the provided DSN.
func New(dsn string) (*Database, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("NewDatabase open postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("NewDatabase access postgres pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return &Database{db: db}, nil
}

// DB returns the underlying GORM database connection.
func (d *Database) DB() *gorm.DB {
	return d.db
}

// Close closes the underlying database connection.
func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
