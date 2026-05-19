package infra

import (
	"context"
	"fmt"

	"markpost/internal/domain"
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

func scopedByIDAndUserID(db *gorm.DB, id, userID int) *gorm.DB {
	return db.Where("id = ? AND user_id = ?", id, userID)
}

// GetByUserID retrieves all delivery channels for a user.
func (r *DeliveryChannelRepository) GetByUserID(ctx context.Context, userID int) ([]delivery.Channel, error) {
	return findAll[delivery.Channel](ctx, r.db.Where("user_id = ?", userID).Order("id asc"), "GetByUserID")
}

// GetByIDAndUserID retrieves a delivery channel by ID and user ID.
func (r *DeliveryChannelRepository) GetByIDAndUserID(ctx context.Context, id int, userID int) (*delivery.Channel, error) {
	ch, err := findFirst[delivery.Channel](ctx, scopedByIDAndUserID(r.db, id, userID), domain.ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("GetByIDAndUserID: %w", err)
	}
	return ch, nil
}

// Create creates a new delivery channel.
func (r *DeliveryChannelRepository) Create(ctx context.Context, channel *delivery.Channel) error {
	if err := r.db.WithContext(ctx).Create(channel).Error; err != nil {
		return fmt.Errorf("Create: %w", err)
	}
	return nil
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
	return updateByID[delivery.Channel](ctx, r.db, channel.ID, updates, "Update")
}

// DeleteByIDAndUserID deletes a delivery channel by ID and user ID.
func (r *DeliveryChannelRepository) DeleteByIDAndUserID(ctx context.Context, id int, userID int) (int64, error) {
	n, err := deleteWhere[delivery.Channel](ctx, scopedByIDAndUserID(r.db, id, userID))
	if err != nil {
		return 0, fmt.Errorf("DeleteByIDAndUserID: %w", err)
	}
	return n, nil
}

// ListAll retrieves all delivery channels with pagination.
func (r *DeliveryChannelRepository) ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, error) {
	return findMany[delivery.Channel](ctx, r.db.Order("id asc"), offset, limit, "ListAll")
}

// CountAll returns the total number of delivery channels.
func (r *DeliveryChannelRepository) CountAll(ctx context.Context) (int64, error) {
	return countQuery(ctx, r.db.Model(&delivery.Channel{}), "CountAll")
}
