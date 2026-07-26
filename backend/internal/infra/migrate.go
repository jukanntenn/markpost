package infra

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// MigrateUp applies all pending up migrations to the given postgres DSN.
func MigrateUp(dsn string) error {
	m, err := newMigrate(dsn)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back n migrations.
func MigrateDown(dsn string, n int) error {
	m, err := newMigrate(dsn)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Steps(-n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down %d: %w", n, err)
	}
	return nil
}

// MigrateForce sets the version without running migrations (baseline an existing DB).
func MigrateForce(dsn string, version int) error {
	m, err := newMigrate(dsn)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	return m.Force(version)
}

// MigrateVersion reports the current migration version.
func MigrateVersion(dsn string) (uint, bool, error) {
	m, err := newMigrate(dsn)
	if err != nil {
		return 0, false, err
	}
	defer func() { _, _ = m.Close() }()
	v, dirty, err := m.Version()
	return v, dirty, err
}

func newMigrate(dsn string) (*migrate.Migrate, error) {
	sub, err := fs.Sub(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate embed sub: %w", err)
	}
	src, err := iofs.New(sub, ".")
	if err != nil {
		return nil, fmt.Errorf("migrate iofs: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return nil, fmt.Errorf("migrate new: %w", err)
	}
	return m, nil
}
