package infra

import (
	"context"
	"fmt"

	"markpost/internal/domain"
	"markpost/internal/domain/settings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettingsRepository provides runtime settings data access (MRFC
// 2026-08-23-github-login-vip-grant-strategy).
type SettingsRepository struct {
	db *gorm.DB
}

// NewSettingsRepository creates a new SettingsRepository instance.
func NewSettingsRepository(db *gorm.DB) settings.Repository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting row by key.
func (r *SettingsRepository) Get(ctx context.Context, key string) (*settings.Setting, error) {
	s, err := findFirst[settings.Setting](ctx, r.db.Where("key = ?", key), domain.ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("Get: %w", err)
	}
	return s, nil
}

// GetAll retrieves every setting row, ordered by key.
func (r *SettingsRepository) GetAll(ctx context.Context) ([]settings.Setting, error) {
	var rows []settings.Setting
	if err := r.db.WithContext(ctx).Order("key asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("GetAll: %w", err)
	}
	return rows, nil
}

// Set upserts a setting row, recording the acting admin.
func (r *SettingsRepository) Set(ctx context.Context, key string, value settings.SettingValue, updatedBy int) error {
	by := int64(updatedBy)
	s := settings.Setting{Key: key, Value: value, UpdatedBy: &by}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at"}),
	}).Create(&s).Error
	if err != nil {
		return fmt.Errorf("Set: %w", err)
	}
	return nil
}

// VIPStrategyEnabled reports whether the GitHub-login VIP grant strategy is on.
func (r *SettingsRepository) VIPStrategyEnabled(ctx context.Context) (bool, error) {
	s, err := r.Get(ctx, settings.KeyVIP)
	if err != nil {
		return false, err
	}
	return s.Value.Enabled, nil
}

// VIPRetentionDays returns the VIP-class retention default. domain.ErrNotFound
// means no row exists; a row with nil Days means no default set — callers map
// both to "follow the global config".
func (r *SettingsRepository) VIPRetentionDays(ctx context.Context) (*int, error) {
	s, err := r.Get(ctx, settings.KeyVIPRetention)
	if err != nil {
		return nil, err
	}
	return s.Value.Days, nil
}
