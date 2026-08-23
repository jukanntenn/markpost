package infra

import (
	"context"
	"errors"
	"testing"

	"markpost/internal/domain"
	"markpost/internal/domain/settings"
)

func TestSettingsRepository(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	t.Run("VIPStrategyEnabled reads the stored value", func(t *testing.T) {
		if err := repo.Set(ctx, settings.KeyVIP, settings.SettingValue{Enabled: true}, 0); err != nil {
			t.Fatalf("seed vip strategy: %v", err)
		}
		enabled, err := repo.VIPStrategyEnabled(ctx)
		if err != nil {
			t.Fatalf("VIPStrategyEnabled: %v", err)
		}
		if !enabled {
			t.Error("expected the stored vip strategy to read enabled")
		}
	})

	t.Run("Set upserts and VIPStrategyEnabled follows", func(t *testing.T) {
		if err := repo.Set(ctx, settings.KeyVIP, settings.SettingValue{Enabled: false}, 1); err != nil {
			t.Fatalf("Set(false): %v", err)
		}
		enabled, err := repo.VIPStrategyEnabled(ctx)
		if err != nil {
			t.Fatalf("VIPStrategyEnabled: %v", err)
		}
		if enabled {
			t.Error("expected vip strategy disabled after Set(false)")
		}
		if err := repo.Set(ctx, settings.KeyVIP, settings.SettingValue{Enabled: true}, 2); err != nil {
			t.Fatalf("Set(true): %v", err)
		}
		s, err := repo.Get(ctx, settings.KeyVIP)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !s.Value.Enabled {
			t.Error("expected vip strategy re-enabled")
		}
		if s.UpdatedBy == nil || *s.UpdatedBy != 2 {
			t.Errorf("expected updated_by=2, got %v", s.UpdatedBy)
		}
	})

	t.Run("GetAll lists rows ordered by key", func(t *testing.T) {
		if err := repo.Set(ctx, "aaa", settings.SettingValue{Enabled: false}, 1); err != nil {
			t.Fatalf("Set(aaa): %v", err)
		}
		rows, err := repo.GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll: %v", err)
		}
		if len(rows) != 2 || rows[0].Key != "aaa" || rows[1].Key != settings.KeyVIP {
			t.Errorf("expected [aaa vip], got %v", rows)
		}
	})

	t.Run("unknown key returns ErrNotFound", func(t *testing.T) {
		_, err := repo.Get(ctx, "nope")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}
