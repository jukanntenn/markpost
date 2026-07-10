// Package infra provides infrastructure layer implementations.
package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
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
	&delivery.Attempt{},
	&delivery.History{},
}

// Database wraps a GORM database connection.
type Database struct {
	db *gorm.DB
}

func ensureSQLiteDir(dsn string) error {
	if dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:") {
		return nil
	}

	path := strings.TrimPrefix(dsn, "file:")
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
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("NewDatabase open mysql: %w", err)
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

	switch cfg.DB.Driver {
	case "sqlite":
		if sqlDB, err2 := db.DB(); err2 == nil {
			sqlDB.SetMaxOpenConns(1)
			defer func() { sqlDB.SetMaxOpenConns(0) }()
			db.Exec("PRAGMA foreign_keys = OFF")
		}
	case "postgresql", "mysql":
		sqlDB, err2 := db.DB()
		if err2 != nil {
			return nil, fmt.Errorf("NewDatabase access %s pool: %w", cfg.DB.Driver, err2)
		}
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
	}

	if err = db.AutoMigrate(allModels...); err != nil {
		return nil, fmt.Errorf("NewDatabase auto migrate: %w", err)
	}

	if cfg.DB.Driver == "sqlite" {
		db.Exec("PRAGMA foreign_keys = ON")
	}

	database := &Database{db: db}

	if err := database.migratePasswordColumn(); err != nil {
		return nil, fmt.Errorf("NewDatabase migrate password column: %w", err)
	}

	if err := database.migrateQIDPrefix(); err != nil {
		return nil, fmt.Errorf("NewDatabase migrate qid prefix: %w", err)
	}

	if err := database.dropStaleChannelsTable(); err != nil {
		return nil, fmt.Errorf("NewDatabase drop stale channels table: %w", err)
	}

	if err := database.migrateChannelConfiguration(); err != nil {
		return nil, fmt.Errorf("NewDatabase migrate channel configuration: %w", err)
	}

	if cfg.DB.Driver == "postgresql" {
		if err := database.migratePostBodyCompressionLZ4(); err != nil {
			return nil, fmt.Errorf("NewDatabase migrate post body lz4: %w", err)
		}
	}

	if err := database.migrateDeliveryIndexesAndOptions(); err != nil {
		return nil, fmt.Errorf("NewDatabase migrate delivery indexes: %w", err)
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

// Close closes the underlying database connection.
func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) userExists(username string) (bool, error) {
	return existsBy[user.User](context.Background(), d.db, "username", username, "userExists")
}

func (d *Database) createUser(u *user.User) error {
	return d.db.Create(u).Error
}

func (d *Database) migratePasswordColumn() error {
	if !d.db.Migrator().HasColumn(&user.User{}, "password") {
		return nil
	}

	result := d.db.Exec("UPDATE users SET password_hash = password WHERE password IS NOT NULL AND (password_hash IS NULL OR password_hash = '')")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("migrated %d user passwords from legacy 'password' column", result.RowsAffected)
	}

	return nil
}

func (d *Database) migrateQIDPrefix() error {
	result := d.db.Model(&post.Post{}).
		Where("qid NOT LIKE ?", "p-%").
		Update("qid", d.db.Statement.Raw("CONCAT('p-', qid)"))
	log.Printf("migrateQIDPrefix: rowsAffected=%d, error=%v", result.RowsAffected, result.Error)
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

func (d *Database) dropStaleChannelsTable() error {
	if d.db.Migrator().HasTable("channels") {
		if err := d.db.Exec("DROP TABLE channels").Error; err != nil {
			return err
		}
		log.Print("dropped stale channels table")
	}
	return nil
}

func (d *Database) migrateChannelConfiguration() error {
	if !d.db.Migrator().HasColumn(&delivery.Channel{}, "webhook_url") {
		return nil
	}

	type legacyChannel struct {
		ID         int    `gorm:"primaryKey"`
		WebhookURL string `gorm:"column:webhook_url"`
	}

	var rows []legacyChannel
	if err := d.db.Table("delivery_channels").Find(&rows).Error; err != nil {
		return fmt.Errorf("read legacy channels: %w", err)
	}

	for _, row := range rows {
		config := delivery.ChannelConfiguration{
			"webhook_url":   row.WebhookURL,
			"card_link_url": "",
		}
		jsonBytes, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("marshal config for channel %d: %w", row.ID, err)
		}
		if err := d.db.Table("delivery_channels").
			Where("id = ?", row.ID).
			Update("configuration", string(jsonBytes)).Error; err != nil {
			return fmt.Errorf("update config for channel %d: %w", row.ID, err)
		}
	}

	if len(rows) > 0 {
		log.Printf("migrated %d delivery channels from webhook_url to configuration", len(rows))
	}

	if err := d.db.Migrator().DropColumn(&delivery.Channel{}, "webhook_url"); err != nil {
		return fmt.Errorf("drop legacy webhook_url column: %w", err)
	}
	log.Print("dropped legacy webhook_url column from delivery_channels")

	return nil
}

