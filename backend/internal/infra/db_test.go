package infra

import (
	"os"
	"path/filepath"
	"testing"

	"markpost/internal/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewTestDatabase(t *testing.T) {
	database, err := NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}
	if database == nil {
		t.Fatal("expected non-nil database")
	}
	defer func() { _ = database.Close() }()

	db := database.DB()
	if db == nil {
		t.Fatal("expected non-nil gorm.DB")
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping error: %v", err)
	}
}

func TestDatabase_DB(t *testing.T) {
	database, err := NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}
	defer func() { _ = database.Close() }()

	if database.DB() == nil {
		t.Fatal("DB() returned nil")
	}
}

func TestDatabase_Close(t *testing.T) {
	database, err := NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
}

func TestNew_WithSQLite(t *testing.T) {
	config.ResetForTest()

	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "test.toml")
	dbPath := filepath.Join(tmpDir, "data", "test.db")
	content := `
post_key_length = 16
[server]
host = "127.0.0.1"
port = 7330
[db]
driver = "sqlite"
dsn = "` + dbPath + `"
[admin]
initial_username = "admin"
initial_password = "admin123"
[jwt]
access_signing_key = "test-access-key-min-32-characters!!"
refresh_signing_key = "test-refresh-key-min-32-characters!!"
[delivery]
request_timeout = "5s"
`
	if err := os.WriteFile(tomlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := config.Load(tomlPath)
	if err != nil {
		t.Fatalf("config.Load error: %v", err)
	}

	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer func() { _ = database.Close() }()

	db := database.DB()
	if db == nil {
		t.Fatal("expected non-nil gorm.DB")
	}

	var count int64
	db.Table("users").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 seeded admin user, got %d", count)
	}
}

func TestNew_UnsupportedDriver(t *testing.T) {
	config.ResetForTest()

	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "test.toml")
	content := `
[server]
host = "127.0.0.1"
port = 7330
[db]
driver = "mysql"
dsn = "test"
[admin]
initial_username = "admin"
initial_password = "admin123"
[jwt]
access_signing_key = "test-access-key-min-32-characters!!"
refresh_signing_key = "test-refresh-key-min-32-characters!!"
[delivery]
request_timeout = "5s"
`
	if err := os.WriteFile(tomlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := config.Load(tomlPath)
	if err != nil {
		t.Fatalf("config.Load error: %v", err)
	}

	_, err = New("test")
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestMigratePasswordColumn(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(allModels...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	database := &Database{db: db}
	if err := database.migratePasswordColumn(); err != nil {
		t.Fatalf("migratePasswordColumn error: %v", err)
	}
}

func TestMigrateQIDPrefix(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(allModels...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	database := &Database{db: db}
	if err := database.migrateQIDPrefix(); err != nil {
		t.Fatalf("migrateQIDPrefix error: %v", err)
	}
}

func TestDropStaleChannelsTable(t *testing.T) {
	t.Run("no stale table", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		if err := db.AutoMigrate(allModels...); err != nil {
			t.Fatalf("migrate: %v", err)
		}

		database := &Database{db: db}
		if err := database.dropStaleChannelsTable(); err != nil {
			t.Fatalf("dropStaleChannelsTable error: %v", err)
		}
	})

	t.Run("drops stale table", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		if err := db.AutoMigrate(allModels...); err != nil {
			t.Fatalf("migrate: %v", err)
		}

		// Create a stale "channels" table
		db.Exec("CREATE TABLE channels (id INTEGER PRIMARY KEY, name TEXT)")

		database := &Database{db: db}
		if err := database.dropStaleChannelsTable(); err != nil {
			t.Fatalf("dropStaleChannelsTable error: %v", err)
		}

		if db.Migrator().HasTable("channels") {
			t.Error("expected stale channels table to be dropped")
		}
	})
}

func TestSetupTestDBWithRepos(t *testing.T) {
	db, userRepo, tokenRepo, postRepo, deliveryRepo := SetupTestDBWithRepos(t)
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	if userRepo == nil {
		t.Fatal("expected non-nil userRepo")
	}
	if tokenRepo == nil {
		t.Fatal("expected non-nil tokenRepo")
	}
	if postRepo == nil {
		t.Fatal("expected non-nil postRepo")
	}
	if deliveryRepo == nil {
		t.Fatal("expected non-nil deliveryRepo")
	}
}

func TestUserExists(t *testing.T) {
	database, err := NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}
	defer func() { _ = database.Close() }()

	exists, err := database.userExists("admin")
	if err != nil {
		t.Fatalf("userExists error: %v", err)
	}
	if exists {
		t.Error("expected user to not exist initially")
	}
}

func TestCreateUser(t *testing.T) {
	database, err := NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}
	defer func() { _ = database.Close() }()

	u, err := makeUser("admin@example.com", "admin", "password123", nil, nil, 16)
	if err != nil {
		t.Fatalf("makeUser error: %v", err)
	}
	if err := database.createUser(u); err != nil {
		t.Fatalf("createUser error: %v", err)
	}

	exists, err := database.userExists("admin")
	if err != nil {
		t.Fatalf("userExists error: %v", err)
	}
	if !exists {
		t.Error("expected user to exist after creation")
	}
}

func TestEnsureSQLiteDirCreatesParentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	err := ensureSQLiteDir("file:./data/markpost.db?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		t.Fatalf("ensureSQLiteDir error: %v", err)
	}

	info, err := os.Stat(filepath.Join(tempDir, "data"))
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected data to be a directory")
	}
}

func TestEnsureSQLiteDirMemoryDSN(t *testing.T) {
	if err := ensureSQLiteDir(":memory:"); err != nil {
		t.Fatalf("unexpected error for :memory: DSN: %v", err)
	}
}

func TestEnsureSQLiteDirFileMemoryDSN(t *testing.T) {
	if err := ensureSQLiteDir("file::memory:?cache=shared"); err != nil {
		t.Fatalf("unexpected error for file::memory: DSN: %v", err)
	}
}

func TestEnsureSQLiteDirBareFilename(t *testing.T) {
	if err := ensureSQLiteDir("markpost.db"); err != nil {
		t.Fatalf("unexpected error for bare filename DSN: %v", err)
	}
}
