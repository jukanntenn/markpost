package delivery

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"markpost/internal/domain/user"
)

type ChannelKind string

const (
	ChannelKindFeishu ChannelKind = "feishu"
)

var validChannelKinds = map[ChannelKind]bool{
	ChannelKindFeishu: true,
}

func (Channel) TableName() string { return "delivery_channels" }

func (k ChannelKind) IsValid() bool {
	return validChannelKinds[k]
}

type FeishuConfiguration struct {
	WebhookURL  string `json:"webhook_url"`
	CardLinkURL string `json:"card_link_url"`
}

type ChannelConfiguration map[string]any

func (c ChannelConfiguration) Feishu() FeishuConfiguration {
	return FeishuConfiguration{
		WebhookURL:  c.stringField("webhook_url"),
		CardLinkURL: c.stringField("card_link_url"),
	}
}

func (c ChannelConfiguration) stringField(key string) string {
	v, ok := c[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func (c ChannelConfiguration) Value() (driver.Value, error) {
	if c == nil {
		return "{}", nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal channel configuration: %w", err)
	}
	return string(b), nil
}

func (c *ChannelConfiguration) Scan(value any) error {
	if value == nil {
		*c = ChannelConfiguration{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("cannot scan %T into ChannelConfiguration", value)
	}

	if len(bytes) == 0 {
		*c = ChannelConfiguration{}
		return nil
	}

	var result ChannelConfiguration
	if err := json.Unmarshal(bytes, &result); err != nil {
		return fmt.Errorf("unmarshal channel configuration: %w", err)
	}
	*c = result
	return nil
}

type Channel struct {
	ID            int                  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int                  `json:"user_id" gorm:"index;not null;column:user_id"`
	User          user.User            `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Kind          ChannelKind          `json:"kind" gorm:"not null;size:32"`
	Name          string               `json:"name" gorm:"not null;default:''"`
	Enabled       bool                 `json:"enabled" gorm:"not null;default:true"`
	Configuration ChannelConfiguration `json:"configuration" gorm:"not null;type:text;column:configuration;default:'{}'"`
	Keywords      string               `json:"keywords" gorm:"not null;type:text;default:''"`
	CreatedAt     time.Time            `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time            `json:"updated_at" gorm:"autoUpdateTime"`
}
