// Package cmd provides CLI commands for the application.
package cmd

import (
	"context"
	"fmt"
	"log"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/infra"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// sqliteModels is the per-table model set used to scan the source SQLite file,
// matching allModels in infra/db.go. Declared separately so this command stays
// decoupled from the migrate target's internal ordering.
var sqliteModels = []any{
	&user.User{},
	&user.RefreshToken{},
	&user.TokenBlacklist{},
	&post.Post{},
	&delivery.Channel{},
}

// RunMigrateSqliteToPostgres reads every table from a SQLite file and copies
// the rows into the PostgreSQL database named by the config's DSN. The target
// database must already exist with its schema — this command calls infra.New,
// which runs AutoMigrate, so pointing it at a fresh Postgres database creates
// the full schema (the same one the server creates on boot) before copying.
//
// Tables present in SQLite but absent from the model set (for example a legacy
// channels table) are ignored. Tables present in the model set but absent from
// SQLite (for example delivery_attempts on an old install) are skipped with a
// notice, since there is nothing to copy.
//
// The copy preserves primary-key ids (the auto-incremented values from the
// source), then advances each Postgres sequence past the largest copied id so
// subsequent server inserts do not collide. The whole copy runs in one
// transaction; any error rolls everything back, leaving the target untouched.
//
// Usage:
//
//	markpost -c <target-postgres-config.toml> migrate-sqlite-to-postgres \
//	    --sqlite /path/to/source.db [--dry-run]
func RunMigrateSqliteToPostgres(configPath, sqlitePath string, dryRun bool) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg := config.Get()
	if cfg.DB.Driver != "postgresql" {
		return fmt.Errorf("config db.driver must be postgresql for the migration target, got %q", cfg.DB.Driver)
	}

	target, err := infra.New(cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize target database: %w", err)
	}
	defer func() {
		if err := target.Close(); err != nil {
			log.Printf("Failed to close target database: %v", err)
		}
	}()

	source, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open source sqlite: %w", err)
	}
	sqlDB, err := source.DB()
	if err != nil {
		return fmt.Errorf("failed to access source sqlite pool: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	ctx := context.Background()
	return target.DB().Transaction(func(tx *gorm.DB) error {
		for _, model := range sqliteModels {
			if err := migrateTable(ctx, source, tx, model, dryRun); err != nil {
				return err
			}
		}
		if dryRun {
			log.Print("dry-run: no changes committed")
			return nil
		}
		return resyncSequences(tx)
	})
}

// migrateTable copies one model's rows from source to tx. Tables missing from
// the source are reported and skipped.
func migrateTable(ctx context.Context, source, tx *gorm.DB, model any, dryRun bool) error {
	table := tableName(model)

	if !source.Migrator().HasTable(model) {
		log.Printf("skip %s: table absent from source sqlite", table)
		return nil
	}

	var total int64
	if err := source.Model(model).Count(&total).Error; err != nil {
		return fmt.Errorf("count source %s: %w", table, err)
	}
	if total == 0 {
		log.Printf("skip %s: source has 0 rows", table)
		return nil
	}

	rows, err := source.Model(model).Rows()
	if err != nil {
		return fmt.Errorf("scan source %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	copied := int64(0)
	for rows.Next() {
		dst := newModel(model)
		if err := source.ScanRows(rows, dst); err != nil {
			return fmt.Errorf("scan row in %s: %w", table, err)
		}
		if dryRun {
			copied++
			continue
		}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(dst).Error; err != nil {
			return fmt.Errorf("insert row into %s: %w", table, err)
		}
		copied++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate %s: %w", table, err)
	}

	log.Printf("%s: copied %d/%d rows", table, copied, total)
	return nil
}

// resyncSequences advances every serial/bigserial sequence in the target past
// the largest copied id, so the server's later inserts pick fresh ids instead
// of colliding with migrated ones.
func resyncSequences(tx *gorm.DB) error {
	type seq struct {
		Table string
	}
	var tables []seq
	if err := tx.Raw(`SELECT table_name AS table FROM information_schema.columns WHERE table_schema = 'public' AND column_name = 'id' AND column_default LIKE 'nextval%'`).Scan(&tables).Error; err != nil {
		return fmt.Errorf("list id sequences: %w", err)
	}
	for _, t := range tables {
		seqName := t.Table + "_id_seq"
		stmt := fmt.Sprintf(`SELECT setval('%s', COALESCE((SELECT max(id) FROM "%s"), 1), true)`, seqName, t.Table)
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("setval %s: %w", seqName, err)
		}
		log.Printf("resync %s -> max(id)", seqName)
	}
	return nil
}

// tableName returns the database table name for a model value, for log
// messages. It mirrors the TableName methods declared on each model; GORM's
// generic helper is avoided so the names stay readable in logs.
func tableName(model any) string {
	switch model.(type) {
	case *user.User:
		return "users"
	case *user.RefreshToken:
		return "refresh_tokens"
	case *user.TokenBlacklist:
		return "token_blacklist"
	case *post.Post:
		return "posts"
	case *delivery.Channel:
		return "delivery_channels"
	default:
		return fmt.Sprintf("%T", model)
	}
}

// newModel allocates a fresh zero value of the same model type. GORM needs a
// pointer to a concrete struct per row; the switch mirrors sqliteModels.
func newModel(model any) any {
	switch model.(type) {
	case *user.User:
		return &user.User{}
	case *user.RefreshToken:
		return &user.RefreshToken{}
	case *user.TokenBlacklist:
		return &user.TokenBlacklist{}
	case *post.Post:
		return &post.Post{}
	case *delivery.Channel:
		return &delivery.Channel{}
	default:
		return model
	}
}
