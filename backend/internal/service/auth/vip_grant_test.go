package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"markpost/internal/domain/settings"
	"markpost/internal/domain/user"
	"markpost/internal/infra"
)

var errStrategyRead = errors.New("strategy read failed")

type failingVIPStrategy struct{}

func (failingVIPStrategy) VIPStrategyEnabled(context.Context) (bool, error) {
	return false, errStrategyRead
}

func (failingVIPStrategy) VIPRetentionDays(context.Context) (*int, error) {
	return nil, errStrategyRead
}

// TestGrantVIPForGitHubLogin covers the grant strategy semantics (MRFC
// 2026-08-23-github-login-vip-grant-strategy): grant while enabled, no write
// in either direction while disabled, fail toward not-granting on a read
// error, and no-op when unwired.
func TestGrantVIPForGitHubLogin(t *testing.T) {
	ctx := context.Background()

	type fixture struct {
		svc     *Service
		users   user.Repository
		setting settings.Repository
	}
	setup := func(t *testing.T) *fixture {
		db := infra.SetupTestDB(t)
		users := infra.NewUserRepository(db, 16)
		setting := infra.NewSettingsRepository(db)
		// Seed explicitly: an earlier test's TRUNCATE may have removed the
		// migration seed, and the hook must not depend on residue.
		if err := setting.Set(ctx, settings.KeyVIP, settings.SettingValue{Enabled: true}, 0); err != nil {
			t.Fatalf("seed vip strategy: %v", err)
		}
		tokens := infra.NewTokenRepository(db)
		jwt := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour)
		svc := NewService(users, tokens, nil, jwt, "markpost", "testpassword").WithVIPStrategy(setting)
		return &fixture{svc: svc, users: users, setting: setting}
	}

	t.Run("grants vip to a non-vip user while enabled", func(t *testing.T) {
		f := setup(t)
		u, err := f.users.Create(ctx, "grant@example.com", "grantuser", "pass")
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		f.svc.grantVIPForGitHubLogin(ctx, u)
		if !u.VIP {
			t.Fatal("expected in-memory user flagged VIP")
		}
		got, _ := f.users.GetByID(ctx, u.ID)
		if !got.VIP {
			t.Error("expected persisted VIP")
		}
	})

	t.Run("skips the write for an already-vip user", func(t *testing.T) {
		f := setup(t)
		u, _ := f.users.Create(ctx, "vip@example.com", "vipuser", "pass")
		if err := f.users.SetUserVIP(ctx, u.ID, true, nil); err != nil {
			t.Fatalf("seed vip: %v", err)
		}
		u.VIP = true
		f.svc.grantVIPForGitHubLogin(ctx, u)
		got, _ := f.users.GetByID(ctx, u.ID)
		if !got.VIP {
			t.Error("expected VIP retained")
		}
	})

	t.Run("leaves vip untouched in both directions while disabled", func(t *testing.T) {
		f := setup(t)
		if err := f.setting.Set(ctx, settings.KeyVIP, settings.SettingValue{Enabled: false}, 1); err != nil {
			t.Fatalf("disable strategy: %v", err)
		}

		plain, _ := f.users.Create(ctx, "closed@example.com", "closeduser", "pass")
		f.svc.grantVIPForGitHubLogin(ctx, plain)
		if plain.VIP {
			t.Error("expected no grant while disabled")
		}

		if err := f.users.SetUserVIP(ctx, plain.ID, true, nil); err != nil {
			t.Fatalf("seed vip: %v", err)
		}
		plain.VIP = true
		f.svc.grantVIPForGitHubLogin(ctx, plain)
		got, _ := f.users.GetByID(ctx, plain.ID)
		if !got.VIP {
			t.Error("expected existing vip NOT revoked while disabled")
		}
	})

	t.Run("read failure grants nothing and does not panic", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		users := infra.NewUserRepository(db, 16)
		tokens := infra.NewTokenRepository(db)
		jwt := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour)
		svc := NewService(users, tokens, nil, jwt, "markpost", "testpassword").WithVIPStrategy(failingVIPStrategy{})
		u, _ := users.Create(ctx, "fail@example.com", "failuser", "pass")
		svc.grantVIPForGitHubLogin(ctx, u)
		if u.VIP {
			t.Error("expected no grant on strategy read failure")
		}
	})

	t.Run("unwired strategy never grants", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		users := infra.NewUserRepository(db, 16)
		tokens := infra.NewTokenRepository(db)
		jwt := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour)
		svc := NewService(users, tokens, nil, jwt, "markpost", "testpassword")
		u, _ := users.Create(ctx, "nowire@example.com", "nowireuser", "pass")
		svc.grantVIPForGitHubLogin(ctx, u)
		if u.VIP {
			t.Error("expected no grant without a wired strategy")
		}
	})
}
