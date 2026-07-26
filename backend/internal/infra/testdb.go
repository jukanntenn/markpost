package infra

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// testPGContainer is the shared postgres container for the test package.
// Started lazily in SetupTestDB; reused across tests in the same package via
// TRUNCATE cleanup between tests.
var testPGContainer *tcpostgres.PostgresContainer

// SetupTestDB starts (or reuses) a postgres testcontainer, applies migrations,
// and returns a connected *gorm.DB with cleanup that truncates all data between tests.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	if testPGContainer == nil {
		startContainer(ctx, t)
	}

	dsn, err := testPGContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("testcontainer dsn: %v", err)
	}

	// Retry migrations: Docker Desktop on WSL2 can briefly reset connections
	// right after the container starts, even though the port is reachable.
	var migrateErr error
	for range 60 {
		migrateErr = MigrateUp(dsn)
		if migrateErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if migrateErr != nil {
		t.Fatalf("apply migrations: %v", migrateErr)
	}

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	t.Cleanup(func() {
		db.Exec("TRUNCATE users, posts, refresh_tokens, token_blacklist, delivery_channels, delivery_attempts, delivery_history RESTART IDENTITY CASCADE")
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func startContainer(ctx context.Context, t *testing.T) {
	c, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("markpost_test"),
		tcpostgres.WithUsername("markpost"),
		tcpostgres.WithPassword("markpost"),
	)
	if err != nil {
		if os.Getenv("TESTCONTAINERS_SKIP") != "" {
			t.Skipf("testcontainers unavailable: %v", err)
		}
		t.Fatalf("start postgres container: %v", err)
	}
	testPGContainer = c
}

// SetupTestDBWithRepos mirrors the old helper's signature so callers are unchanged.
func SetupTestDBWithRepos(t *testing.T) (*gorm.DB, user.Repository, user.TokenRepository, post.Repository, delivery.Repository) {
	t.Helper()
	db := SetupTestDB(t)
	return db,
		NewUserRepository(db, 16),
		NewTokenRepository(db),
		NewPostRepository(db),
		NewDeliveryChannelRepository(db)
}

// keep the package compiling even if unused after refactor
var _ = fmt.Sprintf
var _ = time.Second
