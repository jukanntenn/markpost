// Package delivery provides domain models for delivery channels.
package delivery

import (
	"errors"
	"time"

	"markpost/internal/domain/user"
)

// ChannelKind represents the type of delivery channel.
type ChannelKind string

const (
	// ChannelKindFeishu represents a Feishu/Lark delivery channel.
	ChannelKindFeishu ChannelKind = "feishu"
)

// Channel represents a delivery channel configuration.
type Channel struct {
	ID         int         `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int         `json:"user_id" gorm:"index;not null;column:user_id"`
	User       user.User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Kind       ChannelKind `json:"kind" gorm:"not null;size:32"`
	Name       string      `json:"name" gorm:"not null;default:''"`
	Enabled    bool        `json:"enabled" gorm:"not null;default:true"`
	WebhookURL string      `json:"webhook_url" gorm:"not null;type:text;column:webhook_url"`
	Keywords   string      `json:"keywords" gorm:"not null;type:text;default:''"`
	CreatedAt  time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
}

// ErrNotFound is returned when a record is not found.
var ErrNotFound = errors.New("record not found")

// IsNotFound checks if an error is ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
