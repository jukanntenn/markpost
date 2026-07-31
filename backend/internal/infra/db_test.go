package infra

import (
	"context"
	"testing"
	"time"
)

// TestNew_PinsSessionTimezoneAndNowFunc exercises the production infra.New path
// (not the test-helper gorm.Open) with a non-default timezone, asserting both
// that the Postgres session timezone is pinned and that GORM's NowFunc stamps
// created_at in the configured zone.
func TestNew_PinsSessionTimezoneAndNowFunc(t *testing.T) {
	// SetupTestDB ensures the shared container exists; we only need its DSN.
	_ = SetupTestDB(t)
	ctx := context.Background()
	dsn, err := testPGContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	const tz = "Asia/Shanghai"
	loc, _ := time.LoadLocation(tz)
	savedLocal := time.Local
	t.Cleanup(func() { time.Local = savedLocal })

	dbInst, err := New(dsn, tz)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		dbInst.DB().Exec("TRUNCATE users, posts RESTART IDENTITY CASCADE")
		sqlDB, _ := dbInst.DB().DB()
		_ = sqlDB.Close()
	})

	// (a) Postgres session timezone is pinned to the configured zone.
	var sessionTZ string
	if err := dbInst.DB().Raw("SHOW TIME ZONE").Scan(&sessionTZ).Error; err != nil {
		t.Fatalf("SHOW TIME ZONE: %v", err)
	}
	if sessionTZ != tz {
		t.Errorf("session timezone = %q, want %q", sessionTZ, tz)
	}

	// (b) NowFunc stamps created_at in the configured zone. A post created now
	// should read back with its Location set to the configured zone (same
	// instant, local wall-clock representation).
	uid := createTestUser(t, dbInst.DB())
	repo := NewPostRepository(dbInst.DB())
	p, err := repo.Create(ctx, "tz-probe", "body", uid)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByQID(ctx, p.QID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.CreatedAt.Location().String() != loc.String() {
		t.Errorf("created_at location = %q, want %q", got.CreatedAt.Location(), loc)
	}
}

// TestNew_RejectsUnknownTimezone ensures a bogus IANA name fails fast at boot
// rather than silently falling back to UTC.
func TestNew_RejectsUnknownTimezone(t *testing.T) {
	_ = SetupTestDB(t)
	ctx := context.Background()
	dsn, err := testPGContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	if _, err := New(dsn, "Not/A/Real/Zone"); err == nil {
		t.Fatal("expected error for unknown timezone, got nil")
	}
}

// TestNew_EmptyTimezoneDefaultsToUTC verifies the empty-string fallback.
func TestNew_EmptyTimezoneDefaultsToUTC(t *testing.T) {
	_ = SetupTestDB(t)
	ctx := context.Background()
	dsn, err := testPGContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	if err := MigrateUp(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	dbInst, err := New(dsn, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		dbInst.DB().Exec("TRUNCATE users, posts RESTART IDENTITY CASCADE")
		sqlDB, _ := dbInst.DB().DB()
		_ = sqlDB.Close()
	})
	var sessionTZ string
	if err := dbInst.DB().Raw("SHOW TIME ZONE").Scan(&sessionTZ).Error; err != nil {
		t.Fatalf("SHOW TIME ZONE: %v", err)
	}
	if sessionTZ != "UTC" {
		t.Errorf("session timezone = %q, want UTC", sessionTZ)
	}
}