// migratePostBodyCompressionLZ4 switches the posts.body TOAST compressor to
// lz4 on Postgres 14+. lz4 decompresses ~3x faster than the default pglz at a
// comparable ratio.
//
// The ALTER is gated on the column's current attcompression: it only runs once,
// on the first startup after the upgrade, and becomes a cheap catalog read on
// every subsequent restart. SET COMPRESSION is metadata-only (it does not
// rewrite rows and takes AccessExclusiveLock only for the instant of the
// catalog update), and is idempotent in effect — but re-issuing the DDL every
// boot would needlessly reacquire that lock, so we skip it once the attribute
// is already lz4.
//
// Existing rows keep their old compression until they are rewritten, so a
// one-time `VACUUM FULL posts` is recommended in a maintenance window to
// retrofit them — that is intentionally NOT done here because it takes a
// long-lived AccessExclusiveLock. SQLite has no TOAST and is skipped (caller
// gates on driver).
func (d *Database) migratePostBodyCompressionLZ4() error {
	// pg_attribute.attcompression is a char set by SET COMPRESSION:
	// 'p' = pglz, 'l' = lz4, '\0' = default (see toast_compression.h:
	// TOAST_PGLZ_COMPRESSION / TOAST_LZ4_COMPRESSION / InvalidCompressionMethod).
	// It is NOT an OID into pg_am (that catalog is for index access methods).
	const alreadyLZ4 = `SELECT EXISTS (
		SELECT 1
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relname = 'posts'
		  AND a.attname = 'body'
		  AND a.attcompression = 'l')`
	var already bool
	if err := d.db.Raw(alreadyLZ4).Scan(&already).Error; err != nil {
		return fmt.Errorf("check body compression: %w", err)
	}
	if already {
		return nil
	}
	return d.db.Exec("ALTER TABLE posts ALTER COLUMN body SET COMPRESSION lz4").Error
}

