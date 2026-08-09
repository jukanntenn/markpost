package infra

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
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
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
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
	if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
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
	// migrate's URL-based constructor parses the DSN as a URL and fails with
	// "no scheme" on the libpq key-value form, which GORM and injectTimezone
	// both accept. Open the pool with lib/pq (it takes either form) and hand
	// migrate a ready driver so the two agree on what a valid DSN is.
	// The "postgres" sql driver is registered by lib/pq, imported by the
	// migrate postgres driver package above.
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("migrate open postgres: %w", err)
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate postgres driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		_ = driver.Close()
		return nil, fmt.Errorf("migrate new: %w", err)
	}
	return m, nil
}
