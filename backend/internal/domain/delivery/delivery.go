package delivery

import (
	"errors"
	"time"

	"markpost/internal/domain/user"
)

type ChannelKind string

const (
	ChannelKindFeishu ChannelKind = "feishu"
)

type Channel struct {
	ID         int          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int          `json:"user_id" gorm:"index;not null;column:user_id"`
	User       user.User    `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Kind       ChannelKind  `json:"kind" gorm:"not null;size:32"`
	Name       string       `json:"name" gorm:"not null;default:''"`
	Enabled    bool         `json:"enabled" gorm:"not null;default:true"`
	WebhookURL string       `json:"webhook_url" gorm:"not null;type:text;column:webhook_url"`
	Keywords   string       `json:"keywords" gorm:"not null;type:text;default:''"`
	CreatedAt  time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

var ErrNotFound = errors.New("record not found")

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