// migrateDeliveryIndexesAndOptions creates the per-dialect indexes and (on
// Postgres) the storage reloptions for the delivery queue tables. The claim
// query is WHERE status = <pending> AND next_at <= ? ORDER BY next_at; the
// optimal index for it differs by dialect and cannot be expressed in a GORM
// struct tag:
//
//   - Postgres/SQLite: a partial index (next_at) WHERE status = 0 keeps the
//     index small and, on Postgres, cooperates with HOT updates (once a row
//     leaves pending its index entry is dropped, so later status/next_at
//     updates never touch this index).
//   - MySQL: partial indexes are unsupported (CREATE INDEX has no WHERE
//     grammar), so a composite (status, next_at) covers the claim query.
//     A usable index is load-bearing on InnoDB — without it the claim's record
//     locks escalate to a full-table scan lock.
//
// On Postgres the hot table also gets fillfactor=90 (HOT-friendly page space)
// and aggressive autovacuum reloptions; the append-only history table gets
// fillfactor=100. MySQL and SQLite have no per-table fillfactor/autovacuum
// analog and are left at driver defaults.
//
// The history table carries two indexes, created here (rather than via GORM
// tags) so the IF NOT EXISTS guard and the per-dialect branch stay in one
// place:
//
//   - idx_dh_user_channel_created (user_id, channel_id, created_at DESC): the
//     three-column composite covers both the user-scoped history query
//     (WHERE user_id = ?) and the per-channel query (WHERE user_id = ? AND
//     channel_id = ?) via the leftmost-prefix rule. It supersedes the old
//     idx_dh_user_created, which is dropped.
//   - idx_dh_created (created_at DESC): serves the admin "all history" view
//     (ORDER BY created_at DESC with no user_id predicate), which the
//     user_id-leading composite index cannot serve (no leftmost equality).
func (d *Database) migrateDeliveryIndexesAndOptions() error {
	driver := config.Get().DB.Driver

	switch driver {
	case "postgresql":
		stmts := []string{
			`CREATE INDEX IF NOT EXISTS idx_da_pending ON delivery_attempts (next_at) WHERE status = 0`,
			`DROP INDEX IF EXISTS idx_dh_user_created`,
			`CREATE INDEX IF NOT EXISTS idx_dh_user_channel_created ON delivery_history (user_id, channel_id, created_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_dh_created ON delivery_history (created_at DESC)`,
			`ALTER TABLE delivery_attempts SET (
				fillfactor = 90,
				autovacuum_vacuum_scale_factor = 0.05,
				autovacuum_vacuum_threshold = 1000,
				autovacuum_analyze_scale_factor = 0.02,
				autovacuum_analyze_threshold = 1000
			)`,
			`ALTER TABLE delivery_history SET (fillfactor = 100)`,
		}
		for _, s := range stmts {
			if err := d.db.Exec(s).Error; err != nil {
				return fmt.Errorf("postgres delivery index/option: %w", err)
			}
		}
	case "sqlite":
		stmts := []string{
			`CREATE INDEX IF NOT EXISTS idx_da_pending ON delivery_attempts (next_at) WHERE status = 0`,
			`DROP INDEX IF EXISTS idx_dh_user_created`,
			`CREATE INDEX IF NOT EXISTS idx_dh_user_channel_created ON delivery_history (user_id, channel_id, created_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_dh_created ON delivery_history (created_at DESC)`,
		}
		for _, s := range stmts {
			if err := d.db.Exec(s).Error; err != nil {
				return fmt.Errorf("sqlite delivery index: %w", err)
			}
		}
	case "mysql":
		stmts := []string{
			`CREATE INDEX idx_da_status_next ON delivery_attempts (status, next_at)`,
			`DROP INDEX idx_dh_user_created ON delivery_history`,
			`CREATE INDEX idx_dh_user_channel_created ON delivery_history (user_id, channel_id, created_at DESC)`,
			`CREATE INDEX idx_dh_created ON delivery_history (created_at DESC)`,
		}
		for _, s := range stmts {
			if err := d.db.Exec(s).Error; err != nil {
				if isIndexExistsErr(err) || isUnknownIndexErr(err) {
					continue
				}
				return fmt.Errorf("mysql delivery index: %w", err)
			}
		}
	}
	return nil
}

// isIndexExistsErr reports whether err is MySQL's "duplicate key name" error,
// raised when CREATE INDEX targets an index that already exists. MySQL does not
// support CREATE INDEX ... IF NOT EXISTS (the create_index_stmt grammar at
// sql/sql_yacc.yy has no opt_if_not_exists, and prepare_key raises
// ER_DUP_KEYNAME unconditionally at sql/sql_table.cc:7596), so repeated index
// creation must detect this one error and treat it as success to stay
// idempotent. ER_DUP_KEYNAME is error number 1061, SQLSTATE 42000
// (share/messages_to_clients.txt). Matching on the driver's typed error number
// avoids false positives from string-matching the message.
func isIndexExistsErr(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == errDupKeyName
	}
	return false
}

// isUnknownIndexErr reports whether err is MySQL's "can't drop; index doesn't
// exist" error. MySQL does not support DROP INDEX ... IF EXISTS in the ALTER
// TABLE form used by InnoDB (the drop_index_stmt production at
// sql/sql_yacc.yy has no opt_if_exists), so a DROP on an already-absent index
// raises ER_CANT_DROP_FIELD_OR_KEY (1091). Tolerating it keeps the migration
// idempotent on MySQL, mirroring DROP INDEX IF EXISTS on Postgres/SQLite.
func isUnknownIndexErr(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == errCantDropFieldOrKey
	}
	return false
}

const (
	errDupKeyName         = 1061
	errCantDropFieldOrKey = 1091
)

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

	u, err := makeUser(username, username, cfg.Admin.InitialPassword, nil, nil, cfg.PostKeyLength)
	if err != nil {
		return err
	}
	u.Email = username + "@localhost"

	if err = d.createUser(u); err != nil {
		return err
	}
	log.Printf("initialized user: %s", username)
	return nil
}
