package models

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DeliveryChannelKind string

const (
	DeliveryChannelKindFeishu DeliveryChannelKind = "feishu"
)

type DeliveryChannel struct {
	ID         int                `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int                `json:"user_id" gorm:"index;not null;column:user_id"`
	User       User               `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Kind       DeliveryChannelKind `json:"kind" gorm:"not null;size:32"`
	Name       string             `json:"name" gorm:"not null;default:''"`
	Enabled    bool               `json:"enabled" gorm:"not null;default:true"`
	WebhookURL string             `json:"webhook_url" gorm:"not null;type:text;column:webhook_url"`
	Keywords   string             `json:"keywords" gorm:"not null;type:text;default:''"`
	CreatedAt  time.Time          `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time          `json:"updated_at" gorm:"autoUpdateTime"`
}

func (model *DeliveryChannel) Create(database *Database) error {
	db := database.DB()
	if err := db.Create(model).Error; err != nil {
		return fmt.Errorf("DeliveryChannel.Create: %w", err)
	}
	return nil
}

func (model *DeliveryChannel) Update(database *Database) error {
	db := database.DB()
	updates := map[string]any{
		"kind":        model.Kind,
		"name":        model.Name,
		"enabled":     model.Enabled,
		"webhook_url": model.WebhookURL,
		"keywords":    model.Keywords,
	}
	if err := db.Model(&model).Updates(updates).Error; err != nil {
		return fmt.Errorf("DeliveryChannel.Update: %w", err)
	}
	return nil
}

func GetDeliveryChannel(database *Database, query map[string]any) (*DeliveryChannel, error) {
	db := database.DB()

	var channel DeliveryChannel
	err := db.Take(&channel, query).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetDeliveryChannel: %w", err)
	}

	return &channel, nil
}

func GetDeliveryChannels(database *Database, query map[string]any) ([]DeliveryChannel, error) {
	db := database.DB()

	var channels []DeliveryChannel
	err := db.Where(query).Order("id asc").Find(&channels).Error
	if err != nil {
		return nil, fmt.Errorf("GetDeliveryChannels: %w", err)
	}

	return channels, nil
}

func DeleteDeliveryChannels(database *Database, query map[string]any) (int64, error) {
	db := database.DB()

	tx := db.Where(query).Delete(&DeliveryChannel{})
	if tx.Error != nil {
		return 0, fmt.Errorf("DeleteDeliveryChannels: %w", tx.Error)
	}

	return tx.RowsAffected, nil
}
