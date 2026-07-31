// Package infra provides infrastructure layer implementations.
package infra

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database wraps a GORM database connection.
type Database struct {
	db *gorm.DB
}

// New creates a new Database instance with the provided DSN. The timezone
// pins three things so writes, reads and time.Now() all agree on one zone,
// regardless of the process's TZ env or the server's default:
//   - time.Local is set to the zone, so pgx's timestamptz decode (which uses
//     time.Local) and every time.Now() caller land in the configured zone;
//   - a `timezone=` DSN parameter is injected, which the pgx-backed driver
//     applies on every pooled connection as the Postgres session timezone;
//   - GORM's NowFunc stamps autoCreateTime/autoUpdateTime in the same zone.
func New(dsn string, timezone string) (*Database, error) {
	if timezone == "" {
		timezone = "UTC"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("NewDatabase load timezone %q: %w", timezone, err)
	}
	time.Local = loc
	dsn = injectTimezone(dsn, timezone)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time { return time.Now().In(loc) },
	})
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

// injectTimezone appends a `timezone=<zone>` parameter to the DSN unless one is
// already present. The gorm postgres driver parses this on every pooled
// connection and applies it both as the Postgres session timezone and as the
// pgx timestamp codec's scan location, so the setting is connection-pool safe.
// The zone string has already been validated by time.LoadLocation, so it is a
// real IANA name (letters, '/', '_') and safe to interpolate.
func injectTimezone(dsn, timezone string) string {
	if strings.Contains(dsn, "timezone=") ||
		strings.Contains(dsn, "TimeZone=") ||
		strings.Contains(dsn, "time_zone=") {
		return dsn
	}
	sep := " "
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if strings.Contains(dsn, "?") {
			sep = "&"
		} else {
			sep = "?"
		}
	}
	return dsn + sep + "timezone=" + timezone
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
