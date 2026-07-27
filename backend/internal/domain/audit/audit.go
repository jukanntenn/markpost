// Package audit defines the audit log domain model.
package audit

import (
	"context"
	"encoding/json"
	"time"
)

// Log is an immutable record of an admin write operation.
type Log struct {
	ID         int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	ActorID    int             `json:"actor_id" gorm:"not null;index"`
	Action     string          `json:"action" gorm:"not null;size:64"`
	TargetType string          `json:"target_type" gorm:"not null;size:32"`
	TargetID   string          `json:"target_id" gorm:"size:64"`
	Metadata   json.RawMessage `json:"metadata" gorm:"not null;type:jsonb;default:'{}'::jsonb"`
	IP         string          `json:"ip" gorm:"size:45"`
	CreatedAt  time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

func (Log) TableName() string { return "audit_logs" }

// Entry is the input for recording an audit log.
type Entry struct {
	ActorID    int
	Action     string // e.g. "user.delete", "post.delete", "channel.update"
	TargetType string // "user" | "post" | "channel" | "session"
	TargetID   string
	Metadata   map[string]any
	IP         string
}

// Repository defines the interface for audit log data access.
type Repository interface {
	Record(ctx context.Context, e Entry) error
	List(ctx context.Context, offset, limit int) ([]Log, int64, error)
}
