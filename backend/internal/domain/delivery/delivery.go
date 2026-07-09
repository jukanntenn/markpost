// Package delivery defines the domain models for message delivery channels.
package delivery

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
)

// ChannelKind represents the type of delivery channel.
type ChannelKind string

// Supported delivery channel kinds.
const (
	ChannelKindFeishu ChannelKind = "feishu"
)

var validChannelKinds = map[ChannelKind]bool{
	ChannelKindFeishu: true,
}

// TableName returns the database table name for Channel.
func (Channel) TableName() string { return "delivery_channels" }

// IsValid reports whether the ChannelKind is a recognized value.
func (k ChannelKind) IsValid() bool {
	return validChannelKinds[k]
}

// Status is the lifecycle state of a delivery attempt or history row. It is an
// int8 (not a string) so the column maps to the smallest integer type on each
// dialect via GORM's size-based logic: tinyint on MySQL (1B), smallint on
// Postgres (2B, its floor), integer on SQLite (value-width). StatusPending is 0
// so the database default (default:0) lands on the pending state. The numeric
// mapping is append-only forever: inserting a state in the middle would
// renumber every later state and corrupt existing rows.
type Status int8

// Delivery attempt/history statuses. Order is significant and must stay
// append-only.
const (
	StatusPending   Status = 0 // due or in-flight
	StatusDelivered Status = 1 // terminal — a send succeeded
	StatusFailed    Status = 2 // terminal — retry sequence exhausted
	StatusExpired   Status = 3 // terminal — expiry wall passed
)

// IsTerminal reports whether the status is a terminal state (no further
// transitions).
func (s Status) IsTerminal() bool {
	return s == StatusDelivered || s == StatusFailed || s == StatusExpired
}

// TableName returns the database table name for Attempt.
func (Attempt) TableName() string { return "delivery_attempts" }

// Attempt is a single in-flight delivery job: one post to one channel, retried
// with backoff until it reaches a terminal state. A row lives only while
// delivery is in progress (at most the expiry wall); on any terminal state it
// is archived to History and deleted in the same transaction.
type Attempt struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int       `json:"user_id" gorm:"not null;column:user_id;index"`
	PostID    int       `json:"post_id" gorm:"not null;column:post_id;index"`
	ChannelID int       `json:"channel_id" gorm:"not null;column:channel_id;index"`
	Status    Status    `json:"status" gorm:"not null;default:0"`
	Attempts  int       `json:"attempts" gorm:"not null;default:0"`
	NextAt    int64     `json:"next_at" gorm:"not null"` // epoch ms; when the next attempt may run
	LastError string    `json:"last_error" gorm:"not null;type:text;default:''"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"` // drives the expiry wall
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	User    user.User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Post    post.Post `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Channel Channel   `json:"-" gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE"`
}

// TableName returns the database table name for History.
func (History) TableName() string { return "delivery_history" }

// History is the cold archive of a delivery's terminal outcome, retained for
// the history-retention window (7 days). All foreign keys are nullable with
// ON DELETE SET NULL so deleting a user/post/channel does not cascade-delete
// the history row (which could lock a large row set) — the reference is
// nulled and the row is preserved as an anonymous record.
type History struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    *int      `json:"user_id" gorm:"column:user_id;index"`       // nullable; ON DELETE SET NULL
	PostID    *int      `json:"post_id" gorm:"column:post_id;index"`       // nullable; ON DELETE SET NULL
	ChannelID *int      `json:"channel_id" gorm:"column:channel_id;index"` // nullable; ON DELETE SET NULL
	Status    Status    `json:"status" gorm:"not null"`                    // delivered | failed | expired
	LastError string    `json:"last_error" gorm:"not null;type:text;default:''"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	User    *user.User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
	Post    *post.Post `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:SET NULL"`
	Channel *Channel   `json:"-" gorm:"foreignKey:ChannelID;constraint:OnDelete:SET NULL"`
}

// HistoryRow is the read projection of a delivery_history row joined to its
// post/channel/user at read time (the spec's normalization rule: titles and
// names are JOINed, not snapshotted). The nullable pointers reflect
// ON DELETE SET NULL: a nil field means the referenced row was deleted.
type HistoryRow struct {
	ID          int64     `json:"id" gorm:"column:id"`
	Status      Status    `json:"status" gorm:"column:status"`
	LastError   string    `json:"last_error" gorm:"column:last_error"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	PostTitle   *string   `json:"post_title" gorm:"column:post_title"`
	PostQID     *string   `json:"post_qid" gorm:"column:post_qid"`
	ChannelName *string   `json:"channel_name" gorm:"column:channel_name"`
	Username    *string   `json:"username" gorm:"column:username"`
}

// FeishuConfiguration holds the configuration for a Feishu delivery channel.
type FeishuConfiguration struct {
	WebhookURL  string `json:"webhook_url"`
	CardLinkURL string `json:"card_link_url"`
}

// ChannelConfiguration stores arbitrary key-value pairs for channel settings.
type ChannelConfiguration map[string]any

// Feishu parses and returns the Feishu-specific configuration.
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

// Value implements the driver.Valuer interface for database serialization.
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

// Scan implements the sql.Scanner interface for database deserialization.
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

// Channel represents a delivery channel linked to a user.
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
