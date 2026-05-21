package infra

import (
	"context"
	"errors"
	"fmt"

	"markpost/internal/domain"

	"gorm.io/gorm"
)

func mapNotFound(err error, notFound error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound
	}
	return err
}

func findFirst[T any](ctx context.Context, query *gorm.DB, notFound error) (*T, error) {
	var result T
	if err := query.WithContext(ctx).First(&result).Error; err != nil {
		return nil, mapNotFound(err, notFound)
	}
	return &result, nil
}

func existsBy[T any](ctx context.Context, db *gorm.DB, field string, value any, label string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(new(T)).Where(field+" = ?", value).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("%s: %w", label, err)
	}
	return count > 0, nil
}

func findMany[T any](ctx context.Context, query *gorm.DB, offset, limit int, label string) ([]T, error) {
	var results []T
	if err := query.WithContext(ctx).Offset(offset).Limit(limit).Find(&results).Error; err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return results, nil
}

func findAll[T any](ctx context.Context, query *gorm.DB, label string) ([]T, error) {
	var results []T
	if err := query.WithContext(ctx).Find(&results).Error; err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return results, nil
}

func countQuery(ctx context.Context, query *gorm.DB, label string) (int64, error) {
	var count int64
	if err := query.WithContext(ctx).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("%s: %w", label, err)
	}
	return count, nil
}

func updateByID[T any](ctx context.Context, db *gorm.DB, id int, updates map[string]any, label string) error {
	result := db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("%s: %w", label, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%s: %w", label, domain.ErrNotFound)
	}
	return nil
}

func deleteWhere[T any](ctx context.Context, query *gorm.DB) (int64, error) {
	tx := query.WithContext(ctx).Delete(new(T))
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}
