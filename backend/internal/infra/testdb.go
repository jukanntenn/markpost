package infra

import (
	"testing"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testModels = []any{
	&user.User{},
	&user.RefreshToken{},
	&user.TokenBlacklist{},
	&post.Post{},
	&delivery.Channel{},
}

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(testModels...); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})
	return db
}

func SetupTestDBWithRepos(t *testing.T) (*gorm.DB, user.Repository, user.TokenRepository, post.Repository, delivery.Repository) {
	t.Helper()
	db := SetupTestDB(t)
	return db,
		NewUserRepository(db, 16),
		NewTokenRepository(db),
		NewPostRepository(db),
		NewDeliveryChannelRepository(db)
}
