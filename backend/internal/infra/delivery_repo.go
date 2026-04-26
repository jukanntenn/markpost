// Package infra provides infrastructure layer implementations.
package infra

import (
	"context"
	"errors"

	"markpost/internal/domain/delivery"

	"gorm.io/gorm"
)

// DeliveryChannelRepository provides delivery channel data access operations.
type DeliveryChannelRepository struct {
	db *gorm.DB
}

// NewDeliveryChannelRepository creates a new DeliveryChannelRepository instance.
func NewDeliveryChannelRepository(db *gorm.DB) delivery.Repository {
	return &DeliveryChannelRepository{db: db}
}

// GetByID retrieves a delivery channel by its ID.
func (r *DeliveryChannelRepository) GetByID(ctx context.Context, id int) (*delivery.Channel, error) {
	var c delivery.Channel
	err := r.db.WithContext(ctx).First(&c, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, delivery.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// GetByUserID retrieves all delivery channels for a user.
func (r *DeliveryChannelRepository) GetByUserID(ctx context.Context, userID int) ([]delivery.Channel, error) {
	var channels []delivery.Channel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id asc").Find(&channels).Error
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// GetByIDAndUserID retrieves a delivery channel by ID and user ID.
func (r *DeliveryChannelRepository) GetByIDAndUserID(ctx context.Context, id int, userID int) (*delivery.Channel, error) {
	var c delivery.Channel
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, delivery.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// Create creates a new delivery channel.
func (r *DeliveryChannelRepository) Create(ctx context.Context, channel *delivery.Channel) error {
	return r.db.WithContext(ctx).Create(channel).Error
}

// Update updates an existing delivery channel.
func (r *DeliveryChannelRepository) Update(ctx context.Context, channel *delivery.Channel) error {
	updates := map[string]any{
		"kind":        channel.Kind,
		"name":        channel.Name,
		"enabled":     channel.Enabled,
		"webhook_url": channel.WebhookURL,
		"keywords":    channel.Keywords,
	}
	return r.db.WithContext(ctx).Model(channel).Updates(updates).Error
}

// DeleteByID deletes a delivery channel by its ID.
func (r *DeliveryChannelRepository) DeleteByID(ctx context.Context, id int) (int64, error) {
	tx := r.db.WithContext(ctx).Delete(&delivery.Channel{}, id)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}

// DeleteByIDAndUserID deletes a delivery channel by ID and user ID.
func (r *DeliveryChannelRepository) DeleteByIDAndUserID(ctx context.Context, id int, userID int) (int64, error) {
	tx := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&delivery.Channel{})
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}

// DeleteByUserID deletes all delivery channels for a user.
func (r *DeliveryChannelRepository) DeleteByUserID(ctx context.Context, userID int) (int64, error) {
	tx := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&delivery.Channel{})
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}

// ListAll retrieves all delivery channels with pagination.
func (r *DeliveryChannelRepository) ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, error) {
	var channels []delivery.Channel
	err := r.db.WithContext(ctx).Order("id asc").Offset(offset).Limit(limit).Find(&channels).Error
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// CountAll returns the total number of delivery channels.
func (r *DeliveryChannelRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&delivery.Channel{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
